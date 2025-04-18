package engine

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"

	"github.com/jackparsonss/vertex/internal/codegen"
	vp "github.com/jackparsonss/vertex/internal/codegen/parser"
	"github.com/jackparsonss/vertex/internal/config"
)

type Engine struct {
	Config       config.Config
	vertexParser *vp.VertexParser
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

	return &Engine{vertexParser: vp.NewVertexParser(node), Config: config}
}

func (e *Engine) Compile() {
	functions := e.vertexParser.Parse()

	generator := codegen.NewGenerator(e.Config, functions)
	generator.GenerateServerCode()
	generator.GenerateClientCode()
}
