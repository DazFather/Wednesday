package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/DazFather/brush"
)

const usageSnip = `Use "help" for usage`

// Palette
var (
	danger  = NewPrinter(padding("x"), brush.Red)
	warn    = NewPrinter(padding("!"), brush.Yellow)
	success = NewPrinter(padding("v"), brush.Green)
	running = NewPrinter(padding(">>"), brush.Magenta)
)

func NewPrinter(prefix string, baseTone brush.ANSIColor) func(...any) {
	white := brush.New(brush.BrightWhite, brush.UseColor(baseTone))
	black := brush.New(brush.Black, brush.UseColor(baseTone+8))

	return func(v ...any) {
		var suffix string = "\n"
		if len(v) >= 2 {
			suffix = brush.Paintln(baseTone, nil, v[1:]...).String()
			v[0] = padding(v[0])
			v = v[0:1]
		}
		fmt.Printf("%s%s %s\n", white.Paint(prefix), black.Paint(v...), suffix)
	}
}

func padding(s any) string {
	return fmt.Sprint(" ", s, " ")
}

func ShowUsage() {
	var (
		magenta = brush.New(brush.BrightWhite, brush.UseColor(brush.Magenta)).Paint
		gray    = brush.New(brush.BrightBlack, nil).Embed
		cyan    = brush.New(brush.BrightCyan, nil).Paint

		f = [...]string{"settings", "port"}
		a = [...]string{"mount"}

		flagUsage = map[string]string{
			"settings": "path to the settings file. If not given default one will be used",
			"port":     "server port. If not given :8080 will be used",
		}

		argUsage = map[string]string{
			"mount": "path to the directory to build. If not given it's assumed is the current working directory",
		}

		commands = []struct {
			name  string
			desc  string
			args  []string
			flags []string
		}{
			{name: "help", desc: "Show this helpful text on screen"},
			{name: "run", flags: f[:1], desc: "Run pipeline defined on the \"Run\" settings property"},
			{name: "init", flags: f[:1], desc: "Initialize a new project on the current working directories by creating default files and directories"},
			{name: "build", flags: f[:1], args: a[:], desc: "Compose project components recursively and outputs the files on the directories specified on the project settings"},
			{name: "serve", flags: f[:], args: a[:], desc: "Build the project and launch a local static server on the built directory"},
		}
	)

	fmt.Println("\n\tWednesday usage:")
	for _, c := range commands {
		fmt.Println("\n", magenta(padding(c.name)), c.desc)
		for _, name := range c.args {
			fmt.Println(gray("  <", cyan(name), "> (optional) ", argUsage[name]))
		}
		for _, name := range c.flags {
			fmt.Println(gray("  -", cyan(name[0:1]), " | -", cyan(name), " (optional) ", flagUsage[name]))
		}
	}
}

type indentWriter struct {
	indent []byte
	end    []byte
	*strings.Builder
}

func NewIndentWriter(indent, end string, buf *strings.Builder) *indentWriter {
	if buf == nil {
		buf = new(strings.Builder)
	}

	return &indentWriter{
		indent:  []byte(indent),
		end:     []byte(end),
		Builder: buf,
	}
}

func (w *indentWriter) Write(data []byte) (n int, err error) {
	n = len(data)
	if _, err = w.Builder.Write(data); err != nil {
		return
	}

	if n > 0 && data[n-1] == '\n' {
		data = data[:n-1]
	}

	b := new(bytes.Buffer)
	if _, err = b.Write(w.indent); err == nil {
		if _, err = b.Write(endln.ReplaceAll(data, append([]byte("\n"), w.indent...))); err == nil {
			if _, err = b.Write(w.end); err == nil {
				_, err = os.Stdout.Write(b.Bytes())
			}
		}
	}
	return
}
