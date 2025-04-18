package config

import (
	"fmt"
	"path/filepath"
)

type Config struct {
	InputDir          string
	OutputDir         string
	PackageNameOutput string
	GoModFile         string
}

func NewConfig(outputDir, packageNameOutput string) (Config, error) {
	absInputFile, err := filepath.Abs(".")
	if err != nil {
		return Config{}, fmt.Errorf("error resolving input file path :%v", err)
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return Config{}, fmt.Errorf("error resolving output directory path :%v", err)
	}

	goModFile, err := filepath.Abs("go.mod")
	if err != nil {
		return Config{}, fmt.Errorf("error resolving go.mod file path :%v", err)
	}

	return Config{
		InputDir:          absInputFile,
		OutputDir:         absOutputDir,
		PackageNameOutput: packageNameOutput,
		GoModFile:         goModFile,
	}, nil
}
