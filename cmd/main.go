package main

import (
	"flag"
	"log"

	"github.com/jackparsonss/vertex/engine"
	"github.com/jackparsonss/vertex/internal/config"
)

func main() {
	inputFile := flag.String("input", "", "Go source file to parse (required)")
	outputDir := flag.String("output", "generated", "Directory where generated files will be placed")
	serverPort := flag.Int("port", 8080, "Port for the server to listen on")
	serverEndpoint := flag.String("endpoint", "", "Base URL for the client to connect to (defaults to http://localhost:<port>)")
	packageName := flag.String("package", "generated", "Package name for the generated code")

	flag.Parse()

	c, err := config.NewConfig(*inputFile, *outputDir, *serverEndpoint, *packageName, *serverPort)
	if err != nil {
		log.Fatalf("Error creating config: %v\n", err)
	}

	engine := engine.NewEngine(c)
	engine.Compile()
}
