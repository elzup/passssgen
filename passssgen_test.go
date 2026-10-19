package main

import (
	"bytes"
	"math"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var defaults = Options{Runs: 5, RunLen: 4, Symbols: DefaultSymbols}

func splitRuns(s string) []string {
	var runs []string
	for i := 0; i < len(s); {
		j := i
		for j < len(s) && s[j] == s[i] {
			j++
		}
		runs = append(runs, s[i:j])
		i = j
	}
	return runs
}

func TestGenerateDefaults(t *testing.T) {
	for range 500 {
		res, err := Generate(defaults)
		if err != nil {
			t.Fatal(err)
		}
		pw := res.Password
		if len(pw) != 21 {
			t.Fatalf("len(%q) = %d, want 21", pw, len(pw))
		}
		syms := 0
		var plain strings.Builder
		for _, r := range pw {
			if strings.ContainsRune(DefaultSymbols, r) {
				syms++
				continue
			}
			plain.WriteRune(r)
		}
		if syms != 1 {
			t.Fatalf("%q has %d symbols, want 1", pw, syms)
		}
		runs := splitRuns(plain.String())
		if len(runs) != 5 {
			t.Fatalf("%q has runs %v, want 5", pw, runs)
		}
		for _, r := range runs {
			if len(r) != 4 {
				t.Fatalf("%q has run %q, want length 4", pw, r)
			}
		}
		for _, cl := range classes {
			if !strings.ContainsAny(pw, cl) {
				t.Fatalf("%q misses class %q", pw, cl)
			}
		}
		if strings.ReplaceAll(res.View, " ", "") != pw {
			t.Fatalf("view %q does not match %q", res.View, pw)
		}
	}
}

func TestGenerateJitter(t *testing.T) {
	for _, j := range []int{1, 2} {
		o := Options{Runs: 8, RunLen: 4, Jitter: j}
		lens := map[int]bool{}
		for range 200 {
			res, err := Generate(o)
			if err != nil {
				t.Fatal(err)
			}
			runs := splitRuns(res.Password)
			if len(runs) != 8 {
				t.Fatalf("%q has %d runs, want 8", res.Password, len(runs))
			}
			for _, r := range runs {
				if len(r) < 4-j || len(r) > 4+j {
					t.Fatalf("jitter %d: run %q out of range", j, r)
				}
				lens[len(r)] = true
			}
		}
		if len(lens) != 2*j+1 {
			t.Fatalf("jitter %d: saw run lengths %v, want %d..%d", j, lens, 4-j, 4+j)
		}
	}
}

func TestGenerateNoAmbiguous(t *testing.T) {
	for range 300 {
		res, _ := Generate(defaults)
		if strings.ContainsAny(res.Password, "0O1lIo") {
			t.Fatalf("%q contains a look-alike", res.Password)
		}
	}
}

func TestGenerateAcceptsBoundaryOptions(t *testing.T) {
	for _, o := range []Options{{Runs: 3, RunLen: 1}, {Runs: 5, RunLen: 2, Jitter: 1},
		// ASCII punctuation right next to the alnum and printable edges
		{Runs: 5, RunLen: 4, Symbols: "!~/:@[`{"}} {
		if _, err := Generate(o); err != nil {
			t.Errorf("%+v: %v", o, err)
		}
	}
}

// Every character must count toward its class, including the first one of
// each class; otherwise single-character classes could never be satisfied.
func TestGenerateTinyClasses(t *testing.T) {
	saved := classes
	defer func() { classes = saved }()
	classes = []string{"a", "B", "3"}
	for range 50 {
		res, err := Generate(Options{Runs: 3, RunLen: 1})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Password) != 3 {
			t.Fatalf("%q, want 3 chars", res.Password)
		}
	}
}

func TestGenerateSymbolPositions(t *testing.T) {
	for _, tc := range []struct {
		anywhere bool
		want     []int
	}{
		// 3 runs of 2 leave 4 places between runs, both ends included
		{false, []int{0, 2, 4, 6}},
		// or 7 places when the symbol may split a run
		{true, []int{0, 1, 2, 3, 4, 5, 6}},
	} {
		o := Options{Runs: 3, RunLen: 2, Symbols: "!", Anywhere: tc.anywhere}
		seen := map[int]bool{}
		for range 700 {
			res, err := Generate(o)
			if err != nil {
				t.Fatal(err)
			}
			pos := strings.IndexByte(res.Password, '!')
			if !slices.Contains(tc.want, pos) {
				t.Fatalf("anywhere=%v: symbol at %d in %q", tc.anywhere, pos, res.Password)
			}
			seen[pos] = true
			if strings.ReplaceAll(res.View, " ", "") != res.Password {
				t.Fatalf("view %q does not match %q", res.View, res.Password)
			}
		}
		for _, pos := range tc.want {
			if !seen[pos] {
				t.Errorf("anywhere=%v: symbol never at %d, saw %v", tc.anywhere, pos, seen)
			}
		}
	}
}

func TestGenerateRejectsBadOptions(t *testing.T) {
	for _, o := range []Options{{Runs: 2, RunLen: 4}, {Runs: 5, RunLen: 0}, {Runs: 5, RunLen: 2, Jitter: 2}, {Runs: 5, RunLen: 4, Jitter: -1},
		{Runs: 5, RunLen: 4, Symbols: "!!"}, {Runs: 5, RunLen: 4, Symbols: "a!"}, {Runs: 5, RunLen: 4, Symbols: " "},
		{Runs: 5, RunLen: 4, Symbols: "★"}} {
		if _, err := Generate(o); err == nil {
			t.Errorf("%+v: want error", o)
		}
	}
	for _, c := range "azAZ09\x7f" {
		if _, err := Generate(Options{Runs: 5, RunLen: 4, Symbols: "!" + string(c)}); err == nil {
			t.Errorf("symbol %q: want error", c)
		}
	}
}

// enumerate lists every password Generate can return for o, by length.
func enumerate(o Options, runChars []string) map[int]map[string]bool {
	byLen := map[int]map[string]bool{}
	// cuts holds the offsets between runs, both ends included.
	var walk func(i int, plain string, runs []string, cuts []int)
	walk = func(i int, plain string, runs []string, cuts []int) {
		if i == o.Runs {
			pws := []string{plain}
			if o.Symbols != "" {
				pws = nil
				for _, sym := range o.Symbols {
					for pos := 0; pos <= len(plain); pos++ {
						if o.Anywhere || slices.Contains(cuts, pos) {
							pws = append(pws, plain[:pos]+string(sym)+plain[pos:])
						}
					}
				}
			}
			for _, pw := range pws {
				if byLen[len(pw)] == nil {
					byLen[len(pw)] = map[string]bool{}
				}
				byLen[len(pw)][pw] = true
			}
			return
		}
		for n := o.RunLen - o.Jitter; n <= o.RunLen+o.Jitter; n++ {
			next := plain + strings.Repeat(runs[i], n)
			walk(i+1, next, runs, append(slices.Clone(cuts), len(next)))
		}
	}
	for _, rc := range runChars {
		runs := strings.Split(rc, "")
		walk(0, "", runs, []int{0})
	}
	return byLen
}

func TestEstimateMatchesBruteForce(t *testing.T) {
	saved := classes
	defer func() { classes = saved }()
	classes = []string{"ab", "C", "34"}
	// run characters: no neighbour repeats, every class present
	var runChars []string
	all := strings.Join(classes, "")
	var walk func(prefix string)
	walk = func(prefix string) {
		if len(prefix) == 4 {
			for _, cl := range classes {
				if !strings.ContainsAny(prefix, cl) {
					return
				}
			}
			runChars = append(runChars, prefix)
			return
		}
		for i := 0; i < len(all); i++ {
			if len(prefix) == 0 || prefix[len(prefix)-1] != all[i] {
				walk(prefix + string(all[i]))
			}
		}
	}
	walk("")
	for _, o := range []Options{
		{Runs: 4, RunLen: 1},
		{Runs: 4, RunLen: 3, Symbols: "!@"},
		{Runs: 4, RunLen: 2, Jitter: 1},
		{Runs: 4, RunLen: 3, Jitter: 2},
		{Runs: 4, RunLen: 2, Jitter: 1, Symbols: "!@"},
		{Runs: 4, RunLen: 3, Symbols: "!@", Anywhere: true},
		{Runs: 4, RunLen: 2, Jitter: 1, Symbols: "!@", Anywhere: true},
	} {
		for length, pws := range enumerate(o, runChars) {
			got := Estimate(o, length).Known
			if want := math.Log2(float64(len(pws))); math.Abs(got-want) > 1e-9 {
				t.Errorf("%+v length %d: Estimate = %.6f bits, brute force = %.6f", o, length, got, want)
			}
		}
	}
}

func TestEstimateDefaults(t *testing.T) {
	s := Estimate(defaults, 21)
	if s.Known < 30 || s.Known > 40 {
		t.Fatalf("known = %.1f, want about 35", s.Known)
	}
	if s.Naive <= s.Known {
		t.Fatalf("naive %.1f should exceed known %.1f", s.Naive, s.Known)
	}
}

func TestRun(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-c", "2", "-r", "6"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || len(lines[0]) != 25 {
		t.Fatalf("unexpected stdout: %q", out.String())
	}
	if !strings.Contains(errOut.String(), "known") {
		t.Fatalf("stderr lacks strength: %q", errOut.String())
	}
}

func TestRunQuiet(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-q", "-s", ""}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if errOut.Len() != 0 || len(strings.TrimSpace(out.String())) != 20 {
		t.Fatalf("stdout %q stderr %q", out.String(), errOut.String())
	}
}

func TestRunExitCodes(t *testing.T) {
	for _, tc := range []struct {
		args   []string
		code   int
		stdout string
		stderr string
	}{
		{[]string{"-version"}, 0, "dev\n", ""},
		{[]string{"-h"}, 2, "", "usage: passssgen"},
		{[]string{"-c", "0"}, 2, "", "count must be >= 1"},
		{[]string{"-c", "-1"}, 2, "", "count must be >= 1"},
		{[]string{"-s", "!!"}, 1, "", "symbols must be distinct"},
		{[]string{"-m", "x"}, 2, "", "mode must be runs or block"},
		{[]string{"-m", "block", "-l", "0"}, 1, "", "every block needs"},
		{[]string{"-x"}, 2, "", "usage: passssgen"},
		{[]string{"-r", "2"}, 1, "", "runs must be >= 3"},
	} {
		var out, errOut bytes.Buffer
		if code := run(tc.args, &out, &errOut); code != tc.code {
			t.Errorf("%v: exit %d, want %d", tc.args, code, tc.code)
		}
		if out.String() != tc.stdout || !strings.Contains(errOut.String(), tc.stderr) {
			t.Errorf("%v: stdout %q stderr %q", tc.args, out.String(), errOut.String())
		}
	}
}

func TestGenerateBlocks(t *testing.T) {
	shape := regexp.MustCompile(`^([A-HJ-NP-Z]{6})([!@.\-$?])([2-9]{2})([a-km-np-z]{6})$`)
	o := Options{Block: true, Upper: 6, Digits: 2, Lower: 6, Symbols: DefaultSymbols}
	for range 300 {
		res, err := Generate(o)
		if err != nil {
			t.Fatal(err)
		}
		m := shape.FindStringSubmatch(res.Password)
		if m == nil {
			t.Fatalf("%q does not match the block layout", res.Password)
		}
		if want := strings.Join(m[1:], " "); res.View != want {
			t.Fatalf("view %q, want %q", res.View, want)
		}
	}
}

// Every block mode password must be reachable and equally counted, so the
// distinct outputs of a tiny alphabet must number exactly 2^Known.
func TestEstimateBlocksMatchesOutputs(t *testing.T) {
	saved := classes
	defer func() { classes = saved }()
	classes = []string{"ab", "C", "34"}
	for _, o := range []Options{
		{Block: true, Upper: 2, Digits: 1, Lower: 2, Symbols: "!@"},
		{Block: true, Upper: 1, Digits: 2, Lower: 1},
	} {
		seen := map[string]bool{}
		var length int
		for range 2000 {
			res, err := Generate(o)
			if err != nil {
				t.Fatal(err)
			}
			seen[res.Password] = true
			length = len(res.Password)
		}
		got := Estimate(o, length).Known
		if want := math.Log2(float64(len(seen))); math.Abs(got-want) > 1e-9 {
			t.Errorf("%+v: Estimate = %.3f bits, saw %d passwords (%.3f bits)", o, got, len(seen), want)
		}
	}
}
