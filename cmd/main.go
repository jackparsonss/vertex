package main

import (
	"log"
	"os"

	"github.com/jackparsonss/vertex/engine"
	"github.com/jackparsonss/vertex/internal/config"
)

const (
	RUN_COMMAND   = "run"
	BUILD_COMMAND = "build"
)

var validCommands = map[string]bool{
	RUN_COMMAND:   true,
	BUILD_COMMAND: true,
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("Must provide vertex with command, use 'vertex run' to start up")
	}

	command := os.Args[1]
	if ok := validCommands[command]; !ok {
		log.Fatalln("Invalid command, use 'vertex run' to start up")
	}

	c, err := config.NewConfig("vertex", "vertex")
	if err != nil {
		log.Fatalf("Error creating config: %v\n", err)
	}

	engine, err := engine.NewEngine(c)
	if err != nil {
		log.Fatalf("Error creating engine: %v\n", err)
	}

	err = engine.Compile()
	if err != nil {
		log.Fatalf("Error compiling: %v\n", err)
	}

	if command == RUN_COMMAND {
		err = engine.Run()
		if err != nil {
			log.Fatalf("Error starting up: %v\n", err)
		}
	}
}
