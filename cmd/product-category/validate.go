package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pcat "github.com/ndx-technologies/product-category-go"
	"github.com/ndx-technologies/product-category-go/validate"
)

const DocShortValidate = "validate product categories"

func ValidateCmd(args []string) error {
	var (
		dir      string
		maxPrint int
	)

	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), DocShortValidate, "\n\n")
		fs.PrintDefaults()
	}

	fs.StringVar(&dir, "dir", ".", "directory with category files")
	fs.IntVar(&maxPrint, "max", 5, "how many rows to print per defect")

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

	translations, err := listTranslationFiles(dir)
	if err != nil {
		return fmt.Errorf("cannot list translation files: %w", err)
	}
	if len(translations) == 0 {
		return fmt.Errorf("no translation files in %s", dir)
	}

	fmt.Printf("canonical: %d categories, version %s\n", len(canonical.Categories), canonical.Version)

	var failedFiles []string

	fmt.Printf("\n== canonical: %d rows\n", len(canonical.Categories))

	var errs []error
	for _, err := range []error{
		validate.ValidateProductCategories(canonical.Categories),
		validate.ValidateTree(canonical.Categories),
	} {
		if err != nil {
			if unwrapper, ok := err.(interface{ Unwrap() []error }); ok {
				errs = append(errs, unwrapper.Unwrap()...)
			} else {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		failedFiles = append(failedFiles, canonicalFilename)
	}
	printDefects(errs, maxPrint)

	for _, translation := range translations {
		path := filepath.Join(dir, translation)
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

		fmt.Printf("\n== %s: %d rows\n", translation, len(repo.Categories))

		var errs []error
		for _, err := range []error{
			validate.ValidateProductCategories(repo.Categories),
			validate.ValidateTree(repo.Categories),
			validate.ValidateIDsMatch(repo, canonical),
			validate.ValidateTreesMatch(repo, canonical),
		} {
			if err != nil {
				if unwrapper, ok := err.(interface{ Unwrap() []error }); ok {
					errs = append(errs, unwrapper.Unwrap()...)
				} else {
					errs = append(errs, err)
				}
			}
		}
		if len(errs) > 0 {
			failedFiles = append(failedFiles, translation)
		}
		printDefects(errs, maxPrint)
	}

	if len(failedFiles) > 0 {
		return fmt.Errorf("defects in %s", strings.Join(failedFiles, ", "))
	}

	fmt.Println("\nok")
	return nil
}

func printDefects(defects []error, maxPrint int) {
	var missing, unknown, empty, tooDeep, tooLong, depth, structure, orphan, singleChild []error

	for _, err := range defects {
		var (
			missingErr   validate.ErrMissing
			unknownErr   validate.ErrUnknown
			emptyNode    validate.ErrEmptyNode
			maxDepth     validate.ErrMaxDepth
			maxLength    validate.ErrMaxLength
			depthErr     validate.ErrDepth
			structureErr validate.ErrStructure
			orphanErr    validate.ErrOrphan
			singleErr    validate.ErrSingleChild
		)

		switch {
		case errors.Is(err, validate.ErrEmpty):
			empty = append(empty, err)
		case errors.As(err, &missingErr):
			missing = append(missing, missingErr)
		case errors.As(err, &unknownErr):
			unknown = append(unknown, unknownErr)
		case errors.As(err, &emptyNode):
			empty = append(empty, emptyNode)
		case errors.As(err, &maxDepth):
			tooDeep = append(tooDeep, maxDepth)
		case errors.As(err, &maxLength):
			tooLong = append(tooLong, maxLength)
		case errors.As(err, &depthErr):
			depth = append(depth, depthErr)
		case errors.As(err, &structureErr):
			structure = append(structure, structureErr)
		case errors.As(err, &orphanErr):
			orphan = append(orphan, orphanErr)
		case errors.As(err, &singleErr):
			singleChild = append(singleChild, singleErr)
		}
	}

	printRows("missing", missing, maxPrint)
	printRows("unknown", unknown, maxPrint)
	printRows("empty", empty, maxPrint)
	printRows("max depth", tooDeep, maxPrint)
	printRows("max length", tooLong, maxPrint)
	printRows("depth", depth, maxPrint)
	printRows("structure", structure, maxPrint)
	printRows("orphan", orphan, maxPrint)
	printRows("single child", singleChild, maxPrint)
}

func printRows(label string, errs []error, maxPrint int) {
	if len(errs) == 0 {
		fmt.Printf("   %s: 0\n", label)
		return
	}
	fmt.Printf("   %s: %d\n", label, len(errs))
	for _, err := range errs[:min(len(errs), maxPrint)] {
		fmt.Printf("      %s\n", err)
	}
	if len(errs) > maxPrint {
		fmt.Printf("      ... and %d more\n", len(errs)-maxPrint)
	}
}
