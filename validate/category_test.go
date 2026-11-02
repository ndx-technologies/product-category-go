package validate

import (
	"errors"
	"strings"
	"testing"

	pcat "github.com/ndx-technologies/product-category-go"
)

func TestValidateProductCategory(t *testing.T) {
	t.Run("not empty", func(t *testing.T) {
		if err := ValidateProductCategory("Alimentos / Lanches"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("no value", func(t *testing.T) {
		err := ValidateProductCategory("")
		if !errors.Is(err, ErrEmpty) {
			t.Fatalf("got %v, want ErrEmpty", err)
		}
	})

	t.Run("an empty node", func(t *testing.T) {
		err := ValidateProductCategory("Food /  / Snacks")

		var emptyNode ErrEmptyNode
		if !errors.As(err, &emptyNode) {
			t.Fatalf("got %v, want ErrEmptyNode", err)
		}
		if emptyNode.Node != 2 {
			t.Error(2, emptyNode.Node)
		}
		if got, want := err.Error(), "empty node: 2"; got != want {
			t.Error(want, got)
		}
	})

	t.Run("a trailing empty node", func(t *testing.T) {
		err := ValidateProductCategory("Food / ")

		var emptyNode ErrEmptyNode
		if !errors.As(err, &emptyNode) {
			t.Fatalf("got %v, want ErrEmptyNode", err)
		}
		if emptyNode.Node != 2 {
			t.Error(2, emptyNode.Node)
		}
	})

	t.Run("a path deeper than the taxonomy allows", func(t *testing.T) {
		err := ValidateProductCategory("A / B / C / D / E")

		var maxDepth ErrMaxDepth
		if !errors.As(err, &maxDepth) {
			t.Fatalf("got %v, want ErrMaxDepth", err)
		}
		if maxDepth.Depth != 5 {
			t.Error(5, maxDepth.Depth)
		}
		if got, want := err.Error(), "max depth: 5 (max 4)"; got != want {
			t.Error(want, got)
		}
	})

	t.Run("a value longer than the taxonomy allows", func(t *testing.T) {
		err := ValidateProductCategory(pcat.ProductCategory(strings.Repeat("a", pcat.MaxLength+1)))

		var maxLength ErrMaxLength
		if !errors.As(err, &maxLength) {
			t.Fatalf("got %v, want ErrMaxLength", err)
		}
		if maxLength.Length != pcat.MaxLength+1 {
			t.Error(pcat.MaxLength+1, maxLength.Length)
		}
		if got, want := err.Error(), "max length: 513 (max 512)"; got != want {
			t.Error(want, got)
		}
	})
}

func TestValidateProductCategories(t *testing.T) {
	t.Run("clean values", func(t *testing.T) {
		vs := []pcat.ProductCategory{"Alimentos / Lanches", "Roupas"}
		if err := ValidateProductCategories(vs); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("every bad value, in order", func(t *testing.T) {
		vs := []pcat.ProductCategory{"", "Lanches", "A /  / B"}
		err := ValidateProductCategories(vs)

		if !errors.Is(err, ErrEmpty) {
			t.Error("want ErrEmpty", err)
		}

		var emptyNode ErrEmptyNode
		if !errors.As(err, &emptyNode) {
			t.Fatalf("got %v, want ErrEmptyNode", err)
		}
		if emptyNode.Node != 2 {
			t.Error(2, emptyNode.Node)
		}

		if got, want := err.Error(), "empty\nempty node: 2"; got != want {
			t.Error(want, got)
		}
	})
}
