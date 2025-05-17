//go:build linux && arm

package main

import "embed"

//go:embed embed/linux-arm.zip
var binary embed.FS
