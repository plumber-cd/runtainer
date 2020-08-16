package main

import (
	"github.com/plumber-cd/runtainer/cmd"
	"github.com/plumber-cd/runtainer/log"
)

func main() {
	loggerCallback := log.SetupLog()
	defer loggerCallback()

	log.Info.Print("RunTainer started")

	err := cmd.Execute()
	if err != nil {
		log.Error.Fatal(err)
	}
}
