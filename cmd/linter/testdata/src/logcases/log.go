package logcases

import (
	"log"
	"os"
)

func helper() {
	log.Fatal("stop") // want "log.Fatal is not allowed outside main"
}

func helperExit() {
	os.Exit(1) // want "os.Exit is not allowed outside main"
}

func nested() {
	go func() {
		log.Fatalf("nested") // want "log.Fatal is not allowed outside main"
	}()
}
