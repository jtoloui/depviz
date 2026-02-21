package main

import "github.com/jtoloui/depviz/cmd"

var version = "dev"

func init() {
	cmd.SetVersion(version)
}

func main() {
	cmd.Execute()
}
