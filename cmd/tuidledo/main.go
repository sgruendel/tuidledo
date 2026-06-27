package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/sgruendel/tuidledo/internal/app"
)

var version = "dev"
var clientID = ""
var clientSecret = ""

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Printf("tuidledo %s\n", version)
		return
	}

	program := tea.NewProgram(app.New(clientID, clientSecret))
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tuidledo: %v\n", err)
		os.Exit(1)
	}
}
