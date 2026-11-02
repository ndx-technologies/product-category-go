package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/ndx-technologies/fmtx"
	pcat "github.com/ndx-technologies/product-category-go"
)

const DocShortStats = "report translation coverage"

func StatsCmd(args []string) error {
	var dir string
	var subtree pcat.ProductCategory
	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), DocShortStats, "\n\n")
		fs.PrintDefaults()
	}
	fs.StringVar(&dir, "dir", ".", "directory with category files")
	fs.TextVar(&subtree, "subtree", subtree, "report one category and everything under it")
	if err := fs.Parse(args); err != nil {
		return err
	}

	path := filepath.Join(dir, canonicalFilename)
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", path, err)
	}
	var canonical pcat.ProductCategoryRepositoryCSV
	err = canonical.Read(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", path, err)
	}
	canonicalIDs := make(map[pcat.ProductCategoryID]bool, len(canonical.Categories))
	for _, id := range canonical.IDs {
		if id > 0 {
			canonicalIDs[id] = true
		}
	}
	if len(canonicalIDs) == 0 {
		return fmt.Errorf("%s has no categories", canonicalFilename)
	}

	canonicalByID := make(map[pcat.ProductCategoryID]pcat.ProductCategory, len(canonical.Categories))
	for i, id := range canonical.IDs {
		if id > 0 {
			canonicalByID[id] = canonical.Categories[i]
		}
	}

	firstDepth := 1
	if !subtree.IsZero() {
		kept := canonical.Categories[:0]
		for _, category := range canonical.Categories {
			if category.IsWithin(subtree) {
				kept = append(kept, category)
			}
		}
		if len(kept) == 0 {
			return fmt.Errorf("no categories under %q", subtree)
		}
		canonical.Categories = kept
		firstDepth = subtree.Depth()
	}

	children := make(map[pcat.ProductCategory]int, len(canonical.Categories))
	byDepth := make(map[int]int, pcat.MaxDepth)
	var singleChild int
	var deepest, longest pcat.ProductCategory
	for _, category := range canonical.Categories {
		byDepth[category.Depth()]++
		if parent := category.Parent(); !parent.IsZero() {
			children[parent]++
		}
		if category.Depth() > deepest.Depth() {
			deepest = category
		}
		if len(category) > len(longest) {
			longest = category
		}
	}
	leavesByDepth := make(map[int]int, pcat.MaxDepth)
	for _, category := range canonical.Categories {
		if children[category] == 0 {
			leavesByDepth[category.Depth()]++
		}
	}
	for _, count := range children {
		if count == 1 {
			singleChild++
		}
	}
	leaves := len(canonical.Categories) - len(children)

	subtreeNodes := make(map[pcat.ProductCategory]int, len(canonical.Categories))
	for _, category := range canonical.Categories {
		for _, ancestor := range category.Path() {
			subtreeNodes[ancestor]++
		}
	}

	sizesByDepth := make(map[int][]int, pcat.MaxDepth)
	for _, category := range canonical.Categories {
		depth := category.Depth()
		sizesByDepth[depth] = append(sizesByDepth[depth], subtreeNodes[category])
	}
	giniByDepth := make(map[int]float64, pcat.MaxDepth)
	for depth, sizes := range sizesByDepth {
		slices.Sort(sizes)
		var sum, weighted float64
		for i, size := range sizes {
			sum += float64(size)
			weighted += float64((i + 1) * size)
		}
		if n := len(sizes); n > 0 && sum > 0 {
			giniByDepth[depth] = 2*weighted/(float64(n)*sum) - float64(n+1)/float64(n)
		}
	}

	sackin, depthSquares, leafDepths := 0, 0, 0
	for _, category := range canonical.Categories {
		if children[category] != 0 {
			continue
		}
		depth := category.Depth()
		sackin += depth
		depthSquares += depth * depth
		leafDepths++
	}
	leafDepthAvg, leafDepthVar := 0.0, 0.0
	if leafDepths > 0 {
		leafDepthAvg = float64(sackin) / float64(leafDepths)
		leafDepthVar = float64(depthSquares)/float64(leafDepths) - leafDepthAvg*leafDepthAvg
		if leafDepthVar < 0 {
			leafDepthVar = 0
		}
	}

	fanOutTotal, fanOutSquares, fanOutNodes := 0, 0, 0
	for _, count := range children {
		fanOutTotal += count
		fanOutSquares += count * count
		fanOutNodes++
	}
	fanOutStd, fanOutMean := 0.0, 0.0
	if fanOutNodes > 0 {
		fanOutMean = float64(fanOutTotal) / float64(fanOutNodes)
		fanOutStd = math.Sqrt(float64(fanOutSquares)/float64(fanOutNodes) - fanOutMean*fanOutMean)
	}

	childMin := make(map[int]int, pcat.MaxDepth)
	childMax := make(map[int]int, pcat.MaxDepth)
	childSum := make(map[int]int, pcat.MaxDepth)
	childSquares := make(map[int]int, pcat.MaxDepth)
	childNodes := make(map[int]int, pcat.MaxDepth)
	for _, category := range canonical.Categories {
		count := children[category]
		if count == 0 {
			continue
		}
		depth := category.Depth()
		childSum[depth] += count
		childSquares[depth] += count * count
		childNodes[depth]++
		if min, ok := childMin[depth]; !ok || count < min {
			childMin[depth] = count
		}
		if count > childMax[depth] {
			childMax[depth] = count
		}
	}

	subtreeLeaves := make(map[pcat.ProductCategory]int, len(canonical.Categories))
	for _, category := range canonical.Categories {
		if children[category] != 0 {
			continue
		}
		for _, ancestor := range category.Path() {
			subtreeLeaves[ancestor]++
		}
	}

	leafMin := make(map[int]int, pcat.MaxDepth)
	leafMax := make(map[int]int, pcat.MaxDepth)
	leafSum := make(map[int]int, pcat.MaxDepth)
	leafNodes := make(map[int]int, pcat.MaxDepth)
	for _, category := range canonical.Categories {
		depth := category.Depth()
		count := subtreeLeaves[category]
		leafSum[depth] += count
		leafNodes[depth]++
		if min, ok := leafMin[depth]; !ok || count < min {
			leafMin[depth] = count
		}
		if count > leafMax[depth] {
			leafMax[depth] = count
		}
	}

	fmt.Printf("canonical: %d categories\n\n", len(canonical.Categories))

	depthTw := fmtx.TableWriter{
		Out: os.Stdout,
		Cols: []fmtx.TablCol{
			{Header: "Depth", Width: 5, Alignment: fmtx.AlignRight},
			{Header: "Nodes", Width: 5, Alignment: fmtx.AlignRight},
			{Header: "Leaves", Width: 6, Alignment: fmtx.AlignRight},
			{Header: "Leaf%", Width: 5, Alignment: fmtx.AlignRight},
			{Header: "ChMin", Width: 5, Alignment: fmtx.AlignRight},
			{Header: "ChMax", Width: 5, Alignment: fmtx.AlignRight},
			{Header: "ChAvg", Width: 5, Alignment: fmtx.AlignRight},
			{Header: "ChStd", Width: 5, Alignment: fmtx.AlignRight},
			{Header: "LfMin", Width: 5, Alignment: fmtx.AlignRight},
			{Header: "LfMax", Width: 5, Alignment: fmtx.AlignRight},
			{Header: "LfAvg", Width: 5, Alignment: fmtx.AlignRight},
			{Header: "Gini", Width: 5, Alignment: fmtx.AlignRight},
		},
	}
	depthTw.WriteHeader()
	depthTw.WriteHeaderLine()
	for depth := firstDepth; depth <= pcat.MaxDepth; depth++ {
		var childAvg, leafAvg float64
		if count := childNodes[depth]; count > 0 {
			childAvg = float64(childSum[depth]) / float64(count)
		}
		if count := leafNodes[depth]; count > 0 {
			leafAvg = float64(leafSum[depth]) / float64(count)
		}
		childStd := 0.0
		if count := childNodes[depth]; count > 0 {
			variance := float64(childSquares[depth])/float64(count) - childAvg*childAvg
			if variance < 0 {
				variance = 0
			}
			childStd = math.Sqrt(variance)
		}
		stdCell := fmt.Sprintf("%.1f", childStd)
		switch {
		case childStd > childAvg:
			stdCell = fmtx.RedS(stdCell)
		case childStd > childAvg/2:
			stdCell = fmtx.YellowS(stdCell)
		default:
			stdCell = fmtx.GreenS(stdCell)
		}
		leafPercent := 0.0
		if count := byDepth[depth]; count > 0 {
			leafPercent = 100 * float64(leavesByDepth[depth]) / float64(count)
		}
		giniCell := fmt.Sprintf("%.2f", giniByDepth[depth])
		switch {
		case giniByDepth[depth] > 0.6:
			giniCell = fmtx.RedS(giniCell)
		case giniByDepth[depth] > 0.4:
			giniCell = fmtx.YellowS(giniCell)
		default:
			giniCell = fmtx.GreenS(giniCell)
		}
		depthTw.WriteRow(
			strconv.Itoa(depth),
			strconv.Itoa(byDepth[depth]),
			strconv.Itoa(leavesByDepth[depth]),
			fmt.Sprintf("%.0f%%", leafPercent),
			strconv.Itoa(childMin[depth]),
			strconv.Itoa(childMax[depth]),
			fmt.Sprintf("%.1f", childAvg),
			stdCell,
			strconv.Itoa(leafMin[depth]),
			strconv.Itoa(leafMax[depth]),
			fmt.Sprintf("%.1f", leafAvg),
			giniCell,
		)
	}

	depthVarianceMark := fmtx.GreenS(fmt.Sprintf("%.2f", leafDepthVar))
	if leafDepthVar > 1 {
		depthVarianceMark = fmtx.RedS(fmt.Sprintf("%.2f", leafDepthVar))
	} else if leafDepthVar > 0.5 {
		depthVarianceMark = fmtx.YellowS(fmt.Sprintf("%.2f", leafDepthVar))
	}

	fanOutStdMark := fmtx.GreenS(fmt.Sprintf("%.2f", fanOutStd))
	if fanOutStd > fanOutMean {
		fanOutStdMark = fmtx.RedS(fmt.Sprintf("%.2f", fanOutStd))
	} else if fanOutStd > fanOutMean/2 {
		fanOutStdMark = fmtx.YellowS(fmt.Sprintf("%.2f", fanOutStd))
	}

	fmt.Printf("\nsackin %d, avg leaf depth %.2f, leaf depth variance %s\n", sackin, leafDepthAvg, depthVarianceMark)
	fmt.Printf("fan-out mean %.2f, std %s\n", fanOutMean, fanOutStdMark)

	fmt.Println()
	fmt.Print(`legend
  sackin               sum of leaf depths; avg leaf depth is how many clicks from the top a product sits
  leaf depth variance  spread of those depths; < 0.5 even, > 1.0 some products sit much deeper than others
  fan-out mean/std     children per branching node; std above the mean means there is no typical shape
  gini                 inequality of subtree sizes at that level; 0.2-0.4 balanced, > 0.6 lumpy
`)

	fmt.Printf("\nchildren per node: 0 = %d, 1 = %d, 2+ = %d\n", leaves, singleChild, len(children)-singleChild)
	fmt.Printf("deepest: %s\n", deepest)
	fmt.Printf("longest: %d chars\n\n", len(longest))

	translations, err := listTranslationFiles(dir)
	if err != nil {
		return fmt.Errorf("cannot list translation files: %w", err)
	}

	fileWidth := len("File")
	for _, translation := range translations {
		if len(translation) > fileWidth {
			fileWidth = len(translation)
		}
	}

	tw := fmtx.TableWriter{
		Out: os.Stdout,
		Cols: []fmtx.TablCol{
			{Header: "File", Width: fileWidth},
			{Header: "Coverage", Width: 14, Alignment: fmtx.AlignRight},
			{Header: "Percent", Width: 7, Alignment: fmtx.AlignRight},
			{Header: "Missing", Width: 7, Alignment: fmtx.AlignRight},
			{Header: "Depth", Width: 5, Alignment: fmtx.AlignRight},
			{Header: "Empty", Width: 5, Alignment: fmtx.AlignRight},
			{Header: "Unknown", Width: 8, Alignment: fmtx.AlignRight},
		},
	}
	tw.WriteHeader()
	tw.WriteHeaderLine()

	var noIDs []string

	for _, translation := range translations {
		path := filepath.Join(dir, translation)
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open %s: %w", path, err)
		}
		var repo pcat.ProductCategoryRepositoryCSV
		err = repo.Read(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", path, err)
		}
		if len(repo.IDs) != len(repo.Categories) {
			noIDs = append(noIDs, translation)
		}

		covered, missing, depth, empty, unknown := 0, 0, 0, 0, 0
		seen := make(map[pcat.ProductCategoryID]bool, len(repo.Categories))

		for i, category := range repo.Categories {
			id := repo.IDs[i]
			if id <= 0 {
				continue
			}
			seen[id] = true

			canon, ok := canonicalByID[id]
			if !ok {
				unknown++
				continue
			}

			switch {
			case category.IsZero():
				empty++
			case category.Depth() != canon.Depth():
				depth++
			default:
				covered++
			}
		}

		for id := range canonicalIDs {
			if !seen[id] {
				missing++
			}
		}

		tw.WriteRow(
			translation,
			strconv.Itoa(covered)+"/"+strconv.Itoa(len(canonicalIDs)),
			fmt.Sprintf("%.1f%%", float64(covered)/float64(len(canonicalIDs))*100),
			strconv.Itoa(missing),
			strconv.Itoa(depth),
			strconv.Itoa(empty),
			strconv.Itoa(unknown),
		)
	}

	for _, name := range noIDs {
		fmt.Printf("\n%s carries no ID column; run assign-ids to establish one.\n", name)
	}

	return nil
}
