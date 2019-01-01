package commands

import (
	"log"

	"github.com/fatih/color"
)

func handleError(err error) {
	if err != nil {
		boldRed := color.New(color.FgRed, color.Bold)
		log.Fatal(boldRed.Sprint(err))
	}
}
