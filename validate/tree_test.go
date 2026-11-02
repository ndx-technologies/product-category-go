package validate

import (
	"errors"
	"testing"

	pcat "github.com/ndx-technologies/product-category-go"
)

func TestValidateTree(t *testing.T) {
	t.Run("a tree that branches", func(t *testing.T) {
		tree := []pcat.ProductCategory{
			"Food",
			"Food / Snacks",
			"Food / Drinks",
			"Food / Snacks / Chips",
		}

		if err := ValidateTree(tree); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("one node with one child is left alone", func(t *testing.T) {
		tree := []pcat.ProductCategory{
			"Apparel",
			"Apparel / Bottoms",
			"Apparel / Bottoms / Skirts",
			"Apparel / Tops",
		}

		if err := ValidateTree(tree); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("a row whose parent is not in the tree", func(t *testing.T) {
		tree := []pcat.ProductCategory{
			"Food",
			"Food / Snacks / Chips",
		}

		err := ValidateTree(tree)

		var orphan ErrOrphan
		if !errors.As(err, &orphan) {
			t.Fatalf("got %v, want ErrOrphan", err)
		}
		if orphan.Category != "Food / Snacks / Chips" || orphan.Parent != "Food / Snacks" {
			t.Error("orphan Food / Snacks / Chips under Food / Snacks", orphan)
		}
		if got, want := err.Error(), "orphan: Food / Snacks / Chips (parent Food / Snacks is not in the tree)"; got != want {
			t.Error(want, got)
		}
	})

	t.Run("two nodes in a row with one child each", func(t *testing.T) {
		tree := []pcat.ProductCategory{
			"Food",
			"Food / Snacks",
			"Food / Snacks / Chocolate",
		}

		err := ValidateTree(tree)

		var singleChild ErrSingleChild
		if !errors.As(err, &singleChild) {
			t.Fatalf("got %v, want ErrSingleChild", err)
		}
		if singleChild.Category != "Food" || singleChild.Run != 2 {
			t.Error("run of 2 starting at Food", singleChild)
		}
		if got, want := err.Error(), "single child: Food (2 levels in a row have one child)"; got != want {
			t.Error(want, got)
		}
	})

	t.Run("a run is reported once, at its first node", func(t *testing.T) {
		tree := []pcat.ProductCategory{
			"Food",
			"Food / Snacks",
			"Food / Snacks / Chocolate",
			"Food / Snacks / Chocolate / Dark",
		}

		err := ValidateTree(tree)

		var singleChild ErrSingleChild
		if !errors.As(err, &singleChild) {
			t.Fatalf("got %v, want ErrSingleChild", err)
		}
		if singleChild.Run != 3 {
			t.Error(3, singleChild.Run)
		}
		if got, want := err.Error(), "single child: Food (3 levels in a row have one child)"; got != want {
			t.Error(want, got)
		}
	})
}

func TestValidateIDsMatch(t *testing.T) {
	treeB := pcat.ProductCategoryRepositoryCSV{
		IDs: []pcat.ProductCategoryID{1, 2, 3, 4, 5},
		Categories: []pcat.ProductCategory{
			"Food",
			"Food / Snacks",
			"Food / Drinks",
			"Home",
			"Home / Kitchen",
		},
	}

	t.Run("the same set of IDs, whatever the values", func(t *testing.T) {
		treeA := pcat.ProductCategoryRepositoryCSV{
			IDs:        []pcat.ProductCategoryID{5, 4, 3, 2, 1},
			Categories: []pcat.ProductCategory{"Rent", "Casa", "Mistério", "Lanches", "Alimentos"},
		}

		if err := ValidateIDsMatch(treeA, treeB); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("an ID treeA does not carry", func(t *testing.T) {
		treeA := pcat.ProductCategoryRepositoryCSV{
			IDs:        []pcat.ProductCategoryID{1, 2, 4, 5},
			Categories: []pcat.ProductCategory{"Alimentos", "Alimentos / Lanches", "Casa", "Casa / Cozinha"},
		}

		err := ValidateIDsMatch(treeA, treeB)

		var missing ErrMissing
		if !errors.As(err, &missing) {
			t.Fatalf("got %v, want ErrMissing", err)
		}
		if missing.ID != 3 || missing.Category != "Food / Drinks" {
			t.Error("missing 3 Food / Drinks", missing)
		}
		if got, want := err.Error(), "missing: 3  Food / Drinks"; got != want {
			t.Error(want, got)
		}
	})

	t.Run("an ID treeB does not carry", func(t *testing.T) {
		treeA := pcat.ProductCategoryRepositoryCSV{
			IDs:        []pcat.ProductCategoryID{1, 2, 3, 4, 5, 9},
			Categories: []pcat.ProductCategory{"Alimentos", "Alimentos / Lanches", "Alimentos / Bebidas", "Casa", "Casa / Cozinha", "Mistério"},
		}

		err := ValidateIDsMatch(treeA, treeB)

		var unknown ErrUnknown
		if !errors.As(err, &unknown) {
			t.Fatalf("got %v, want ErrUnknown", err)
		}
		if unknown.ID != 9 || unknown.Category != "Mistério" {
			t.Error("unknown 9 Mistério", unknown)
		}
		if got, want := err.Error(), "unknown: 9  Mistério"; got != want {
			t.Error(want, got)
		}
	})
}

func TestValidateTreesMatch(t *testing.T) {
	treeB := pcat.ProductCategoryRepositoryCSV{
		IDs: []pcat.ProductCategoryID{1, 2, 3, 4, 5},
		Categories: []pcat.ProductCategory{
			"Food",
			"Food / Snacks",
			"Food / Drinks",
			"Home",
			"Home / Kitchen",
		},
	}

	t.Run("the same taxonomy translated", func(t *testing.T) {
		treeA := pcat.ProductCategoryRepositoryCSV{
			IDs: []pcat.ProductCategoryID{1, 2, 3, 4, 5},
			Categories: []pcat.ProductCategory{
				"Alimentos",
				"Alimentos / Lanches",
				"Alimentos / Bebidas",
				"Casa",
				"Casa / Cozinha",
			},
		}

		if err := ValidateTreesMatch(treeA, treeB); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("a row with fewer segments than the row it pairs with", func(t *testing.T) {
		treeA := pcat.ProductCategoryRepositoryCSV{
			IDs:        []pcat.ProductCategoryID{1, 2, 3, 4, 5},
			Categories: []pcat.ProductCategory{"Alimentos", "Alimentos / Lanches", "Bebidas", "Casa", "Casa / Cozinha"},
		}

		err := ValidateTreesMatch(treeA, treeB)

		var depth ErrDepth
		if !errors.As(err, &depth) {
			t.Fatalf("got %v, want ErrDepth", err)
		}
		if depth.ID != 3 || depth.Depth != 1 || depth.ExpectedDepth != 2 {
			t.Error("depth 1 want 2 for 3", depth)
		}
		if got, want := err.Error(), "depth: 3  Food / Drinks (depth 1, want 2)"; got != want {
			t.Error(want, got)
		}
	})

	t.Run("a row under the wrong parent", func(t *testing.T) {
		treeA := pcat.ProductCategoryRepositoryCSV{
			IDs:        []pcat.ProductCategoryID{1, 2, 3, 4, 5},
			Categories: []pcat.ProductCategory{"Alimentos", "Alimentos / Lanches", "Casa / Bebidas", "Casa", "Casa / Cozinha"},
		}

		err := ValidateTreesMatch(treeA, treeB)

		var structure ErrStructure
		if !errors.As(err, &structure) {
			t.Fatalf("got %v, want ErrStructure", err)
		}
		if structure.ID != 3 || structure.Parent != "Casa" || structure.ExpectedParent != "Alimentos" {
			t.Error("structure 3 parent Casa want Alimentos", structure)
		}
		if got, want := err.Error(), "structure: 3  Food / Drinks (parent Casa, want Alimentos)"; got != want {
			t.Error(want, got)
		}
	})
}
