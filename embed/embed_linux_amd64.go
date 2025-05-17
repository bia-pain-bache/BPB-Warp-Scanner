//go:build amd64

package main

import "embed"

//go:embed bin/linux-amd64.zip
var binary embed.FS
