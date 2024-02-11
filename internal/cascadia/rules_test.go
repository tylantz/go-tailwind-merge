package cascadia

import (
	"bytes"
	"reflect"
	"testing"
)

func TestExtractRulesFile2(t *testing.T) {

	input := `
	.class1 .class2 {
		color: red;
	}

	@media (min-width: 640px) {
		.sm\:grid-cols-2 {
		grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}

	/* this one should be ignored */
	#test {
		color: red;
	}

	.pb-10 {
		padding-bottom: 2.5rem;
	}

	.read-only\:p-2:read-only {
		padding: 0.5rem;
	}

	.ring {
		--tw-ring-offset-shadow: var(--tw-ring-inset) 0 0 0 var(--tw-ring-offset-width) var(--tw-ring-offset-color);
		--tw-ring-shadow: var(--tw-ring-inset) 0 0 0 calc(3px + var(--tw-ring-offset-width)) var(--tw-ring-color);
		box-shadow: var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow, 0 0 #0000);
	}
	`

	want := []CssRule{
		{
			Selector:     CombinedSelector{first: ClassSelector{Class: "class1"}, combinator: ' ', second: ClassSelector{Class: "class2"}},
			Declarations: []CssDeclaration{{Property: "color", Value: "red"}},
			condition:    "",
		},
		{
			Selector:     ClassSelector{Class: "sm:grid-cols-2"},
			Declarations: []CssDeclaration{{Property: "grid-template-columns", Value: "repeat(2,minmax(0,1fr))"}},
			condition:    "(min-width:640px)",
		},
		{
			Selector:     IdSelector{id: "test"},
			Declarations: []CssDeclaration{{Property: "color", Value: "red"}},
			condition:    "",
		},
		{
			Selector:     ClassSelector{Class: "pb-10"},
			Declarations: []CssDeclaration{{Property: "padding-bottom", Value: "2.5rem"}},
			condition:    "",
		},
		{
			Selector:     CompoundSelector{selectors: []Sel{ClassSelector{Class: "read-only:p-2"}, ReadOnlyPseudoClassSelector{}}, pseudoElement: ""},
			Declarations: []CssDeclaration{{Property: "padding", Value: "0.5rem"}},
			condition:    "",
		},
		{
			Selector: ClassSelector{Class: "ring"},
			Declarations: []CssDeclaration{
				{Property: "--tw-ring-offset-shadow", Value: "var(--tw-ring-inset) 0 0 0 var(--tw-ring-offset-width) var(--tw-ring-offset-color)"},
				{Property: "--tw-ring-shadow", Value: "var(--tw-ring-inset) 0 0 0 calc(3px + var(--tw-ring-offset-width)) var(--tw-ring-color)"},
				{Property: "box-shadow", Value: "var(--tw-ring-offset-shadow),var(--tw-ring-shadow),var(--tw-shadow,0 0 #0000)"},
			},
			condition: "",
		},
	}

	t.Run("ExtractRulesMultiple", func(t *testing.T) {
		r := bytes.NewBufferString(input)
		got, err := ExtractRules(r, false)
		if err != nil {
			t.Fatalf("ExtractRules returned error: %v", err)
		}
		if len(got) != len(want) {
			t.Errorf("ExtractRules returned %d items, want %d items", len(got), len(want))
		}
		for i := 0; i < len(got); i++ {
			if !reflect.DeepEqual(got[i].Selector, want[i].Selector) {
				t.Errorf("ExtractRules failed on test %d with non-matching selectors:\nreturned %v,\nwant %v ", i, got[i].Selector, want[i].Selector)
			}
			if !reflect.DeepEqual(got[i].Declarations, want[i].Declarations) {
				t.Errorf("ExtractRules failed on test %d with non-matching declarations:\nreturned %v,\nwant %v ", i, got[i].Declarations, want[i].Declarations)
			}
			if got[i].condition != want[i].condition {
				t.Errorf("ExtractRules failed on test %d with non-matching conditions:\nreturned %v,\nwant %v ", i, got[i].condition, want[i].condition)
			}
		}
	})
}
