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
	o := Options{Runs: 8, RunLen: 4, Jitter: 1}
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
			if len(r) < 3 || len(r) > 5 {
				t.Fatalf("run %q out of range", r)
			}
			lens[len(r)] = true
		}
	}
	if len(lens) != 3 {
		t.Fatalf("saw run lengths %v, want 3..5", lens)
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

func TestGenerateRejectsBadOptions(t *testing.T) {
	for _, o := range []Options{{Runs: 2, RunLen: 4}, {Runs: 5, RunLen: 0}, {Runs: 5, RunLen: 2, Jitter: 2}, {Runs: 5, RunLen: 4, Jitter: -1}} {
		if _, err := Generate(o); err == nil {
			t.Errorf("%+v: want error", o)
		}
	}
}

// bruteCount enumerates run characters for tiny alphabets to check Estimate's DP.
func TestEstimateMatchesBruteForce(t *testing.T) {
	saved := classes
	defer func() { classes = saved }()
	classes = []string{"ab", "C", "34"}
	all := strings.Join(classes, "")
	o := Options{Runs: 4, RunLen: 1}
	count := 0
	var walk func(prefix string)
	walk = func(prefix string) {
		if len(prefix) == o.Runs {
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
	got := Estimate(o, o.Runs).Known
	if want := math.Log2(float64(count)); math.Abs(got-want) > 1e-9 {
		t.Fatalf("Estimate = %.6f bits, brute force = %.6f", got, want)
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
