//go:build linux && arm64

package main

import "embed"

//go:embed embed/linux-arm64.zip
var binary embed.FS
