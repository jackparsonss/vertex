package codegen

import (
	"embed"
	"fmt"
	"html/template"
	"os"

	"github.com/jackparsonss/vertex/internal/config"
)

//go:embed templates/*.tmpl
var templates embed.FS

type Generator struct {
	Config    config.Config
	functions []FunctionInfo
}

func NewGenerator(config config.Config, functions []FunctionInfo) *Generator {
	return &Generator{Config: config, functions: functions}
}

func (g *Generator) GenerateClientCode() {
	tmpl := template.Must(template.ParseFS(templates, "templates/client.tmpl"))

	packageName := ""
	if len(g.functions) > 0 {
		packageName = g.functions[0].PackageName
	}

	templateData := struct {
		PackageName string
		Functions   []FunctionInfo
	}{
		PackageName: packageName,
		Functions:   g.functions,
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
	if len(g.functions) > 0 {
		packageName = g.functions[0].PackageName
	}

	structFuncs := make(map[string][]FunctionInfo)
	var standaloneFuncs []FunctionInfo

	for _, fn := range g.functions {
		if fn.IsMethod {
			structFuncs[fn.StructName] = append(structFuncs[fn.StructName], fn)
		} else {
			standaloneFuncs = append(standaloneFuncs, fn)
		}
	}

	allFunctions := make([]FunctionInfo, 0)
	for _, fns := range structFuncs {
		allFunctions = append(allFunctions, fns...)
	}
	allFunctions = append(allFunctions, standaloneFuncs...)

	templateData := struct {
		PackageName     string
		StructFuncs     map[string][]FunctionInfo
		StandaloneFuncs []FunctionInfo
		AllFunctions    []FunctionInfo
	}{
		PackageName:     packageName,
		StructFuncs:     structFuncs,
		StandaloneFuncs: standaloneFuncs,
		AllFunctions:    allFunctions,
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
