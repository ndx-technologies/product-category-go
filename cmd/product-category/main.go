package main

import (
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"slices"
)

type command struct {
	docShort string
	run      func(args []string) error
}

const doc = `Product Category Toolkit

Tools for working with NDX Product Category taxonomy.

## Releasing a New Version

Work in a directory holding categories.txt and the translation files.

1. 'assign-ids -dir <dir>' — write the ID column of every file, 1..N in file order. Run it once, when a file carries no IDs or stale ones. This is the only command that reads row position, and it refuses to run unless every file holds the same number of categories. IDs are reassigned on every 'normalize', so they differ between versions: join on category path, never on ID.
2. Edit categories.txt with the change. Additions, renames and deletions all work, since 'normalize' rebuilds the list from scratch.
3. Apply the same change to every translation file, giving each new category the same ID it carries in categories.txt. Row order does not matter: translations are joined on ID.
4. 'normalize -dir <dir>' — sorts by path, renumbers 1..N, and remaps translations. It fails if any category has no translation; pass '-allow_partial' to write a knowingly incomplete result.
5. 'validate -dir <dir>'
6. 'stats -dir <dir>' — expect full coverage in every file, with Depth, Empty and Missing at zero.
7. 'bump-version -dir <dir>'
8. commit in 'product-category' repo, make new tag and push.sh
`

var commands = map[string]command{
	"stats":        {docShort: DocShortStats, run: StatsCmd},
	"validate":     {docShort: DocShortValidate, run: ValidateCmd},
	"assign-ids":   {docShort: DocShortAssignIDs, run: AssignIDsCmd},
	"normalize":    {docShort: DocShortNormalize, run: NormalizeCmd},
	"bump-version": {docShort: DocShortBumpVersion, run: BumpVersionCmd},
}

func main() {
	cmdNames := slices.Collect(maps.Keys(commands))
	slices.Sort(cmdNames)

	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprint(w, doc)
		fmt.Fprint(w, "Usage:\n\n")
		for _, name := range cmdNames {
			fmt.Fprintf(w, "  %s — %s\n", name, commands[name].docShort)
		}
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	cmd, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		flag.Usage()
		os.Exit(1)
	}
	if err := cmd.run(args[1:]); err != nil {
		log.Fatal(err)
	}
}
