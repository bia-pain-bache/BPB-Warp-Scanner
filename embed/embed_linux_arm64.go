//go:build arm64

package main

import "embed"

//go:embed bin/linux-arm64.zip
var binary embed.FS
