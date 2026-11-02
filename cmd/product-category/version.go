package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	pcat "github.com/ndx-technologies/product-category-go"
)

const DocShortBumpVersion = "increment version in all category files"

func BumpVersionCmd(args []string) error {
	var dir string

	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.Usage = func() {
		fs.PrintDefaults()
	}

	fs.StringVar(&dir, "dir", ".", "directory with category files")

	if err := fs.Parse(args); err != nil {
		return err
	}

	path := filepath.Join(dir, canonicalFilename)
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", path, err)
	}
	var canonical pcat.ProductCategoryRepositoryCSV
	err = canonical.Read(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", path, err)
	}
	nextVersion, err := canonical.Version.Next()
	if err != nil {
		return err
	}

	fmt.Printf("bumping version: %s → %s\n", canonical.Version, nextVersion)

	paths, err := taxonomyFiles(dir)
	if err != nil {
		return fmt.Errorf("cannot list taxonomy files: %w", err)
	}

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open %s: %w", path, err)
		}
		var repo pcat.ProductCategoryRepositoryCSV
		err = repo.Read(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", path, err)
		}
		repo.Version = nextVersion

		f, err = os.Create(path)
		if err != nil {
			return fmt.Errorf("cannot create %s: %w", path, err)
		}
		if err := repo.Write(f); err != nil {
			f.Close()
			return fmt.Errorf("cannot write %s: %w", path, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("cannot write %s: %w", path, err)
		}

		fmt.Printf("  %s\n", filepath.Base(path))
	}

	return nil
}
