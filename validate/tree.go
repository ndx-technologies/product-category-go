package validate

import (
	"errors"
	"strconv"

	pcat "github.com/ndx-technologies/product-category-go"
)

// ErrOrphan is a row whose parent is not a row of its own.
// The path is in the file, but the node it hangs from is not, so it cannot be walked to from a root.
type ErrOrphan struct{ Category, Parent pcat.ProductCategory }

func (e ErrOrphan) Error() string {
	return "orphan: " + e.Category.String() +
		" (parent " + e.Parent.String() + " is not in the tree)"
}

// ErrSingleChild is a run of nodes that each have exactly one child.
// Those levels distinguish nothing: the run could be one node, and a run that reaches
// the deepest allowed level has spent the whole depth budget on one line.
type ErrSingleChild struct {
	Category pcat.ProductCategory // the first node of the run
	Run      int                  // how many nodes in a row have exactly one child
}

func (e ErrSingleChild) Error() string {
	return "single child: " + e.Category.String() +
		" (" + strconv.Itoa(e.Run) + " levels in a row have one child)"
}

// ValidateTree ensures properties of a good tree.
//
//  1. every row hangs from a row that is in the tree;
//  2. no two nodes in a row each have exactly one child — such a run adds a level
//     of path without adding a distinction.
//
// A single node with one child is not reported. Only a run of two or more is,
// since one child under one node is an ordinary shape: "Bottoms" holding only
// "Skirts" is still a level that a reader may want.
func ValidateTree(tree []pcat.ProductCategory) error {
	present := make(map[pcat.ProductCategory]bool, len(tree))
	children := make(map[pcat.ProductCategory]int, len(tree))
	onlyChild := make(map[pcat.ProductCategory]pcat.ProductCategory, len(tree))
	for _, category := range tree {
		present[category] = true
	}

	for _, category := range tree {
		parent := category.Parent()
		if parent.IsZero() {
			continue
		}
		children[parent]++
		onlyChild[parent] = category
	}

	var errs []error

	for _, category := range tree {
		if category.IsZero() {
			continue
		}
		parent := category.Parent()
		if parent.IsZero() {
			continue
		}
		if !present[parent] {
			errs = append(errs, ErrOrphan{Category: category, Parent: parent})
		}
	}

	for _, category := range tree {
		if category.IsZero() || children[category] != 1 {
			continue
		}
		if parent := category.Parent(); !parent.IsZero() && children[parent] == 1 {
			continue
		}

		run, node := 0, category
		for children[node] == 1 {
			run++
			node = onlyChild[node]
		}
		if run >= 2 {
			errs = append(errs, ErrSingleChild{Category: category, Run: run})
		}
	}

	return errors.Join(errs...)
}

type ErrMissing struct {
	ID       pcat.ProductCategoryID
	Category pcat.ProductCategory
}

func (e ErrMissing) Error() string { return "missing: " + e.ID.String() + "  " + e.Category.String() }

type ErrUnknown struct {
	ID       pcat.ProductCategoryID
	Category pcat.ProductCategory
}

func (e ErrUnknown) Error() string { return "unknown: " + e.ID.String() + "  " + e.Category.String() }

// ErrDepth is a row whose path has a different number of segments than the row with the same ID in the other tree.
type ErrDepth struct {
	ID            pcat.ProductCategoryID
	Category      pcat.ProductCategory
	Depth         int
	ExpectedDepth int
}

func (e ErrDepth) Error() string {
	return "depth: " + e.ID.String() + "  " + e.Category.String() +
		" (depth " + strconv.Itoa(e.Depth) + ", want " + strconv.Itoa(e.ExpectedDepth) + ")"
}

// ErrStructure is a row that sits under a different parent than the row with the same ID in the other tree.
type ErrStructure struct {
	ID                               pcat.ProductCategoryID
	Category, Parent, ExpectedParent pcat.ProductCategory
}

func (e ErrStructure) Error() string {
	return "structure: " + e.ID.String() + "  " + e.Category.String() +
		" (parent " + e.Parent.String() + ", want " + e.ExpectedParent.String() + ")"
}

func ValidateIDsMatch(dbA, dbB pcat.ProductCategoryRepositoryCSV) error {
	inA := make(map[pcat.ProductCategoryID]bool, len(dbA.IDs))
	for _, id := range dbA.IDs {
		inA[id] = true
	}

	inB := make(map[pcat.ProductCategoryID]bool, len(dbB.IDs))
	for _, id := range dbB.IDs {
		inB[id] = true
	}

	var errs []error

	for i, id := range dbB.IDs {
		if !inA[id] {
			errs = append(errs, ErrMissing{ID: id, Category: dbB.Categories[i]})
		}
	}

	for i, id := range dbA.IDs {
		if !inB[id] {
			errs = append(errs, ErrUnknown{ID: id, Category: dbA.Categories[i]})
		}
	}

	return errors.Join(errs...)
}

// ValidateTreesMatch ensures that two trees have the same structure.
func ValidateTreesMatch(treeA, treeB pcat.ProductCategoryRepositoryCSV) error {
	// using tree with IDs is conenient, since it gives immediately tree stucture to copmare,
	// which avoids expensive serach for tree structure match on both sides.

	categoryOfB := make(map[pcat.ProductCategoryID]pcat.ProductCategory, len(treeB.IDs))
	idOfB := make(map[pcat.ProductCategory]pcat.ProductCategoryID, len(treeB.IDs))
	for i, id := range treeB.IDs {
		categoryOfB[id] = treeB.Categories[i]
		idOfB[treeB.Categories[i]] = id
	}

	categoryOfA := make(map[pcat.ProductCategoryID]pcat.ProductCategory, len(treeA.IDs))
	for i, id := range treeA.IDs {
		categoryOfA[id] = treeA.Categories[i]
	}

	var errs []error

	for i, id := range treeA.IDs {
		want, ok := categoryOfB[id]
		if !ok {
			continue
		}

		value := treeA.Categories[i]
		if value.IsZero() {
			continue
		}
		if value.Depth() != want.Depth() {
			errs = append(errs, ErrDepth{
				ID:            id,
				Category:      want,
				Depth:         value.Depth(),
				ExpectedDepth: want.Depth(),
			})
			continue
		}

		parentID, ok := idOfB[want.Parent()]
		if !ok {
			continue
		}
		expectedParent, ok := categoryOfA[parentID]
		if !ok {
			continue
		}
		if value.Parent() != expectedParent {
			errs = append(errs, ErrStructure{
				ID:             id,
				Category:       want,
				Parent:         value.Parent(),
				ExpectedParent: expectedParent,
			})
		}
	}

	return errors.Join(errs...)
}
