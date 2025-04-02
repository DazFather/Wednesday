package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DazFather/brush"
)

type color func(v ...any) brush.Highlighted

func palette(noColor bool) (color, color, color) {
	var (
		magenta = brush.New(brush.BrightWhite, brush.UseColor(brush.Magenta))
		gray    = brush.New(brush.BrightBlack, nil)
		cyan    = brush.New(brush.BrightCyan, nil)
	)

	if noColor {
		magenta.Disable, gray.Disable, cyan.Disable = true, true, true
	}

	return magenta.Embed, gray.Embed, cyan.Embed
}

func mainUsage(noColor bool) {
	var (
		wed                 = cutExt(filepath.Base(os.Args[0]))
		magenta, gray, cyan = palette(noColor)
	)

	fmt.Println(`
        Wednesday Usage
  `, gray(wed, ` <command> [optional] --flag`), `

Commands:

`, magenta(" build "), `Compile the project into a static site

`, magenta(" init "), `Generate a default project `, gray(`
   [`, cyan("mount"), `] directory to put the project into`), `

`, magenta(" serve "), `Build and serve your project statically via http`, gray(`
   -`, cyan("p"), ` | --`, cyan("port"), ` specify the server port. Default :8080
   -`, cyan("l"), ` | --`, cyan("live"), ` enable automatic rebuilding at specified time interval`), `

`, magenta(" run <command> "), `Execute a user-defined pipeline of commands

`, magenta(" lib <command> "), `Manage and interact with external libraries`, gray(`
   `, cyan("search"), ` [pattern] Search for components within trusted libraries
   `, cyan("trust"), ` <url> Trust a library and download it's manifest
   `, cyan("untrust"), ` <library> Remove the manifest of the specified library
   `, cyan("use"), ` <component> Use a specific component in the current project`), `

`, magenta(" h ", gray("|"), " help "), `Show detailed usage`, gray(`
   -`, cyan("nc"), ` | --`, cyan("no-color"), ` force disable of colored output
   [`, cyan("command"), `] command to obtain usage about. If not provided this message is shown`), `

`, magenta(" v ", gray("|"), " version "), "Display the current", wed, `version

`, flagsUsage(noColor))

}

func libUsage(noColor bool) {
	var magenta, gray, cyan = palette(noColor)

	fmt.Println(`
        Usage`, magenta(" lib "), `command

Sub-commands:

`, magenta(" search "), `Search for components within trusted libraries`, gray(`
   [`, cyan("pattern"), `] regex pattern for component name matching. If '--tags' is not provided, it will match tags as well.
   -`, cyan("i"), ` enable case-insensitive pattern matching
   -`, cyan("t"), ` | --`, cyan("tags"), ` specify a specific pattern to match component tags`), `

`, magenta(" trust "), `Trust a library and download or copy locally it's manifest`, gray(`
   <`, cyan("link"), `> url or path to the library manifest
   -`, cyan("r"), ` | --`, cyan("rename"), ` rename (locally) the trusted library`), `

`, magenta(" untrust "), `Remove the manifest of the specified library`, gray(`
   <`, cyan("lib"), `> library unique name`), `

`, magenta(" use "), `Download a specific component in the current project`, gray(`
   <`, cyan("component"), `> the component name or the library name followed by '/' and then the component name`), `

`, flagsUsage(noColor))
}

func flagsUsage(noColor bool) string {
	var _, gray, cyan = palette(noColor)

	return fmt.Sprintln(`Global flags:`, gray(`

   -`, cyan("s"), ` | --`, cyan("settings"), ` Specify the path to the project settings file, by default 'wed-settings.json' will be used.
    If not exists 'build' is used as 'OutputDir' and the current working directory as 'InputDir'
   -`, cyan("h"), ` | --`, cyan("help"), ` Display help and detailed usage of a specific command`), `
`)

}

func serveUsage(noColor bool) {
	var magenta, gray, cyan = palette(noColor)

	fmt.Println(`
        Usage`, magenta(" serve "), `command

Build the project and serve the 'Outputdir' statically via an http server.
If build phase fails, program exit without server running 

Command flags:`, gray(`

  -`, cyan("p"), ` | --`, cyan("port"), ` Specify the server port by default :8080 will be used. Character ':' at the beginning is optional
  -`, cyan("l"), ` | --`, cyan("live"), ` Enable rebuilding at a specified time interval. If 0 or value not provided there will be no rebuilding`), `

`, flagsUsage(noColor))

}

func initUsage(noColor bool) {
	var magenta, _, _ = palette(noColor)

	fmt.Println(`
        Usage`, magenta(" init "), `command

Creates a default project.
Optionally you can specify a directory just after the command.
If not provided the current working directory will be used instead.
If provided but do not exist it be created

`, flagsUsage(noColor))

}

func buildUsage(noColor bool) {
	var magenta, gray, _ = palette(noColor)

	fmt.Println(`
        Usage`, magenta(" build "), `command

Compile the project into a static site.
If not by 'InputDir' the current working directory will be used as entrypoint
and all subdirectory will be checked recursively.
The program will treats all files with extention '.wed.html' as components and
'.tmpl' as pages.

`, gray(`Where do things go to:`), `
The output will be located at 'OutputDir' ('./build' by default). In the specific
  CSS styles into 'style' subdirectory
  JS scripts into 'script' subdirectory
All the pages at the top level inside 'OutputDir'

`, flagsUsage(noColor))

}

func runUsage(noColor bool) {
	var (
		wed              = cutExt(filepath.Base(os.Args[0]))
		magenta, gray, _ = palette(noColor)
	)

	fmt.Println(`
        Usage`, magenta(" run "), `command

Execute a user-defined pipeline.
Douring execution if one fails the program terminates.
All output generated by the commands will be shown on standard output and if
it's also possible to interact via standard input.
On Windows environment the os variable 'COMSPEC' will be used to detect preference.
if not found all command will be launched via 'cmd' 
On other environments the os variable 'SHELL' will be used instead and
if not found 'sh' will be used

`, gray(`How to set a pipeline:`), `
A pipeline can be set using the 'Commands' property on the project file settings.
This property is a map: command name -> sequence of operation to execute.
Therefore two commands cannot have an identical name. Subcommand are not natively supported.
example:
`+codeBlock(`
"Commands": {
	"update": [
		"git fetch",
		"git pull"
	],
	"live": ["`+wed+` serve --port=4200 --live=10s"]
}
`, noColor), `

`, gray(`How call a pipeline:`), `
In relation to the previous example a pipeline called 'update' can be called by simply: 'wed run update'
All flags passed after are not shared with any of the commands of the pipeline

`, flagsUsage(noColor))

}

func libUseUsage(noColor bool) {
	var magenta, gray, _ = palette(noColor)

	fmt.Println(`
        Usage`, magenta(" lib use "), `command

Use a component in the current project.
The component must be from an already trusted library.
It's possible to specify the name by only using the name, or to avoid homonymous
by prefixing it with the name of the library followed by '/'
When using this command http call(s) will be made to download the component and
if present it's dependencies.

`, gray(`Where do things go to:`), `
All components will be download inside a subdirectory of 'InputDir' (by default
the current directory) called with the same library name of the requested one
In order to avoid homonymous they will be renamed by prefixing it with the
library name of the requested one followed by '-'

`, flagsUsage(noColor))

}

func libTrustUsage(noColor bool) {
	var magenta, gray, cyan = palette(noColor)

	fmt.Println(`
        Usage`, magenta(" lib trust "), `command

Trust a library and download it's manifest by following the provided link.
If starts with 'http' then an HTTP GET request will be made to retreive the file.
Or else it's assumed is a path to a local file and therefore it will be copied it

`, gray(`Where do things go to:`), `
The library manifest will be created in the user configuration directory at
wednesday/trusted/<library name>.json

Command flags:`, gray(`

  -`, cyan("r"), ` | --`, cyan("rename"), ` Provide a name for the library.
   By default the name is extrapolated from the provided link

`), flagsUsage(noColor))

}

func libSearchUsage(noColor bool) {
	var magenta, gray, cyan = palette(noColor)

	fmt.Println(`
        Usage`, magenta(" lib search "), `command'

Obtain a detailed view of matching components within trusted libraries.
By default the provided pattern will be used for matching components name or tags

Command flags:`, gray(`

  -`, cyan("i"), ` Enable case-insensitive pattern matching
  -`, cyan("t"), ` | --`, cyan("tags"), `Filter only matching component tags with provided pattern

`), flagsUsage(noColor))

}

func libUntrustUsage(noColor bool) {
	var magenta, _, _ = palette(noColor)

	fmt.Println(`
        Usage`, magenta(" lib untrust "), `command

Trusting a library is essentialy coping a manifest into wednesday config.
By untrusting it you remove that file therefore components provided will not be shown
by 'lib search' or esaily available by 'lib use'
In case a library has been named you must provide the same name.
You can trust back a library at any moment.
Components used by the untrusted library will not be removed from any project

`, flagsUsage(noColor))
}

func codeBlock(txt string, noColor bool) string {
	if noColor {
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
		str        = regexp.MustCompile(`".*?"`)
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
