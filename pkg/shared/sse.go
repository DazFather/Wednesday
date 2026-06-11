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

func (h *SSEHandler) Handler(acao string, handleErr func(error), ping, retry *time.Duration) http.HandlerFunc {
	if handleErr == nil {
		handleErr = func(_ error) {}
	}

	send := func(f *http.ResponseController, w http.ResponseWriter, base string, args ...any) {
		_, err := fmt.Fprintf(w, base+"\n\n", args...)
		if err == nil {
			err = f.Flush()
		}
		if err != nil {
			handleErr(fmt.Errorf("Unable to send data over SSE: %w", err))
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", acao)

		var (
			values             = h.connect()
			flusher            = http.NewResponseController(w)
			connection, cancel = context.WithCancel(r.Context())
		)
		defer h.close(values)
		defer cancel()

		if retry != nil {
			send(flusher, w, "retry: %d", retry.Milliseconds())
		}

		var pinger *time.Ticker
		if ping != nil {
			pinger = time.NewTicker(*ping)
		} else {
			pinger = time.NewTicker(10 * time.Second)
		}
		defer pinger.Stop()

		for {
			select {
			case <-connection.Done():
				return
			case val := <-*values:
				if val != "" {
					send(flusher, w, val)
				}
			case <-pinger.C:
				send(flusher, w, "event: ping\ndata: keepalive")
			}
		}
	}
}
