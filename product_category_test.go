package pcat_test

import (
	"fmt"
	"slices"
	"testing"

	pcat "github.com/ndx-technologies/product-category-go"
)

func TestProductCategory_IsChildOf(t *testing.T) {
	children := map[pcat.ProductCategory][]pcat.ProductCategory{
		"": {"food", "electronics"},
		"food": {
			"food / fruit",
			"food / dairy",
		},
		"food / fruit": {
			"food / fruit / apple",
			"food / fruit / banana",
		},
		"electronics": {
			"electronics / phones",
			"electronics / laptops",
		},
		"electronics / phones": {
			"electronics / phones / apple",
			"electronics / phones / samsung",
		},
	}
	for parent, children := range children {
		for _, child := range children {
			if !child.IsChildOf(parent) {
				t.Error(child, parent)
			}
		}
	}

	notChildren := []struct{ Parent, Child pcat.ProductCategory }{
		{"food / fruit", "food / dairy"},
		{"food / fruit", "food / fruit / banana / nano-banana"}, // grandchild is not child
		{"food / fruit / apple", "food / dairy"},
		{"food / fruit / apple", "food / fruit / banana"},
		{"electronics / phones", "electronics / laptops"},
		{"electronics / phones / apple", "electronics / phones / samsung"},
	}
	for _, pair := range notChildren {
		if pair.Child.IsChildOf(pair.Parent) {
			t.Error(pair.Child, pair.Parent)
		}
	}
}

func ExampleProductCategory_Path() {
	for _, category := range pcat.ProductCategory("Food / Dairy / Milk").Path() {
		fmt.Println(category)
	}
	// Output:
	// Food
	// Food / Dairy
	// Food / Dairy / Milk
}

func ExampleProductCategory_PathNodes() {
	cat := pcat.ProductCategory("food / fruit / banana / nano-banana")
	fmt.Println(cat.PathNodes())
	// Output: [food fruit banana nano-banana]
}

func TestProductCategory_IsZero(t *testing.T) {
	tests := []struct {
		v    pcat.ProductCategory
		want bool
	}{
		{"", true},
		{"food", false},
		{"food / fruit", false},
	}

	for _, tc := range tests {
		t.Run(string(tc.v), func(t *testing.T) {
			if got := tc.v.IsZero(); got != tc.want {
				t.Errorf("(%q).IsZero() = %v, want %v", tc.v, got, tc.want)
			}
		})
	}
}

func TestProductCategory_UnmarshalText(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		var c pcat.ProductCategory
		if err := c.UnmarshalText([]byte("food / fruit")); err != nil {
			t.Fatal(err)
		}
		if c != "food / fruit" {
			t.Error(c)
		}
	})

	t.Run("too long", func(t *testing.T) {
		var c pcat.ProductCategory
		longString := make([]byte, 513)
		for i := range longString {
			longString[i] = 'a'
		}
		if err := c.UnmarshalText(longString); err == nil {
			t.Error("expected error for long string")
		}
	})
}

func TestProductCategory_Path(t *testing.T) {
	tests := []struct {
		cat  pcat.ProductCategory
		path []pcat.ProductCategory
	}{
		{"", nil},
		{"food", []pcat.ProductCategory{"food"}},
		{"food / dairy", []pcat.ProductCategory{"food", "food / dairy"}},
		{"food / dairy / cheese", []pcat.ProductCategory{"food", "food / dairy", "food / dairy / cheese"}},
		{"a / b / c / d", []pcat.ProductCategory{"a", "a / b", "a / b / c", "a / b / c / d"}},
	}
	for _, tc := range tests {
		t.Run(string(tc.cat), func(t *testing.T) {
			if !slices.Equal(tc.cat.Path(), tc.path) {
				t.Error(tc, tc.cat.Path())
			}
		})
	}
}

func TestProductCategory_Parent(t *testing.T) {
	tests := []struct {
		cat  pcat.ProductCategory
		want pcat.ProductCategory
	}{
		{"", ""},
		{"food", ""},
		{"food / fruit", "food"},
		{"food / fruit / apple", "food / fruit"},
		{"a / b / c / d", "a / b / c"},
	}
	for _, tc := range tests {
		t.Run(string(tc.cat), func(t *testing.T) {
			if got := tc.cat.Parent(); got != tc.want {
				t.Errorf("(%q).Parent() = %q, want %q", tc.cat, got, tc.want)
			}
		})
	}
}

func TestProductCategory_IsWithin(t *testing.T) {
	within := []struct{ Cat, Parent pcat.ProductCategory }{
		{"food", "food"},
		{"food / fruit", "food"},
		{"food / fruit / apple", "food"},
		{"food / fruit / apple", "food / fruit"},
		{"food / fruit / apple", "food / fruit / apple"},
		{"electronics / phones / samsung", "electronics"},
		{"food", ""}, // zero parent matches everything
	}
	for _, pair := range within {
		if !pair.Cat.IsWithin(pair.Parent) {
			t.Error(pair.Cat, "not within", pair.Parent)
		}
	}

	notWithin := []struct{ Cat, Parent pcat.ProductCategory }{
		{"food / fruit", "electronics"},
		{"food / fruit / apple", "food / dairy"},
		{"food / dairy", "food / fruit"},
		{"electronics / phones", "electronics / laptops"},
		{"electronics / phones / apple", "electronics / laptops / thinkpad"},
		{"", "food"}, // empty category is not within any specific category
	}
	for _, pair := range notWithin {
		if pair.Cat.IsWithin(pair.Parent) {
			t.Error(pair.Cat, "within", pair.Parent)
		}
	}
}

func TestNormalizeCategory(t *testing.T) {
	tests := []struct{ v, c pcat.ProductCategory }{
		{"Food / Dairy / Milk", "Food / Dairy / Milk"},
		{"Food/Dairy/Milk", "Food / Dairy / Milk"},
		{"Food\\Dairy\\Milk", "Food / Dairy / Milk"},
		{"Food / Dairy\\Milk", "Food / Dairy / Milk"},
		{"  Food  /  Dairy  /  Milk  ", "Food / Dairy / Milk"},
		{"Food / Dairy / Milk / Whole / Organic", "Food / Dairy / Milk / Whole"},
		{"Food // Dairy / / Milk", "Food / Dairy / Milk"},
		{"Food", "Food"},
		{"Food/Dairy", "Food / Dairy"},
	}

	for _, tc := range tests {
		t.Run(tc.v.String(), func(t *testing.T) {
			result := pcat.NormalizeCategory(tc.v)
			if result != tc.c {
				t.Error(tc.c, result)
			}
		})
	}
}

func TestEnsureIntermediateNodes(t *testing.T) {
	vs := []pcat.ProductCategory{
		"Food / Dairy / Milk",
		"Food / Dairy / Cheese",
		"Food / Bakery",
		"Beverages",
	}

	exp := []pcat.ProductCategory{
		"Food",
		"Food / Dairy",
		"Food / Dairy / Milk",
		"Food / Dairy / Cheese",
		"Food / Bakery",
		"Beverages",
	}

	var categories []pcat.ProductCategory
	for _, s := range vs {
		cat, err := pcat.ProductCategoryFromString(s.String())
		if err != nil {
			t.Error(err)
		}
		categories = append(categories, cat)
	}

	result := pcat.EnsureIntermediateNodes(categories)

	if len(result) != len(exp) {
		t.Fatalf("NormalizeCategoriesWithExpansion() returned %d categories, want %d", len(result), len(exp))
	}

	resultMap := make(map[string]bool)
	for _, cat := range result {
		resultMap[cat.String()] = true
	}

	for _, exp := range exp {
		if !resultMap[exp.String()] {
			t.Errorf("NormalizeCategoriesWithExpansion() missing expected category: %q", exp)
		}
	}
}

func TestNormalizeCategory_Empty(t *testing.T) {
	var zero pcat.ProductCategory
	result := pcat.NormalizeCategory(zero)

	if !result.IsZero() {
		t.Errorf("NormalizeCategory(empty) should return empty category, got %q", result.String())
	}
}
