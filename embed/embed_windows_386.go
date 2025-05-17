//go:build windows && 386

package main

import "embed"

//go:embed bin/windows-386.zip
var binary embed.FS
