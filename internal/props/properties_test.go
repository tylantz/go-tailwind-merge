package props

import (
	"reflect"
	"testing"
)

func TestUnMarshalPropsShortHands(t *testing.T) {
	t.Parallel()
	tt := []struct {
		name         string
		wantKey      string
		wantProp     Property
		wantComputed []string
	}{
		{
			name:    "background",
			wantKey: "background",
			wantProp: Property{
				name: "background",
				property: property{
					Computed: []string{
						"background-image",
						"background-position",
						"background-size",
						"background-repeat",
						"background-origin",
						"background-clip",
						"background-attachment",
						"background-color",
					},
					AlsoAppliesTo: []AlsoAppliesTo{
						"::first-letter",
						"::first-line",
						"::placeholder",
					},
					Status: "standard",
				},
			},
			wantComputed: []string{
				"background-image",
				"background-position",
				"background-size",
				"background-repeat",
				"background-origin",
				"background-clip",
				"background-attachment",
				"background-color",
			},
		},
		{
			name:    "inset",
			wantKey: "inset",
			wantProp: Property{
				name: "inset",
				property: property{
					Computed: []string{
						"top",
						"bottom",
						"left",
						"right",
					},
					Status: "standard",
				},
			},
			wantComputed: []string{"top", "bottom", "left", "right"},
		},
		{
			name:    "background-image",
			wantKey: "background-image",
			wantProp: Property{
				name: "background-image",
				property: property{
					Computed: "asSpecifiedURLsAbsolute",
					Status:   "standard",
					AlsoAppliesTo: []AlsoAppliesTo{
						"::first-letter",
						"::first-line",
						"::placeholder",
					},
				},
			},
			wantComputed: []string{"background-image"},
		},
	}

	props, err := GetProperties()
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := props[tc.wantKey]

			if !ok {
				t.Fatalf("key %s not found", tc.wantKey)
			}
			if !reflect.DeepEqual(got.ComputedProps(), tc.wantComputed) {
				t.Errorf("ComputedProps different, got %v; want %v", got.ComputedProps(), tc.wantComputed)
			}
			if !reflect.DeepEqual(got.AlsoAppliesTo(), tc.wantProp.property.AlsoAppliesTo) {
				t.Errorf("AlsoAppliesTo different got %v; want %v", got.AlsoAppliesTo(), tc.wantProp.AlsoAppliesTo())
			}
			if got.Status() != tc.wantProp.property.Status {
				t.Errorf("Status different got %v; want %v", got, tc.wantProp)
			}
			if got.Name() != tc.wantProp.name {
				t.Errorf("Name different got %v; want %v", got, tc.wantProp)
			}
		})
	}
}
