package clues

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"
)

const hashTruncateLen = 16

type hashAlg int

const (
	SHA256 hashAlg = iota
	HMAC_SHA256
	Plaintext
	Flatmask
)

var (
	initial = makeDefaultHash()
	config  = DefaultHash()
)

type HashCfg struct {
	HashAlg hashAlg
	HMACKey []byte
}

// SetHasher sets the hashing configuration used in
// all clues concealer structs, and clues.Conceal()
// and clues.Hash() calls.
func SetHasher(sc HashCfg) {
	config = sc
}

// NoHash provides a secrets configuration with
// no hashing or masking of values.
func NoHash() HashCfg {
	return HashCfg{Plaintext, nil}
}

// DefaultHash creates a secrets configuration using the
// HMAC_SHA256 hash with a random key.  This value is already
// set upon initialization of the package.
func DefaultHash() HashCfg {
	return HashCfg{initial.HashAlg, initial.HMACKey}
}

func makeDefaultHash() HashCfg {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		b = []byte(fmt.Sprintf("%d", time.Now().UnixNano()))[:16]
	}

	return HashCfg{HMAC_SHA256, b}
}

// ---------------------------------------------------------------------------
// types and interfaces
// ---------------------------------------------------------------------------

type PlainConcealer interface {
	PlainStringer
	Concealer
}

// PlainStringer is the opposite of conceal.
// Useful for if you want to retrieve the raw value of a secret.
type PlainStringer interface {
	PlainString() string
}

type Concealer interface {
	Conceal() string
	// Concealers also need to comply with Format
	// It's a bit overbearing, but complying with Concealer
	// doesn't guarantee any caller that thee variable won't
	// pass into fmt.Printf("%v") and skip the whole hash.
	// This is for your protection, too.
	Format(fs fmt.State, verb rune)
}

// compliance guarantees
var (
	_ Concealer     = &secret{}
	_ PlainStringer = &secret{}
)

type secret struct {
	s string
	v any
	h string
}

// use the hashed string in any fmt verb.
func (s secret) Format(fs fmt.State, verb rune) { io.WriteString(fs, s.h) }
func (s secret) String() string                 { return s.h }
func (s secret) Conceal() string                { return s.h }
func (s secret) PlainString() string            { return s.s }
func (s secret) V() any                         { return s.v }

// ---------------------------------------------------------------------------
// concealer constructors
// ---------------------------------------------------------------------------

// Hide embeds the value in a secret struct where the
// Conceal() call contains a truncated hash of value.
// The hash function defaults to SHA256, but can be
// changed through configuration.
func Hide(a any) secret {
	str := marshal(a)

	return secret{
		s: str,
		v: a,
		h: Conceal(str),
	}
}

// HideAll is a quality-of-life wrapper for transforming
// multiple values to secret structs.
func HideAll(a ...any) []secret {
	sl := make([]secret, 0, len(a))

	for _, v := range a {
		sl = append(sl, Hide(v))
	}

	return sl
}

// Mask embeds the value in a secret struct where the
// Conceal() call always returns a flat string: "***"
func Mask(a any) secret {
	return secret{
		s: marshal(a),
		v: a,
		h: "***",
	}
}

// Conceal runs the currently configured hashing algorithm
// on the parameterized value.
func Conceal(a any) string {
	return ConcealWith(config.HashAlg, marshal(a))
}

// Conceal runs one of clues' hashing algorithms on
// the provided string.
func ConcealWith(alg hashAlg, s string) string {
	if len(s) == 0 {
		return ""
	}

	switch alg {
	case HMAC_SHA256:
		return hashHmacSha256(s)

	case Plaintext:
		return s

	case Flatmask:
		return "***"

	default:
		return hashSha256(s)
	}
}

// ---------------------------------------------------------------------------
// hashing algs
// ---------------------------------------------------------------------------

func hashHmacSha256(s string) string {
	sig := hmac.New(sha256.New, config.HMACKey)
	sig.Write([]byte(s))

	return hex.EncodeToString(sig.Sum(nil))[:hashTruncateLen]
}

func hashSha256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))

	return hex.EncodeToString(h.Sum(nil))[:hashTruncateLen]
}
