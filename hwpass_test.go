package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestGenerateShape(t *testing.T) {
	re := regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{4}(-[0-9A-HJKMNP-TV-Z]{4}){3}$`)
	for i := 0; i < 1000; i++ {
		pw, err := Generate(4, 4)
		if err != nil {
			t.Fatal(err)
		}
		if !re.MatchString(pw) {
			t.Fatalf("unexpected shape: %q", pw)
		}
	}
}

func TestGenerateUsesWholeAlphabet(t *testing.T) {
	seen := map[rune]bool{}
	for i := 0; i < 200; i++ {
		pw, _ := Generate(4, 4)
		for _, r := range strings.ReplaceAll(pw, "-", "") {
			seen[r] = true
		}
	}
	if len(seen) != len(Alphabet) {
		t.Fatalf("saw %d symbols, want %d", len(seen), len(Alphabet))
	}
}

func TestGenerateRejectsBadArgs(t *testing.T) {
	if _, err := Generate(0, 4); err == nil {
		t.Fatal("want error for 0 groups")
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"7KQM-2HXD-9RTF-4WBN", "7KQM-2HXD-9RTF-4WBN"},
		{"7kqm 2hxd 9rtf 4wbn", "7KQM-2HXD-9RTF-4WBN"},
		{"oil0", "0110"},
		{"7KQM2HXD9", "7KQM-2HXD-9"},
	}
	for _, c := range cases {
		got, err := Normalize(c.in, 4)
		if err != nil || got != c.want {
			t.Errorf("Normalize(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestNormalizeRejects(t *testing.T) {
	for _, in := range []string{"", "--", "ABCU", "AB#C"} {
		if _, err := Normalize(in, 4); err == nil {
			t.Errorf("Normalize(%q): want error", in)
		}
	}
}

func TestRunGenerate(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-g", "3", "-n", "2"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || len(lines[0]) != 14 {
		t.Fatalf("unexpected output: %q", out.String())
	}
	if errOut.String() != "60 bits\n" {
		t.Fatalf("unexpected stderr: %q", errOut.String())
	}
}

func TestRunNormalize(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"normalize", "7kqm", "2hxd"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if out.String() != "7KQM-2HXD\n" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}
