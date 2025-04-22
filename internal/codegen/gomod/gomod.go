package gomod

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type GoMod struct {
	path string
}

func NewGoMod(path string) *GoMod {
	return &GoMod{path: path}
}

func ParseGoModule(goModPath string) (string, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "module ") {
			moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			return moduleName, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("no module declaration found in %s", goModPath)
}

func (gm *GoMod) AddReplace() error {
	content, err := os.ReadFile(gm.path)
	if err != nil {
		return err
	}

	contentStr := string(content)

	replacePattern := regexp.MustCompile(`(?m)^replace\s+vertex\s*=>\s*\./vertex`)
	if replacePattern.MatchString(contentStr) {
		return nil
	}

	file, err := os.OpenFile(gm.path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if len(contentStr) > 0 && !strings.HasSuffix(contentStr, "\n") {
		if _, err := file.WriteString("\n"); err != nil {
			return err
		}
	}

	_, err = file.WriteString("\nreplace vertex => ./vertex\n")
	return err
}

func (gm *GoMod) Tidy() error {
	dir := filepath.Dir(gm.path)

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %v\nOutput: %s", err, output)
	}

	return nil
}
