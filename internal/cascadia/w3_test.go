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
	"log"
	"os"
	"reflect"
	"testing"

	"golang.org/x/net/html"
)

func TestInvalidSelectors(t *testing.T) {
	c, err := os.ReadFile("test_resources/invalid_selectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var tests []invalidSelector
	if err = json.Unmarshal(c, &tests); err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		_, err := ParseGroupWithPseudoElements(test.Selector)
		if err == nil {
			t.Fatalf("%s -> expected error on invalid selector : %s", test.Name, test.Selector)
		}
	}
}

func parseReference(filename string) *html.Node {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	node, err := html.Parse(f)
	if err != nil {
		log.Fatal(err)
	}
	return node
}

func getId(n *html.Node) string {
	for _, attr := range n.Attr {
		if attr.Key == "id" {
			return attr.Val
		}
	}
	return ""
}

func isEqual(m map[string]int, l []string) bool {
	expected := map[string]int{}
	for _, s := range l {
		expected[s]++
	}
	return reflect.DeepEqual(m, expected)
}
