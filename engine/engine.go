package engine

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"

	"github.com/jackparsonss/vertex/internal/codegen"
	"github.com/jackparsonss/vertex/internal/config"
)

type Engine struct {
	Config       config.Config
	vertexParser *codegen.VertexParser
}

func NewEngine(config config.Config) *Engine {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, config.InputFile, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	return &Engine{vertexParser: codegen.NewVertexParser(node), Config: config}
}

func (e *Engine) Compile() {
	functions := e.vertexParser.Parse()

	generator := codegen.NewGenerator(e.Config, functions)
	generator.GenerateServerCode(functions)
	generator.GenerateClientCode(functions)
}
