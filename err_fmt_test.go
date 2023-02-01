package clues_test

import (
	stderr "errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/pkg/errors"

	"github.com/alcionai/clues"
)

const ()

func plusRE(lines ...string) string {
	var s string

	for i := 0; i < len(lines); i += 2 {
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

func (c checkFmt) check(t *testing.T, err error) {
	t.Run(c.tmpl, func(t *testing.T) {
		result := fmt.Sprintf(c.tmpl, err)

		if len(c.expect) > 0 && result != c.expect {
			t.Errorf("unexpected format for template %#v\nexpected \"%s\"\ngot \"%s\"", c.tmpl, c.expect, result)
		}

		if c.reExpect != nil && !c.reExpect.MatchString(result) {
			t.Errorf("unexpected format for template %#v\nexpected \"%v\"\ngot %#v", c.tmpl, c.reExpect, result)
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

	cluesWrap  = func(err error) error { return clues.Wrap(err, "clues wrap") }
	cluesStack = func(err error) error { return clues.Stack(globalSentinel, err) }
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`clues wrap\n`, `err_fmt_test.go:\d+\n`,
					`sentinel\n`, "",
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`mid\n`, ``,
					`top$`, ``,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`mid\n`, ``,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`top\n`, ``,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`mid\n`, ``,
					`top$`, ``,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`mid\n`, `err_fmt_test.go:\d+\n`,
					`top\n`, `err_fmt_test.go:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`rhs\n`, `err_fmt_test.go:\d+\n`,
					`mid\n`, ``,
					`top\n`, ``,
					`lhs\n`, `err_fmt_test.go:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`rhs\n`, `err_fmt_test.go:\d+\n`,
					`mid\n`, ``,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`top\n`, ``,
					`github.com/alcionai/clues_test.TestFmt\n`, `err_fmt_test.go:\d+\n`,
					`testing.tRunner\n`, `testing.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`lhs\n`, `err_fmt_test.go:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`rhs\n`, `err_fmt_test.go:\d+\n`,
					`mid\n`, ``,
					`top\n`, ``,
					`lhs\n`, `err_fmt_test.go:\d+$`,
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
					`github.com/alcionai/clues_test.init\n`, `err_fmt_test.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.doInit\n`, `proc.go:\d+\n`,
					`runtime.main\n`, `proc.go:\d+\n`,
					`runtime.goexit\n`, `runtime/.*:\d+\n`,
					`rhs\n`, `err_fmt_test.go:\d+\n`,
					`mid\n`, `err_fmt_test.go:\d+\n`,
					`top\n`, `err_fmt_test.go:\d+\n`,
					`lhs\n`, `err_fmt_test.go:\d+$`,
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
