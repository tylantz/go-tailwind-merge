// It is based on the cascadia package kept here: github.com/andybalholm/cascadia

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
	"strconv"
	"strings"
)

// implements the reverse operation Sel -> string

var specialCharReplacer *strings.Replacer

func init() {
	var pairs []string
	for _, s := range ",!\"#$%&'()*+ -./:;<=>?@[\\]^`{|}~" {
		pairs = append(pairs, string(s), "\\"+string(s))
	}
	specialCharReplacer = strings.NewReplacer(pairs...)
}

// espace special CSS char
func escape(s string) string { return specialCharReplacer.Replace(s) }

func (c TagSelector) String() string {
	return c.tag
}

func (c IdSelector) String() string {
	return "#" + escape(c.id)
}

func (c ClassSelector) String() string {
	return "." + escape(c.Class)
}

func (c AttrSelector) String() string {
	val := c.val
	if c.operation == "#=" {
		val = c.regexp.String()
	} else if c.operation != "" {
		val = fmt.Sprintf(`"%s"`, val)
	}

	ignoreCase := ""

	if c.insensitive {
		ignoreCase = " i"
	}

	return fmt.Sprintf(`[%s%s%s%s]`, c.key, c.operation, val, ignoreCase)
}

func (c IsPseudoClassSelector) String() string {
	return fmt.Sprintf(":is(%s)", c.match.String())
}

func (c WherePseudoClassSelector) String() string {
	return fmt.Sprintf(":where(%s)", c.match.String())
}

func (c RelativePseudoClassSelector) String() string {
	return fmt.Sprintf(":%s(%s)", c.name, c.match.String())
}

func (c ContainsPseudoClassSelector) String() string {
	s := "contains"
	if c.own {
		s += "Own"
	}
	return fmt.Sprintf(`:%s("%s")`, s, c.value)
}

func (c RegexpPseudoClassSelector) String() string {
	s := "matches"
	if c.own {
		s += "Own"
	}
	return fmt.Sprintf(":%s(%s)", s, c.regexp.String())
}

func (c NthPseudoClassSelector) String() string {
	if c.a == 0 && c.b == 1 { // special cases
		s := ":first-"
		if c.last {
			s = ":last-"
		}
		if c.ofType {
			s += "of-type"
		} else {
			s += "child"
		}
		return s
	}
	var name string
	switch [2]bool{c.last, c.ofType} {
	case [2]bool{true, true}:
		name = "nth-last-of-type"
	case [2]bool{true, false}:
		name = "nth-last-child"
	case [2]bool{false, true}:
		name = "nth-of-type"
	case [2]bool{false, false}:
		name = "nth-child"
	}
	s := fmt.Sprintf("+%d", c.b)
	if c.b < 0 { // avoid +-8 invalid syntax
		s = strconv.Itoa(c.b)
	}
	return fmt.Sprintf(":%s(%dn%s)", name, c.a, s)
}

func (c OnlyChildPseudoClassSelector) String() string {
	if c.ofType {
		return ":only-of-type"
	}
	return ":only-child"
}

func (c InputPseudoClassSelector) String() string {
	return ":input"
}

func (c EmptyElementPseudoClassSelector) String() string {
	return ":empty"
}

func (c RootPseudoClassSelector) String() string {
	return ":root"
}

func (c LinkPseudoClassSelector) String() string {
	return ":link"
}

func (c LangPseudoClassSelector) String() string {
	return fmt.Sprintf(":lang(%s)", c.lang)
}

func (c EnabledPseudoClassSelector) String() string {
	return ":enabled"
}

func (c DisabledPseudoClassSelector) String() string {
	return ":disabled"
}

func (c CheckedPseudoClassSelector) String() string {
	return ":checked"
}

func (c VisitedPseudoClassSelector) String() string {
	return ":visited"
}

func (c HoverPseudoClassSelector) String() string {
	return ":hover"
}

func (c ActivePseudoClassSelector) String() string {
	return ":active"
}

func (c FocusPseudoClassSelector) String() string {
	return ":focus"
}

func (c TargetPseudoClassSelector) String() string {
	return ":target"
}

func (c ReadOnlyPseudoClassSelector) String() string {
	return ":read-only"
}

func (c PopoverPseudoClassSelector) String() string {
	return ":popover"
}

func (c abstractPseudoClass) String() string {
	return fmt.Sprintf(":%s", c.name)
}

func (c CompoundSelector) String() string {
	if len(c.selectors) == 0 && c.pseudoElement == "" {
		return "*"
	}
	chunks := make([]string, len(c.selectors))
	for i, sel := range c.selectors {
		chunks[i] = sel.String()
	}
	s := strings.Join(chunks, "")
	if c.pseudoElement != "" {
		s += "::" + c.pseudoElement
	}
	return s
}

func (c CombinedSelector) String() string {
	start := c.first.String()
	if c.second != nil {
		start += fmt.Sprintf(" %s %s", string(c.combinator), c.second.String())
	}
	return start
}

func (c SelectorGroup) String() string {
	ck := make([]string, len(c))
	for i, s := range c {
		ck[i] = s.String()
	}
	return strings.Join(ck, ", ")
}
