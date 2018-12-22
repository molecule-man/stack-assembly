package commands

import "log"

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
