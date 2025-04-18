package codegen

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"os"

	"github.com/jackparsonss/vertex/internal/codegen/types"
	"github.com/jackparsonss/vertex/internal/config"
	"golang.org/x/tools/imports"
)

//go:embed templates/*.tmpl
var templates embed.FS

type Generator struct {
	Config config.Config
	Vertex types.Vertex
}

func NewGenerator(config config.Config, v types.Vertex) *Generator {
	return &Generator{Config: config, Vertex: v}
}

func (g *Generator) GenerateClientCode() error {
	tmpl := template.Must(template.ParseFS(templates, "templates/client.tmpl"))

	packageName := ""
	if len(g.Vertex.Functions) > 0 {
		packageName = g.Vertex.Functions[0].PackageName
	}

	templateData := struct {
		PackageName  string
		Functions    []types.FunctionInfo
		GoModPackage string
	}{
		PackageName:  packageName,
		Functions:    g.Vertex.Functions,
		GoModPackage: g.Vertex.GoModPackage,
	}

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, templateData)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s/client.go", g.Config.OutputDir)
	formattedFile, err := imports.Process(filename, buf.Bytes(), nil)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, formattedFile, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (g *Generator) GenerateServerCode() error {
	tmpl := template.Must(template.ParseFS(templates, "templates/server.tmpl"))

	packageName := ""
	if len(g.Vertex.Functions) > 0 {
		packageName = g.Vertex.Functions[0].PackageName
	}

	structFuncs := make(map[string][]types.FunctionInfo)
	var standaloneFuncs []types.FunctionInfo

	for _, fn := range g.Vertex.Functions {
		if fn.IsMethod {
			structFuncs[fn.StructName] = append(structFuncs[fn.StructName], fn)
		} else {
			standaloneFuncs = append(standaloneFuncs, fn)
		}
	}

	allFunctions := make([]types.FunctionInfo, 0)
	for _, fns := range structFuncs {
		allFunctions = append(allFunctions, fns...)
	}
	allFunctions = append(allFunctions, standaloneFuncs...)

	templateData := struct {
		PackageName     string
		StructFuncs     map[string][]types.FunctionInfo
		StandaloneFuncs []types.FunctionInfo
		AllFunctions    []types.FunctionInfo
		GoModPackage    string
	}{
		PackageName:     packageName,
		StructFuncs:     structFuncs,
		StandaloneFuncs: standaloneFuncs,
		AllFunctions:    allFunctions,
		GoModPackage:    g.Vertex.GoModPackage,
	}

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, templateData)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s/server.go", g.Config.OutputDir)
	formattedFile, err := imports.Process(filename, buf.Bytes(), nil)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, formattedFile, 0644)
	if err != nil {
		return err
	}

	return nil
}
