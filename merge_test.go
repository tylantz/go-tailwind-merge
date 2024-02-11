package merge

import (
	"bytes"
	"os"
	"slices"
	"testing"
)

func TestCustomVarRegex(t *testing.T) {

	tt := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "simple",
			in:   "var(--tw-ring-color)",
			want: []string{"--tw-ring-color"},
		},
		{
			name: "shadow",
			in:   "var(--tw-ring-offset-shadow, 0 0 #0000), var(--tw-ring-shadow, 0 0 #0000), var(--tw-shadow)",
			want: []string{"--tw-ring-offset-shadow", "--tw-ring-shadow", "--tw-shadow"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := customVarRegex.FindAllStringSubmatch(tc.in, -1)
			for i, match := range got {
				if match[1] != tc.want[i] {
					t.Errorf("Resolve returned %v, want %v", match[1], tc.want[i])
				}
			}
		})
	}
}

func TestSortSubset(t *testing.T) {
	tt := []struct {
		name   string
		inFull []string
		inSub  []string
		want   []string
	}{
		{
			name:   "simple",
			inFull: []string{"a", "b", "c", "d", "e"},
			inSub:  []string{"e", "a", "c"},
			want:   []string{"a", "c", "e"},
		},
		{
			name:   "simple reversed",
			inFull: []string{"e", "d", "c", "b", "a"},
			inSub:  []string{"e", "a", "c"},
			want:   []string{"e", "c", "a"},
		},
		{
			name:   "with duplicates in full",
			inFull: []string{"e", "d", "c", "b", "e", "a"},
			inSub:  []string{"e", "a", "c"},
			want:   []string{"c", "e", "a"},
		},
		{
			name:   "empty full",
			inFull: []string{},
			inSub:  []string{"e", "a", "c"},
			want:   []string{"e", "a", "c"},
		},
		{
			name:   "nil full",
			inFull: nil,
			inSub:  []string{"e", "a", "c"},
			want:   []string{"e", "a", "c"},
		},
		{
			name:   "nil sub",
			inFull: []string{"e", "d", "c"},
			inSub:  nil,
			want:   nil,
		},
		{
			name:   "both nil",
			inFull: nil,
			inSub:  nil,
			want:   nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			sortSubset(tc.inSub, tc.inFull)
			if !slices.Equal(tc.inSub, tc.want) {
				t.Errorf("Resolve returned %v, want %v", tc.inSub, tc.want)
			}
		})
	}
}

func BenchmarkMergeNoCache(b *testing.B) {
	by, err := os.ReadFile("./internal/cascadia/test_resources/test_output.css")
	if err != nil {
		b.Fatalf("ReadFile returned error: %v", err)
	}
	rm := NewMerger(nil, false)
	err = rm.AddRules(bytes.NewBuffer(by), false)
	if err != nil {
		b.Fatalf("AddRules returned error: %v", err)
	}

	by, err = os.ReadFile("./internal/cascadia/test_resources/classList.txt")
	if err != nil {
		b.Fatalf("ReadFile returned error: %v", err)
	}
	classList := string(by)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.Merge(classList)
	}
}

func BenchmarkMergeCache(b *testing.B) {
	by, err := os.ReadFile("./internal/cascadia/test_resources/test_output.css")
	if err != nil {
		b.Fatalf("ReadFile returned error: %v", err)
	}
	cache := NewCache()
	rm := NewMerger(cache, false)
	err = rm.AddRules(bytes.NewBuffer(by), false)
	if err != nil {
		b.Fatalf("AddRules returned error: %v", err)
	}

	by, err = os.ReadFile("./internal/cascadia/test_resources/classList.txt")
	if err != nil {
		b.Fatalf("ReadFile returned error: %v", err)
	}
	classList := string(by)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.Merge(classList)
	}
}

func TestMerge(t *testing.T) {
	tt := []struct {
		in   string
		want string
	}{
		// color variants
		{
			in:   "border-white border-white/10",
			want: "border-white/10",
		},
		{
			in:   "border-white/10 border-white",
			want: "border-white",
		},
		// handles arbitrary property conflicts correctly
		{
			in:   "[paint-order:markers] [paint-order:normal]",
			want: "[paint-order:normal]",
		},
		// handles arbitrary property conflicts with modifiers correctly
		{
			in:   "[paint-order:markers] hover:[paint-order:normal]",
			want: "[paint-order:markers] hover:[paint-order:normal]",
		},
		{
			in:   "hover:[paint-order:markers] hover:[paint-order:normal]",
			want: "hover:[paint-order:normal]",
		},
		{
			in:   "hover:focus:[paint-order:markers] focus:hover:[paint-order:normal]",
			want: "focus:hover:[paint-order:normal]",
		},
		// handles simple conflicts with arbitrary values correctly
		{
			in:   "m-[2px] m-[10px]",
			want: "m-[10px]",
		},
		{
			in:   "z-20 z-[99]",
			want: "z-[99]",
		},
		{
			in:   "my-[2px] m-[10rem]",
			want: "m-[10rem]",
		},
		{
			in:   "cursor-pointer cursor-[grab]",
			want: "cursor-[grab]",
		},
		{
			in:   "m-[calc(100%-var(--arbitrary))] m-[2px]",
			want: "m-[2px]",
		},
		{
			in:   "m-[2px] m-[length:var(--mystery-var)]",
			want: "m-[length:var(--mystery-var)]",
		},
		{
			in:   "opacity-10 opacity-[0.025]",
			want: "opacity-[0.025]",
		},
		{
			in:   "scale-75 scale-[1.7]",
			want: "scale-[1.7]",
		},
		{
			in:   "brightness-90 brightness-[1.75]",
			want: "brightness-[1.75]",
		},
		{
			in:   "min-h-[0.5px] min-h-[0]",
			want: "min-h-[0]",
		},
		{
			in:   "text-[0.5px] text-[color:0]",
			want: "text-[0.5px] text-[color:0]",
		},
		{
			in:   "text-[0.5px] text-[--my-0]",
			want: "text-[0.5px] text-[--my-0]",
		},
		// handles arbitrary length conflicts with labels and modifiers correctly
		{
			in:   "hover:m-[2px] hover:m-[length:var(--c)]",
			want: "hover:m-[length:var(--c)]",
		},
		{
			in:   "hover:focus:m-[2px] focus:hover:m-[length:var(--c)]",
			want: "focus:hover:m-[length:var(--c)]",
		},
		// handles complex arbitrary value conflicts correctly
		{
			in:   "grid-rows-[1fr,auto] grid-rows-2",
			want: "grid-rows-2",
		},
		{
			in:   "grid-rows-[repeat(20,minmax(0,1fr))] grid-rows-3",
			want: "grid-rows-3",
		},
		// handles ambiguous arbitrary values correctly
		{
			in:   "mt-2 mt-[calc(theme(fontSize.4xl)/1.125)]",
			want: "mt-[calc(theme(fontSize.4xl)/1.125)]",
		},
		{
			in:   "p-2 p-[calc(theme(fontSize.4xl)/1.125)_10px]",
			want: "p-[calc(theme(fontSize.4xl)/1.125)_10px]",
		},
		{
			in:   "bg-cover bg-[percentage:30%] bg-[length:200px_100px]",
			want: "bg-[length:200px_100px]",
		},
		// basic arbitrary variants
		{
			in:   "[&>*]:underline [&>*]:line-through",
			want: "[&>*]:line-through",
		},
		{
			in:   "[&>*]:underline [&>*]:line-through [&_div]:line-through",
			want: "[&>*]:line-through [&_div]:line-through",
		},
		{
			in:   "supports-[display:grid]:flex supports-[display:grid]:grid",
			want: "supports-[display:grid]:grid",
		},
		// arbitrary variants with modifiers
		{
			in:   "dark:lg:hover:[&>*]:underline dark:lg:hover:[&>*]:line-through",
			want: "dark:lg:hover:[&>*]:line-through",
		},
		{
			in:   "dark:lg:hover:[&>*]:underline dark:hover:lg:[&>*]:line-through",
			want: "dark:hover:lg:[&>*]:line-through",
		},
		{
			in:   "hover:[&>*]:underline [&>*]:hover:line-through",
			want: "hover:[&>*]:underline [&>*]:hover:line-through",
		},
		// arbitrary variants with attribute selectors
		{
			in:   "[&[data-open]]:underline [&[data-open]]:line-through",
			want: "[&[data-open]]:line-through",
		},
		// multiple arbitrary variants
		{
			in:   "[&>*]:[&_div]:underline [&>*]:[&_div]:line-through",
			want: "[&>*]:[&_div]:line-through",
		},
		{
			in:   "[&>*]:[&_div]:underline [&_div]:[&>*]:line-through",
			want: "[&>*]:[&_div]:underline [&_div]:[&>*]:line-through",
		},
		// arbitrary variants with arbitrary properties
		{
			in:   "[&>*]:[color:red] [&>*]:[color:blue]",
			want: "[&>*]:[color:blue]",
		},
		// merges classes from same group correctly
		{
			in:   "overflow-x-auto overflow-x-hidden",
			want: "overflow-x-hidden",
		},
		{
			in:   "basis-full basis-auto",
			want: "basis-auto",
		},
		{
			in:   "w-full w-fit",
			want: "w-fit",
		},
		{
			in:   "overflow-x-auto overflow-x-hidden overflow-x-scroll",
			want: "overflow-x-scroll",
		},
		{
			in:   "overflow-x-auto hover:overflow-x-hidden overflow-x-scroll",
			want: "hover:overflow-x-hidden overflow-x-scroll",
		},
		{
			in:   "col-span-1 col-span-full",
			want: "col-span-full",
		},
		// merges classes from Font Variant Numeric section correctly
		{
			in:   "lining-nums tabular-nums diagonal-fractions",
			want: "lining-nums tabular-nums diagonal-fractions",
		},
		{
			in:   "normal-nums tabular-nums diagonal-fractions",
			want: "tabular-nums diagonal-fractions",
		},
		{
			in:   "tabular-nums diagonal-fractions normal-nums",
			want: "normal-nums",
		},
		{
			in:   "tabular-nums proportional-nums",
			want: "proportional-nums",
		},
		// handles color conflicts properly
		{
			in:   "hover:bg-destructive/90 hover:bg-accent",
			want: "hover:bg-accent",
		},
		{
			in:   "stroke-[hsl(350_80%_0%)] stroke-[10px]",
			want: "stroke-[hsl(350_80%_0%)] stroke-[10px]",
		},
		// handles conflicts across class groups correctly
		{
			in:   "inset-1 inset-x-1",
			want: "inset-1 inset-x-1",
		},
		{
			in:   "inset-x-1 inset-1",
			want: "inset-1",
		},
		{
			in:   "inset-x-1 left-1 inset-1",
			want: "inset-1",
		},
		{
			in:   "inset-x-1 inset-1 left-1",
			want: "inset-1 left-1",
		},
		{
			in:   "inset-x-1 right-1 inset-1",
			want: "inset-1",
		},
		{
			in:   "inset-x-1 right-1 inset-x-1",
			want: "inset-x-1",
		},
		{
			in:   "inset-x-1 right-1 inset-y-1",
			want: "inset-x-1 right-1 inset-y-1",
		},
		{
			in:   "right-1 inset-x-1 inset-y-1",
			want: "inset-x-1 inset-y-1",
		},
		{
			in:   "inset-x-1 hover:left-1 inset-1",
			want: "hover:left-1 inset-1",
		},
		// ring and shadow classes do not create conflict
		{
			in:   "ring shadow",
			want: "ring shadow",
		},
		{
			in:   "ring-2 shadow-md",
			want: "ring-2 shadow-md",
		},
		{
			in:   "shadow ring",
			want: "shadow ring",
		},
		{
			in:   "shadow-md ring-2",
			want: "shadow-md ring-2",
		},
		// touch classes do create conflicts correctly
		{
			in:   "touch-pan-x touch-pan-right",
			want: "touch-pan-right",
		},
		{
			in:   "touch-none touch-pan-x",
			want: "touch-pan-x",
		},
		{
			in:   "touch-pan-x touch-none",
			want: "touch-none",
		},
		{
			in:   "touch-pan-x touch-pan-y touch-pinch-zoom",
			want: "touch-pan-x touch-pan-y touch-pinch-zoom",
		},
		{
			in:   "touch-manipulation touch-pan-x touch-pan-y touch-pinch-zoom",
			want: "touch-pan-x touch-pan-y touch-pinch-zoom",
		},
		{
			in:   "touch-pan-x touch-pan-y touch-pinch-zoom touch-auto",
			want: "touch-auto",
		},
		// line-clamp classes do create conflicts correctly
		{
			in:   "overflow-auto inline line-clamp-1",
			want: "line-clamp-1",
		},
		{
			in:   "line-clamp-1 overflow-auto inline",
			want: "line-clamp-1 overflow-auto inline",
		},
		// merges content utilities correctly
		{
			in:   "content-['hello'] content-[attr(data-content)]",
			want: "content-[attr(data-content)]",
		},
		// merges tailwind classes with important modifier correctly
		{
			in:   "!font-medium !font-bold",
			want: "!font-bold",
		},
		{
			in:   "!font-medium !font-bold font-thin",
			want: "!font-bold font-thin",
		},
		{
			in:   "!right-2 !-inset-x-px",
			want: "!-inset-x-px",
		},
		{
			in:   "focus:!inline focus:!block",
			want: "focus:!block",
		},
		// conflicts across prefix modifiers
		{
			in:   "hover:block hover:inline",
			want: "hover:inline",
		},
		{
			in:   "hover:block hover:focus:inline",
			want: "hover:block hover:focus:inline",
		},
		{
			in:   "hover:block hover:focus:inline focus:hover:inline",
			want: "hover:block focus:hover:inline",
		},
		{
			in:   "focus-within:inline focus-within:block",
			want: "focus-within:block",
		},
		// conflicts across postfix modifiers
		{
			in:   "text-lg/7 text-lg/8",
			want: "text-lg/8",
		},
		{
			in:   "text-lg/none leading-9",
			want: "text-lg/none leading-9",
		},
		{
			in:   "leading-9 text-lg/none",
			want: "text-lg/none",
		},
		{
			in:   "w-full w-1/2",
			want: "w-1/2",
		},
		// handles negative value conflicts correctly
		{
			in:   "-m-2 -m-5",
			want: "-m-5",
		},
		{
			in:   "top-12 -top-12 ",
			want: "-top-12",
		},
		// handles conflicts between positive and negative values correctly
		{
			in:   "-m-2 m-auto",
			want: "m-auto",
		},
		// handles conflicts across groups with negative values correctly
		{
			in:   "-right-1 inset-x-1",
			want: "inset-x-1",
		},
		{
			in:   "hover:focus:-right-1 focus:hover:inset-x-1",
			want: "focus:hover:inset-x-1",
		},

		// merges non-conflicting classes correctly
		{
			in:   "border-t border-white/10",
			want: "border-t border-white/10",
		},
		{
			in:   "border-t border-white",
			want: "border-t border-white",
		},
		{
			in:   "text-2xl text-black",
			want: "text-2xl text-black",
		},
		// handles pseudo variants conflicts properly
		{
			in:   "empty:p-2 empty:p-3",
			want: "empty:p-3",
		},
		{
			in:   "hover:empty:p-2 hover:empty:p-3",
			want: "hover:empty:p-3",
		},
		{
			in:   "read-only:p-2 read-only:p-3",
			want: "read-only:p-3",
		},
		// handles pseudo variant group conflicts properly
		{
			in:   "group-empty:p-2 group-empty:p-3",
			want: "group-empty:p-3",
		},
		{
			in:   "peer-empty:p-2 peer-empty:p-3",
			want: "peer-empty:p-3",
		},
		{
			in:   "group-empty:p-2 peer-empty:p-3",
			want: "group-empty:p-2 peer-empty:p-3",
		},
		{
			in:   "hover:group-empty:p-2 hover:group-empty:p-3",
			want: "hover:group-empty:p-3",
		},
		{
			in:   "group-read-only:p-2 group-read-only:p-3",
			want: "group-read-only:p-3",
		},
		// merges standalone classes from same group correctly
		{
			in:   "inline block",
			want: "block",
		},
		{
			in:   "hover:block hover:inline",
			want: "hover:inline",
		},
		{
			in:   "hover:block hover:block",
			want: "hover:block",
		},
		{
			in:   "inline hover:inline focus:inline hover:block hover:focus:block",
			want: "inline focus:inline hover:block hover:focus:block",
		},
		{
			in:   "underline line-through",
			want: "line-through",
		},
		{
			in:   "line-through no-underline",
			want: "no-underline",
		},
		// supports Tailwind CSS v3.3 features
		{
			in:   "hyphens-auto hyphens-manual",
			want: "hyphens-manual",
		},
		{
			in:   "caption-top caption-bottom",
			want: "caption-bottom",
		},
		{
			in:   "line-clamp-2 line-clamp-none line-clamp-[10]",
			want: "line-clamp-[10]",
		},
		{
			in:   "delay-150 delay-0 duration-150 duration-0",
			want: "delay-0 duration-0",
		},
		{
			in:   "justify-normal justify-center justify-stretch",
			want: "justify-stretch",
		},
		{
			in:   "content-normal content-center content-stretch",
			want: "content-stretch",
		},
		{
			in:   "whitespace-nowrap whitespace-break-spaces",
			want: "whitespace-break-spaces",
		},
		// supports Tailwind CSS v3.4 features
		{
			in:   "h-svh h-dvh w-svw w-dvw",
			want: "h-dvh w-dvw",
		},
		{
			in:   "text-wrap text-pretty",
			want: "text-pretty",
		},
		{
			in:   "w-5 h-3 size-10 w-12",
			want: "size-10 w-12",
		},
		{
			in:   "grid-cols-2 grid-cols-subgrid grid-rows-5 grid-rows-subgrid",
			want: "grid-cols-subgrid grid-rows-subgrid",
		},
		{
			in:   "min-w-0 min-w-px max-w-0 max-w-px",
			want: "min-w-px max-w-px",
		},
		{
			in:   "forced-color-adjust-none forced-color-adjust-auto",
			want: "forced-color-adjust-auto",
		},
		{
			in:   "appearance-none appearance-auto",
			want: "appearance-auto",
		},
		{
			in:   "float-start float-end clear-start clear-end",
			want: "float-end clear-end",
		},
		{
			in:   "*:p-10 *:p-20 hover:*:p-10 hover:*:p-20",
			want: "*:p-20 hover:*:p-20",
		},
		{
			in:   "mix-blend-normal mix-blend-multiply",
			want: "mix-blend-multiply",
		},
		{
			in:   "h-10 h-min",
			want: "h-min",
		},
		{
			in:   "stroke-black stroke-1",
			want: "stroke-black stroke-1",
		},
		{
			in:   "stroke-2 stroke-[3]",
			want: "stroke-[3]",
		},
		{
			in:   "outline-black outline-1",
			want: "outline-black outline-1",
		},
		{
			in:   "grayscale-0 grayscale-[50%]",
			want: "grayscale-[50%]",
		},
		{
			in:   "grow grow-[2]",
			want: "grow-[2]",
		},
		// keeps unknown classes
		{
			in:   "grow grow-[2] unrecognized-class",
			want: "grow-[2] unrecognized-class",
		},
		{
			in:   "unrecognized-class grow grow-[2]",
			want: "unrecognized-class grow-[2]",
		},
		// handles :is() pseudo-class
		{
			in:   "dark:bg-green-500/20 dark:bg-blue-500/20",
			want: "dark:bg-blue-500/20",
		},
		{
			in:   "dark:bg-blue-500/20 dark:bg-green-500/20",
			want: "dark:bg-green-500/20",
		},
		// single class
		{
			in:   "p-1 ",
			want: "p-1 ",
		},
		// simple conflict
		{
			in:   "p-1 p-2",
			want: "p-2",
		},
		{
			in:   "p-2 p-1",
			want: "p-1",
		},
		// conditional
		{
			in:   "read-only:p-2 p-1",
			want: "read-only:p-2 p-1",
		},
		{
			in:   "space-x-16 space-x-2",
			want: "space-x-2",
		},
		// handles important
		{
			in:   "p-3Important p-2",
			want: "p-3Important p-2",
		},
		{
			in:   "class2 class3",
			want: "class2 class3",
		},
	}
	by, err := os.ReadFile("./internal/cascadia/test_resources/test_output.css")
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	r := NewMerger(nil, true)
	err = r.AddRules(bytes.NewBuffer(by), false)
	if err != nil {
		t.Fatalf("AddRules returned error: %v", err)
	}
	failed := 0
	passed := 0
	for _, tc := range tt {
		t.Run(tc.in, func(t *testing.T) {
			got := r.Merge(tc.in)
			if got != tc.want {
				failed++
				t.Errorf("TestMerge fail: in %v, want %v, got %v", tc.in, tc.want, got)
			} else {
				passed++
			}
		})
	}
	if len(tt)-failed-passed > 0 {
		t.Errorf("TestMerge failed %d, passed %d", failed, passed)
	}
}
