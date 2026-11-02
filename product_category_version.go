package pcat

import (
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	productCategoryVersionWeekFactor  = 10_000
	productCategoryVersionYearFactor  = 100 * productCategoryVersionWeekFactor
	ProductCategoryVersionSequenceMax = productCategoryVersionWeekFactor - 1
)

var (
	ErrInvalidProductCategoryVersion           = errors.New("invalid product category version")
	ErrProductCategoryVersionSequenceExhausted = errors.New("product category version sequence exhausted")
	ErrProductCategoryVersionNegative          = errors.New("product category version cannot be negative")
	ErrProductCategoryVersionScan              = errors.New("cannot scan value into ProductCategoryVersion")
)

// ProductCategoryVersion encodes ISO year, ISO week, and within-week sequence into `vYYWWNNNN`.
// It keeps versions short, sortable, and decodable back into UTC ISO week.
type ProductCategoryVersion uint32

func ParseProductCategoryVersion(s string) (ProductCategoryVersion, error) {
	var v ProductCategoryVersion
	if err := v.UnmarshalText([]byte(s)); err != nil {
		return 0, err
	}
	return v, nil
}

func (s ProductCategoryVersion) IsZero() bool { return s == 0 }

func (s ProductCategoryVersion) Value() (driver.Value, error) {
	if s.IsZero() {
		return nil, nil
	}
	return int64(s), nil
}

func (s *ProductCategoryVersion) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*s = 0
	case int64:
		if v < 0 {
			return ErrProductCategoryVersionNegative
		}
		*s = ProductCategoryVersion(v)
	case int32:
		if v < 0 {
			return ErrProductCategoryVersionNegative
		}
		*s = ProductCategoryVersion(v)
	case []byte:
		return s.scanText(string(v))
	case string:
		return s.scanText(v)
	default:
		return ErrProductCategoryVersionScan
	}
	return nil
}

func (s *ProductCategoryVersion) scanText(v string) error {
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return ErrProductCategoryVersionScan
	}
	*s = ProductCategoryVersion(n)
	return nil
}

func NewProductCategoryVersion() ProductCategoryVersion {
	year, week := time.Now().UTC().ISOWeek()
	return ProductCategoryVersion((year%100)*productCategoryVersionYearFactor + week*productCategoryVersionWeekFactor)
}

func (s ProductCategoryVersion) ISOYearWeek() (int, int) {
	v := int(s)
	return 2000 + v/productCategoryVersionYearFactor, (v % productCategoryVersionYearFactor) / productCategoryVersionWeekFactor
}

func (s ProductCategoryVersion) Sequence() int { return int(s) % productCategoryVersionWeekFactor }

func (s ProductCategoryVersion) Next() (ProductCategoryVersion, error) {
	if s.IsZero() {
		return NewProductCategoryVersion(), nil
	}

	base := NewProductCategoryVersion()

	y, w := s.ISOYearWeek()
	if ny, nw := base.ISOYearWeek(); y != ny || w != nw {
		return base, nil
	}

	if s.Sequence() >= ProductCategoryVersionSequenceMax {
		return 0, ErrProductCategoryVersionSequenceExhausted
	}
	return s + 1, nil
}

func (s ProductCategoryVersion) String() string {
	digits := strconv.Itoa(int(s))
	if len(digits) < 8 {
		digits = strings.Repeat("0", 8-len(digits)) + digits
	}
	return "v" + digits
}

func (s ProductCategoryVersion) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

func (s *ProductCategoryVersion) UnmarshalText(text []byte) error {
	if len(text) != 9 || text[0] != 'v' {
		return ErrInvalidProductCategoryVersion
	}

	v, err := strconv.Atoi(string(text[1:]))
	if err != nil {
		return ErrInvalidProductCategoryVersion
	}

	*s = ProductCategoryVersion(v)
	return nil
}
