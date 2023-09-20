package clues_test

import (
	stderr "errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/alcionai/clues"
)

// if this panics, you added an uneven number of
// entries.  Lines should have an even number len.
// You might need to pair the line you added with
// an empty string.
func plusRE(lines ...string) string {
	var s string
	ll := len(lines)

	for i := 0; i < ll; i += 2 {
		s += lines[i]

		if len(lines[i+1]) > 0 {
			s += `\t.+/` + lines[i+1]
		}
	}

	return s
}

type checkFmt struct {
	tmpl     string
	expect   string
	reExpect *regexp.Regexp
}

func prettyStack(s string) string {
	s = strings.ReplaceAll(s, "\n", string('\n'))
	s = strings.ReplaceAll(s, "\t", "    ")
	return s
}

func (c checkFmt) check(t *testing.T, err error) {
	t.Run(c.tmpl, func(t *testing.T) {
		result := fmt.Sprintf(c.tmpl, err)

		if len(c.expect) > 0 && result != c.expect {
			t.Errorf(
				"unexpected fmt result for template %#v"+
					"\n\nexpected (raw)\n\"%s\""+
					"\n\ngot (raw)\n%#v"+
					"\n\ngot (fmt)\n\"%s\"",
				c.tmpl, c.expect, result, result,
			)
		}

		if c.reExpect != nil && !c.reExpect.MatchString(result) {
			t.Errorf(
				"unexpected fmt result for template %#v"+
					"\n\nexpected (raw)\n\"%s\""+
					"\n\ngot (raw)\n%#v"+
					"\n\ngot (fmt)\n\"%s\"",
				c.tmpl, c.reExpect, result, result,
			)
		}
	})
}

var globalSentinel = errors.New("sentinel")

func makeOnion(base error, mid, top func(error) error) error {
	var (
		bottom = func() error { return base }
		middle = func() error { return mid(bottom()) }
		outer  = func() error { return top(middle()) }
	)

	return outer()
}

func self(err error) error { return err }

var (
	errStd  = stderr.New("an error")
	errErrs = errors.New("an error")
	fmtErrf = fmt.Errorf("an error")
	cluErr  = clues.New("an error")

	cluesWrap       = func(err error) error { return clues.Wrap(err, "clues wrap") }
	cluesPlainStack = func(err error) error { return clues.Stack(err) }
	cluesStack      = func(err error) error { return clues.Stack(globalSentinel, err) }
)

func TestFmt(t *testing.T) {
	type expect struct {
		v    string
		plus string
		hash string
		s    string
		q    string
	}

	table := []struct {
		name   string
		onion  error
		expect expect
	}{
		// ---------------------------------------------------------------------------
		// litmus
		// ---------------------------------------------------------------------------
		{
			name: "litmus wrap stderr.New",
			onion: makeOnion(errStd,
				func(err error) error { return errors.Wrap(err, "errors wrap") },
				self),
			expect: expect{
				v:    "errors wrap: an error",
				hash: "errors wrap: an error",
				s:    "errors wrap: an error",
				q:    `"errors wrap: an error"`,
				plus: plusRE(
					`an error\n`, "",
					`errors wrap\n`, "",
					`github.com/alcionai/clues_test.TestFmt.func1\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.makeOnion.func2\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.makeOnion.func3\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.makeOnion\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
				),
			},
		},
		{
			name: "litmus wrap errors.New",
			onion: makeOnion(errErrs,
				func(err error) error { return errors.Wrap(err, "errors wrap") },
				self),
			expect: expect{
				v:    "errors wrap: an error",
				hash: "errors wrap: an error",
				s:    "errors wrap: an error",
				q:    `"errors wrap: an error"`,
				plus: plusRE(
					`an error\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues/.+go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`errors wrap\n`, "",
					`github.com/alcionai/clues_test.TestFmt.func2\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.makeOnion.func2\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.makeOnion.func3\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.makeOnion\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
				),
			},
		},
		{
			name: "litmus wrap fmt.Errorf",
			onion: makeOnion(fmtErrf,
				func(err error) error { return errors.Wrap(err, "errors wrap") },
				self),
			expect: expect{
				v:    "errors wrap: an error",
				hash: "errors wrap: an error",
				s:    "errors wrap: an error",
				q:    `"errors wrap: an error"`,
				plus: plusRE(
					`an error\n`, "",
					`errors wrap\n`, "",
					`github.com/alcionai/clues_test.TestFmt.func3\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.makeOnion.func2\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.makeOnion.func3\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.makeOnion\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
				),
			},
		},
		{
			name: "litmus wrap clues.New",
			onion: makeOnion(cluErr,
				func(err error) error { return errors.Wrap(err, "errors wrap") },
				self),
			expect: expect{
				v:    "errors wrap: an error",
				hash: "errors wrap: an error",
				s:    "errors wrap: an error",
				q:    `"errors wrap: an error"`,
				plus: plusRE(
					`an error\n`, `err_fmt_test.go:\d+\n`,
					`errors wrap\n`, "",
					`github.com/alcionai/clues_test.TestFmt.func4\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.makeOnion.func2\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.makeOnion.func3\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.makeOnion\n`, `err_fmt_test.go:\d+\n`,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
				),
			},
		},
		// ---------------------------------------------------------------------------
		// single error
		// ---------------------------------------------------------------------------
		{
			name:  "stderr.New",
			onion: makeOnion(errStd, self, self),
			expect: expect{
				v:    "an error",
				hash: `&errors.errorString{s:"an error"}`,
				s:    "an error",
				q:    `"an error"`,
				plus: plusRE(
					"an error$", "",
				),
			},
		},
		{
			name:  "errors.New",
			onion: makeOnion(errErrs, self, self),
			expect: expect{
				v:    "an error",
				hash: "an error",
				s:    "an error",
				q:    `"an error"`,
				plus: plusRE(
					`an error\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
				),
			},
		},
		{
			name:  "fmt.Errorf",
			onion: makeOnion(fmtErrf, self, self),
			expect: expect{
				v:    "an error",
				hash: `&errors.errorString{s:"an error"}`,
				s:    "an error",
				q:    `"an error"`,
				plus: plusRE(
					"an error$", "",
				),
			},
		},
		{
			name:  "clues.New",
			onion: makeOnion(cluErr, self, self),
			expect: expect{
				v:    "an error",
				hash: "an error",
				s:    "an error",
				q:    `"an error"`,
				plus: plusRE(
					"an error\n", `err_fmt_test.go:\d+$`,
				),
			},
		},
		// ---------------------------------------------------------------------------
		// wrapped error
		// ---------------------------------------------------------------------------
		{
			name:  "clues.Wrap stderr.New",
			onion: makeOnion(errStd, cluesWrap, self),
			expect: expect{
				v:    "clues wrap: an error",
				hash: "clues wrap: an error",
				s:    "clues wrap: an error",
				q:    `"clues wrap": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`clues wrap\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.Wrap errors.New",
			onion: makeOnion(errErrs, cluesWrap, self),
			expect: expect{
				v:    "clues wrap: an error",
				hash: "clues wrap: an error",
				s:    "clues wrap: an error",
				q:    `"clues wrap": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`clues wrap\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.Wrap fmt.Errorf",
			onion: makeOnion(fmtErrf, cluesWrap, self),
			expect: expect{
				v:    "clues wrap: an error",
				hash: "clues wrap: an error",
				s:    "clues wrap: an error",
				q:    `"clues wrap": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`clues wrap`, "",
				),
			},
		},
		{
			name:  "clues.Wrap clues.New",
			onion: makeOnion(cluErr, cluesWrap, self),
			expect: expect{
				v:    "clues wrap: an error",
				hash: "clues wrap: an error",
				s:    "clues wrap: an error",
				q:    `"clues wrap": "an error"`,
				plus: plusRE(
					"an error\n", `err_fmt_test.go:\d+\n`,
					`clues wrap\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		// ---------------------------------------------------------------------------
		// plain stacked sentinel
		// ---------------------------------------------------------------------------
		{
			name:  "clues.PlainStack stderr.New",
			onion: makeOnion(errStd, cluesPlainStack, self),
			expect: expect{
				v:    "an error",
				hash: "an error",
				s:    "an error",
				q:    `"an error"`,
				plus: plusRE(
					`an error\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.PlainStack errors.New",
			onion: makeOnion(errErrs, cluesPlainStack, self),
			expect: expect{
				v:    "an error",
				hash: "an error",
				s:    "an error",
				q:    `"an error"`,
				plus: plusRE(
					`an error\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.PlainStack fmt.Errorf",
			onion: makeOnion(fmtErrf, cluesPlainStack, self),
			expect: expect{
				v:    "an error",
				hash: "an error",
				s:    "an error",
				q:    `"an error"`,
				plus: plusRE(
					`an error\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.PlainStack clues.New",
			onion: makeOnion(cluErr, cluesPlainStack, self),
			expect: expect{
				v:    "an error",
				hash: "an error",
				s:    "an error",
				q:    `"an error"`,
				plus: plusRE(
					`an error\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		// ---------------------------------------------------------------------------
		// stacked sentinel
		// ---------------------------------------------------------------------------
		{
			name:  "clues.Stack stderr.New",
			onion: makeOnion(errStd, cluesStack, self),
			expect: expect{
				v:    "sentinel: an error",
				hash: "sentinel: an error",
				s:    "sentinel: an error",
				q:    `"sentinel": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.Stack errors.New",
			onion: makeOnion(errErrs, cluesStack, self),
			expect: expect{
				v:    "sentinel: an error",
				hash: "sentinel: an error",
				s:    "sentinel: an error",
				q:    `"sentinel": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.Stack fmt.Errorf",
			onion: makeOnion(fmtErrf, cluesStack, self),
			expect: expect{
				v:    "sentinel: an error",
				hash: "sentinel: an error",
				s:    "sentinel: an error",
				q:    `"sentinel": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.Stack clues.New",
			onion: makeOnion(cluErr, cluesStack, self),
			expect: expect{
				v:    "sentinel: an error",
				hash: "sentinel: an error",
				s:    "sentinel: an error",
				q:    `"sentinel": "an error"`,
				plus: plusRE(
					`an error\n`, `err_fmt_test.go:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		// ---------------------------------------------------------------------------
		// wrapped, stacked errors
		// ---------------------------------------------------------------------------
		{
			name:  "clues.Wrap clues.Stack stderr.New",
			onion: makeOnion(errStd, cluesStack, cluesWrap),
			expect: expect{
				v:    "clues wrap: sentinel: an error",
				hash: "clues wrap: sentinel: an error",
				s:    "clues wrap: sentinel: an error",
				q:    `"clues wrap": "sentinel": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`clues wrap\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.Wrap clues.Stack errors.New",
			onion: makeOnion(errErrs, cluesStack, cluesWrap),
			expect: expect{
				v:    "clues wrap: sentinel: an error",
				hash: "clues wrap: sentinel: an error",
				s:    "clues wrap: sentinel: an error",
				q:    `"clues wrap": "sentinel": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`clues wrap\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.Wrap clues.Stack fmt.Errorf",
			onion: makeOnion(fmtErrf, cluesStack, cluesWrap),
			expect: expect{
				v:    "clues wrap: sentinel: an error",
				hash: "clues wrap: sentinel: an error",
				s:    "clues wrap: sentinel: an error",
				q:    `"clues wrap": "sentinel": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`clues wrap\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.Wrap clues.Stack clues.New",
			onion: makeOnion(cluErr, cluesStack, cluesWrap),
			expect: expect{
				v:    "clues wrap: sentinel: an error",
				hash: "clues wrap: sentinel: an error",
				s:    "clues wrap: sentinel: an error",
				q:    `"clues wrap": "sentinel": "an error"`,
				plus: plusRE(
					`an error\n`, `err_fmt_test.go:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`clues wrap\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		// ---------------------------------------------------------------------------
		// stacked, wrapped errors
		// ---------------------------------------------------------------------------
		{
			name:  "clues.Stack clues.Wrap stderr.New",
			onion: makeOnion(errStd, cluesWrap, cluesStack),
			expect: expect{
				v:    "sentinel: clues wrap: an error",
				hash: "sentinel: clues wrap: an error",
				s:    "sentinel: clues wrap: an error",
				q:    `"sentinel": "clues wrap": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`clues wrap\n`, `err_fmt_test.go:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.Stack clues.Wrap errors.New",
			onion: makeOnion(errErrs, cluesWrap, cluesStack),
			expect: expect{
				v:    "sentinel: clues wrap: an error",
				hash: "sentinel: clues wrap: an error",
				s:    "sentinel: clues wrap: an error",
				q:    `"sentinel": "clues wrap": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`clues wrap\n`, `err_fmt_test.go:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.Stack clues.Wrap fmt.Errorf",
			onion: makeOnion(fmtErrf, cluesWrap, cluesStack),
			expect: expect{
				v:    "sentinel: clues wrap: an error",
				hash: "sentinel: clues wrap: an error",
				s:    "sentinel: clues wrap: an error",
				q:    `"sentinel": "clues wrap": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`clues wrap\n`, `err_fmt_test.go:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:  "clues.Stack clues.Wrap clues.New",
			onion: makeOnion(cluErr, cluesWrap, cluesStack),
			expect: expect{
				v:    "sentinel: clues wrap: an error",
				hash: "sentinel: clues wrap: an error",
				s:    "sentinel: clues wrap: an error",
				q:    `"sentinel": "clues wrap": "an error"`,
				plus: plusRE(
					`an error\n`, `err_fmt_test.go:\d+\n`,
					`clues wrap\n`, `err_fmt_test.go:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		// ---------------------------------------------------------------------------
		// compound stacking
		// ---------------------------------------------------------------------------
		{
			name: "multi-stack stderr.New",
			onion: clues.Stack(
				clues.Stack(
					stderr.New("top"),
					stderr.New("mid"),
				),
				clues.Stack(
					globalSentinel,
					stderr.New("bot"),
				),
			),
			expect: expect{
				v:    "top: mid: sentinel: bot",
				hash: "top: mid: sentinel: bot",
				s:    "top: mid: sentinel: bot",
				q:    `"top": "mid": "sentinel": "bot"`,
				plus: plusRE(
					`bot\n`, ``,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`mid\n`, ``,
					`top\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name: "multi-stack errors.New",
			onion: clues.Stack(
				clues.Stack(
					errors.New("top"),
					errors.New("mid"),
				),
				clues.Stack(
					globalSentinel,
					errors.New("bot"),
				),
			),
			expect: expect{
				v:    "top: mid: sentinel: bot",
				hash: "top: mid: sentinel: bot",
				s:    "top: mid: sentinel: bot",
				q:    `"top": "mid": "sentinel": "bot"`,
				plus: plusRE(
					`bot\n`, ``,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`mid\n`, ``,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`top\n`, ``,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name: "multi-stack fmt.Errorf",
			onion: clues.Stack(
				clues.Stack(
					fmt.Errorf("top"),
					fmt.Errorf("mid"),
				),
				clues.Stack(
					globalSentinel,
					fmt.Errorf("bot"),
				),
			),
			expect: expect{
				v:    "top: mid: sentinel: bot",
				hash: "top: mid: sentinel: bot",
				s:    "top: mid: sentinel: bot",
				q:    `"top": "mid": "sentinel": "bot"`,
				plus: plusRE(
					`bot\n`, ``,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`mid\n`, ``,
					`top\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name: "multi-stack clues.New",
			onion: clues.Stack(
				clues.Stack(
					clues.New("top"),
					clues.New("mid"),
				),
				clues.Stack(
					globalSentinel,
					clues.New("bot"),
				),
			),
			expect: expect{
				v:    "top: mid: sentinel: bot",
				hash: "top: mid: sentinel: bot",
				s:    "top: mid: sentinel: bot",
				q:    `"top": "mid": "sentinel": "bot"`,
				plus: plusRE(
					`bot\n`, `err_fmt_test.go:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`mid\n`, `err_fmt_test.go:\d+\n`,
					`top\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		// ---------------------------------------------------------------------------
		// wrapped, compound stacking
		// ---------------------------------------------------------------------------
		{
			name: "wrapped multi-stack stderr.New",
			onion: clues.Stack(
				clues.Wrap(clues.Stack(
					stderr.New("top"),
					stderr.New("mid"),
				), "lhs"),
				clues.Wrap(clues.Stack(
					globalSentinel,
					stderr.New("bot"),
				), "rhs"),
			),
			expect: expect{
				v:    "lhs: top: mid: rhs: sentinel: bot",
				hash: "lhs: top: mid: rhs: sentinel: bot",
				s:    "lhs: top: mid: rhs: sentinel: bot",
				q:    `"lhs": "top": "mid": "rhs": "sentinel": "bot"`,
				plus: plusRE(
					`bot\n`, ``,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`rhs\n`, `err_fmt_test.go:\d+\n`,
					`mid\n`, ``,
					`top\n`, `err_fmt_test.go:\d+\n`,
					`lhs\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name: "wrapped multi-stack errors.New",
			onion: clues.Stack(
				clues.Wrap(clues.Stack(
					errors.New("top"),
					errors.New("mid"),
				), "lhs"),
				clues.Wrap(clues.Stack(
					globalSentinel,
					errors.New("bot"),
				), "rhs"),
			),
			expect: expect{
				v:    "lhs: top: mid: rhs: sentinel: bot",
				hash: "lhs: top: mid: rhs: sentinel: bot",
				s:    "lhs: top: mid: rhs: sentinel: bot",
				q:    `"lhs": "top": "mid": "rhs": "sentinel": "bot"`,
				plus: plusRE(
					`bot\n`, ``,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`rhs\n`, `err_fmt_test.go:\d+\n`,
					`mid\n`, ``,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`top\n`, ``,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`lhs\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name: "wrapped multi-stack fmt.Errorf",
			onion: clues.Stack(
				clues.Wrap(clues.Stack(
					fmt.Errorf("top"),
					fmt.Errorf("mid"),
				), "lhs"),
				clues.Wrap(clues.Stack(
					globalSentinel,
					fmt.Errorf("bot"),
				), "rhs"),
			),
			expect: expect{
				v:    "lhs: top: mid: rhs: sentinel: bot",
				hash: "lhs: top: mid: rhs: sentinel: bot",
				s:    "lhs: top: mid: rhs: sentinel: bot",
				q:    `"lhs": "top": "mid": "rhs": "sentinel": "bot"`,
				plus: plusRE(
					`bot\n`, ``,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`rhs\n`, `err_fmt_test.go:\d+\n`,
					`mid\n`, ``,
					`top\n`, `err_fmt_test.go:\d+\n`,
					`lhs\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name: "wrapped multi-stack clues.New",
			onion: clues.Stack(
				clues.Wrap(clues.Stack(
					clues.New("top"),
					clues.New("mid"),
				), "lhs"),
				clues.Wrap(clues.Stack(
					globalSentinel,
					clues.New("bot"),
				), "rhs"),
			),
			expect: expect{
				v:    "lhs: top: mid: rhs: sentinel: bot",
				hash: "lhs: top: mid: rhs: sentinel: bot",
				s:    "lhs: top: mid: rhs: sentinel: bot",
				q:    `"lhs": "top": "mid": "rhs": "sentinel": "bot"`,
				plus: plusRE(
					`bot\n`, `err_fmt_test.go:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`rhs\n`, `err_fmt_test.go:\d+\n`,
					`mid\n`, `err_fmt_test.go:\d+\n`,
					`top\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`lhs\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			formats := []checkFmt{
				{"%v", test.expect.v, nil},
				{"%+v", "", regexp.MustCompile(test.expect.plus)},
				{"%#v", test.expect.hash, nil},
				{"%s", test.expect.s, nil},
				{"%q", test.expect.q, nil},
			}

			for _, f := range formats {
				f.check(t, test.onion)
			}
		})
	}
}

// wrapped

func bottomWrap(err error) error {
	return clues.Wrap(err, "bottom wrap")
}

func midWrap(err error) error {
	return clues.Wrap(bottomWrap(err), "mid wrap")
}

func topWrap(err error) error {
	return clues.Wrap(midWrap(err), "top wrap")
}

// plain-stacked

func bottomPlainStack(err error) error {
	return clues.Stack(err)
}

func midPlainStack(err error) error {
	return clues.Stack(bottomPlainStack(err))
}

func topPlainStack(err error) error {
	return clues.Stack(midPlainStack(err))
}

// stacked

func bottomStack(err error) error {
	return clues.Stack(clues.New("bottom"), err)
}

func midStack(err error) error {
	return clues.Stack(clues.New("mid"), bottomStack(err))
}

func topStack(err error) error {
	return clues.Stack(clues.New("top"), midStack(err))
}

func TestFmt_nestedFuncs(t *testing.T) {
	type expect struct {
		v    string
		plus string
		hash string
		s    string
		q    string
	}

	table := []struct {
		name   string
		fn     func(error) error
		source error
		expect expect
	}{
		// ---------------------------------------------------------------------------
		// wrapped error
		// ---------------------------------------------------------------------------
		{
			name:   "clues.Wrap stderr.New",
			fn:     topWrap,
			source: errStd,
			expect: expect{
				v:    "top wrap: mid wrap: bottom wrap: an error",
				hash: "top wrap: mid wrap: bottom wrap: an error",
				s:    "top wrap: mid wrap: bottom wrap: an error",
				q:    `"top wrap": "mid wrap": "bottom wrap": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`bottom wrap\n`, `err_fmt_test.go:\d+\n`,
					`mid wrap\n`, `err_fmt_test.go:\d+\n`,
					`top wrap\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:   "clues.Wrap errors.New",
			fn:     topWrap,
			source: errErrs,
			expect: expect{
				v:    "top wrap: mid wrap: bottom wrap: an error",
				hash: "top wrap: mid wrap: bottom wrap: an error",
				s:    "top wrap: mid wrap: bottom wrap: an error",
				q:    `"top wrap": "mid wrap": "bottom wrap": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`bottom wrap\n`, `err_fmt_test.go:\d+\n`,
					`mid wrap\n`, `err_fmt_test.go:\d+\n`,
					`top wrap\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:   "clues.Wrap fmt.Errorf",
			fn:     topWrap,
			source: fmtErrf,
			expect: expect{
				v:    "top wrap: mid wrap: bottom wrap: an error",
				hash: "top wrap: mid wrap: bottom wrap: an error",
				s:    "top wrap: mid wrap: bottom wrap: an error",
				q:    `"top wrap": "mid wrap": "bottom wrap": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`bottom wrap\n`, `err_fmt_test.go:\d+\n`,
					`mid wrap\n`, `err_fmt_test.go:\d+\n`,
					`top wrap\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:   "clues.Wrap clues.New",
			fn:     topWrap,
			source: cluErr,
			expect: expect{
				v:    "top wrap: mid wrap: bottom wrap: an error",
				hash: "top wrap: mid wrap: bottom wrap: an error",
				s:    "top wrap: mid wrap: bottom wrap: an error",
				q:    `"top wrap": "mid wrap": "bottom wrap": "an error"`,
				plus: plusRE(
					"an error\n", `err_fmt_test.go:\d+\n`,
					`bottom wrap\n`, `err_fmt_test.go:\d+\n`,
					`mid wrap\n`, `err_fmt_test.go:\d+\n`,
					`top wrap\n`, `err_fmt_test.go:\d+$`,
				),
			},
		},
		// ---------------------------------------------------------------------------
		// plain stacked
		// ---------------------------------------------------------------------------
		{
			name:   "clues.PlainStack stderr.New",
			fn:     topPlainStack,
			source: errStd,
			expect: expect{
				v:    "an error",
				hash: "an error",
				s:    "an error",
				q:    `"an error"`,
				plus: plusRE(
					`an error\n`, "",
					"", `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:   "clues.PlainStack errors.New",
			fn:     topPlainStack,
			source: errErrs,
			expect: expect{
				v:    "an error",
				hash: "an error",
				s:    "an error",
				q:    `"an error"`,
				plus: plusRE(
					`an error\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:   "clues.PlainStack fmt.Errorf",
			fn:     topPlainStack,
			source: fmtErrf,
			expect: expect{
				v:    "an error",
				hash: "an error",
				s:    "an error",
				q:    `"an error"`,
				plus: plusRE(
					`an error\n`, "",
					"", `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:   "clues.PlainStack clues.New",
			fn:     topPlainStack,
			source: cluErr,
			expect: expect{
				v:    "an error",
				hash: "an error",
				s:    "an error",
				q:    `"an error"`,
				plus: plusRE(
					`an error\n`, ``,
					"", `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		// ---------------------------------------------------------------------------
		// stacked tree
		// ---------------------------------------------------------------------------
		{
			name:   "clues.Stack stderr.New",
			fn:     topStack,
			source: errStd,
			expect: expect{
				v:    "top: mid: bottom: an error",
				hash: "top: mid: bottom: an error",
				s:    "top: mid: bottom: an error",
				q:    `"top": "mid": "bottom": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`bottom\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`mid\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`top\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:   "clues.Stack errors.New",
			fn:     topStack,
			source: errErrs,
			expect: expect{
				v:    "top: mid: bottom: an error",
				hash: "top: mid: bottom: an error",
				s:    "top: mid: bottom: an error",
				q:    `"top": "mid": "bottom": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`github.com/alcionai/clues_test.init\n`, `clues_benchmark_test.go:\d+\n`,
					`runtime.doInit1\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`bottom\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`mid\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`top\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:   "clues.Stack fmt.Errorf",
			fn:     topStack,
			source: fmtErrf,
			expect: expect{
				v:    "top: mid: bottom: an error",
				hash: "top: mid: bottom: an error",
				s:    "top: mid: bottom: an error",
				q:    `"top": "mid": "bottom": "an error"`,
				plus: plusRE(
					`an error\n`, "",
					`bottom\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`mid\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`top\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
		{
			name:   "clues.Stack clues.New",
			fn:     topStack,
			source: cluErr,
			expect: expect{
				v:    "top: mid: bottom: an error",
				hash: "top: mid: bottom: an error",
				s:    "top: mid: bottom: an error",
				q:    `"top": "mid": "bottom": "an error"`,
				plus: plusRE(
					`an error\n`, `err_fmt_test.go:\d+\n`,
					`bottom\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`mid\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+\n`,
					`top\n`, `err_fmt_test.go:\d+\n`,
					"", `err_fmt_test.go:\d+$`,
				),
			},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			formats := []checkFmt{
				{"%v", test.expect.v, nil},
				{"%+v", "", regexp.MustCompile(test.expect.plus)},
				{"%#v", test.expect.hash, nil},
				{"%s", test.expect.s, nil},
				{"%q", test.expect.q, nil},
			}

			for _, f := range formats {
				f.check(t, test.fn(test.source))
			}
		})
	}
}

func TestWithTrace(t *testing.T) {
	table := []struct {
		name   string
		tracer func(err error) error
		expect string
	}{
		{
			name: "error -1",
			tracer: func(err error) error {
				return withTraceWrapper(err, -1)
			},
			expect: plusRE(`an error\n`, `err_test.go:\d+$`),
		},
		{
			name: "error 0",
			tracer: func(err error) error {
				return withTraceWrapper(err, 0)
			},
			expect: plusRE(`an error\n`, `err_test.go:\d+$`),
		},
		{
			name: "error 1",
			tracer: func(err error) error {
				return withTraceWrapper(err, 1)
			},
			expect: plusRE(`an error\n`, `err_fmt_test.go:\d+$`),
		},
		{
			name: "error 2",
			tracer: func(err error) error {
				return withTraceWrapper(err, 2)
			},
			expect: plusRE(`an error\n`, `err_fmt_test.go:\d+$`),
		},
		{
			name: "error 3",
			tracer: func(err error) error {
				return withTraceWrapper(err, 3)
			},
			expect: plusRE(`an error\n`, `testing/testing.go:\d+$`),
		},
		{
			name: "error 4",
			tracer: func(err error) error {
				return withTraceWrapper(err, 4)
			},
			expect: plusRE(`an error\n`, `runtime/.*:\d+$`),
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			checkFmt{"%+v", "", regexp.MustCompile(test.expect)}.
				check(t, test.tracer(cluErr))
		})
	}
	table2 := []struct {
		name   string
		tracer func(err *clues.Err) error
		expect string
	}{
		{
			name: "clues.Err -1",
			tracer: func(err *clues.Err) error {
				return cluesWithTraceWrapper(err, -1)
			},
			expect: plusRE(`an error\n`, `err_test.go:\d+$`),
		},
		{
			name: "clues.Err 0",
			tracer: func(err *clues.Err) error {
				return cluesWithTraceWrapper(err, 0)
			},
			expect: plusRE(`an error\n`, `err_test.go:\d+$`),
		},
		{
			name: "clues.Err 1",
			tracer: func(err *clues.Err) error {
				return cluesWithTraceWrapper(err, 1)
			},
			expect: plusRE(`an error\n`, `err_fmt_test.go:\d+$`),
		},
		{
			name: "clues.Err 2",
			tracer: func(err *clues.Err) error {
				return cluesWithTraceWrapper(err, 2)
			},
			expect: plusRE(`an error\n`, `err_fmt_test.go:\d+$`),
		},
		{
			name: "clues.Err 3",
			tracer: func(err *clues.Err) error {
				return cluesWithTraceWrapper(err, 3)
			},
			expect: plusRE(`an error\n`, `testing/testing.go:\d+$`),
		},
		{
			name: "clues.Err 4",
			tracer: func(err *clues.Err) error {
				return cluesWithTraceWrapper(err, 4)
			},
			expect: plusRE(`an error\n`, `runtime/.*:\d+$`),
		},
	}
	for _, test := range table2 {
		t.Run(test.name, func(t *testing.T) {
			checkFmt{"%+v", "", regexp.MustCompile(test.expect)}.
				check(t, test.tracer(cluErr))
		})
	}
}

func TestErrCore_String(t *testing.T) {
	table := []struct {
		name        string
		core        *clues.ErrCore
		expectS     string
		expectVPlus string
	}{
		{
			name:        "nil",
			core:        nil,
			expectS:     `<nil>`,
			expectVPlus: `<nil>`,
		},
		{
			name: "all values",
			core: clues.
				New("message").
				With("key", "value").
				Label("label").
				Core(),
			expectS:     `{"message", [label], {key:value}}`,
			expectVPlus: `{msg:"message", labels:[label], values:{key:value}}`,
		},
		{
			name: "message only",
			core: clues.
				New("message").
				Core(),
			expectS:     `{"message"}`,
			expectVPlus: `{msg:"message", labels:[], values:{}}`,
		},
		{
			name: "labels only",
			core: clues.
				New("").
				Label("label").
				Core(),
			expectS:     `{[label]}`,
			expectVPlus: `{msg:"", labels:[label], values:{}}`,
		},
		{
			name: "values only",
			core: clues.
				New("").
				With("key", "value").
				Core(),
			expectS:     `{{key:value}}`,
			expectVPlus: `{msg:"", labels:[], values:{key:value}}`,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			tc := test.core
			if tc != nil {
				if _, ok := tc.Values["clues_trace"]; !ok {
					t.Error("expected core values to contain key [clues_trace]")
				}
				delete(tc.Values, "clues_trace")
			}

			result := tc.String()
			if result != test.expectS {
				t.Errorf("expected string\n%s\ngot %s", test.expectS, result)
			}

			result = fmt.Sprintf("%s", test.core)
			if result != test.expectS {
				t.Errorf("expected %%s\n%s\ngot %s", test.expectS, result)
			}

			result = fmt.Sprintf("%v", test.core)
			if result != test.expectS {
				t.Errorf("expected %%s\n%s\ngot %s", test.expectS, result)
			}

			result = fmt.Sprintf("%+v", test.core)
			if result != test.expectVPlus {
				t.Errorf("expected %%s\n%s\ngot %s", test.expectVPlus, result)
			}

			result = fmt.Sprintf("%#v", test.core)
			if result != test.expectVPlus {
				t.Errorf("expected %%s\n%s\ngot %s", test.expectVPlus, result)
			}
		})
	}
}
