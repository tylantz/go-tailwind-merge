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
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func loadValidSelectors(t *testing.T) []validSelector {
	c, err := os.ReadFile("test_resources/valid_selectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var tests []validSelector
	if err = json.Unmarshal(c, &tests); err != nil {
		t.Fatal(err)
	}
	return tests
}

func TestSerialize(t *testing.T) {
	var testSer []string
	for _, test := range selectorTests {
		testSer = append(testSer, test.selector)
	}
	for _, test := range testsPseudo {
		testSer = append(testSer, test.selector)
	}
	for _, test := range loadValidSelectors(t) {
		if test.Xfail {
			continue
		}
		testSer = append(testSer, test.Selector)
	}

	for _, test := range testSer {
		s, err := ParseGroupWithPseudoElements(test)
		if err != nil {
			t.Fatalf("error compiling %q: %s", test, err)
		}

		serialized := s.String()
		s2, err := ParseGroupWithPseudoElements(serialized)
		if err != nil {
			t.Errorf("error compiling %q: %s %T (original : %s)", serialized, err, s, test)
		}

		if !reflect.DeepEqual(s, s2) {
			t.Errorf("can't retrieve selector from serialized : %s (original : %s, sel : %#v)", serialized, test, s)
		}
	}
}
