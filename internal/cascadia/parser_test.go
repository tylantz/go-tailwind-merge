/*
Copyright (c) 2011 Andy Balholm. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

  - Redistributions of source code must retain the above copyright

notice, this list of conditions and the following disclaimer.
  - Redistributions in binary form must reproduce the above

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
	"testing"
)

var identifierTests = map[string]string{
	"x":             "x",
	"96":            "",
	"-x":            "-x",
	`r\e9 sumé`:     "résumé",
	`r\0000e9 sumé`: "résumé",
	`r\0000e9sumé`:  "résumé",
	`a\"b`:          `a"b`,
}

func TestParseIdentifier(t *testing.T) {
	for source, want := range identifierTests {
		p := &parser{s: source}
		got, err := p.parseIdentifier()
		if err != nil {
			if want == "" {
				// It was supposed to be an error.
				continue
			}
			t.Errorf("parsing %q: got error (%s), want %q", source, err, want)
			continue
		}

		if want == "" {
			if err == nil {
				t.Errorf("parsing %q: got %q, want error", source, got)
			}
			continue
		}

		if p.i < len(source) {
			t.Errorf("parsing %q: %d bytes left over", source, len(source)-p.i)
			continue
		}

		if got != want {
			t.Errorf("parsing %q: got %q, want %q", source, got, want)
		}
	}
}

var stringTests = map[string]string{
	`"x"`:             "x",
	`'x'`:             "x",
	`'x`:              "",
	"'x\\\r\nx'":      "xx",
	`"r\e9 sumé"`:     "résumé",
	`"r\0000e9 sumé"`: "résumé",
	`"r\0000e9sumé"`:  "résumé",
	`"a\"b"`:          `a"b`,
}

func TestParseString(t *testing.T) {
	for source, want := range stringTests {
		p := &parser{s: source}
		got, err := p.parseString()
		if err != nil {
			if want == "" {
				// It was supposed to be an error.
				continue
			}
			t.Errorf("parsing %q: got error (%s), want %q", source, err, want)
			continue
		}

		if want == "" {
			if err == nil {
				t.Errorf("parsing %q: got %q, want error", source, got)
			}
			continue
		}

		if p.i < len(source) {
			t.Errorf("parsing %q: %d bytes left over", source, len(source)-p.i)
			continue
		}

		if got != want {
			t.Errorf("parsing %q: got %q, want %q", source, got, want)
		}
	}
}
