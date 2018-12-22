// Package main provides cmd stas application
package main

import (
	"log"

	"github.com/molecule-man/stack-assembly/cmd/commands"
)

func main() {
	if err := commands.RootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
