package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DazFather/brush"
)

func hint(vals ...any) {
	if !settings.quiet {
		fmt.Print(vals...)
	}
}

func printlnFailed(command string, vals ...any) {
	if !brush.Disable {
		if command == "" {
			command = cutExt(filepath.Base(os.Args[0]))
		}
		if settings.arg != "" {
			command += " " + settings.arg
		}
		fmt.Printf("%s%s ", red.Paint(" x "), red.UseBgColor(brush.Red).Paint(" ", command, " "))
	} else {
		fmt.Print("[x] ")
	}
	fmt.Fprintln(os.Stderr, vals...)
}

func printlnDone(command string, vals ...any) {
	if settings.quiet {
		return
	}

	if settings.arg != "" {
		command += " " + settings.arg
	}
	if !brush.Disable {
		fmt.Printf("%s%s ", green.Paint(" v "), green.UseBgColor(brush.Green).Paint(" ", command, " "))
	} else {
		fmt.Print("[v] ")
	}
	fmt.Println(vals...)
}

var (
	// Palette
	magenta = brush.New(brush.BrightWhite, brush.UseColor(brush.Magenta))
	gray    = brush.New(brush.BrightBlack, nil)
	cyan    = brush.New(brush.BrightCyan, nil)
	green   = brush.New(brush.BrightWhite, brush.UseColor(brush.BrightGreen))
	red     = brush.New(brush.BrightWhite, brush.UseColor(brush.BrightRed))
)

func mainUsage() {
	var wed = cutExt(filepath.Base(os.Args[0]))

	fmt.Println(`
        Wednesday Usage
  `, gray.Paint(wed, ` <command> [optional] --flag`), `

Commands:

`, magenta.Paint(" build "), `Compile the project into a static site

`, magenta.Paint(" init "), `Generate a default project `, gray.Embed(`
   [`, cyan.Paint("mount"), `] directory to put the project into`), `

`, magenta.Paint(" serve "), `Build and serve your project statically via http`, gray.Embed(`
   -`, cyan.Paint("l"), ` | --`, cyan.Paint("live"), ` enable automatic rebuilding or at specified time interval
   -`, cyan.Paint("p"), ` | --`, cyan.Paint("port"), ` specify the server port. Default :8080`), `

`, magenta.Paint(" run <command> "), `Execute a user-defined pipeline of commands

`, magenta.Paint(" lib <command> "), `Manage and interact with external libraries`, gray.Embed(`
   `, cyan.Paint("search"), ` [pattern] Search for components within trusted libraries
   `, cyan.Paint("trust"), ` <url> Trust a library and download it's manifest
   `, cyan.Paint("untrust"), ` <library> Remove the manifest of the specified library
   `, cyan.Paint("use"), ` <component> Use a specific component in the current project`), `

`, magenta.Embed(" h ", gray.Paint("|"), " help "), `Show detailed usage`, gray.Embed(`
   [`, cyan.Paint("command"), `] command to obtain usage about. If not provided this message is shown`), `

`, magenta.Embed(" v ", gray.Paint("|"), " version "), "Display the current", wed, `version

`, flagsUsage())

}

func libUsage() {
	fmt.Println(`
        Usage`, magenta.Paint(" lib "), `command

Sub-commands:

`, magenta.Paint(" search "), `Search for components within trusted libraries`, gray.Embed(`
   [`, cyan.Paint("pattern"), `] regex pattern for component name matching. If '--tags' is not provided, it will match tags as well.
   -`, cyan.Paint("i"), ` enable case-insensitive pattern matching
   -`, cyan.Paint("t"), ` | --`, cyan.Paint("tags"), ` specify a specific pattern to match component tags`), `

`, magenta.Paint(" trust "), `Trust a library and download or copy locally it's manifest`, gray.Embed(`
   <`, cyan.Paint("link"), `> url or path to the library manifest
   -`, cyan.Paint("r"), ` | --`, cyan.Paint("rename"), ` rename (locally) the trusted library
   -`, cyan.Paint("d"), ` | --`, cyan.Paint("download"), ` download all components and edit local manifest for offline usage`), `

`, magenta.Paint(" untrust "), `Remove the manifest of the specified library`, gray.Embed(`
   <`, cyan.Paint("lib"), `> library unique name`), `

`, magenta.Paint(" use "), `Download a specific component in the current project`, gray.Embed(`
   <`, cyan.Paint("component"), `> the component name or the library name followed by '/' and then the component name`), `

`, flagsUsage())
}

func flagsUsage() string {
	return fmt.Sprintln(`Global flags:`, gray.Embed(`

   -`, cyan.Paint("s"), ` | --`, cyan.Paint("settings"), ` Specify the path to the project settings file, by default 'wed-settings.json' will be used.
    If not exists 'build' is used as 'output_dir' and the current working directory as 'input_dir'
   -`, cyan.Paint("nc"), ` | --`, cyan.Paint("no-color"), ` Disable terminal colored output
   -`, cyan.Paint("h"), ` | --`, cyan.Paint("help"), ` Display help and detailed usage of a specific command`), `
`)

}

func serveUsage() {
	fmt.Println(`
        Usage`, magenta.Paint(" serve "), `command

Build the project and serve the 'Outputdir' statically via an http server.
If build phase fails, program exit without server running 

Command flags:`, gray.Embed(`

   -`, cyan.Paint("l"), ` | --`, cyan.Paint("live"), ` Enable automatic rebuilding. If a non 0 time interval is specified, site will be rebuilt at that interval
    If nothing is specified site will be rebuilt on each changes detected from the 'input_dir' recursively (except for the 'output_dir')
   -`, cyan.Paint("p"), ` | --`, cyan.Paint("port"), ` Specify the server port by default :8080 will be used. Character ':' at the beginning is optional`), `

`, flagsUsage())

}

func initUsage() {
	fmt.Println(`
        Usage`, magenta.Paint(" init "), `command

Creates a default project.
Optionally you can specify a directory just after the command.
If not provided the current working directory will be used instead.
If provided but do not exist it be created

`, flagsUsage())

}

func buildUsage() {
	fmt.Println(`
        Usage`, magenta.Paint(" build "), `command

Compile the project into a static site.
If not by 'input_dir' the current working directory will be used as entrypoint
and all subdirectory will be checked recursively.
The program will treats all files with extention '.wed.html' as components and
'.tmpl' as pages.

`, gray.Paint(`Where do things go to:`), `
The output will be located at 'output_dir' ('./build' by default). In the specific
  CSS styles into 'style' subdirectory
  JS scripts into 'script' subdirectory
All the pages at the top level inside 'output_dir'

`, flagsUsage())

}

func runUsage() {
	var wed = cutExt(filepath.Base(os.Args[0]))

	fmt.Println(`
        Usage`, magenta.Paint(" run "), `command

Execute a user-defined pipeline.
Douring execution if one fails the program terminates.
All output generated by the commands will be shown on standard output and if
it's also possible to interact via standard input.
On Windows environment the os variable 'COMSPEC' will be used to detect preference.
if not found all command will be launched via 'cmd' 
On other environments the os variable 'SHELL' will be used instead and
if not found 'sh' will be used

`, gray.Paint(`How to set a pipeline:`), `
A pipeline can be set using the 'commands' property on the project file settings.
This property is a map: command name -> sequence of operation to execute.
Therefore two commands cannot have an identical name. Subcommand are not natively supported.
example:
`+codeBlock(`
"commands": {
	"update": [
		"git fetch",
		"git pull"
	],
	"live": ["`+wed+` serve --port=4200 --live=10s"]
}
`), `

`, gray.Paint(`How call a pipeline:`), `
In relation to the previous example a pipeline called 'update' can be called by simply: 'wed run update'
All flags passed after are not shared with any of the commands of the pipeline

`, flagsUsage())

}

func libUseUsage() {
	fmt.Println(`
        Usage`, magenta.Paint(" lib use "), `command

Use a component in the current project.
The component must be from an already trusted library.
It's possible to specify the name by only using the name, or to avoid homonymous
by prefixing it with the name of the library followed by '/'
When using this command http call(s) will be made to download the component and
if present it's dependencies.

`, gray.Paint(`Where do things go to:`), `
All components will be download inside a subdirectory of 'input_dir' (by default
the current directory) called with the same library name of the requested one
In order to avoid homonymous they will be renamed by prefixing it with the
library name of the requested one followed by '-'

`, flagsUsage())

}

func libTrustUsage() {
	fmt.Println(`
        Usage`, magenta.Paint(" lib trust "), `command

Trust a library and download it's manifest by following the provided link.
If starts with 'http' then an HTTP GET request will be made to retreive the file.
Or else it's assumed is a path to a local file and therefore it will be copied it

`, gray.Paint(`Where do things go to:`), `
The library manifest will be created in the user configuration directory at
wednesday/trusted/<library name>.json

Command flags:`, gray.Embed(`

  -`, cyan.Paint("r"), ` | --`, cyan.Paint("rename"), ` Provide a name for the library.
   By default the name is extrapolated from the provided link
  -`, cyan.Paint("d"), ` | --`, cyan.Paint("download"), ` Download all the components of the
   library beforehand and change local manifest to point to the file just downloaded

`), flagsUsage())

}

func libSearchUsage() {
	fmt.Println(`
        Usage`, magenta.Paint(" lib search "), `command'

Obtain a detailed view of matching components within trusted libraries.
By default the provided pattern will be used for matching components name or tags

Command flags:`, gray.Embed(`

  -`, cyan.Paint("i"), ` Enable case-insensitive pattern matching
  -`, cyan.Paint("t"), ` | --`, cyan.Paint("tags"), `Filter only matching component tags with provided pattern

`), flagsUsage())

}

func libUntrustUsage() {
	fmt.Println(`
        Usage`, magenta.Paint(" lib untrust "), `command

Trusting a library is essentialy coping a manifest into wednesday config.
By untrusting it you remove that file therefore components provided will not be shown
by 'lib search' or esaily available by 'lib use'
In case a library has been named you must provide the same name.
You can trust back a library at any moment.
Components used by the untrusted library will not be removed from any project

`, flagsUsage())
}

func codeBlock(txt string) string {
	if brush.Disable {
		n := 0
		return " 0 | " + regexp.MustCompile(`\n`).ReplaceAllStringFunc(txt, func(endln string) string {
			n++
			return fmt.Sprint(endln, " ", n, " | ")
		})
	}

	var (
		background = brush.UseColor(brush.GrayScale(1))
		base       = brush.New(brush.BrightWhite.ToExtended(), background)
		syntax     = brush.New(brush.Green.ToExtended(), background)
		number     = brush.New(brush.Red.ToExtended(), brush.UseColor(brush.GrayScale(0)))
		str        = regexp.MustCompile(`".*?"|'.*?'`)
	)

	res := ""
	for i, line := range strings.Split(txt, "\n") {
		res += base.Embed(
			"\n", number.Paint(" ", i+1, " "),
			syntax.Highlight(line, str),
			" ",
		).String()
	}

	return res + "\n"
}
