// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
package main

import (
	"fmt"
	"os"

	"github.com/houcemdevops007/d9s/internal/app"
	"github.com/houcemdevops007/d9s/internal/config"
	"github.com/houcemdevops007/d9s/pkg/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("d9s version %s (%s) built %s\n", version.Version, version.GitCommit, version.BuildDate)
			os.Exit(0)
		case "--help", "-h":
			printHelp()
			os.Exit(0)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "d9s: config error: %v\n", err)
		os.Exit(1)
	}

	application := app.New(cfg)
	if err := application.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "d9s: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`d9s - Docker TUI Manager

Usage:
  d9s [flags]

Flags:
  --version, -v    Print version and exit
  --help, -h       Print this help message

Key Bindings:
  Tab              Switch panel (Contexts → Projects → Containers)
  ↑/↓              Navigate list
  Enter            Select
  /                Search containers
  l                View Logs
  e                View Events
  i                Inspect container
  s                Stats view
  S                Open shell (exec)
  r                Restart container
  x                Stop container
  R/Delete         Remove container
  u                Compose up
  d                Compose down
  p                Compose pull
  b                Compose build
  ?                Help overlay
  q / Ctrl+C       Quit
`)
}
