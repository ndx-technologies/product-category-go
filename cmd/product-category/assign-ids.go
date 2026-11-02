package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	pcat "github.com/ndx-technologies/product-category-go"
)

const (
	DocShortAssignIDs = "regenerate category ids for a version"

	DocAssignIDs = `rewrite the ID column of every recognized file so that IDs run 1..N in file order.

This is the migration that lets the rest of the toolkit join on ID rather than
guessing from row position: it is the last command that reads position, and it uses it
only to establish the IDs every later command relies on.`
)

func AssignIDsCmd(args []string) error {
	var dir string

	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), DocAssignIDs, "\n\n")
		fs.PrintDefaults()
	}
	fs.StringVar(&dir, "dir", ".", "directory with category files")

	if err := fs.Parse(args); err != nil {
		return err
	}

	paths, err := taxonomyFiles(dir)
	if err != nil {
		return fmt.Errorf("cannot list taxonomy files: %w", err)
	}

	repos := make([]pcat.ProductCategoryRepositoryCSV, len(paths))
	for i, path := range paths {
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
		repos[i] = repo

		if i > 0 && len(repos[i].Categories) != len(repos[0].Categories) {
			return fmt.Errorf("%s has %d categories, %s has %d: refusing to assign IDs by position",
				filepath.Base(path), len(repos[i].Categories),
				filepath.Base(paths[0]), len(repos[0].Categories))
		}
	}

	for i, path := range paths {
		repo := repos[i]
		repo.IDs = make([]pcat.ProductCategoryID, len(repo.Categories))
		for j := range repo.Categories {
			repo.IDs[j] = pcat.ProductCategoryID(j + 1)
		}

		f, err := os.Create(path)
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
		fmt.Printf("  %-24s %d categories\n", filepath.Base(path), len(repo.Categories))
	}

	return nil
}
