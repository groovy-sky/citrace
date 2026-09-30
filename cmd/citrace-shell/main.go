package main

import (
	"os"

	"github.com/citrace/citrace-shell/internal/app"
	"github.com/citrace/citrace-shell/internal/version"
)

var buildVersion = "dev"

func main() {
	version.Value = buildVersion
	os.Exit(app.Run(os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr))
}
