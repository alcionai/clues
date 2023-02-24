package clues

import "testing"

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
				t.Errorf(`expected hash result "%s", got "%s"`, test.expect, result)
			}
		})
	}
}

func TestMask(t *testing.T) {
	table := []struct {
		name   string
		input  any
		expect string
	}{
		{
			name:   "string",
			input:  "fnords",
			expect: "fnords",
		},
		{
			name:   "stringer",
			input:  mockStringer{"fnords"},
			expect: "{s:fnords}",
		},
		{
			name:   "map",
			input:  map[string]string{"fnords": "smarf"},
			expect: `{"fnords":"smarf"}`,
		},
		{
			name:   "nil",
			input:  nil,
			expect: ``,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			m := Mask(test.input)
			if m.Conceal() != "***" {
				t.Errorf(`expected Conceal() result "***", got "%s"`, m.Conceal())
			}
			if m.String() != test.expect {
				t.Errorf(`expected String() result "%s", got "%s"`, test.expect, m.String())
			}
		})
	}
}

func TestHide(t *testing.T) {
	table := []struct {
		name       string
		input      any
		expectHash string
		expectStr  string
	}{
		{
			name:       "string",
			input:      "fnords",
			expectHash: "7745164c2e6b3c97",
			expectStr:  "fnords",
		},
		{
			name:       "int",
			input:      1,
			expectHash: "1e29272d274ab30f",
			expectStr:  "1",
		},
		{
			name:       "stringer",
			input:      mockStringer{"fnords"},
			expectHash: "553c83b5702ada92",
			expectStr:  "{s:fnords}",
		},
		{
			name:       "map",
			input:      map[string]string{"fnords": "smarf"},
			expectHash: "1502957923bb4cc8",
			expectStr:  `{"fnords":"smarf"}`,
		},
		{
			name:       "nil",
			input:      nil,
			expectHash: "",
			expectStr:  ``,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			h := Hide(test.input)
			if h.Conceal() != test.expectHash {
				t.Errorf(`expected Conceal() result "%s", got "%s"`, test.expectHash, h.Conceal())
			}
			if h.String() != test.expectStr {
				t.Errorf(`expected String() result "%s", got "%s"`, test.expectStr, h.String())
			}
		})
	}
}

func TestHideAll(t *testing.T) {
	table := []struct {
		name       string
		input      []any
		expectHash []string
		expectStr  []string
	}{
		{
			name:       "string, int",
			input:      []any{"fnords", 1},
			expectHash: []string{"7745164c2e6b3c97", "1e29272d274ab30f"},
			expectStr:  []string{"fnords", "1"},
		},
		{
			name:       "stringer",
			input:      []any{mockStringer{"fnords"}, mockStringer{"smarf"}},
			expectHash: []string{"553c83b5702ada92", "71e19af12aa87603"},
			expectStr:  []string{"{s:fnords}", "{s:smarf}"},
		},
		{
			name:       "map",
			input:      []any{map[string]string{"fnords": "smarf"}},
			expectHash: []string{"1502957923bb4cc8"},
			expectStr:  []string{`{"fnords":"smarf"}`},
		},
		{
			name:       "nil",
			input:      []any{nil, nil},
			expectHash: []string{"", ""},
			expectStr:  []string{"", ""},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			hs := HideAll(test.input...)
			for i, h := range hs {
				expectHash := test.expectHash[i]
				expectStr := test.expectStr[i]

				if h.Conceal() != expectHash {
					t.Errorf(`expected Conceal() result "%s", got "%s"`, expectHash, h.Conceal())
				}
				if h.String() != expectStr {
					t.Errorf(`expected String() result "%s", got "%s"`, expectStr, h.String())
				}
			}
		})
	}
}
