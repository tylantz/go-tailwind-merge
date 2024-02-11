<div align="center">
    <br />
    <a href="https://github.com/tylantz/go-tailwind-merge">
        <img src="assets/go-tailwind-merge-surfer-logo.svg" alt="go-tailwind-merge-surfer-logo" height="300px" />
    </a>
</div>

# go-tailwind-merge

<a href="https://pkg.go.dev/github.com/tylantz/go-tailwind-merge"><img src="https://pkg.go.dev/badge/github.com/tylantz/go-tailwind-merge.svg" alt="Go Reference" /></a>

A utility for resolving CSS class conflicts. Inspired by [dcastil/tailwind-merge](https://github.com/dcastil/tailwind-merge). Useful for tailwind and non-tailwind CSS.

```go
import (
	"fmt"
	"strings"

	merge "github.com/tylantz/go-tailwind-merge"
)

func main() {
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
	// However, we want p-1 because it is defined later in the string
	fmt.Println(merger.Merge("p-2 p-1"))
	// Output: p-1
}
```

## The problem

TLDR: One cannot consistently override Tailwind CSS classes by adding additional class names to the class attribute.

We often build base components like buttons, cards, etc. that we want to customise without creating a whole new component.
Using tailwind alone, one has to recreate the whole component to alter it.
However, some components take <b>many</b> classes to establish a style, and it can be tedious and difficult to maintain different versions of that class list for small variations. For example, check out the class attribute on a rendered [shadcn-ui button](https://ui.shadcn.com/docs/components/button#installation):

```html
<button
  class="inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 bg-primary text-primary-foreground shadow hover:bg-primary/90 h-9 px-4 py-2"
>
  Button
</button>
```

Because tailwind generates very normal stylesheets, and the browser prioritises CSS rules defined later in the stylesheet (when they otherwise have equal specificity), new classes added to the attribute may have no impact on the rendered style.

## The solution

This package! Mostly...

Using this library, one can customise base components by merging the default classes with override classes. However it has some limitations. See below.

## Differences from the JS version

The javascript library analyses class strings lexicographically: it tries to identify tailwind class names and resolve conflicts based on a long set of rules informed by the authors' understanding of how tailwind classes are named and structured.

In contrast, this library parses one or more stylesheets and, instead of identifying common tailwind names, it identifies class conflicts based on the actual rule definitions. This approach allows a user to merge classes from any source, not just tailwind. The drawback is one has to instantiate a Merger struct, give it the stylesheet to parse, and pass it around or use a singleton within a package. In the Go context, this approach makes sense because the same Go server that is serving html is probably also serving the stylesheet(s), and therefore has access to it to parse. It's also pretty fast because there is limited use of regex required and there is no need to recursively walk down the class names in the html.

This package is not optimised, but initial merges take 0.117 miliseconds and subsequent merges using the provided sync.Map-based cache take 21 nanoseconds on a gnarly class list with 31 class names.

```
cpu: 12th Gen Intel(R) Core(TM) i7-1260P
BenchmarkMergeNoCache-16
    9112	    117416 ns/op	   48501 B/op	    1126 allocs/op
BenchmarkMergeMapCache-16
58723353	        20.77 ns/op	       0 B/op	       0 allocs/op
```

## Example

I recommend using a real template library such as [template/html](https://pkg.go.dev/html/template) or [templ](https://github.com/a-h/templ). This is a basic example without one.

```go
import (
	merge "github.com/tylantz/go-tailwind-merge"
)

func button(merger *Merger, content string, class string) string {
	baseClass := "inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 bg-green-500"

	class = merger.Merge(baseClass + class)

	return fmt.Sprintf(`<button type="submit" class="%s">%s</button>`, class, content)
}

func formWithButton(merger *Merger) string {
	button := button(merger, "Submit", "bg-blue-500")

	form := `
	<form action="/submit" method="post">
		<label for="name">Name:</label><br>
		<input type="text" id="name" name="name"><br>
		%s
	</form>`
	return fmt.Sprintf(form, button)
}
```

## Limitations

### Primary limitation

The big one is the point of this package is to short-circuit the cascade provided by CSS, but the cascade considers more than just class names. Using this library, we only have access to the class attribute so we cannot predict how the cascade would prioritize one class over another if a rule is to be applied based on more than one class condition (e.g., #id or div.class). Importantly, if you are using css in the way recommended by the creators of tailwind, this is not an issue.

To illustrate, consider the following:

```css
.class1 .class2 {
  padding: 10px;
}
.class3 {
  padding: 20px;
}
```

```html
<div class="class1">
  <p class="{{ merger.Merge('class2 class3') }}">Content</p>
</div>
```

If the merger encounters "class2 class3" on an element, it will know from the parsed stylesheet that there are two rules with conflicting style definitions, one with a ".class1 .class2" selector and another with a ".class3" selector.

If the p element has a parent with "class1", the browser will prioritise any shared properties defined by the ".class1 .class2" rule because it has greater specificity.

The problem is the merger struct only has access to the class attribute on one element, the \<p> element, so it cannot know that the \<p> is a descendant of a "class1" element because it doesn't have access to the parent node.

The merge algorithm tries to assess conflicting style properties within the context of the situation in which they would be applied. In this example, because class2 is dependent on having a class1 parent and class3 has no depenedenc, the algorithm will keep both classes because class2's dependency cannot be checked and it would have greater specificity if it were met. If there is no class1 parent element, class2 won't be applied by the browser anyway so we end up with the desired behaviour.

### Other limitations

- We treat the class name to rule relationship as one-to-one so if multiple rules use the same class name within it's selector, for instance in a combined selector, the rule defined last in the stylesheet using that class name is considered in the merge algorithm.
  - If using tailwind as recommended by its creators, this should not be an issue.
  - This may create unwanted behaviour when used with a library like daisyui that uses tailwind utilities in very complicated combined selectors that frankly go against the recommended way to use tailwind. I have not tested this.
- There is currently no consideration for the cascade defined by [@layer](https://developer.mozilla.org/en-US/docs/Web/CSS/@layer). If only using tailwind, this does not matter: tailwind has it's own @layer implementation, I think? To be checked.
- Rules that are applied under certain circumstances (at-rules), for example based on screen-size, are only compared with other rules that are applied under the same circumanstances.
  - For instance, if a class is "w-7/12 md:w-1/2 w-full md:w-full", the algorithm resolves "w-7/12" vs. "w-full" and "md:w-1/2" vs. "md:w-full" separately and the resulting class will be "w-full md:w-full".
  - This works well for most standard use cases, but it could potentially cause uncertain behaviour for other at-rules (untested).

## Acknowledgments

- This package uses a modified version of [andybalholm/cascadia](https://github.com/andybalholm/cascadia)'s css selector parser
- It also uses [tdewolff/parse/v2](https://github.com/tdewolff/parse/v2) to parse rule definitions
- And finally, development relied heavily on more than 100 unit tests from [dcastil/tailwind-merge](https://github.com/dcastil/tailwind-merge)

## Todo

- [ ] Remove html parsing from internal/cascadia to drop dependency on net/html
- [ ] Remove unused CSS property elements from internal/props
- [ ] Add support of CSS-native [@layer rule](https://developer.mozilla.org/en-US/docs/Web/CSS/@layer)
