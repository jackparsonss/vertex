package codegen

import (
	"embed"
	"fmt"
	"html/template"
	"os"

	"github.com/jackparsonss/vertex/internal/codegen/types"
	"github.com/jackparsonss/vertex/internal/config"
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

func (g *Generator) GenerateClientCode() {
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

	file, err := os.Create(fmt.Sprintf("%s/client.go", g.Config.OutputDir))
	if err != nil {
		fmt.Printf("Error creating client file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := tmpl.Execute(file, templateData); err != nil {
		fmt.Printf("Error executing client template: %v\n", err)
		os.Exit(1)
	}
}

func (g *Generator) GenerateServerCode() {
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

	file, err := os.Create(fmt.Sprintf("%s/server.go", g.Config.OutputDir))
	if err != nil {
		fmt.Printf("Error creating server file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := tmpl.Execute(file, templateData); err != nil {
		fmt.Printf("Error executing server template: %v\n", err)
		os.Exit(1)
	}
}
