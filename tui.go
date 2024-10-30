package main

import (
	"fmt"

	"github.com/DazFather/brush"
)

const usageSnip = `Use "help" for usage`

// Palette
var (
	danger  = NewPrinter(padding("x"), brush.Red)
	warn    = NewPrinter(padding("!"), brush.Yellow)
	success = NewPrinter(padding("v"), brush.Green)
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
			"settings": "path to the settings file. if not given default one will be used",
			"port":     "server port. If not given :8080 will be used",
		}

		argUsage = map[string]string{
			"mount": "path to the directory to build. if not given it's assumed is the current working directory",
		}

		commands = []struct {
			name  string
			desc  string
			args  []string
			flags []string
		}{
			{name: "help", desc: "show this helpful text on screen"},
			{name: "init", flags: f[:1], desc: "Initialize a new project on the current working directories by creating default files and directories"},
			{name: "build", flags: f[:1], args: a[:], desc: "Compose project components recursively and outputs the files on the directories specified on the project settings"},
			{name: "serve", flags: f[:], args: a[:], desc: "Build the project and launch a local static server on the built directory"},
		}
	)

	fmt.Println("\n\tWednesday usage:")
	for _, c := range commands {
		fmt.Println("\n", magenta(padding(c.name)), c.desc)
		for _, name := range c.args {
			fmt.Println(gray("  ", cyan(name), " (optional) ", argUsage[name]))
		}
		for _, name := range c.flags {
			fmt.Println(gray("  -", cyan(name[0:1]), " | -", cyan(name), " (optional) ", flagUsage[name]))
		}
	}
}
