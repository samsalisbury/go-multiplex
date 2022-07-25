package multiplex

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// These tests should not be run in parallel as they all share a single,
// global io.Writer (testStdout).

var testStdout io.Writer = os.Stdout

func write(s string) { fmt.Fprintln(testStdout, s) }

func lines(s ...string) string { return strings.Join(s, "\n") + "\n" }

//
// Define types to use in tests.
//

// Doer is the example interface we are multiplexing.
type Doer interface{ Do() }

// compoundDoer is itself a Doer and contains two other Doers.
type compoundDoer struct {
	Doer1  doer1
	Doer2  *doer2
	suffix string
}

func (cd *compoundDoer) Do() { write("compoundDoer" + cd.suffix) }

// doerContainer is not itself a Doer, but it contains Doers.
type doerContainer struct {
	Doer1 doer1
	Doer2 *doer2
}

// doer1 and doer2 are simple doers.
type doer1 struct{ suffix string }
type doer2 struct{ suffix string }

func (d1 *doer1) Do() { write("doer1" + d1.suffix) }
func (d2 *doer2) Do() { write("doer2" + d2.suffix) }

// multiDoer is a Doer and exists solely to group other Doers and call their Do
// methods one at a time.
type multiDoer []Doer

func (md *multiDoer) Collect(d Doer) { *md = append(*md, d) }

func (md *multiDoer) Do() {
	for _, d := range *md {
		d.Do()
	}
}

func TestInterface_ok(t *testing.T) {

	cases := []struct {
		desc     string
		toplevel any
		options  []Option
		want     string
	}{
		{
			desc:     "single doer",
			toplevel: &doer1{"-hi"},
			options:  nil,
			want:     lines("doer1-hi"),
		},
		{
			desc: "compound doer",
			toplevel: &compoundDoer{
				Doer1: doer1{"-hallo"},
				Doer2: &doer2{"-two"},
			},
			options: nil,
			want:    lines("doer1-hallo", "doer2-two", "compoundDoer"),
		},
		{
			desc: "compound doer one nil field default (create)",
			toplevel: &compoundDoer{
				Doer1: doer1{"-one"},
				Doer2: nil,
			},
			options: nil,
			want:    lines("doer1-one", "doer2", "compoundDoer"),
		},
		{
			desc: "compound doer one nil field create",
			toplevel: &compoundDoer{
				Doer1: doer1{"-one"},
				Doer2: nil,
			},
			options: []Option{OptCreateNilFields},
			want:    lines("doer1-one", "doer2", "compoundDoer"),
		},
		{
			desc: "compound doer one nil field skip",
			toplevel: &compoundDoer{
				Doer1: doer1{"-alone"},
				Doer2: nil,
			},
			options: []Option{OptSkipNilFields},
			want:    lines("doer1-alone", "compoundDoer"),
		},
		{
			desc: "doer container",
			toplevel: &doerContainer{
				Doer1: doer1{"-blargle"},
				Doer2: &doer2{"-brrr"},
			},
			options: nil,
			want:    lines("doer1-blargle", "doer2-brrr"),
		},
		{
			desc: "doer container - one nil field - default (create)",
			toplevel: &doerContainer{
				Doer1: doer1{"-blargle"},
				Doer2: nil,
			},
			options: nil,
			want:    lines("doer1-blargle", "doer2"),
		},
		{
			desc: "doer container - one nil field - create",
			toplevel: &doerContainer{
				Doer1: doer1{"-blargle"},
				Doer2: nil,
			},
			options: []Option{OptCreateNilFields},
			want:    lines("doer1-blargle", "doer2"),
		},
		{
			desc: "doer container - one nil field - skip",
			toplevel: &doerContainer{
				Doer1: doer1{"-blargle"},
				Doer2: nil,
			},
			options: []Option{OptSkipNilFields},
			want:    lines("doer1-blargle"),
		},
	}

	for _, c := range cases {
		desc, toplevel, options, want := c.desc, c.toplevel, c.options, c.want
		t.Run(desc, func(t *testing.T) {
			buf := &bytes.Buffer{}
			testStdout = buf
			multi := Interface[Doer](toplevel, &multiDoer{}, options...)
			multi.Do()
			got := buf.String()
			if got != want {
				t.Errorf("got %q; want %q", got, want)
			}
		})
	}

}
