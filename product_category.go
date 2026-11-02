package pcat

import (
	"database/sql/driver"
	"errors"
	"slices"
	"strconv"
	"strings"
)

const (
	MaxDepth           = 4
	MaxLength          = 512
	Separator          = "/"
	separatorWithSpace = " / "
)

// ProductCategoryID identifies a category within a taxonomy.
// This value may change between different taxonomy versions.
type ProductCategoryID int

var (
	ErrProductCategoryIDScan = errors.New("cannot scan value into ProductCategoryID")
	ErrProductCategoryScan   = errors.New("cannot scan value into ProductCategory")
)

func (s ProductCategoryID) IsZero() bool { return s == 0 }

func (s ProductCategoryID) String() string { return strconv.Itoa(int(s)) }

func (s ProductCategoryID) Value() (driver.Value, error) {
	if s.IsZero() {
		return nil, nil
	}
	return int64(s), nil
}

func (s *ProductCategoryID) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*s = 0
	case int64:
		*s = ProductCategoryID(v)
	case int32:
		*s = ProductCategoryID(v)
	case int16:
		*s = ProductCategoryID(v)
	case []byte:
		return s.scanText(string(v))
	case string:
		return s.scanText(v)
	default:
		return ErrProductCategoryIDScan
	}
	return nil
}

func (s *ProductCategoryID) scanText(v string) error {
	n, err := strconv.Atoi(v)
	if err != nil {
		return ErrProductCategoryIDScan
	}
	*s = ProductCategoryID(n)
	return nil
}

func ProductCategoryIDFromString(s string) (ProductCategoryID, error) {
	idInt, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return ProductCategoryID(idInt), nil
}

type ProductCategory string

func ProductCategoryFromString(s string) (ProductCategory, error) { return ProductCategory(s), nil }

func (s ProductCategory) IsZero() bool { return s == "" }

func (s ProductCategory) String() string { return string(s) }

func (s ProductCategory) Value() (driver.Value, error) {
	if s.IsZero() {
		return nil, nil
	}
	return string(s), nil
}

func (s *ProductCategory) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*s = ""
	case string:
		*s = ProductCategory(v)
	case []byte:
		*s = ProductCategory(v)
	default:
		return ErrProductCategoryScan
	}
	return nil
}

func (s ProductCategory) MarshalText() ([]byte, error) { return []byte(s), nil }

func (s *ProductCategory) UnmarshalText(b []byte) error {
	if len(b) > MaxLength {
		return errors.New("category length must be <= 512")
	}
	*s = ProductCategory(b)
	return nil
}

func (s ProductCategory) IsChildOf(parent ProductCategory) bool {
	if parent.IsZero() {
		return !s.IsZero() && !strings.Contains(string(s), separatorWithSpace)
	}
	suffix, ok := strings.CutPrefix(string(s), string(parent)+separatorWithSpace)
	return ok && !strings.Contains(suffix, separatorWithSpace)
}

func (s ProductCategory) Depth() int {
	if s.IsZero() {
		return 0
	}
	return strings.Count(string(s), separatorWithSpace) + 1
}

func (s ProductCategory) Path() []ProductCategory {
	if s.IsZero() {
		return nil
	}
	parts := strings.Split(string(s), separatorWithSpace)
	path := make([]ProductCategory, len(parts))
	for i := range parts {
		path[i] = ProductCategory(strings.Join(parts[:i+1], separatorWithSpace))
	}
	return path
}

func (s ProductCategory) PathNodes() []string {
	if s.IsZero() {
		return nil
	}
	return strings.Split(string(s), separatorWithSpace)
}

func (s ProductCategory) Parent() ProductCategory {
	path := s.Path()
	if len(path) < 2 {
		return ""
	}
	return path[len(path)-2]
}

func (s ProductCategory) IsWithin(parent ProductCategory) bool {
	if parent.IsZero() {
		return true
	}
	return slices.Contains(s.Path(), parent)
}

// NormalizeCategory removes typical minor encoding and formatting errors.
func NormalizeCategory(category ProductCategory) ProductCategory {
	if category.IsZero() {
		return category
	}

	s := category.String()
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ">", Separator)
	s = strings.ReplaceAll(s, "\\", Separator)

	var parts []string
	for p := range strings.SplitSeq(s, Separator) {
		if p := strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}

	category, _ = ProductCategoryFromString(strings.Join(parts, separatorWithSpace))
	return category
}

func EnsureIntermediateNodes(categories []ProductCategory) []ProductCategory {
	seen := make(map[ProductCategory]bool)
	var result []ProductCategory

	for _, category := range categories {
		for _, cat := range category.Path() {
			if !seen[cat] {
				seen[cat] = true
				result = append(result, cat)
			}
		}
	}

	return result
}
