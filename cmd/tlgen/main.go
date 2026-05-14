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
	sourceDir := flag.String("source", "compiler/source", "directory containing .tl files")
	outDir := flag.String("out", "tg", "output directory for generated Go files")
	layer := flag.Int("layer", 224, "API layer number")
	e2eSchema := flag.String("e2e", "", "path to e2e secret chat TL schema (generates tg/e2e/ if set)")
	flag.Parse()

	files, err := filepath.Glob(filepath.Join(*sourceDir, "*.tl"))
	if err != nil {
		log.Fatal(err)
	}
	if len(files) == 0 {
		log.Fatalf("no .tl files found in %s", *sourceDir)
	}

	var allCombos []tlgen.Combinator
	for _, f := range files {
		file, err := os.Open(f)
		if err != nil {
			log.Fatal(err)
		}
		combos, err := tlgen.Parse(file)
		file.Close()
		if err != nil {
			log.Fatalf("parse %s: %v", f, err)
		}
		allCombos = append(allCombos, combos...)
		fmt.Printf("parsed %s: %d combinators\n", filepath.Base(f), len(combos))
	}

	fmt.Printf("total: %d combinators\n", len(allCombos))

	if err := tlgen.GeneratePackageFiles(*outDir, "tg", *layer); err != nil {
		log.Fatal(err)
	}

	if err := tlgen.GenerateGroupedTypes(*outDir, allCombos, *layer); err != nil {
		log.Fatal("generate grouped types:", err)
	}
	if err := tlgen.GenerateGroupedFunctions(*outDir, allCombos, *layer); err != nil {
		log.Fatal("generate functions:", err)
	}
	if err := tlgen.GenerateNamesMap(*outDir, allCombos); err != nil {
		log.Fatal("generate names map:", err)
	}
	if err := tlgen.GenerateGroupedConstructors(*outDir, allCombos); err != nil {
		log.Fatal("generate constructors map:", err)
	}
	if err := tlgen.GenerateFunctionsMap(*outDir, allCombos); err != nil {
		log.Fatal("generate functions map:", err)
	}

	if *e2eSchema != "" {
		f, err := os.Open(*e2eSchema)
		if err != nil {
			log.Fatal(err)
		}
		e2eCombos, err := tlgen.Parse(f)
		f.Close()
		if err != nil {
			log.Fatalf("parse e2e schema: %v", err)
		}
		fmt.Printf("e2e: %d combinators\n", len(e2eCombos))

		e2eDir := filepath.Join(*outDir, "e2e")
		if err := tlgen.GenerateE2EPackage(e2eDir, e2eCombos, *layer); err != nil {
			log.Fatal("generate e2e:", err)
		}
	}

	fmt.Println("generation complete")
}
