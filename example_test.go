package merge_test

import (
	"fmt"
	"strings"

	merge "github.com/tylantz/go-tailwind-merge"
)

func Example() {

	// Define the css rules. This can come from a stylesheet.
	rules := `
	.p-1 {
		padding: 0.25rem;
	}
	.p-2 {
		padding: 0.5rem;
	}
	`
	// Create a new resolver with the default configuration
	merger := merge.NewMerger(nil, true)

	// Add the stylesheet to the resolver
	merger.AddRules(strings.NewReader(rules), false)

	// p-2 is defined after p-1, so it would be applied by the browser if both classes were present.
	// But we want p-1 because it is defined later in the string
	fmt.Println(merger.Merge("p-2 p-1"))
	// Output: p-1
}
