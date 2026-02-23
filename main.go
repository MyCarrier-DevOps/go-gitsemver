package main

import "go-gitsemver/cmd"

var version = "dev"

func main() {
	cmd.Version = version
	cmd.Execute()
}
