package main

import (
	"log"

	"github.com/nikhilsbhat/yamll/cmd"
	"github.com/spf13/cobra/doc"
)

//go:generate go run github.com/nikhilsbhat/yamll/docs
func main() {
	commands := cmd.SetYamllCommands()

	if err := doc.GenMarkdownTree(commands, "doc"); err != nil {
		log.Fatal(err)
	}
}
