package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	pcat "github.com/ndx-technologies/product-category-go"
)

const (
	DocShortNormalize = "sort alphabetically, reassign sequential IDs, remap all translations"

	DocNormalize = `sort categories by path, reassign IDs 1..N, and remap every translation onto them.

Additions, renames and deletions all work: the list is rebuilt from scratch.
Translations are matched to the canonical on ID, so their row order does not matter.

The command writes nothing when a category has no translation in some language. Pass
-allow_partial to write a knowingly incomplete result.`
)

func NormalizeCmd(args []string) error {
	var (
		dir          string
		allowPartial bool
	)

	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), DocNormalize, "\n\n")
		fs.PrintDefaults()
	}

	fs.StringVar(&dir, "dir", ".", "directory with category files")
	fs.BoolVar(&allowPartial, "allow_partial", false, "write the result even when categories have no translation")

	if err := fs.Parse(args); err != nil {
		return err
	}

	canonicalPath := filepath.Join(dir, canonicalFilename)
	f, err := os.Open(canonicalPath)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", canonicalPath, err)
	}
	var canonical pcat.ProductCategoryRepositoryCSV
	err = canonical.Read(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", canonicalPath, err)
	}

	// The list is rebuilt from scratch: deduplicated, then sorted by path.
	seen := make(map[pcat.ProductCategory]bool, len(canonical.Categories))
	sorted := make([]pcat.ProductCategory, 0, len(canonical.Categories))
	for _, category := range canonical.Categories {
		if !seen[category] {
			seen[category] = true
			sorted = append(sorted, category)
		}
	}
	slices.Sort(sorted)

	oldIDs := make(map[pcat.ProductCategory]pcat.ProductCategoryID, len(canonical.Categories))
	for i, category := range canonical.Categories {
		oldIDs[category] = canonical.IDs[i]
	}

	fmt.Printf("canonical: %d categories\n", len(sorted))
	fmt.Printf("current version: %s\n", canonical.Version)

	translations, err := listTranslationFiles(dir)
	if err != nil {
		return fmt.Errorf("cannot list translation files: %w", err)
	}

	// Translations are joined on ID, never on row position, so a row may sit
	// anywhere in the file. Everything is remapped before anything is written, so
	// a missing translation fails the run instead of leaving a half-rewritten
	// directory behind.
	type remapped struct {
		filename string
		repo     pcat.ProductCategoryRepositoryCSV
		missing  []pcat.ProductCategory
	}

	results := make([]remapped, 0, len(translations))

	for _, translation := range translations {
		path := filepath.Join(dir, translation)
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open %s: %w", path, err)
		}
		var loaded pcat.ProductCategoryRepositoryCSV
		err = loaded.Read(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", path, err)
		}
		byID := make(map[pcat.ProductCategoryID]pcat.ProductCategory, len(loaded.Categories))
		for i, id := range loaded.IDs {
			if id > 0 {
				byID[id] = loaded.Categories[i]
			}
		}

		repo := pcat.ProductCategoryRepositoryCSV{
			Version:    canonical.Version,
			Categories: make([]pcat.ProductCategory, 0, len(sorted)),
			IDs:        make([]pcat.ProductCategoryID, 0, len(sorted)),
			Index:      make(map[pcat.ProductCategory]pcat.ProductCategoryID, len(sorted)),
		}

		var missing []pcat.ProductCategory

		for i, category := range sorted {
			newID := pcat.ProductCategoryID(i + 1)

			text, ok := byID[oldIDs[category]]
			if !ok {
				missing = append(missing, category)
				continue
			}

			repo.Categories = append(repo.Categories, text)
			repo.IDs = append(repo.IDs, newID)
			repo.Index[text] = newID
		}

		fmt.Printf("  %-10s: %d/%d translated\n", translation, len(repo.Categories), len(sorted))
		for _, category := range missing {
			fmt.Printf("      no translation: %s\n", category)
		}

		results = append(results, remapped{filename: translation, repo: repo, missing: missing})
	}

	for _, r := range results {
		if len(r.missing) > 0 && !allowPartial {
			return fmt.Errorf("%s is missing %d translations; add them or pass -allow_partial", r.filename, len(r.missing))
		}
	}

	canonicalRepo := pcat.ProductCategoryRepositoryCSV{
		Version:    canonical.Version,
		Categories: sorted,
		IDs:        make([]pcat.ProductCategoryID, len(sorted)),
		Index:      make(map[pcat.ProductCategory]pcat.ProductCategoryID, len(sorted)),
	}
	for i, category := range sorted {
		id := pcat.ProductCategoryID(i + 1)
		canonicalRepo.IDs[i] = id
		canonicalRepo.Index[category] = id
	}

	path := filepath.Join(dir, canonicalFilename)
	f, err = os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", path, err)
	}
	if err := canonicalRepo.Write(f); err != nil {
		f.Close()
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	fmt.Printf("wrote %s: %d categories\n", canonicalFilename, len(sorted))

	for _, r := range results {
		path := filepath.Join(dir, r.filename)
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("cannot create %s: %w", path, err)
		}
		if err := r.repo.Write(f); err != nil {
			f.Close()
			return fmt.Errorf("cannot write %s: %w", path, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("cannot write %s: %w", path, err)
		}
	}

	return nil
}
