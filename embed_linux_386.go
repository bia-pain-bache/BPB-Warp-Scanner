//go:build linux && 386

package main

import "embed"

//go:embed embed/linux-386.zip
var binary embed.FS
