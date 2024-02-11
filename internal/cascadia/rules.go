package cascadia

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

/*
	DeclarationGrammar: This refers to a grammar rule that matches a declaration in CSS.
	A declaration includes a property and a value, separated by a colon (:), and ends with a semicolon (;).

	EndRulesetGrammar: This likely refers to a grammar rule that matches the end of a ruleset. This would be the closing curly brace (}) of a declaration block

	ErrorGrammar: This likely refers to a grammar rule that matches erroneous or unexpected input in the CSS. It could be used to handle and recover from errors in a graceful manner.

	AtRuleGrammar: This refers to a grammar rule that matches at-rules in CSS. At-rules start with an at sign (@) followed by an identifier (like media, import, keyframes, etc.) and include everything up to the next semicolon (;) or the next CSS block, whichever comes first.

	EndAtRuleGrammar: This likely refers to a grammar rule that matches the end of an at-rule. This could be a semicolon (;) or the end of a CSS block.

	RulesetGrammar: This refers to a grammar rule that matches a ruleset in CSS. A ruleset includes a selector (or a group of selectors) followed by a declaration block (which is enclosed in curly braces {} and contains declarations).
*/

// CssRule represents a CSS rule, which includes a selector and a list of declarations.
// It also includes a condition, which is used to represent the condition of an at-rule (e.g., @media).
// The condition can also be a pseudo-element or pseudo-class for a compound selector (e.g., :hover, ::before).
type CssRule struct {
	Selector     Sel              // Selector is the selector for the rule
	Declarations []CssDeclaration // Declarations is a list of declarations for the rule (e.g., property-value pairs)
	condition    string           // Condition is the condition for the rule (e.g., for an at-rule like @media)
}

func (r CssRule) String() string {
	return fmt.Sprintf("Selector: %v, Declarations: %v, Condition: %v", r.Selector, r.Declarations, r.condition)
}

func (r CssRule) GetSelector() string {
	return r.Selector.String()
}

func (r CssRule) GetCondition() string {
	if t, ok := r.Selector.(CompoundSelector); ok {
		return t.PseudoElementsString()
	}
	return r.condition
}

func (r CssRule) ToCssFormat() string {
	dec := strings.Builder{}
	for i, d := range r.Declarations {
		dec.WriteString(fmt.Sprintf("%s: %s;", d.Property, d.Value))
		if i < len(r.Declarations)-1 {
			dec.WriteByte(' ')
		}
	}
	return fmt.Sprintf("%s { %s }", r.Selector.String(), dec.String())
}

// CssDeclaration represents a CSS declaration, which includes a property and a value.
type CssDeclaration struct {
	Property string // Property is the property for the declaration (e.g., "color")
	Value    string // Value is the value for the declaration (e.g., "red")
}

// getSelector parses the selector from the tokens
func getSelector(data []byte, tokens []css.Token) (Sel, error) {
	fullSelector := strings.Builder{}
	for _, val := range tokens {
		fullSelector.Write(val.Data)
	}
	casc, err := ParseWithPseudoElement(fullSelector.String())
	return casc, err
}

// parses the name of an at-rule and its values if it's a media query
// @media (min-width: 600px) { /* styles */ } => "media", "(min-width: 600px)"
func parseAtRuleName(data []byte, values []css.Token) (string, string) {
	if string(data) != "@media" && string(data) != "@supports" {
		return "", ""
	}
	if len(values) == 0 {
		panic("no values")
	}
	ruleBuilder := strings.Builder{}
	for _, val := range values {
		ruleBuilder.Write(val.Data)
	}
	return string(data), ruleBuilder.String()
}

func CssUnescape(b []byte) string {

	var buf bytes.Buffer

	var i int
	for i = 0; i < len(b); i++ {
		if b[i] == '\\' {
			// set up buf with the stuff we've already scanned
			buf.Grow(len(b))
			buf.Write(b[:i])
			goto foundEsc
		}
		continue
	}
	// no escaping needed
	return string(b)

foundEsc:
	inEsc := false
	for ; i < len(b); i++ {
		if b[i] == '\\' && !inEsc {
			inEsc = true
			continue
		}
		buf.WriteByte(b[i])
		inEsc = false
	}
	return buf.String()
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func ExtractRules(r io.Reader, inline bool) ([]CssRule, error) {
	p := css.NewParser(parse.NewInput(r), inline)
	rules := make([]CssRule, 0)
	var err error
	var currentRule CssRule
	var atRuleCondition string
	ignore := false
	ruleSetErr := false
	for {
		gt, _, data := p.Next()
		if ignore && gt != css.EndAtRuleGrammar {
			continue
		}

		if ruleSetErr && gt != css.EndRulesetGrammar {
			continue
		} else if ruleSetErr && gt == css.EndRulesetGrammar {
			ruleSetErr = false
		}

		switch gt {
		case css.ErrorGrammar:
			err = p.Err()
			if err == io.EOF {
				err = nil
			}
			if err != nil {
				err = fmt.Errorf("encountered error parsing CSS: %v", err)
			}
			return rules, err
		case css.BeginAtRuleGrammar:
			name, condition := parseAtRuleName(data, p.Values())
			if name == "" {
				atRuleCondition = ""
				ignore = true
				continue
			}
			atRuleCondition = condition
		case css.EndAtRuleGrammar:
			atRuleCondition = ""
			ignore = false
		case css.BeginRulesetGrammar:
			currentRule = CssRule{}
			currentRule.condition = atRuleCondition
			sel, err := getSelector(data, p.Values())
			if err != nil {
				log.Println("error parsing rule:", err) // TODO: LOG this better
				ruleSetErr = true
				currentRule = CssRule{}
				continue
			}
			currentRule.Selector = sel
		case css.DeclarationGrammar:
			declaration := buildDeclaration(p, data)
			currentRule.Declarations = append(currentRule.Declarations, declaration)
		case css.EndRulesetGrammar:
			rules = append(rules, currentRule)
		case css.CustomPropertyGrammar:
			declaration := buildDeclaration(p, data)
			currentRule.Declarations = append(currentRule.Declarations, declaration)
		}
	}
}

func buildDeclaration(p *css.Parser, data []byte) CssDeclaration {
	vals := strings.Builder{}
	for _, val := range p.Values() {
		vals.Write(val.Data)
	}
	s := normalizeWhitespace(vals.String())
	declaration := CssDeclaration{Property: string(data), Value: CssUnescape([]byte(s))}
	return declaration
}
