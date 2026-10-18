package main

import (
	"bytes"
	"math"
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
	for i := 0; i < 500; i++ {
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
		for i := 0; i < 200; i++ {
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
	for i := 0; i < 300; i++ {
		res, _ := Generate(defaults)
		if strings.ContainsAny(res.Password, "0O1lIo") {
			t.Fatalf("%q contains a look-alike", res.Password)
		}
	}
}

func TestGenerateAcceptsBoundaryOptions(t *testing.T) {
	for _, o := range []Options{{Runs: 3, RunLen: 1}, {Runs: 5, RunLen: 2, Jitter: 1}} {
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
	for i := 0; i < 50; i++ {
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
	o := Options{Runs: 3, RunLen: 1, Symbols: "!"}
	seen := map[int]bool{}
	for i := 0; i < 500; i++ {
		res, err := Generate(o)
		if err != nil {
			t.Fatal(err)
		}
		pos := strings.IndexByte(res.Password, '!')
		seen[pos] = true
		if strings.ReplaceAll(res.View, " ", "") != res.Password {
			t.Fatalf("view %q does not match %q", res.View, res.Password)
		}
	}
	// 3 run characters leave 4 places for the symbol, both ends included.
	for pos := 0; pos <= 3; pos++ {
		if !seen[pos] {
			t.Errorf("symbol never at %d, saw %v", pos, seen)
		}
	}
}

func TestGenerateRejectsBadOptions(t *testing.T) {
	for _, o := range []Options{{Runs: 2, RunLen: 4}, {Runs: 5, RunLen: 0}, {Runs: 5, RunLen: 2, Jitter: 2}, {Runs: 5, RunLen: 4, Jitter: -1}} {
		if _, err := Generate(o); err == nil {
			t.Errorf("%+v: want error", o)
		}
	}
}

// bruteCount enumerates run characters for tiny alphabets to check Estimate's DP.
func bruteCount(runs int) int {
	all := strings.Join(classes, "")
	count := 0
	var walk func(prefix string)
	walk = func(prefix string) {
		if len(prefix) == runs {
			for _, cl := range classes {
				if !strings.ContainsAny(prefix, cl) {
					return
				}
			}
			count++
			return
		}
		for i := 0; i < len(all); i++ {
			if len(prefix) > 0 && prefix[len(prefix)-1] == all[i] {
				continue
			}
			walk(prefix + string(all[i]))
		}
	}
	walk("")
	return count
}

func TestEstimateMatchesBruteForce(t *testing.T) {
	saved := classes
	defer func() { classes = saved }()
	classes = []string{"ab", "C", "34"}
	runChars := math.Log2(float64(bruteCount(4)))
	for _, tc := range []struct {
		o      Options
		length int
		want   float64
	}{
		{Options{Runs: 4, RunLen: 1}, 4, runChars},
		// each run length has 3 choices
		{Options{Runs: 4, RunLen: 2, Jitter: 1}, 8, runChars + 4*math.Log2(3)},
		{Options{Runs: 4, RunLen: 3, Jitter: 2}, 12, runChars + 4*math.Log2(5)},
		// 2 symbols at any of 4*3+1 positions
		{Options{Runs: 4, RunLen: 3, Symbols: "!@"}, 13, runChars + 1 + math.Log2(13)},
	} {
		if got := Estimate(tc.o, tc.length).Known; math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("%+v: Estimate = %.6f bits, brute force = %.6f", tc.o, got, tc.want)
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
