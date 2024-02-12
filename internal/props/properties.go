package props

import (
	_ "embed"
	"encoding/json"
	"errors"

	"github.com/tylantz/go-tailwind-merge/internal/props/gen"
)

//go:embed gen/properties.json
var propsJson []byte

// See https://github.com/mdn/data/blob/main/css/properties.md for explainer on the properties.json file

// Property represents a CSS property.
type Property struct {
	name string
	property
}

func (p *Property) Name() string {
	return p.name
}

func (p *Property) ComputedProps() []string {
	return p.Computed
}

// Status returns the status of the property.
// This is a string that can be "standard", "nonstandard", "experimental", or "obsolete".
func (p *Property) Status() gen.Status {
	return p.property.Status
}

func (p *Property) AlsoAppliesTo() []AlsoAppliesTo {
	return p.property.AlsoAppliesTo
}

type AlsoAppliesTo string

const (
	FirstLetter AlsoAppliesTo = "::first-letter"
	FirstLine   AlsoAppliesTo = "::first-line"
	Placeholder AlsoAppliesTo = "::placeholder"
)

type property struct {
	Computed      []string        `json:"computed"` // this diverges from CSS-spec. We take the name of the property or a slice of strings (e.g., padding-bottom, etc.)
	Status        gen.Status      `json:"status"`
	Appliesto     gen.Appliesto   `json:"appliesto"`
	AlsoAppliesTo []AlsoAppliesTo `json:"alsoAppliesTo,omitempty"`
}

// GetProperties returns a map of all CSS properties.
// The key is the name of the property, and the value is the property itself.
func GetProperties() (map[string]Property, error) {

	props := make(map[string]map[string]interface{})
	err := json.Unmarshal(propsJson, &props)
	if err != nil {
		return nil, err
	}

	propMap := make(map[string]Property)
	for name, prop := range props {

		computed := computedProps(prop["computed"], name)
		prop["computed"] = computed

		jsonValue, err := json.Marshal(prop)
		if err != nil {
			return nil, err
		}

		var p Property
		err = json.Unmarshal(jsonValue, &p)
		if err != nil {
			return nil, errors.New("could not unmarshal property")
		}
		p.name = name
		propMap[name] = p
	}
	return propMap, nil
}

// this is janky....
// take the "Computed" property as an interface
// it might be a string or a slice of strings
// if it is a string, return the property name in a slice
// if it is a slice of strings, return that
func computedProps(p interface{}, name string) []string {

	jsonValue, err := json.Marshal(p)
	if err != nil {
		return nil
	}

	var computed []string
	err = json.Unmarshal(jsonValue, &computed)
	if err != nil {
		computed = []string{name}
	}

	return computed
}
