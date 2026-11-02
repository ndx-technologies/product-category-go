package pcat_test

import (
	"bytes"
	_ "embed"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	pcat "github.com/ndx-technologies/product-category-go"
)

//go:embed testdata/categories.txt
var testdataCategories string

func TestProductCategoryRepositoryCSV_ReadTestdata(t *testing.T) {
	var repo pcat.ProductCategoryRepositoryCSV
	if err := repo.Read(strings.NewReader(testdataCategories)); err != nil {
		t.Fatal(err)
	}

	if want := pcat.ProductCategoryVersion(26390002); repo.Version != want {
		t.Errorf("Version = %s, want %s", repo.Version, want)
	}

	if got, want := len(repo.Categories), 3813; got != want {
		t.Fatalf("read %d categories, want %d", got, want)
	}
	if len(repo.IDs) != len(repo.Categories) {
		t.Fatalf("IDs has %d entries, Categories has %d: every row must carry an ID column",
			len(repo.IDs), len(repo.Categories))
	}
	if len(repo.Index) != len(repo.Categories) {
		t.Errorf("Index has %d entries, Categories has %d: the snapshot must not repeat a path",
			len(repo.Index), len(repo.Categories))
	}

	for i, category := range repo.Categories {
		if category.IsZero() {
			t.Fatalf("row %d carries an empty category", i+1)
		}
		if got, want := repo.IDs[i], pcat.ProductCategoryID(i+1); got != want {
			t.Fatalf("IDs[%d] = %d, want %d: the snapshot is numbered 1..N in file order", i, got, want)
		}
		if got := repo.Index[category]; got != repo.IDs[i] {
			t.Fatalf("Index[%q] = %d, want %d", category, got, repo.IDs[i])
		}
		if depth := category.Depth(); depth > pcat.MaxDepth {
			t.Errorf("%q has depth %d, want <= %d", category, depth, pcat.MaxDepth)
		}
	}

	if !slices.IsSorted(repo.Categories) {
		t.Error("categories are not sorted by path")
	}
}

func TestProductCategoryRepositoryCSV_Read(t *testing.T) {
	withID := map[pcat.ProductCategory]pcat.ProductCategoryID{"Food": 1, "Food / Dairy": 2}
	onlyFood := map[pcat.ProductCategory]pcat.ProductCategoryID{"Food": 1}

	tests := []struct {
		name       string
		in         string
		version    pcat.ProductCategoryVersion
		categories []pcat.ProductCategory
		ids        []pcat.ProductCategoryID
		index      map[pcat.ProductCategory]pcat.ProductCategoryID
	}{
		{
			name:       "ID column",
			in:         "# Version: v26010001\n1,Food\n2,Food / Dairy\n",
			version:    26010001,
			categories: []pcat.ProductCategory{"Food", "Food / Dairy"},
			ids:        []pcat.ProductCategoryID{1, 2},
			index:      withID,
		},
		{
			name:       "no ID column is read positionally",
			in:         "Food\nFood / Dairy\n",
			categories: []pcat.ProductCategory{"Food", "Food / Dairy"},
			ids:        []pcat.ProductCategoryID{0, 0},
			index:      map[pcat.ProductCategory]pcat.ProductCategoryID{},
		},
		{
			name:       "comments and blank lines are skipped",
			in:         "# NDX Technologies Product Category\n\n# Version: v26010001\n1,Food\n",
			version:    26010001,
			categories: []pcat.ProductCategory{"Food"},
			ids:        []pcat.ProductCategoryID{1},
			index:      onlyFood,
		},
		{
			name:       "zero ID is read but not indexed",
			in:         "0,Food\n",
			categories: []pcat.ProductCategory{"Food"},
			ids:        []pcat.ProductCategoryID{0},
			index:      map[pcat.ProductCategory]pcat.ProductCategoryID{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var repo pcat.ProductCategoryRepositoryCSV
			if err := repo.Read(strings.NewReader(tc.in)); err != nil {
				t.Fatal(err)
			}

			if repo.Version != tc.version {
				t.Errorf("Version = %s, want %s", repo.Version, tc.version)
			}
			if !slices.Equal(repo.Categories, tc.categories) {
				t.Errorf("Categories = %q, want %q", repo.Categories, tc.categories)
			}
			if !slices.Equal(repo.IDs, tc.ids) {
				t.Errorf("IDs = %v, want %v", repo.IDs, tc.ids)
			}
			if !maps.Equal(repo.Index, tc.index) {
				t.Errorf("Index = %v, want %v", repo.Index, tc.index)
			}
		})
	}
}

func TestProductCategoryRepositoryCSV_ReadVersion(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		version pcat.ProductCategoryVersion
		want    error
	}{
		{
			name:    "version header",
			in:      "# Version: v26010001\n1,Food\n",
			version: 26010001,
		},
		{
			name: "version without the v prefix",
			in:   "# Version: 26010001\n1,Food\n",
			want: pcat.ErrInvalidProductCategoryVersion,
		},
		{
			name: "version that is not a number",
			in:   "# Version: garbage\n1,Food\n",
			want: pcat.ErrInvalidProductCategoryVersion,
		},
		{
			name: "no version header",
			in:   "1,Food\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var repo pcat.ProductCategoryRepositoryCSV
			err := repo.Read(strings.NewReader(tc.in))

			if tc.want != nil {
				if !errors.Is(err, tc.want) {
					t.Fatalf("Read() = %v, want %v", err, tc.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if repo.Version != tc.version {
				t.Errorf("Version = %s, want %s", repo.Version, tc.version)
			}
		})
	}
}

func TestProductCategoryRepositoryCSV_ReadDuplicate(t *testing.T) {
	t.Run("an ID carried by two rows", func(t *testing.T) {
		var repo pcat.ProductCategoryRepositoryCSV
		err := repo.Read(strings.NewReader("1,Food\n1,Drinks\n"))

		want := pcat.ErrDuplicateProductCategoryID{ID: 1}
		got, ok := errors.AsType[pcat.ErrDuplicateProductCategoryID](err)
		if !ok {
			t.Fatalf("Read() = %v, want %+v", err, want)
		}
		if got != want {
			t.Errorf("Read() = %+v, want %+v", got, want)
		}
	})

	t.Run("a name carried by two rows", func(t *testing.T) {
		var repo pcat.ProductCategoryRepositoryCSV
		err := repo.Read(strings.NewReader("1,Food\n2,Food\n"))

		want := pcat.ErrDuplicateProductCategory{Category: "Food"}
		got, ok := errors.AsType[pcat.ErrDuplicateProductCategory](err)
		if !ok {
			t.Fatalf("Read() = %v, want %+v", err, want)
		}
		if got != want {
			t.Errorf("Read() = %+v, want %+v", got, want)
		}
	})

	t.Run("a name carried by two rows without an ID column", func(t *testing.T) {
		var repo pcat.ProductCategoryRepositoryCSV
		err := repo.Read(strings.NewReader("Food\nFood\n"))

		want := pcat.ErrDuplicateProductCategory{Category: "Food"}
		got, ok := errors.AsType[pcat.ErrDuplicateProductCategory](err)
		if !ok {
			t.Fatalf("Read() = %v, want %+v", err, want)
		}
		if got != want {
			t.Errorf("Read() = %+v, want %+v", got, want)
		}
	})

	t.Run("unique rows", func(t *testing.T) {
		var repo pcat.ProductCategoryRepositoryCSV
		if err := repo.Read(strings.NewReader("1,Food\n2,Drinks\n")); err != nil {
			t.Fatal(err)
		}
	})
}

func TestProductCategoryRepositoryCSV_ReadResets(t *testing.T) {
	var repo pcat.ProductCategoryRepositoryCSV
	if err := repo.Read(strings.NewReader("# Version: v26010001\n1,Food\n2,Food / Dairy\n")); err != nil {
		t.Fatal(err)
	}
	if err := repo.Read(strings.NewReader("1,Drinks\n")); err != nil {
		t.Fatal(err)
	}

	if want := []pcat.ProductCategory{"Drinks"}; !slices.Equal(repo.Categories, want) {
		t.Errorf("Categories = %q, want %q: a re-read must not append to the previous file", repo.Categories, want)
	}
	if want := []pcat.ProductCategoryID{1}; !slices.Equal(repo.IDs, want) {
		t.Errorf("IDs = %v, want %v", repo.IDs, want)
	}
	if _, ok := repo.Index["Food"]; ok {
		t.Error(`Index still carries "Food" from the previous file`)
	}
	if !repo.Version.IsZero() {
		t.Errorf("Version = %s, want zero: the second file carries no version header", repo.Version)
	}
}

func TestProductCategoryRepositoryCSV_Write(t *testing.T) {
	t.Run("writes the header and one id,category row per line", func(t *testing.T) {
		repo := pcat.ProductCategoryRepositoryCSV{
			Version:    26010001,
			Categories: []pcat.ProductCategory{"Food", "Food / Dairy"},
			IDs:        []pcat.ProductCategoryID{1, 2},
			Index:      map[pcat.ProductCategory]pcat.ProductCategoryID{"Food": 1, "Food / Dairy": 2},
		}

		want := "# NDX Technologies Product Category\n" +
			"# Version: v26010001\n" +
			"1,Food\n" +
			"2,Food / Dairy\n"

		var buf bytes.Buffer
		if err := repo.Write(&buf); err != nil {
			t.Fatal(err)
		}
		if got := buf.String(); got != want {
			t.Errorf("Write() =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("Index is the fallback when no ID column is carried", func(t *testing.T) {
		repo := pcat.ProductCategoryRepositoryCSV{
			Categories: []pcat.ProductCategory{"Food"},
			Index:      map[pcat.ProductCategory]pcat.ProductCategoryID{"Food": 7},
		}

		var buf bytes.Buffer
		if err := repo.Write(&buf); err != nil {
			t.Fatal(err)
		}
		if want := "# NDX Technologies Product Category\n# Version: v00000000\n7,Food\n"; buf.String() != want {
			t.Errorf("Write() = %q, want %q", buf.String(), want)
		}
	})

	t.Run("ID column wins over Index", func(t *testing.T) {
		repo := pcat.ProductCategoryRepositoryCSV{
			Categories: []pcat.ProductCategory{"Food"},
			IDs:        []pcat.ProductCategoryID{3},
			Index:      map[pcat.ProductCategory]pcat.ProductCategoryID{"Food": 7},
		}

		var buf bytes.Buffer
		if err := repo.Write(&buf); err != nil {
			t.Fatal(err)
		}
		if want := "# NDX Technologies Product Category\n# Version: v00000000\n3,Food\n"; buf.String() != want {
			t.Errorf("Write() = %q, want %q", buf.String(), want)
		}
	})

	t.Run("a row with no ID is written without a comma", func(t *testing.T) {
		repo := pcat.ProductCategoryRepositoryCSV{
			Categories: []pcat.ProductCategory{"Food"},
		}

		var buf bytes.Buffer
		if err := repo.Write(&buf); err != nil {
			t.Fatal(err)
		}
		if want := "# NDX Technologies Product Category\n# Version: v00000000\nFood\n"; buf.String() != want {
			t.Errorf("Write() = %q, want %q", buf.String(), want)
		}
	})
}

func TestProductCategoryRepositoryCSV_RoundTripTestdata(t *testing.T) {
	var repo pcat.ProductCategoryRepositoryCSV
	if err := repo.Read(strings.NewReader(testdataCategories)); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := repo.Write(&buf); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(buf.Bytes(), []byte(testdataCategories)) {
		t.Fatalf("writing the taxonomy back does not reproduce testdata/categories.txt byte for byte: got %d bytes, want %d",
			buf.Len(), len(testdataCategories))
	}
}
