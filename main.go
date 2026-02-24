package main

import "github.com/MyCarrier-DevOps/go-gitsemver/cmd"

var version = "dev"

func main() {
	cmd.Version = version
	cmd.Execute()
}
