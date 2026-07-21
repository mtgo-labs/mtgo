package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mtgo-labs/mtgo/compiler/tlgen"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	sourceDir := flag.String("source", "compiler/source", "directory containing .tl files")
	outDir := flag.String("out", "tg", "output directory for generated Go files")
	layer := flag.Int("layer", 228, "API layer number")
	e2eSchema := flag.String("e2e", "compiler/e2e.tl", "path to e2e secret chat TL schema")
	flag.Parse()

	files, err := filepath.Glob(filepath.Join(*sourceDir, "*.tl"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no .tl files found in %s", *sourceDir)
	}

	var allCombos []tlgen.Combinator
	for _, f := range files {
		combos, err := parseSchemaFile(f)
		if err != nil {
			return fmt.Errorf("parse %s: %w", f, err)
		}
		allCombos = append(allCombos, combos...)
		fmt.Printf("parsed %s: %d combinators\n", filepath.Base(f), len(combos))
	}

	fmt.Printf("total: %d combinators\n", len(allCombos))

	if *e2eSchema == "" {
		return fmt.Errorf("e2e schema path must not be empty")
	}
	e2eCombos, err := parseSchemaFile(*e2eSchema)
	if err != nil {
		return fmt.Errorf("parse e2e schema: %w", err)
	}
	fmt.Printf("e2e: %d combinators\n", len(e2eCombos))

	parentDir := filepath.Dir(*outDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return err
	}
	stageDir, err := os.MkdirTemp(parentDir, ".tlgen-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageDir)

	if err := tlgen.GeneratePackageFiles(stageDir, "tg", *layer); err != nil {
		return err
	}
	if err := tlgen.GenerateGroupedTypes(stageDir, allCombos, *layer); err != nil {
		return fmt.Errorf("generate grouped types: %w", err)
	}
	if err := tlgen.GenerateGroupedFunctions(stageDir, allCombos, *layer); err != nil {
		return fmt.Errorf("generate functions: %w", err)
	}
	if err := tlgen.GenerateNamesMap(stageDir, allCombos); err != nil {
		return fmt.Errorf("generate names map: %w", err)
	}
	if err := tlgen.GenerateGroupedConstructors(stageDir, allCombos); err != nil {
		return fmt.Errorf("generate constructors map: %w", err)
	}
	if err := tlgen.GenerateFunctionsMap(stageDir, allCombos); err != nil {
		return fmt.Errorf("generate functions map: %w", err)
	}

	e2eDir := filepath.Join(stageDir, "e2e")
	if err := tlgen.GenerateE2EPackage(e2eDir, e2eCombos, *layer); err != nil {
		return fmt.Errorf("generate e2e: %w", err)
	}
	if err := tlgen.InstallGeneratedFiles(*outDir, stageDir); err != nil {
		return err
	}

	fmt.Println("generation complete")
	return nil
}

func parseSchemaFile(path string) ([]tlgen.Combinator, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	combos, parseErr := tlgen.Parse(file)
	closeErr := file.Close()
	if parseErr != nil {
		return nil, parseErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return combos, nil
}
