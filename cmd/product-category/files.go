package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	canonicalFilename = "categories.txt"
	translationPrefix = "categories_"
	translationSuffix = ".txt"
)

func listTranslationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		code, ok := strings.CutPrefix(name, translationPrefix)
		if !ok {
			continue
		}
		code, ok = strings.CutSuffix(code, translationSuffix)
		if !ok || code == "" || strings.Contains(code, ".") {
			continue
		}

		files = append(files, name)
	}

	slices.Sort(files)

	return files, nil
}

func taxonomyFiles(dir string) ([]string, error) {
	translations, err := listTranslationFiles(dir)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(translations)+1)
	paths = append(paths, filepath.Join(dir, canonicalFilename))
	for _, translation := range translations {
		paths = append(paths, filepath.Join(dir, translation))
	}

	return paths, nil
}
