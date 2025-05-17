//go:build windows && amd64

package main

import "embed"

//go:embed embed/windows-amd64.zip
var binary embed.FS
