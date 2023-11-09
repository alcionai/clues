package clues

import (
	"fmt"
	"testing"
)

// set the hash alg key for consistency
func init() {
	SetHasher(HashCfg{HMAC_SHA256, []byte("gobbledeygook-believe-it-or-not-this-is-randomly-generated")})
}

type mockStringer struct {
	s string
}

func (ms mockStringer) String() string { return "{s:" + ms.s + "}" }

func TestConceal(t *testing.T) {
	input := "brunhaldi"

	table := []struct {
		name   string
		alg    hashAlg
		expect string
	}{
		{
			name:   "plainText",
			alg:    Plaintext,
			expect: input,
		},
		{
			name:   "sha256",
			alg:    SHA256,
			expect: "5fa99f4a1bb5f651",
		},
		{
			name:   "hmac_sha256",
			alg:    HMAC_SHA256,
			expect: "cddff495fc4a46ef",
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			result := ConcealWith(test.alg, input)
			if result != test.expect {
				t.Errorf(`expected hash result %q, got %q`, test.expect, result)
			}
		})
	}
}

func TestMask(t *testing.T) {
	table := []struct {
		name        string
		input       any
		expectPlain string
	}{
		{
			name:        "string",
			input:       "fnords",
			expectPlain: "fnords",
		},
		{
			name:        "stringer",
			input:       mockStringer{"fnords"},
			expectPlain: "{s:fnords}",
		},
		{
			name:        "map",
			input:       map[string]string{"fnords": "smarf"},
			expectPlain: `map[fnords:smarf]`,
		},
		{
			name:        "nil",
			input:       nil,
			expectPlain: ``,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			expect := "***"

			m := Mask(test.input)
			if m.Conceal() != expect {
				t.Errorf(`expected Conceal() result %q, got %q`, expect, m.Conceal())
			}
			if m.String() != expect {
				t.Errorf(`expected String() result %q, got %q`, expect, m.String())
			}
			if m.PlainString() != test.expectPlain {
				t.Errorf(`expected PlainString() result %q, got %q`, test.expectPlain, m.PlainString())
			}
			result := fmt.Sprintf("%s", m)
			if result != expect {
				t.Errorf(`expected %%s fmt result %q, got %q`, expect, result)
			}
			result = fmt.Sprintf("%v", m)
			if result != expect {
				t.Errorf(`expected %%v fmt result %q, got %q`, expect, result)
			}
			result = fmt.Sprintf("%+v", m)
			if result != expect {
				t.Errorf(`expected %%+v fmt result %q, got %q`, expect, result)
			}
			result = fmt.Sprintf("%#v", m)
			if result != expect {
				t.Errorf(`expected %%#v fmt result %q, got %q`, expect, result)
			}
		})
	}
}

func TestHide(t *testing.T) {
	table := []struct {
		name        string
		input       any
		expectHash  string
		expectPlain string
	}{
		{
			name:        "string",
			input:       "fnords",
			expectHash:  "7745164c2e6b3c97",
			expectPlain: "fnords",
		},
		{
			name:        "int",
			input:       1,
			expectHash:  "1e29272d274ab30f",
			expectPlain: "1",
		},
		{
			name:        "stringer",
			input:       mockStringer{"fnords"},
			expectHash:  "553c83b5702ada92",
			expectPlain: "{s:fnords}",
		},
		{
			name:        "map",
			input:       map[string]string{"fnords": "smarf"},
			expectHash:  "e9e09313140d3c4e",
			expectPlain: `map[fnords:smarf]`,
		},
		{
			name:        "nil",
			input:       nil,
			expectHash:  "",
			expectPlain: ``,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			h := Hide(test.input)
			if h.Conceal() != test.expectHash {
				t.Errorf(`expected Conceal() result %q, got %q`, test.expectHash, h.Conceal())
			}
			if h.String() != test.expectHash {
				t.Errorf(`expected String() result %q, got %q`, test.expectHash, h.String())
			}
			if h.PlainString() != test.expectPlain {
				t.Errorf(`expected PlainString() result %q, got %q`, test.expectPlain, h.PlainString())
			}
			result := fmt.Sprintf("%s", h)
			if result != test.expectHash {
				t.Errorf(`expected %%s fmt result %q, got %q`, test.expectHash, result)
			}
			result = fmt.Sprintf("%v", h)
			if result != test.expectHash {
				t.Errorf(`expected %%v fmt result %q, got %q`, test.expectHash, result)
			}
			result = fmt.Sprintf("%+v", h)
			if result != test.expectHash {
				t.Errorf(`expected %%+v fmt result %q, got %q`, test.expectHash, result)
			}
			result = fmt.Sprintf("%#v", h)
			if result != test.expectHash {
				t.Errorf(`expected %%#v fmt result %q, got %q`, test.expectHash, result)
			}
		})
	}
}

func TestHide_hideAConcealer(t *testing.T) {
	table := []struct {
		name        string
		input       any
		expectHash  string
		expectPlain string
	}{
		{
			name:        "string",
			input:       Hide("fnords"),
			expectHash:  "7745164c2e6b3c97",
			expectPlain: "fnords",
		},
		{
			name:        "int",
			input:       Hide(1),
			expectHash:  "1e29272d274ab30f",
			expectPlain: "1",
		},
		{
			name:        "stringer",
			input:       Hide(mockStringer{"fnords"}),
			expectHash:  "553c83b5702ada92",
			expectPlain: "{s:fnords}",
		},
		{
			name:        "map",
			input:       Hide(map[string]string{"fnords": "smarf"}),
			expectHash:  "e9e09313140d3c4e",
			expectPlain: `map[fnords:smarf]`,
		},
		{
			name:        "nil",
			input:       Hide(nil),
			expectHash:  "",
			expectPlain: ``,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			h := Hide(test.input)
			if h.Conceal() != test.expectHash {
				t.Errorf(`expected Conceal() result %q, got %q`, test.expectHash, h.Conceal())
			}
			if h.String() != test.expectHash {
				t.Errorf(`expected String() result %q, got %q`, test.expectHash, h.String())
			}
			if h.PlainString() != test.expectPlain {
				t.Errorf(`expected PlainString() result %q, got %q`, test.expectPlain, h.PlainString())
			}
			result := fmt.Sprintf("%s", h)
			if result != test.expectHash {
				t.Errorf(`expected %%s fmt result %q, got %q`, test.expectHash, result)
			}
			result = fmt.Sprintf("%v", h)
			if result != test.expectHash {
				t.Errorf(`expected %%v fmt result %q, got %q`, test.expectHash, result)
			}
			result = fmt.Sprintf("%+v", h)
			if result != test.expectHash {
				t.Errorf(`expected %%+v fmt result %q, got %q`, test.expectHash, result)
			}
			result = fmt.Sprintf("%#v", h)
			if result != test.expectHash {
				t.Errorf(`expected %%#v fmt result %q, got %q`, test.expectHash, result)
			}
		})
	}
}

func TestHideAll(t *testing.T) {
	table := []struct {
		name        string
		input       []any
		expectHash  []string
		expectPlain []string
	}{
		{
			name:        "string, int",
			input:       []any{"fnords", 1},
			expectHash:  []string{"7745164c2e6b3c97", "1e29272d274ab30f"},
			expectPlain: []string{"fnords", "1"},
		},
		{
			name:        "stringer",
			input:       []any{mockStringer{"fnords"}, mockStringer{"smarf"}},
			expectHash:  []string{"553c83b5702ada92", "71e19af12aa87603"},
			expectPlain: []string{"{s:fnords}", "{s:smarf}"},
		},
		{
			name:        "map",
			input:       []any{map[string]string{"fnords": "smarf"}},
			expectHash:  []string{"e9e09313140d3c4e"},
			expectPlain: []string{`map[fnords:smarf]`},
		},
		{
			name:        "nil",
			input:       []any{nil, nil},
			expectHash:  []string{"", ""},
			expectPlain: []string{"", ""},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			hs := HideAll(test.input...)
			for i, h := range hs {
				expectHash := test.expectHash[i]

				if h.Conceal() != expectHash {
					t.Errorf(`expected Conceal() result %q, got %q`, expectHash, h.Conceal())
				}
				if h.String() != expectHash {
					t.Errorf(`expected String() result %q, got %q`, expectHash, h.String())
				}
				if h.PlainString() != test.expectPlain[i] {
					t.Errorf(`expected PlainString() result %q, got %q`, test.expectPlain[i], h.String())
				}
			}
		})
	}
}
