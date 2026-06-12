package shared

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type SSEHandler struct {
	connections Smap[*chan string, struct{}]
}

func (h *SSEHandler) Broadcast(value string) {
	var conns []chan string

	h.connections.Range(func(ch *chan string, _ struct{}) bool {
		conns = append(conns, *ch)
		return true
	})

	for _, ch := range conns {
		select {
		case ch <- value:
		default:
			// optional: drop slow client
		}
	}
}

func (h *SSEHandler) connect() *chan string {
	var ch = make(chan string, 1)
	h.connections.Store(&ch, struct{}{})
	return &ch
}

func (h *SSEHandler) close(ch *chan string) {
	close(*ch)
	h.connections.Delete(ch)
}

type SSEHandlerOpt struct {
	CrossOriginHeader string
	HandleErr         func(error)
	Ping, Retry       time.Duration
}

func (opt SSEHandlerOpt) send(f *http.ResponseController, w http.ResponseWriter, base string, args ...any) {
	_, err := fmt.Fprintf(w, base+"\n\n", args...)
	if err == nil {
		err = f.Flush()
	}
	if err != nil && opt.HandleErr != nil {
		opt.HandleErr(fmt.Errorf("Unable to send data over SSE: %w", err))
	}
}

func (opt SSEHandlerOpt) pinger() *time.Ticker {
	const def = 10 * time.Second
	if opt.Ping == 0 {
		opt.Ping = def
	}
	return time.NewTicker(opt.Ping)
}

func (h *SSEHandler) Handler(opt *SSEHandlerOpt) http.HandlerFunc {
	if opt == nil {
		opt = new(SSEHandlerOpt)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		if opt.CrossOriginHeader != "" {
			w.Header().Set("Access-Control-Allow-Origin", opt.CrossOriginHeader)
		}

		var (
			values             = h.connect()
			flusher            = http.NewResponseController(w)
			connection, cancel = context.WithCancel(r.Context())
			pinger             = opt.pinger()
		)
		defer h.close(values)
		defer cancel()
		defer pinger.Stop()

		if opt.Retry != 0 {
			opt.send(flusher, w, "retry: %d", opt.Retry.Milliseconds())
		}

		for {
			select {
			case <-connection.Done():
				return
			case val := <-*values:
				if val != "" {
					opt.send(flusher, w, val)
				}
			case <-pinger.C:
				opt.send(flusher, w, "event: ping\ndata: keepalive")
			}
		}
	}
}
