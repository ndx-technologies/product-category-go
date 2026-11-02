package pcat_test

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"

	pcat "github.com/ndx-technologies/product-category-go"
)

func TestProductCategoryVersion_MarshalText_UnmarshalText(t *testing.T) {
	tests := []struct {
		time    time.Time
		version string
	}{
		{time: time.Date(2026, time.April, 16, 12, 0, 0, 0, time.UTC), version: "v26160000"},
		{time: time.Date(2027, time.January, 1, 12, 0, 0, 0, time.UTC), version: "v26530000"},
	}

	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				delta := time.Until(tc.time)
				if delta < 0 {
					t.Fatalf("target time %v is before bubble time %v", tc.time, time.Now())
				}
				time.Sleep(delta)
				synctest.Wait()

				v := pcat.NewProductCategoryVersion()
				if s := v.String(); s != tc.version {
					t.Error(s)
				}

				b, err := v.MarshalText()
				if err != nil {
					t.Fatal(err)
				}
				if string(b) != tc.version {
					t.Error(string(b))
				}

				var decoded pcat.ProductCategoryVersion
				if err := (&decoded).UnmarshalText(b); err != nil {
					t.Fatal(err)
				}
				if decoded != v {
					t.Error(decoded, v)
				}

				if v, err := pcat.ParseProductCategoryVersion(tc.version); err != nil || v != decoded {
					t.Error(err, v)
				}
			})
		})
	}
}

func TestProductCategoryVersion_Next(t *testing.T) {
	target := time.Date(2026, time.April, 16, 12, 0, 0, 0, time.UTC)

	t.Run("when empty, then start current week", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			delta := time.Until(target)
			if delta < 0 {
				t.Fatalf("target time %v is before bubble time %v", target, time.Now())
			}
			time.Sleep(delta)
			synctest.Wait()

			v, err := pcat.ProductCategoryVersion(0).Next()
			if err != nil {
				t.Fatal(err)
			}
			if s := v.String(); s != "v26160000" {
				t.Error(s)
			}
		})
	})

	t.Run("when same week, then increment sequence", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			delta := time.Until(target)
			if delta < 0 {
				t.Fatalf("target time %v is before bubble time %v", target, time.Now())
			}
			time.Sleep(delta)
			synctest.Wait()

			v, err := pcat.ProductCategoryVersion(26160042).Next()
			if err != nil {
				t.Fatal(err)
			}
			if s := v.String(); s != "v26160043" {
				t.Error(s)
			}
		})
	})

	t.Run("when new week, then reset sequence", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			delta := time.Until(target)
			if delta < 0 {
				t.Fatalf("target time %v is before bubble time %v", target, time.Now())
			}
			time.Sleep(delta)
			synctest.Wait()

			v, err := pcat.ProductCategoryVersion(26150042).Next()
			if err != nil {
				t.Fatal(err)
			}
			if s := v.String(); s != "v26160000" {
				t.Error(s)
			}
		})
	})

	t.Run("when sequence exhausted, then error", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			delta := time.Until(target)
			if delta < 0 {
				t.Fatalf("target time %v is before bubble time %v", target, time.Now())
			}
			time.Sleep(delta)
			synctest.Wait()

			_, err := pcat.ProductCategoryVersion(26169999).Next()
			if !errors.Is(err, pcat.ErrProductCategoryVersionSequenceExhausted) {
				t.Error(err)
			}
		})
	})
}
