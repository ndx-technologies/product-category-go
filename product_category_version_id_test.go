package pcat_test

import (
	"testing"

	pcat "github.com/ndx-technologies/product-category-go"
)

func TestParseVersionedProductCategoryID(t *testing.T) {
	v, err := pcat.ParseVersionedProductCategoryID("v26010001-42")
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != pcat.ProductCategoryVersion(26010001) {
		t.Fatal(v)
	}
	if v.ID != pcat.ProductCategoryID(42) {
		t.Fatal(v)
	}
	if v.String() != "v26010001-42" {
		t.Fatal(v.String())
	}

	var unmarshaled pcat.VersionedProductCategoryID
	if err := unmarshaled.UnmarshalText([]byte("v26010001-42")); err != nil {
		t.Fatal(err)
	}
	if unmarshaled != v {
		t.Fatal(unmarshaled)
	}
}

func TestParseVersionedProductCategoryID_Invalid(t *testing.T) {
	for _, tc := range []string{"", "v26010001", "bad-42", "v26010001-bad", "v26010001-42-x"} {
		t.Run(tc, func(t *testing.T) {
			if _, err := pcat.ParseVersionedProductCategoryID(tc); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
