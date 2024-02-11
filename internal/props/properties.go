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
	name     string
	computed []string
	property
}

func (p *Property) Name() string {
	return p.name
}

// TODO MOVE THIS TO CREATION OF PROPERTY
func (p *Property) ComputedProps() []string {
	if len(p.computed) > 0 {
		return p.computed
	}
	jsonValue, err := json.Marshal(p.property.Computed)
	if err != nil {
		return nil
	}

	var computed []string
	err = json.Unmarshal(jsonValue, &computed)
	if err != nil {
		computed = []string{p.name}
	}
	p.computed = computed
	return computed
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
	Syntax        string           `json:"syntax"`
	Media         interface{}      `json:"media"` // can be string or array of strings
	Inherited     bool             `json:"inherited"`
	AnimationType interface{}      `json:"animationType"` // can be string or array of strings
	Percentages   interface{}      `json:"percentages"`   // can be string or array of strings
	Groups        []gen.GroupsElem `json:"groups"`
	Initial       interface{}      `json:"initial"` // can be string or array of strings
	Appliesto     gen.Appliesto    `json:"appliesto"`
	Computed      interface{}      `json:"computed"` // can be string or array of strings
	Order         gen.Order        `json:"order"`
	Status        gen.Status       `json:"status"`
	AlsoAppliesTo []AlsoAppliesTo  `json:"alsoAppliesTo,omitempty"`
	MdnUrl        string           `json:"mdn_url,omitempty"`
	Stacking      bool             `json:"stacking,omitempty"`
}

// GetProperties returns a map of all CSS properties.
// The key is the name of the property, and the value is the property itself.
func GetProperties() (map[string]Property, error) {

	props := make(map[string]interface{})
	err := json.Unmarshal(propsJson, &props)
	if err != nil {
		return nil, err
	}

	propMap := make(map[string]Property)
	for name, prop := range props {
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
