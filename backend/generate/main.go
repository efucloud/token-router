package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/efucloud/token-router/pkg/apis"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/emicklei/go-restful/v3"
)

func main() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	outputDir := filepath.Clean(filepath.Join(dir, "..", "frontend", "src", "services"))
	if err = os.RemoveAll(outputDir); err != nil {
		panic(err)
	}
	if err = os.MkdirAll(outputDir, 0o755); err != nil {
		panic(err)
	}
	generateTypescriptDefine()
	gen := NewRestAPI(config.FrontApiTag)
	ws := apis.GetWebServices(restful.DefaultContainer)
	for _, route := range ws.Routes() {
		gen.AddRoute(route)
	}
	gen.GenerateToDir(outputDir)
	_ = os.Remove(filepath.Join(outputDir, "entries.json"))
	if err = normalizeGeneratedFiles(outputDir); err != nil {
		panic(err)
	}
	fmt.Printf("frontend services generated in %s\n", outputDir)
}

func normalizeGeneratedFiles(outputDir string) error {
	files, err := filepath.Glob(filepath.Join(outputDir, "*.ts"))
	if err != nil {
		return err
	}
	for _, file := range files {
		content, readErr := os.ReadFile(file)
		if readErr != nil {
			return readErr
		}
		lines := strings.Split(string(content), "\n")
		for index := range lines {
			lines[index] = strings.TrimRight(lines[index], " \t")
		}
		normalized := strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
		if writeErr := os.WriteFile(file, []byte(normalized), 0o644); writeErr != nil {
			return writeErr
		}
		if chmodErr := os.Chmod(file, 0o644); chmodErr != nil {
			return chmodErr
		}
	}
	return nil
}
