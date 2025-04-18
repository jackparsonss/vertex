package config

import (
	"fmt"
	"path/filepath"
)

type Config struct {
	InputFile         string
	OutputDir         string
	ServerPort        int
	ServerEndpoint    string
	PackageNameOutput string
	GoModFile         string
}

func NewConfig(inputFile, outputDir, serverEndpoint, packageNameOutput string, serverPort int) (Config, error) {
	if inputFile == "" {
		return Config{}, fmt.Errorf("input file is required")
	}

	if serverEndpoint == "" {
		serverEndpoint = fmt.Sprintf("http://localhost:%d", serverPort)
	}

	absInputFile, err := filepath.Abs(inputFile)
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
		InputFile:         absInputFile,
		OutputDir:         absOutputDir,
		ServerPort:        serverPort,
		ServerEndpoint:    serverEndpoint,
		PackageNameOutput: packageNameOutput,
		GoModFile:         goModFile,
	}, nil
}
