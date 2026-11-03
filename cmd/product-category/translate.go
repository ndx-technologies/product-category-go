package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	pcat "github.com/ndx-technologies/product-category-go"
)

const DocShortTranslate = "fill missing translations from a glossary and the file's own memory"

const DocTranslate = `fill the translations that canonical has and a language file does not.

This is step 3 of the release: the canonical file has been edited, and every translation file has to
carry the same categories with the same IDs. A translation file is joined on ID, so row order does not
matter and a row is composed as "translated parent / translated name", which is what keeps the tree
shape identical to canonical's - validate reports that as its structure counter.

A name is translated in this order, and the first hit wins:

  1. the glossary with its parent: lang,parent,english,translation;
  2. the glossary without a parent: lang,,english,translation - the contexts that share one wording;
  3. the file's own memory, where the same name was translated under the same parent;
  4. the file's own memory, where the same name has exactly one translation anywhere;
  5. the identity, for en only, since the canonical path is already English.

A name the memory holds with two or more translations under different parents is refused rather than
guessed: 化妆工具 for "Tools" is right under makeup and wrong for a plumber's kit. That is the whole
reason the glossary exists, and an unresolved name is reported instead of written.

Existing rows are never rewritten: the command appends only IDs the file does not have.

Usage, as step 3 of a release:

  product-category translate -dir <dir> -glossary glossary.csv
  product-category normalize -dir <dir>
`

type Language = string

type glossaryKey struct {
	lang    Language
	parent  string
	english string
}

type translator struct {
	lang      Language
	glossary  map[glossaryKey]string
	byContext map[[2]string][]string
	byName    map[string][]string
}

func (t *translator) name(name, parent string) (string, string, bool) {
	if v, ok := t.glossary[glossaryKey{t.lang, parent, name}]; ok {
		return v, "glossary", true
	}
	if v, ok := t.glossary[glossaryKey{t.lang, "", name}]; ok {
		return v, "glossary", true
	}
	if v, ok := unique(t.byContext[[2]string{parent, name}]); ok {
		return v, "memory", true
	}
	if v, ok := unique(t.byName[name]); ok {
		return v, "memory", true
	}
	if t.lang == "en" {
		return name, "identity", true
	}
	return "", "", false
}

// unique returns the single value of a memory list, and refuses a list that holds more than one.
func unique(values []string) (string, bool) {
	if len(values) != 1 {
		return "", false
	}
	return values[0], true
}

// category translates one canonical path, recursively through its parent.
// memo holds "" for a path that could not be translated, so a subtree is visited once.
func (t *translator) category(c pcat.ProductCategory, memo map[pcat.ProductCategory]string, source map[string]int, unresolved map[string]pcat.ProductCategory) (string, bool) {
	if v, ok := memo[c]; ok {
		return v, v != ""
	}

	nodes := c.PathNodes()
	name := nodes[len(nodes)-1]

	var translatedParent, parentName string
	if len(nodes) > 1 {
		parent := c.Parent()
		value, ok := t.category(parent, memo, source, unresolved)
		if !ok {
			memo[c] = ""
			return "", false
		}
		translatedParent = value
		parentName = nodes[len(nodes)-2]
	}

	translatedName, from, ok := t.name(name, parentName)
	if !ok {
		memo[c] = ""
		if _, seen := unresolved[name]; !seen {
			unresolved[name] = c
		}
		return "", false
	}

	source[from]++
	out := translatedName
	if translatedParent != "" {
		out = translatedParent + " / " + translatedName
	}
	memo[c] = out
	return out, true
}

func TranslateCmd(args []string) error {
	var (
		dir          string
		glossaryPath string
		langsCSV     string
		dry          bool
		maxPrint     int
	)

	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), DocTranslate, "\n\n")
		fs.PrintDefaults()
	}
	fs.StringVar(&dir, "dir", ".", "directory with category files")
	fs.StringVar(&glossaryPath, "glossary", "", "glossary csv with header lang,parent,english,translation")
	fs.StringVar(&langsCSV, "lang", "", "comma separated languages to fill (default: every translation file in dir)")
	fs.BoolVar(&dry, "dry", false, "report only, write nothing")
	fs.IntVar(&maxPrint, "max", 20, "how many untranslated names to print per language")

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
	canonicalByID := make(map[pcat.ProductCategoryID]pcat.ProductCategory, len(canonical.Categories))
	for i, id := range canonical.IDs {
		canonicalByID[id] = canonical.Categories[i]
	}

	glossary := make(map[glossaryKey]string)
	if glossaryPath != "" {
		glossary, err = readGlossary(glossaryPath)
		if err != nil {
			return err
		}
	}

	langs, err := languages(dir, langsCSV)
	if err != nil {
		return err
	}

	fmt.Printf("canonical  %d categories, version %s, %s\n", len(canonical.Categories), canonical.Version, canonicalPath)
	fmt.Printf("glossary   %d keys, %s\n", len(glossary), glossaryPath)

	var failed []string
	for _, lang := range langs {
		filename := translationPrefix + lang + translationSuffix
		path := filepath.Join(dir, filename)

		translatedFile, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open %s: %w", path, err)
		}
		var translated pcat.ProductCategoryRepositoryCSV
		err = translated.Read(translatedFile)
		translatedFile.Close()
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", path, err)
		}
		have := make(map[pcat.ProductCategoryID]bool, len(translated.IDs))
		for _, id := range translated.IDs {
			have[id] = true
		}

		t := &translator{
			lang:      lang,
			glossary:  glossary,
			byContext: make(map[[2]string][]string),
			byName:    make(map[string][]string),
		}
		for i, id := range translated.IDs {
			c, ok := canonicalByID[id]
			if !ok {
				continue
			}
			name := c.PathNodes()[len(c.PathNodes())-1]
			var parentName string
			if parent := c.Parent(); !parent.IsZero() {
				nodes := parent.PathNodes()
				parentName = nodes[len(nodes)-1]
			}
			translatedName := translated.Categories[i].PathNodes()[len(translated.Categories[i].PathNodes())-1]
			t.byContext[[2]string{parentName, name}] = appendUnique(t.byContext[[2]string{parentName, name}], translatedName)
			t.byName[name] = appendUnique(t.byName[name], translatedName)
		}

		missing := make([]pcat.ProductCategoryID, 0, len(canonical.Categories))
		for _, id := range canonical.IDs {
			if !have[id] {
				missing = append(missing, id)
			}
		}
		slices.Sort(missing)

		memo := make(map[pcat.ProductCategory]string, len(missing))
		// Seed the memo with the rows the file already carries, so a child inherits the
		// parent's own wording instead of a recomposed one that may differ.
		for i, id := range translated.IDs {
			if c, ok := canonicalByID[id]; ok {
				memo[c] = translated.Categories[i].String()
			}
		}
		source := make(map[string]int)
		unresolved := make(map[string]pcat.ProductCategory)

		rows := make([][2]string, 0, len(missing))
		for _, id := range missing {
			c := canonicalByID[id]
			value, ok := t.category(c, memo, source, unresolved)
			if !ok {
				continue
			}
			if pcat.ProductCategory(value).Depth() != c.Depth() {
				return fmt.Errorf("%s: %s: translation %q has depth %d, canonical has %d", lang, id, value, pcat.ProductCategory(value).Depth(), c.Depth())
			}
			rows = append(rows, [2]string{id.String(), value})
		}

		names := make([]string, 0, len(unresolved))
		for name := range unresolved {
			names = append(names, name)
		}
		sort.Strings(names)

		// Two names under one parent can translate to the same word, and the file would then
		// carry a path twice. That is refused here rather than left for the reader to fail on.
		seen := make(map[string]bool, len(translated.Categories)+len(rows))
		for _, category := range translated.Categories {
			seen[category.String()] = true
		}
		kept := rows[:0]
		var conflicts []string
		for _, row := range rows {
			if seen[row[1]] {
				id, _ := pcat.ProductCategoryIDFromString(row[0])
				conflicts = append(conflicts, row[1]+"  <- "+canonicalByID[id].String())
				continue
			}
			seen[row[1]] = true
			kept = append(kept, row)
		}
		rows = kept

		fmt.Printf("\n== %s: %d rows\n", filename, len(translated.Categories))
		fmt.Printf("   missing %d\n", len(missing))
		for _, key := range sortedKeys(source) {
			fmt.Printf("   from %-12s %d\n", key, source[key])
		}
		fmt.Printf("   unresolved %d\n", len(names))
		for i, name := range names {
			if i >= maxPrint {
				fmt.Printf("      ... and %d more\n", len(names)-maxPrint)
				break
			}
			fmt.Printf("      %s  (e.g. %s)\n", name, unresolved[name])
		}
		fmt.Printf("   duplicate translations %d\n", len(conflicts))
		for i, conflict := range conflicts {
			if i >= maxPrint {
				fmt.Printf("      ... and %d more\n", len(conflicts)-maxPrint)
				break
			}
			fmt.Printf("      %s\n", conflict)
		}
		if len(names) > 0 || len(conflicts) > 0 {
			failed = append(failed, filename)
		}

		if dry || len(rows) == 0 {
			continue
		}
		if err := appendRows(path, rows); err != nil {
			return err
		}
		fmt.Printf("   written %d rows to %s\n", len(rows), path)
	}

	if dry {
		fmt.Println("\ndry: nothing written")
	}
	if len(failed) > 0 {
		return fmt.Errorf("untranslated names in %s", strings.Join(failed, ", "))
	}
	fmt.Println("\nok")
	return nil
}

func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// languages returns the languages to fill: the ones asked for, or every translation file in the directory.
func languages(dir, langsCSV string) ([]Language, error) {
	if langsCSV != "" {
		var langs []Language
		for _, lang := range strings.Split(langsCSV, ",") {
			lang = strings.TrimSpace(lang)
			if lang == "" {
				return nil, errors.New("empty language in -lang")
			}
			if _, err := os.Stat(filepath.Join(dir, translationPrefix+lang+translationSuffix)); err != nil {
				return nil, fmt.Errorf("no file for language %s: %w", lang, err)
			}
			langs = append(langs, lang)
		}
		return langs, nil
	}

	files, err := listTranslationFiles(dir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no translation files in %s", dir)
	}
	langs := make([]Language, 0, len(files))
	for _, file := range files {
		langs = append(langs, strings.TrimSuffix(strings.TrimPrefix(file, translationPrefix), translationSuffix))
	}
	return langs, nil
}

// readGlossary reads lang,parent,english,translation rows.
// parent is the English name of the parent node, empty when the wording holds in every context.
func readGlossary(path string) (map[glossaryKey]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = 4
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s is empty", path)
	}

	glossary := make(map[glossaryKey]string, len(rows))
	for i, row := range rows[1:] {
		key := glossaryKey{lang: row[0], parent: row[1], english: row[2]}
		translation := row[3]
		if key.lang == "" || key.english == "" || translation == "" {
			return nil, fmt.Errorf("%s:%d: lang, english and translation are required", path, i+2)
		}
		if strings.Contains(translation, pcat.Separator) {
			return nil, fmt.Errorf("%s:%d: translation %q contains the separator", path, i+2, translation)
		}
		if _, ok := glossary[key]; ok {
			return nil, fmt.Errorf("%s:%d: duplicate key %s/%s/%s", path, i+2, key.lang, key.parent, key.english)
		}
		glossary[key] = translation
	}
	return glossary, nil
}

// appendRows adds "id,path" lines to a translation file, in the file's own unquoted format.
func appendRows(path string, rows [][2]string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", path, err)
	}
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(row[0])
		b.WriteString(",")
		b.WriteString(row[1])
		b.WriteString("\n")
	}
	if _, err := f.WriteString(b.String()); err != nil {
		f.Close()
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	return f.Close()
}
