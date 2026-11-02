package pcat

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type ProductCategoryRepositoryCSV struct {
	Categories []ProductCategory
	IDs        []ProductCategoryID
	Version    ProductCategoryVersion
	Index      map[ProductCategory]ProductCategoryID
}

type ErrDuplicateProductCategoryID struct{ ID ProductCategoryID }

func (e ErrDuplicateProductCategoryID) Error() string {
	return "duplicate product category ID: " + e.ID.String()
}

type ErrDuplicateProductCategory struct{ Category ProductCategory }

func (e ErrDuplicateProductCategory) Error() string {
	return "duplicate product category: " + e.Category.String()
}

func (s *ProductCategoryRepositoryCSV) Read(r io.Reader) error {
	scanner := bufio.NewScanner(r)

	s.Categories = s.Categories[:0]
	s.IDs = s.IDs[:0]
	s.Version = 0
	if s.Index == nil {
		s.Index = make(map[ProductCategory]ProductCategoryID)
	} else {
		clear(s.Index)
	}

	seenIDs := make(map[ProductCategoryID]bool)
	seenCategories := make(map[ProductCategory]bool)

	for scanner.Scan() {
		text := scanner.Text()
		if len(text) == 0 {
			continue
		}
		if text[0] == '#' {
			if after, ok := strings.CutPrefix(text, "# Version: "); ok {
				parsed, err := ParseProductCategoryVersion(after)
				if err != nil {
					return fmt.Errorf("# Version: %s: %w", after, err)
				}
				s.Version = parsed
			}
			continue
		}

		var categoryStr string
		var id ProductCategoryID
		if commaIdx := strings.Index(text, ","); commaIdx > 0 {
			idInt, _ := strconv.Atoi(text[:commaIdx])
			id = ProductCategoryID(idInt)
			categoryStr = text[commaIdx+1:]
		} else {
			categoryStr = text
		}

		category, err := ProductCategoryFromString(categoryStr)
		if err != nil {
			return err
		}
		if id > 0 {
			if seenIDs[id] {
				return ErrDuplicateProductCategoryID{ID: id}
			}
			seenIDs[id] = true
			s.Index[category] = id
		}

		if seenCategories[category] {
			return ErrDuplicateProductCategory{Category: category}
		}
		seenCategories[category] = true

		s.Categories = append(s.Categories, category)
		s.IDs = append(s.IDs, id)
	}

	return scanner.Err()
}

func (s ProductCategoryRepositoryCSV) Write(w io.Writer) error {
	if _, err := w.Write([]byte("# NDX Technologies Product Category\n")); err != nil {
		return err
	}
	if _, err := w.Write([]byte("# Version: " + s.Version.String() + "\n")); err != nil {
		return err
	}
	for i, category := range s.Categories {
		id := s.Index[category]
		if len(s.IDs) == len(s.Categories) {
			id = s.IDs[i]
		}

		if id > 0 {
			if _, err := w.Write([]byte(id.String())); err != nil {
				return err
			}
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}

		if _, err := w.Write([]byte(category.String())); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}
