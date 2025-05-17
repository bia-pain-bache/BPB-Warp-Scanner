//go:build windows && amd64

package main

import "embed"

//go:embed bin/windows-amd64.zip
var binary embed.FS
