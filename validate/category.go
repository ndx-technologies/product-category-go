package validate

import (
	"errors"
	"strconv"
	"strings"

	pcat "github.com/ndx-technologies/product-category-go"
)

var ErrEmpty = errors.New("empty")

// ErrEmptyNode is a value whose path has a segment with nothing in it, such as "Food /  / Snacks".
type ErrEmptyNode struct {
	Node int // 1-based position of the first empty segment
}

func (e ErrEmptyNode) Error() string { return "empty node: " + strconv.Itoa(e.Node) }

type ErrMaxDepth struct{ Depth int }

func (e ErrMaxDepth) Error() string {
	return "max depth: " + strconv.Itoa(e.Depth) + " (max " + strconv.Itoa(pcat.MaxDepth) + ")"
}

type ErrMaxLength struct{ Length int }

func (e ErrMaxLength) Error() string {
	return "max length: " + strconv.Itoa(e.Length) + " (max " + strconv.Itoa(pcat.MaxLength) + ")"
}

func ValidateProductCategory(v pcat.ProductCategory) error {
	if v.IsZero() {
		return ErrEmpty
	}

	var errs []error

	nodes := strings.Split(v.String(), pcat.Separator)
	for i, node := range nodes {
		if strings.TrimSpace(node) == "" {
			errs = append(errs, ErrEmptyNode{Node: i + 1})
			break
		}
	}

	if len(nodes) > pcat.MaxDepth {
		errs = append(errs, ErrMaxDepth{Depth: len(nodes)})
	}

	if len(v) > pcat.MaxLength {
		errs = append(errs, ErrMaxLength{Length: len(v)})
	}

	return errors.Join(errs...)
}

func ValidateProductCategories(vs []pcat.ProductCategory) error {
	var errs []error
	for _, v := range vs {
		errs = append(errs, ValidateProductCategory(v))
	}
	return errors.Join(errs...)
}
