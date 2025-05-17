//go:build arm

package main

import "embed"

//go:embed bin/linux-arm.zip
var binary embed.FS
