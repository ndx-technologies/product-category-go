package pcat

import (
	"errors"
	"strings"
)

var ErrInvalidVersionedProductCategoryID = errors.New("invalid versioned product category id")

type VersionedProductCategoryID struct {
	Version ProductCategoryVersion
	ID      ProductCategoryID
}

func (s VersionedProductCategoryID) IsZero() bool { return s.Version.IsZero() && s.ID == 0 }

func ParseVersionedProductCategoryID(s string) (VersionedProductCategoryID, error) {
	versionText, idText, ok := strings.Cut(s, "-")
	if !ok {
		return VersionedProductCategoryID{}, ErrInvalidVersionedProductCategoryID
	}
	version, err := ParseProductCategoryVersion(versionText)
	if err != nil {
		return VersionedProductCategoryID{}, err
	}
	id, err := ProductCategoryIDFromString(idText)
	if err != nil {
		return VersionedProductCategoryID{}, err
	}
	return VersionedProductCategoryID{Version: version, ID: id}, nil
}

func (s VersionedProductCategoryID) String() string {
	return s.Version.String() + "-" + s.ID.String()
}

func (s VersionedProductCategoryID) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *VersionedProductCategoryID) UnmarshalText(text []byte) error {
	v, err := ParseVersionedProductCategoryID(string(text))
	if err != nil {
		return err
	}
	*s = v
	return nil
}
