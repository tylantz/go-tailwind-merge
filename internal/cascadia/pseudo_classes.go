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
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// This file implements the pseudo classes selectors,
// which share the implementation of PseudoElement() and Specificity()

type abstractPseudoClass struct {
	name string
}

func (s abstractPseudoClass) Match(n *html.Node) bool {
	return false
}

func (s abstractPseudoClass) Specificity() Specificity {
	return Specificity{0, 1, 0}
}

func (c abstractPseudoClass) PseudoElement() string {
	return ""
}

func (c abstractPseudoClass) IsClass(n *html.Node) bool {
	return false
}

type IsPseudoClassSelector struct {
	match SelectorGroup
}

func (c IsPseudoClassSelector) Selectors() []Sel {
	return c.match
}

func (s IsPseudoClassSelector) Match(n *html.Node) bool {
	for _, sel := range s.match {
		if sel.Match(n) {
			return true
		}
	}
	return false
}

func (s IsPseudoClassSelector) Specificity() Specificity {
	// is() takes the specificity of its most specific argument
	var max Specificity
	for _, sel := range s.match {
		newSpe := sel.Specificity()
		if max.Less(newSpe) {
			max = newSpe
		}
	}
	return max
}

func (c IsPseudoClassSelector) PseudoElement() string {
	return ""
}

type WherePseudoClassSelector struct {
	match SelectorGroup
}

func (c WherePseudoClassSelector) Selectors() []Sel {
	return c.match
}

func (s WherePseudoClassSelector) Match(n *html.Node) bool {
	for _, sel := range s.match {
		if sel.Match(n) {
			return true
		}
	}
	return false
}

func (s WherePseudoClassSelector) Specificity() Specificity {
	// :where() always has 0 specificity
	return Specificity{0, 0, 0}
}

func (c WherePseudoClassSelector) PseudoElement() string {
	return ""
}

type RelativePseudoClassSelector struct {
	name  string // one of "not", "has", "haschild"
	match SelectorGroup
}

func (s RelativePseudoClassSelector) Match(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch s.name {
	case "not":
		// matches elements that do not match a.
		return !s.match.Match(n)
	case "has":
		//  matches elements with any descendant that matches a.
		return hasDescendantMatch(n, s.match)
	case "haschild":
		// matches elements with a child that matches a.
		return hasChildMatch(n, s.match)
	default:
		panic(fmt.Sprintf("unsupported relative pseudo class selector : %s", s.name))
	}
}

// hasChildMatch returns whether n has any child that matches a.
func hasChildMatch(n *html.Node, a Matcher) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if a.Match(c) {
			return true
		}
	}
	return false
}

// hasDescendantMatch performs a depth-first search of n's descendants,
// testing whether any of them match a. It returns true as soon as a match is
// found, or false if no match is found.
func hasDescendantMatch(n *html.Node, a Matcher) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if a.Match(c) || (c.Type == html.ElementNode && hasDescendantMatch(c, a)) {
			return true
		}
	}
	return false
}

// Specificity returns the specificity of the most specific selectors
// in the pseudo-class arguments.
// See https://www.w3.org/TR/selectors/#specificity-rules
func (s RelativePseudoClassSelector) Specificity() Specificity {
	var max Specificity
	for _, sel := range s.match {
		newSpe := sel.Specificity()
		if max.Less(newSpe) {
			max = newSpe
		}
	}
	return max
}

func (c RelativePseudoClassSelector) PseudoElement() string {
	return ""
}

type ContainsPseudoClassSelector struct {
	abstractPseudoClass
	value string
	own   bool
}

func (s ContainsPseudoClassSelector) Match(n *html.Node) bool {
	var text string
	if s.own {
		// matches nodes that directly contain the given text
		text = strings.ToLower(nodeOwnText(n))
	} else {
		// matches nodes that contain the given text.
		text = strings.ToLower(nodeText(n))
	}
	return strings.Contains(text, s.value)
}

type RegexpPseudoClassSelector struct {
	abstractPseudoClass
	regexp *regexp.Regexp
	own    bool
}

func (s RegexpPseudoClassSelector) Match(n *html.Node) bool {
	var text string
	if s.own {
		// matches nodes whose text directly matches the specified regular expression
		text = nodeOwnText(n)
	} else {
		// matches nodes whose text matches the specified regular expression
		text = nodeText(n)
	}
	return s.regexp.MatchString(text)
}

// writeNodeText writes the text contained in n and its descendants to b.
func writeNodeText(n *html.Node, b *bytes.Buffer) {
	switch n.Type {
	case html.TextNode:
		b.WriteString(n.Data)
	case html.ElementNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			writeNodeText(c, b)
		}
	}
}

// nodeText returns the text contained in n and its descendants.
func nodeText(n *html.Node) string {
	var b bytes.Buffer
	writeNodeText(n, &b)
	return b.String()
}

// nodeOwnText returns the contents of the text nodes that are direct
// children of n.
func nodeOwnText(n *html.Node) string {
	var b bytes.Buffer
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
	}
	return b.String()
}

type NthPseudoClassSelector struct {
	abstractPseudoClass
	a, b         int
	last, ofType bool
}

func (s NthPseudoClassSelector) Match(n *html.Node) bool {
	if s.a == 0 {
		if s.last {
			return simpleNthLastChildMatch(s.b, s.ofType, n)
		} else {
			return simpleNthChildMatch(s.b, s.ofType, n)
		}
	}
	return nthChildMatch(s.a, s.b, s.last, s.ofType, n)
}

// nthChildMatch implements :nth-child(an+b).
// If last is true, implements :nth-last-child instead.
// If ofType is true, implements :nth-of-type instead.
func nthChildMatch(a, b int, last, ofType bool, n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	parent := n.Parent
	if parent == nil {
		return false
	}

	i := -1
	count := 0
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if (c.Type != html.ElementNode) || (ofType && c.Data != n.Data) {
			continue
		}
		count++
		if c == n {
			i = count
			if !last {
				break
			}
		}
	}

	if i == -1 {
		// This shouldn't happen, since n should always be one of its parent's children.
		return false
	}

	if last {
		i = count - i + 1
	}

	i -= b
	if a == 0 {
		return i == 0
	}

	return i%a == 0 && i/a >= 0
}

// simpleNthChildMatch implements :nth-child(b).
// If ofType is true, implements :nth-of-type instead.
func simpleNthChildMatch(b int, ofType bool, n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	parent := n.Parent
	if parent == nil {
		return false
	}

	count := 0
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || (ofType && c.Data != n.Data) {
			continue
		}
		count++
		if c == n {
			return count == b
		}
		if count >= b {
			return false
		}
	}
	return false
}

// simpleNthLastChildMatch implements :nth-last-child(b).
// If ofType is true, implements :nth-last-of-type instead.
func simpleNthLastChildMatch(b int, ofType bool, n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	parent := n.Parent
	if parent == nil {
		return false
	}

	count := 0
	for c := parent.LastChild; c != nil; c = c.PrevSibling {
		if c.Type != html.ElementNode || (ofType && c.Data != n.Data) {
			continue
		}
		count++
		if c == n {
			return count == b
		}
		if count >= b {
			return false
		}
	}
	return false
}

type OnlyChildPseudoClassSelector struct {
	abstractPseudoClass
	ofType bool
}

// Match implements :only-child.
// If `ofType` is true, it implements :only-of-type instead.
func (s OnlyChildPseudoClassSelector) Match(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	parent := n.Parent
	if parent == nil {
		return false
	}

	count := 0
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if (c.Type != html.ElementNode) || (s.ofType && c.Data != n.Data) {
			continue
		}
		count++
		if count > 1 {
			return false
		}
	}

	return count == 1
}

type InputPseudoClassSelector struct {
	abstractPseudoClass
}

// Matches input, select, textarea and button elements.
func (s InputPseudoClassSelector) Match(n *html.Node) bool {
	return n.Type == html.ElementNode && (n.Data == "input" || n.Data == "select" || n.Data == "textarea" || n.Data == "button")
}

type EmptyElementPseudoClassSelector struct {
	abstractPseudoClass
}

// Matches empty elements.
func (s EmptyElementPseudoClassSelector) Match(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.ElementNode:
			return false
		case html.TextNode:
			if strings.TrimSpace(nodeText(c)) == "" {
				continue
			} else {
				return false
			}
		}
	}

	return true
}

type RootPseudoClassSelector struct {
	abstractPseudoClass
}

// Match implements :root
func (s RootPseudoClassSelector) Match(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	if n.Parent == nil {
		return false
	}
	return n.Parent.Type == html.DocumentNode
}

func hasAttr(n *html.Node, attr string) bool {
	return matchAttribute(n, attr, func(string) bool { return true })
}

type LinkPseudoClassSelector struct {
	abstractPseudoClass
}

// Match implements :link
func (s LinkPseudoClassSelector) Match(n *html.Node) bool {
	return (n.DataAtom == atom.A || n.DataAtom == atom.Area || n.DataAtom == atom.Link) && hasAttr(n, "href")
}

type LangPseudoClassSelector struct {
	abstractPseudoClass
	lang string
}

func (s LangPseudoClassSelector) Match(n *html.Node) bool {
	own := matchAttribute(n, "lang", func(val string) bool {
		return val == s.lang || strings.HasPrefix(val, s.lang+"-")
	})
	if n.Parent == nil {
		return own
	}
	return own || s.Match(n.Parent)
}

type EnabledPseudoClassSelector struct {
	abstractPseudoClass
}

func (s EnabledPseudoClassSelector) Match(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch n.DataAtom {
	case atom.A, atom.Area, atom.Link:
		return hasAttr(n, "href")
	case atom.Optgroup, atom.Menuitem, atom.Fieldset:
		return !hasAttr(n, "disabled")
	case atom.Button, atom.Input, atom.Select, atom.Textarea, atom.Option:
		return !hasAttr(n, "disabled") && !inDisabledFieldset(n)
	}
	return false
}

type DisabledPseudoClassSelector struct {
	abstractPseudoClass
}

func (s DisabledPseudoClassSelector) Match(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch n.DataAtom {
	case atom.Optgroup, atom.Menuitem, atom.Fieldset:
		return hasAttr(n, "disabled")
	case atom.Button, atom.Input, atom.Select, atom.Textarea, atom.Option:
		return hasAttr(n, "disabled") || inDisabledFieldset(n)
	}
	return false
}

func hasLegendInPreviousSiblings(n *html.Node) bool {
	for s := n.PrevSibling; s != nil; s = s.PrevSibling {
		if s.DataAtom == atom.Legend {
			return true
		}
	}
	return false
}

func inDisabledFieldset(n *html.Node) bool {
	if n.Parent == nil {
		return false
	}
	if n.Parent.DataAtom == atom.Fieldset && hasAttr(n.Parent, "disabled") &&
		(n.DataAtom != atom.Legend || hasLegendInPreviousSiblings(n)) {
		return true
	}
	return inDisabledFieldset(n.Parent)
}

type CheckedPseudoClassSelector struct {
	abstractPseudoClass
}

func (s CheckedPseudoClassSelector) Match(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch n.DataAtom {
	case atom.Input, atom.Menuitem:
		return hasAttr(n, "checked") && matchAttribute(n, "type", func(val string) bool {
			t := toLowerASCII(val)
			return t == "checkbox" || t == "radio"
		})
	case atom.Option:
		return hasAttr(n, "selected")
	}
	return false
}

type VisitedPseudoClassSelector struct {
	abstractPseudoClass
}

func (s VisitedPseudoClassSelector) Match(n *html.Node) bool {
	return false
}

type HoverPseudoClassSelector struct {
	abstractPseudoClass
}

func (s HoverPseudoClassSelector) Match(n *html.Node) bool {
	return false
}

type ActivePseudoClassSelector struct {
	abstractPseudoClass
}

func (s ActivePseudoClassSelector) Match(n *html.Node) bool {
	return false
}

type FocusPseudoClassSelector struct {
	abstractPseudoClass
}

func (s FocusPseudoClassSelector) Match(n *html.Node) bool {
	return false
}

type TargetPseudoClassSelector struct {
	abstractPseudoClass
}

func (s TargetPseudoClassSelector) Match(n *html.Node) bool {
	return false
}

type ReadOnlyPseudoClassSelector struct {
	abstractPseudoClass
}

func (s ReadOnlyPseudoClassSelector) Match(n *html.Node) bool {
	return false
}

type PopoverPseudoClassSelector struct {
	abstractPseudoClass
}

func (s PopoverPseudoClassSelector) Match(n *html.Node) bool {
	return false
}

func IsPseudoElement(sel Sel) bool {
	switch sel.(type) {
	case abstractPseudoClass:
		return true
	case RelativePseudoClassSelector:
		return true
	case ContainsPseudoClassSelector:
		return true
	case RegexpPseudoClassSelector:
		return true
	case NthPseudoClassSelector:
		return true
	case OnlyChildPseudoClassSelector:
		return true
	case InputPseudoClassSelector:
		return true
	case EmptyElementPseudoClassSelector:
		return true
	case RootPseudoClassSelector:
		return true
	case LinkPseudoClassSelector:
		return true
	case LangPseudoClassSelector:
		return true
	case EnabledPseudoClassSelector:
		return true
	case DisabledPseudoClassSelector:
		return true
	case CheckedPseudoClassSelector:
		return true
	case VisitedPseudoClassSelector:
		return true
	case HoverPseudoClassSelector:
		return true
	case ActivePseudoClassSelector:
		return true
	case FocusPseudoClassSelector:
		return true
	case TargetPseudoClassSelector:
		return true
	case ReadOnlyPseudoClassSelector:
		return true
	case PopoverPseudoClassSelector:
		return true
	default:
		return false
	}
}
