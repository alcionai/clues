package cecrets

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/alcionai/clues/internal/stringify"
)

const hashTruncateLen = 16

type hashAlg int

const (
	SHA256 hashAlg = iota
	//nolint:revive
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
// all concealer structs, and Conceal() and Hash() calls.
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

type Concealer interface {
	// Conceal produces an obfuscated representation of the value.
	Conceal() string
	// Concealers also need to comply with Format.
	// Complying with Conceal() alone doesn't guarantee that
	// the variable won't pass into fmt.Printf("%v") and skip
	// the whole conceal process.
	Format(fs fmt.State, verb rune)
	// PlainStringer is the opposite of conceal.
	// Useful for if you want to retrieve the raw value of a secret.
	PlainString() string
}

// compliance guarantees
var _ Concealer = &secret{}

type secret struct {
	plainText string
	value     any
	hashText  string
}

// use the hashed string in any fmt verb.
func (s secret) Format(fs fmt.State, verb rune) {
	fmt.Fprint(fs, s.hashText)
}

func (s secret) String() string      { return s.hashText }
func (s secret) Conceal() string     { return s.hashText }
func (s secret) PlainString() string { return s.plainText }
func (s secret) V() any              { return s.value }

// ---------------------------------------------------------------------------
// concealer constructors
// ---------------------------------------------------------------------------

// Hide embeds the value in a secret struct where the
// Conceal() call contains a truncated hash of value.
// The hash function defaults to SHA256, but can be
// changed through configuration.
func Hide(a any) secret {
	if ac, ok := a.(Concealer); ok {
		return secret{
			hashText:  ac.Conceal(),
			plainText: ac.PlainString(),
			value:     a,
		}
	}

	return secret{
		hashText:  Conceal(a),
		plainText: stringify.Fmt(a)[0],
		value:     a,
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
		hashText:  "***",
		plainText: stringify.Fmt(a)[0],
		value:     a,
	}
}

// Conceal runs the currently configured hashing algorithm
// on the parameterized value.
func Conceal(a any) string {
	// marshal with false or else we hit a double hash (at best)
	// or an infinite loop (at worst).
	return ConcealWith(config.HashAlg, stringify.Fmt(a)[0])
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
