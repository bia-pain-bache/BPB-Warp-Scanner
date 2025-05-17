//go:build 386

package main

import "embed"

//go:embed bin/linux-386.zip
var binary embed.FS
