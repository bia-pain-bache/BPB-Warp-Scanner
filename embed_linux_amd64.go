//go:build linux && amd64

package main

import "embed"

//go:embed embed/linux-amd64.zip
var binary embed.FS
