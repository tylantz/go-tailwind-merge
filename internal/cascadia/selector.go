// This is based on the cascadia package kept here: github.com/andybalholm/cascadia

/*
Copyright (c) 2011 Andy Balholm. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package cascadia

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

// Matcher is the interface for basic selector functionality.
// Match returns whether a selector matches n.
type Matcher interface {
	Match(n *html.Node) bool
}

// Sel is the interface for all the functionality provided by selectors.
type Sel interface {
	Matcher
	Specificity() Specificity

	// Returns a CSS input compiling to this selector.
	String() string

	// Returns a pseudo-element, or an empty string.
	PseudoElement() string
}

// Parse parses a selector. Use `ParseWithPseudoElement`
// if you need support for pseudo-elements.
func Parse(sel string) (Sel, error) {
	p := &parser{s: sel}
	compiled, err := p.parseSelector()
	if err != nil {
		return nil, err
	}

	if p.i < len(sel) {
		return nil, fmt.Errorf("parsing %q: %d bytes left over", sel, len(sel)-p.i)
	}

	return compiled, nil
}

// ParseWithPseudoElement parses a single selector,
// with support for pseudo-element.
func ParseWithPseudoElement(sel string) (Sel, error) {
	p := &parser{s: sel, acceptPseudoElements: true}
	compiled, err := p.parseSelector()
	if err != nil {
		return nil, err
	}

	if p.i < len(sel) {
		return nil, fmt.Errorf("parsing %q: %d bytes left over", sel, len(sel)-p.i)
	}

	return compiled, nil
}

// ParseGroup parses a selector, or a group of selectors separated by commas.
// Use `ParseGroupWithPseudoElements`
// if you need support for pseudo-elements.
func ParseGroup(sel string) (SelectorGroup, error) {
	p := &parser{s: sel}
	compiled, err := p.parseSelectorGroup()
	if err != nil {
		return nil, err
	}

	if p.i < len(sel) {
		return nil, fmt.Errorf("parsing %q: %d bytes left over", sel, len(sel)-p.i)
	}

	return compiled, nil
}

// ParseGroupWithPseudoElements parses a selector, or a group of selectors separated by commas.
// It supports pseudo-elements.
func ParseGroupWithPseudoElements(sel string) (SelectorGroup, error) {
	p := &parser{s: sel, acceptPseudoElements: true}
	compiled, err := p.parseSelectorGroup()
	if err != nil {
		return nil, err
	}

	if p.i < len(sel) {
		return nil, fmt.Errorf("parsing %q: %d bytes left over", sel, len(sel)-p.i)
	}

	return compiled, nil
}

func queryInto(n *html.Node, m Matcher, storage []*html.Node) []*html.Node {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if m.Match(child) {
			storage = append(storage, child)
		}
		storage = queryInto(child, m, storage)
	}

	return storage
}

// QueryAll returns a slice of all the nodes that match m, from the descendants
// of n.
func QueryAll(n *html.Node, m Matcher) []*html.Node {
	return queryInto(n, m, nil)
}

// Query returns the first node that matches m, from the descendants of n.
// If none matches, it returns nil.
func Query(n *html.Node, m Matcher) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if m.Match(c) {
			return c
		}
		if matched := Query(c, m); matched != nil {
			return matched
		}
	}

	return nil
}

// Filter returns the nodes that match m.
func Filter(nodes []*html.Node, m Matcher) (result []*html.Node) {
	for _, n := range nodes {
		if m.Match(n) {
			result = append(result, n)
		}
	}
	return result
}

type TagSelector struct {
	tag string
}

// Matches elements with a given tag name.
func (t TagSelector) Match(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == t.tag
}

func (c TagSelector) Specificity() Specificity {
	return Specificity{0, 0, 1}
}

func (c TagSelector) PseudoElement() string {
	return ""
}

type ClassSelector struct {
	Class string
}

// Matches elements by class attribute.
func (t ClassSelector) Match(n *html.Node) bool {
	return matchAttribute(n, "class", func(s string) bool {
		return matchInclude(t.Class, s, false)
	})
}

func (c ClassSelector) Specificity() Specificity {
	return Specificity{0, 1, 0}
}

func (c ClassSelector) PseudoElement() string {
	return ""
}

type IdSelector struct {
	id string
}

// Matches elements by id attribute.
func (t IdSelector) Match(n *html.Node) bool {
	return matchAttribute(n, "id", func(s string) bool {
		return s == t.id
	})
}

func (c IdSelector) Specificity() Specificity {
	return Specificity{1, 0, 0}
}

func (c IdSelector) PseudoElement() string {
	return ""
}

type AttrSelector struct {
	key, val, operation string
	regexp              *regexp.Regexp
	insensitive         bool
}

// Matches elements by attribute value.
func (t AttrSelector) Match(n *html.Node) bool {
	switch t.operation {
	case "":
		return matchAttribute(n, t.key, func(string) bool { return true })
	case "=":
		return matchAttribute(n, t.key, func(s string) bool { return matchInsensitiveValue(s, t.val, t.insensitive) })
	case "!=":
		return attributeNotEqualMatch(t.key, t.val, n, t.insensitive)
	case "~=":
		// matches elements where the attribute named key is a whitespace-separated list that includes val.
		return matchAttribute(n, t.key, func(s string) bool { return matchInclude(t.val, s, t.insensitive) })
	case "|=":
		return attributeDashMatch(t.key, t.val, n, t.insensitive)
	case "^=":
		return attributePrefixMatch(t.key, t.val, n, t.insensitive)
	case "$=":
		return attributeSuffixMatch(t.key, t.val, n, t.insensitive)
	case "*=":
		return attributeSubstringMatch(t.key, t.val, n, t.insensitive)
	case "#=":
		return attributeRegexMatch(t.key, t.regexp, n)
	default:
		panic(fmt.Sprintf("unsuported operation : %s", t.operation))
	}
}

// matches elements where we ignore (or not) the case of the attribute value
// the user attribute is the value set by the user to match elements
// the real attribute is the attribute value found in the code parsed
func matchInsensitiveValue(userAttr string, realAttr string, ignoreCase bool) bool {
	if ignoreCase {
		return strings.EqualFold(userAttr, realAttr)
	}
	return userAttr == realAttr

}

// matches elements where the attribute named key satisfies the function f.
func matchAttribute(n *html.Node, key string, f func(string) bool) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for _, a := range n.Attr {
		if a.Key == key && f(a.Val) {
			return true
		}
	}
	return false
}

// attributeNotEqualMatch matches elements where
// the attribute named key does not have the value val.
func attributeNotEqualMatch(key, val string, n *html.Node, ignoreCase bool) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for _, a := range n.Attr {
		if a.Key == key && matchInsensitiveValue(a.Val, val, ignoreCase) {
			return false
		}
	}
	return true
}

// returns true if s is a whitespace-separated list that includes val.
func matchInclude(val string, s string, ignoreCase bool) bool {
	for s != "" {
		i := strings.IndexAny(s, " \t\r\n\f")
		if i == -1 {
			return matchInsensitiveValue(s, val, ignoreCase)
		}
		if matchInsensitiveValue(s[:i], val, ignoreCase) {
			return true
		}
		s = s[i+1:]
	}
	return false
}

// matches elements where the attribute named key equals val or starts with val plus a hyphen.
func attributeDashMatch(key, val string, n *html.Node, ignoreCase bool) bool {
	return matchAttribute(n, key,
		func(s string) bool {
			if matchInsensitiveValue(s, val, ignoreCase) {
				return true
			}
			if len(s) <= len(val) {
				return false
			}
			if matchInsensitiveValue(s[:len(val)], val, ignoreCase) && s[len(val)] == '-' {
				return true
			}
			return false
		})
}

// attributePrefixMatch returns a Selector that matches elements where
// the attribute named key starts with val.
func attributePrefixMatch(key, val string, n *html.Node, ignoreCase bool) bool {
	return matchAttribute(n, key,
		func(s string) bool {
			if strings.TrimSpace(s) == "" {
				return false
			}
			if ignoreCase {
				return strings.HasPrefix(strings.ToLower(s), strings.ToLower(val))
			}
			return strings.HasPrefix(s, val)
		})
}

// attributeSuffixMatch matches elements where
// the attribute named key ends with val.
func attributeSuffixMatch(key, val string, n *html.Node, ignoreCase bool) bool {
	return matchAttribute(n, key,
		func(s string) bool {
			if strings.TrimSpace(s) == "" {
				return false
			}
			if ignoreCase {
				return strings.HasSuffix(strings.ToLower(s), strings.ToLower(val))
			}
			return strings.HasSuffix(s, val)
		})
}

// attributeSubstringMatch matches nodes where
// the attribute named key contains val.
func attributeSubstringMatch(key, val string, n *html.Node, ignoreCase bool) bool {
	return matchAttribute(n, key,
		func(s string) bool {
			if strings.TrimSpace(s) == "" {
				return false
			}
			if ignoreCase {
				return strings.Contains(strings.ToLower(s), strings.ToLower(val))
			}
			return strings.Contains(s, val)
		})
}

// attributeRegexMatch  matches nodes where
// the attribute named key matches the regular expression rx
func attributeRegexMatch(key string, rx *regexp.Regexp, n *html.Node) bool {
	return matchAttribute(n, key,
		func(s string) bool {
			return rx.MatchString(s)
		})
}

func (c AttrSelector) Specificity() Specificity {
	return Specificity{0, 1, 0}
}

func (c AttrSelector) PseudoElement() string {
	return ""
}

// see pseudo_classes.go for pseudo classes selectors

type CompoundSelector struct {
	selectors     []Sel
	pseudoElement string
}

// Matches elements if each sub-selectors matches.
func (t CompoundSelector) Match(n *html.Node) bool {
	if len(t.selectors) == 0 {
		return n.Type == html.ElementNode
	}

	for _, sel := range t.selectors {
		if !sel.Match(n) {
			return false
		}
	}
	return true
}

func (s CompoundSelector) Specificity() Specificity {
	var out Specificity
	for _, sel := range s.selectors {
		out = out.Add(sel.Specificity())
	}
	if s.pseudoElement != "" {
		// https://drafts.csswg.org/selectors-3/#specificity
		out = out.Add(Specificity{0, 0, 1})
	}
	return out
}

func (c CompoundSelector) PseudoElement() string {
	return c.pseudoElement
}

func (c CompoundSelector) PseudoElements() []Sel {
	elements := make([]Sel, 0, len(c.selectors))
	for _, sel := range c.selectors {
		if !IsPseudoElement(sel) {
			continue
		}
		elements = append(elements, sel)
	}
	return elements
}

func (c CompoundSelector) PseudoElementsString() string {
	elements := c.PseudoElements()
	// sorting here because combinations are semantically the same
	// regardless of order (e.g., focus:hover and hover:focus)
	strs := make([]string, 0, len(elements))
	for _, sel := range elements {
		strs = append(strs, sel.String())
	}
	slices.Sort[[]string](strs)
	return strings.Join(strs, "")
}

func (c CompoundSelector) Selectors() []Sel {
	return c.selectors
}

// CombinedSelector is a selector that combines two selectors with a combinator.
// If the selector is more than two parts, the first contains all but the last
// part, and the second contains the last part.
type CombinedSelector struct {
	first      Sel
	combinator byte
	second     Sel
}

func (t CombinedSelector) Match(n *html.Node) bool {
	if t.first == nil {
		return false // maybe we should panic
	}
	switch t.combinator {
	case 0:
		return t.first.Match(n)
	case ' ':
		return descendantMatch(t.first, t.second, n)
	case '>':
		return childMatch(t.first, t.second, n)
	case '+':
		return siblingMatch(t.first, t.second, true, n)
	case '~':
		return siblingMatch(t.first, t.second, false, n)
	default:
		panic("unknown combinator")
	}
}

func (c CombinedSelector) First() Sel {
	return c.first
}

func (c CombinedSelector) Second() Sel {
	return c.second
}

// matches an element if it matches d and has an ancestor that matches a.
func descendantMatch(a, d Matcher, n *html.Node) bool {
	if !d.Match(n) {
		return false
	}

	for p := n.Parent; p != nil; p = p.Parent {
		if a.Match(p) {
			return true
		}
	}

	return false
}

// matches an element if it matches d and its parent matches a.
func childMatch(a, d Matcher, n *html.Node) bool {
	return d.Match(n) && n.Parent != nil && a.Match(n.Parent)
}

// matches an element if it matches s2 and is preceded by an element that matches s1.
// If adjacent is true, the sibling must be immediately before the element.
func siblingMatch(s1, s2 Matcher, adjacent bool, n *html.Node) bool {
	if !s2.Match(n) {
		return false
	}

	if adjacent {
		for n = n.PrevSibling; n != nil; n = n.PrevSibling {
			if n.Type == html.TextNode || n.Type == html.CommentNode {
				continue
			}
			return s1.Match(n)
		}
		return false
	}

	// Walk backwards looking for element that matches s1
	for c := n.PrevSibling; c != nil; c = c.PrevSibling {
		if s1.Match(c) {
			return true
		}
	}

	return false
}

func (s CombinedSelector) Specificity() Specificity {
	spec := s.first.Specificity()
	if s.second != nil {
		spec = spec.Add(s.second.Specificity())
	}
	return spec
}

// on combinedSelector, a pseudo-element only makes sense on the last
// selector, although others increase specificity.
func (c CombinedSelector) PseudoElement() string {
	if c.second == nil {
		return ""
	}
	return c.second.PseudoElement()
}

// A SelectorGroup is a list of selectors, which matches if any of the
// individual selectors matches.
type SelectorGroup []Sel

// Match returns true if the node matches one of the single selectors.
func (s SelectorGroup) Match(n *html.Node) bool {
	for _, sel := range s {
		if sel.Match(n) {
			return true
		}
	}
	return false
}
