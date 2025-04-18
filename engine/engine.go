package engine

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	"github.com/jackparsonss/vertex/internal/codegen"
	vp "github.com/jackparsonss/vertex/internal/codegen/parser"
	"github.com/jackparsonss/vertex/internal/config"
)

type Engine struct {
	Config       config.Config
	vertexParser *vp.VertexParser
}

func NewEngine(config config.Config) (*Engine, error) {
	nodes, err := getNodes(config.InputDir)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, err
	}

	return &Engine{vertexParser: vp.NewVertexParser(nodes, config), Config: config}, nil
}

func getNodes(root string) ([]*ast.File, error) {
	fset := token.NewFileSet()
	var astFiles []*ast.File

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse dir %s: %w", path, err)
		}

		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				astFiles = append(astFiles, file)
			}
		}

		return nil
	})

	return astFiles, err
}

func (e *Engine) Compile() error {
	v, err := e.vertexParser.Parse()
	if err != nil {
		return err
	}

	generator := codegen.NewGenerator(e.Config, v)
	err = generator.GenerateServerCode()
	if err != nil {
		return err
	}

	err = generator.GenerateClientCode()
	if err != nil {
		return err
	}

	return nil
}
