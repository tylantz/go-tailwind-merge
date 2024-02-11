package merge

import (
	"cmp"
	"io"
	"log"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/tylantz/go-tailwind-merge/internal/cascadia"
	"github.com/tylantz/go-tailwind-merge/internal/props"
)

// Cache is an interface for any cache implementation that can be used to store the results of the Merge function.
type Cache interface {
	Get(key string) (string, bool) // Get a value from the cache
	Set(key string, data string)   // Set a value in the cache
	Clear()                        // Clear the cache
}

// Merger is a struct that resolved conflicting css class rules.
type Merger struct {
	mu         sync.Mutex // mutex is only used when adding rules
	rules      map[string]cascadia.CssRule
	cache      Cache
	properties map[string]props.Property
	keepSort   bool // keep the original sort order of the classes
}

// NewMerger creates a new instance of Merger.
// cache is a Cache interface that can be used to store the results of the Merge function.
// keepSort is a boolean value indicating whether to keep the order of the classes in the class list.
// This is useful for debugging, but it is not necessary in production.
// Returns a pointer to the created Merger.
func NewMerger(cache Cache, keepSort bool) *Merger {
	p, err := props.GetProperties()
	if err != nil {
		log.Fatal(err)
	}

	return &Merger{
		rules:      make(map[string]cascadia.CssRule),
		cache:      cache,
		properties: p,
		keepSort:   keepSort,
	}
}

// Rules returns the map of css class rules with class names as keys and CssRule structs as values
func (r *Merger) Rules() map[string]cascadia.CssRule {
	return r.rules
}

// walk walks a selector and returns a slice of component selectors.
// It may return a single selector in a slice or many in a slice.
func walk(selector cascadia.Sel) []cascadia.Sel {
	var selectors []cascadia.Sel
	switch t := selector.(type) {
	case cascadia.CombinedSelector:
		subSelectors := walk(t.First())
		selectors = append(selectors, subSelectors...)
		subSelectors = walk(t.Second())
		selectors = append(selectors, subSelectors...)
	case cascadia.CompoundSelector:
		for _, subSelector := range t.Selectors() {
			subSelectors := walk(subSelector)
			selectors = append(selectors, subSelectors...)
		}
	case cascadia.IsPseudoClassSelector:
		for _, subSelector := range t.Selectors() {
			subSelectors := walk(subSelector)
			selectors = append(selectors, subSelectors...)
		}
	case cascadia.WherePseudoClassSelector:
		for _, subSelector := range t.Selectors() {
			subSelectors := walk(subSelector)
			selectors = append(selectors, subSelectors...)
		}
	default:
		selectors = append(selectors, t)
	}
	return selectors
}

// AddRules adds rules to the Merger from a reader.
// It takes a reader and a boolean value indicating whether the rules are inline.
// Returns an error if the rules could not be parsed.
// If the cache is not nil, it is cleared.
// New rules with the same class will overwrite existing rules.
func (r *Merger) AddRules(reader io.Reader, inline bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cache != nil {
		r.cache.Clear()
	}
	rules, err := cascadia.ExtractRules(reader, inline)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		selectors := walk(rule.Selector)
		for _, selector := range selectors {
			if t, ok := selector.(cascadia.ClassSelector); ok {
				r.rules[t.Class] = rule
			}
		}

	}
	return nil
}

func (r *Merger) getAffectedProps(rule cascadia.CssRule) []string {
	affectedProps := make([]string, 0)
	for _, dec := range rule.Declarations {
		prop, ok := r.properties[dec.Property]
		if !ok {
			// Allowing these through, maybe they shouldn't be but
			// this allows properties like stroke, fill, etc. to be used
			// which are not in the mdn official list of props
			// IMPORTANT: we are relying on this to let custom properties through
			affectedProps = append(affectedProps, dec.Property)
		}
		affectedProps = append(affectedProps, prop.ComputedProps()...)
	}
	return unique(affectedProps)
}

var importantRegex = regexp.MustCompile(`!important`)      // matches !important in a css declaration value
var customVarRegex = regexp.MustCompile(`var\((--[\w-]+)`) // matches custom variables in a css declaration value

func getCustomVarsInDec(dec cascadia.CssDeclaration) []string {
	matches := customVarRegex.FindAllStringSubmatch(dec.Value, -1)
	if matches == nil {
		return nil
	}
	vars := make([]string, 0, len(matches))
	for _, match := range matches {
		vars = append(vars, match[1])
	}
	return vars
}

// classWithPseudo checks if a selector is a compound selector
// with a class and one or more pseudo element or class.
// It returns true if the selector is a class with pseudo elements or classes.
func classWithPseudo(sel cascadia.Sel) bool {
	_, ok := sel.(cascadia.CompoundSelector)
	if !ok {
		return false
	}
	selectors := walk(sel)
	for i, s := range selectors {
		if i == 0 {
			_, firstIsClass := s.(cascadia.ClassSelector)
			if !firstIsClass {
				return false
			}
			continue
		}
		ok := cascadia.IsPseudoElement(s)
		if !ok {
			return false
		}
	}
	return true
}

func isClass(sel cascadia.Sel) bool {
	if _, ok := sel.(cascadia.ClassSelector); ok {
		return true
	}
	return false
}

// Merge resolves conflicting css class rules.
// It takes a string of space-separated class names.
// Returns a string of space-separated class names with the conflicting classes removed.
// It prioritises the last class in the list for each property.
// If a class name is not found in the rules, it is kept in the output.
// Important properties are prioritised over non-important properties.
// If the cache is not nil, it will store the result of the merge to skip re-calculating the merge later.
func (r *Merger) Merge(inClass string) string {
	if r.cache != nil {
		val, ok := r.cache.Get(inClass)
		if ok {
			return val
		}
	}
	split := strings.Split(strings.TrimSpace(inClass), " ")
	if len(split) < 2 {
		return inClass
	}

	keepClasses := make([]string, 0, len(split))
	selectorToClass := make(map[string]string, len(split))

	// propsToClasses is a map of properties that are shared between classes
	// The property name may have a condition (pseudo or media) appended to it (e.g., "height:hover": ["h-10", "h-20"])
	propsToSelectors := make(map[string][]string, len(split))
	importantPropsToSelectors := make(map[string][]string)
	customVarsToSelectors := make(map[string][]string) // map of custom vars to the class that set them
	propsToCustomVars := make(map[string][]string)     // map of props to the custom vars that it uses
	for _, class := range split {
		rule, ok := r.rules[class]
		if !ok {
			// log.Println("rule not found for class:", class)
			keepClasses = append(keepClasses, class)
			continue
		}

		selectorToClass[rule.Selector.String()] = class

		for _, prop := range r.getAffectedProps(rule) {
			// if the rule has a condition (:hover, :focus, etc.), add the condition to the property name
			if isClass(rule.Selector) || classWithPseudo(rule.Selector) {
				prop = prop + rule.GetCondition()
			} else {
				// use the selector with the class name removed to codify what the selector is being applied to
				s := cascadia.CssUnescape([]byte(rule.Selector.String()))
				prop = prop + strings.Replace(s, class, "", 1)
			}

			for _, dec := range rule.Declarations {

				if !strings.HasPrefix(prop, "--") {
					// if the property has a custom var, add it to the propsToCustomVars map
					customVars := getCustomVarsInDec(dec)
					// overwrite the customVars so we prioritize the last class that sets the property
					propsToCustomVars[prop] = customVars
				}

				// if the property is marked !important, add the class to the importantProps map
				if importantRegex.MatchString(dec.Value) {
					importantPropsToSelectors[prop] = append(importantPropsToSelectors[prop], rule.Selector.String())
				}
			}

			// if it has a custom prop, add it to the customVars map
			if strings.HasPrefix(prop, "--") {
				customVarsToSelectors[prop] = append(customVarsToSelectors[prop], rule.Selector.String())
				continue
			}

			// add all classes to the propsToClasses map
			propsToSelectors[prop] = append(propsToSelectors[prop], rule.Selector.String())
		}
	}

	// keep the last class in the list for each property
	// importantly, this keeps classes that uniquely define a property, even if it has properties that conflict with other classes
	for _, selList := range propsToSelectors {
		sel := selList[len(selList)-1]
		class := selectorToClass[sel]
		keepClasses = append(keepClasses, class)
	}

	// If a class has an !important property, it is kept unless another class comes later in the class string and it is marked !important on the same property.
	for _, selList := range importantPropsToSelectors {
		// This does not remove the class that the the important class is overriding,
		// but it shouldn't matter because the important class will override the other,
		// and the other class may have other properties that are not being overridden
		sel := selList[len(selList)-1]
		class := selectorToClass[sel]
		keepClasses = append(keepClasses, class)
	}

	// keep the class that sets the last definition of each custom property if that custom property is actually used
	for custVar, selList := range customVarsToSelectors {
		for _, vars := range propsToCustomVars {
			// if the custom property is actually used, keep the class that sets it
			if !slices.Contains(vars, custVar) {
				continue
			}
			sel := selList[len(selList)-1]
			class := selectorToClass[sel]
			keepClasses = append(keepClasses, class)
			break
		}
	}

	keepClasses = unique(keepClasses)
	if r.keepSort {
		sortSubset(keepClasses, split)
	}
	out := strings.Join(keepClasses, " ")
	if r.cache != nil {
		r.cache.Set(inClass, out)
	}
	return out
}

// unique returns a slice with all duplicate elements removed.
// The sort order of the elements is not preserved
// unless the elements are already sorted. It does not zero pointers so it can leak.
// Lazy? Yes. But it's a good way to remove duplicates from a slice.
func unique[S ~[]E, E cmp.Ordered](x S) S {
	slices.Sort(x)
	return slices.Compact(x)
}

// sortSubset sorts a subset of strings based on the order of the full set of strings.
// if duplicates are present in the full, this function takes the final position for a value (left-to-right).
func sortSubset(sub []string, full []string) {
	indexMap := make(map[string]int, len(full))
	for i, v := range full {
		indexMap[v] = i
	}

	// Sort subset based on the indices in indexMap
	sort.Slice(sub, func(i, j int) bool {
		return indexMap[sub[i]] < indexMap[sub[j]]
	})
}

/*
The CSS properties that can accept multiple values and layer them, rather than overriding them, are:

box-shadow: Applies one or more shadows to an element. Shadows are applied in the order specified and layered on top of each other,
	with the first shadow on top.

text-shadow: Applies one or more shadows to the text content of an element.
	Like box-shadow, shadows are layered with the first shadow on top.

background: When using multiple backgrounds, they are layered atop one another with the first background you provide on top (closest to the viewer)
	and the last one specified underneath.

transform: Multiple transform functions can be applied to an element. They are applied in the order specified, from left to right.
	However, if they are applied by different selectors with the same specificity, the one that comes later in the CSS will override the earlier one.

transition: Multiple transitions can be applied to an element. They are applied in the order specified, from left to right.
	However, like transform, if they are applied by different selectors with the same specificity, the later one will override the earlier one.

animation: Multiple animations can be applied to an element. They are applied in the order specified, from left to right.
	Again, like transform and transition, if they are applied by different selectors with the same specificity, the later one will override the earlier one.
*/
