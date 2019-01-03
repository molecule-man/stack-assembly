package commands

import (
	"log"

	"github.com/fatih/color"
)

func handleError(err error) {
	if err != nil {
		c := color.New(color.FgRed, color.Bold)
		log.Fatal(c.Sprint(err))
	}
}
