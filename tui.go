package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func mainUsage() {
	var wed = cutExt(filepath.Base(os.Args[0]))

	fmt.Println("Usage:", wed, `<command> [options]

Commands:
  
  build          Compile the project into a static site
 
  init [dir]     Create a default project optionally specifing the directory

  serve          Build and serve your project statically.
    -p, --port <port>      Specify the server port (default: :8080)
    -l, --live <interval>  Enable automatic rebuilding at specified time interval

  run <command>  Execute a user-defined command.

  help, h        Show this message

  version, v     Display the current program version

  lib <command>  Manage and interact with external libraries
    search [pattern]    Search for components within trusted libraries
    trust <url>         Trust a library and download it's manifest
    distrust <library>  Remove the manifest of the specified library
    use <component>     Use a specific component in the current project


Global Flags:

  -s, --settings <file>  Specify the path to the project settings file,
                         By default 'wed-settings.json' will be used.
                         If not exists 'build' is used as 'OutputDir' and
                         the current working directory as 'InputDir'

  -h, --help             Display help for a command

Run '`, wed, "help <command>' for detailed usage of a specific command")
}

func libUsage() {
	commandUsage("lib <command>", `Commands:

  search [pattern]  Search for components within trusted libraries.
                    If '--tags' is not provided, it will match tags as well.
    -i                   Enable case-insensitive pattern matching.
    --tags, -t <pattern> Specify a pattern to match component tags.

  trust <url>       Trust a library and download or copy locally it's manifest.
    --rename, -r <name>  Rename (locally) the trusted library.

  distrust <lib>    Remove the manifest of the specified library.

  use <component>   Download a specific component in the current project. Specify
                    either the component name or the library name followed by '/'
                    and then the component name.
`)
}

func commandUsage(command, usage string) {
	var wed = cutExt(filepath.Base(os.Args[0]))

	fmt.Println("Usage:", wed, command, "\n\n", usage, `

Global Flags:

  -s, --settings <file>  Specify the path to the project settings file,
                         By default 'wed-settings.json' will be used.
                         If not exists './build' is used as 'OutputDir' and
                         the current working directory as 'InputDir'

  -h, --help             Display help for a command

Run '`, wed, "help' for general usage")

}
