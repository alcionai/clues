package clues

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

type hashAlg int

const (
	SHA256 hashAlg = iota
	HMAC_SHA256
	Plaintext
)

var hashingAlgorithm = HMAC_SHA256

type Concealer interface {
	Conceal() string
}

type secret struct {
	s string
	v any
	h string
}

func (s secret) String() string  { return s.s }
func (s secret) Conceal() string { return s.h }

// Mask embeds the value in a clues. secret where the
// Conceal() call always returns a flat string: "***"
func Mask(a any) secret {
	str := marshal(a)

	return secret{
		s: str,
		v: a,
		h: "***",
	}
}

// Hide embeds the value in a clues. secret where the
// Conceal() call contains a truncated hash of value.
// The hash function defaults to SHA256, but can be
// changed through configuration.
func Hide(a any) secret {
	str := marshal(a)

	return secret{
		s: str,
		v: a,
		h: Conceal(hashingAlgorithm, str),
	}
}

// HideAll is a quality-of-life wrapper for transforming
// multiple values to clues.secrete structs.
func HideAll(a ...any) []secret {
	sl := make([]secret, 0, len(a))

	for _, v := range a {
		sl = append(sl, Hide(v))
	}

	return sl
}

// Conceal runs one of clues' hashing algorithms on
// the provided string.
func Conceal(alg hashAlg, s string) string {
	if len(s) == 0 {
		return ""
	}

	switch alg {
	case HMAC_SHA256:
		return hashHmacSha256(s)

	case Plaintext:
		return s

	default:
		return hashSha256(s)
	}
}

func hashHmacSha256(s string) string {
	var (
		// need to set up key establishment.
		key = []byte("TODO-rjjATxL6KRlCaDGyRpIc3T4PUYAUXIz8")
		sig = hmac.New(sha256.New, key)
	)

	sig.Write([]byte(s))

	return hex.EncodeToString(sig.Sum(nil))[:16]
}

func hashSha256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))

	return hex.EncodeToString(h.Sum(nil))[:16]
}
