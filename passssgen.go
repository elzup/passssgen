package main

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"strings"
)

// Character classes with look-alikes removed (0 O o, 1 l I, etc).
var classes = []string{
	"abcdefghijkmnpqrstuvwxyz", // lower: no l, o
	"ABCDEFGHJKLMNPQRSTUVWXYZ", // upper: no I, O
	"23456789",                 // digit: no 0, 1
}

// DefaultSymbols are accepted by most sites and sit on the first symbol
// layer of phone keyboards.
const DefaultSymbols = "!@.-$?"

// Options controls the shape of a password.
type Options struct {
	Runs    int    // number of runs (blocks of one repeated character)
	RunLen  int    // characters per run (the centre when Jitter > 0)
	Jitter  int    // run lengths are random in RunLen-Jitter..RunLen+Jitter
	Symbols string // candidates for the single inserted symbol; empty disables it
}

func (o Options) validate() error {
	if o.Runs < len(classes) {
		return fmt.Errorf("runs must be >= %d to use every class", len(classes))
	}
	if o.Jitter < 0 {
		return fmt.Errorf("jitter must be >= 0")
	}
	if o.RunLen-o.Jitter < 1 {
		return fmt.Errorf("run length minus jitter must be >= 1")
	}
	return nil
}

// Result is a generated password plus a view that shows its runs.
type Result struct {
	Password string
	View     string // runs separated by spaces, for reading while typing
}

func randInt(n int) (int, error) {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(v.Int64()), nil
}

// Generate builds Runs runs of one repeated character each. Neighbouring runs
// never share a character, every class appears at least once, and one symbol
// is inserted at a random position. Retrying on a missing class keeps the
// result uniform over all valid passwords.
func Generate(o Options) (Result, error) {
	if err := o.validate(); err != nil {
		return Result{}, err
	}
	all := strings.Join(classes, "")
	for {
		runs := make([]string, o.Runs)
		used := make([]bool, len(classes))
		prev := byte(0)
		for i := range runs {
			var c byte
			for {
				k, err := randInt(len(all))
				if err != nil {
					return Result{}, err
				}
				if c = all[k]; c != prev {
					break
				}
			}
			prev = c
			for ci, cl := range classes {
				if strings.IndexByte(cl, c) >= 0 {
					used[ci] = true
				}
			}
			n := o.RunLen
			if o.Jitter > 0 {
				k, err := randInt(2*o.Jitter + 1)
				if err != nil {
					return Result{}, err
				}
				n += k - o.Jitter
			}
			runs[i] = strings.Repeat(string(c), n)
		}
		if !allTrue(used) {
			continue
		}
		return insertSymbol(runs, o.Symbols)
	}
}

func allTrue(bs []bool) bool {
	for _, b := range bs {
		if !b {
			return false
		}
	}
	return true
}

func insertSymbol(runs []string, symbols string) (Result, error) {
	plain := strings.Join(runs, "")
	if symbols == "" {
		return Result{plain, strings.Join(runs, " ")}, nil
	}
	k, err := randInt(len(symbols))
	if err != nil {
		return Result{}, err
	}
	pos, err := randInt(len(plain) + 1)
	if err != nil {
		return Result{}, err
	}
	sym := string(symbols[k])
	// Place the symbol inside the view too, so the view reads the same.
	view := make([]string, len(runs))
	offset := 0
	placed := false
	for i, r := range runs {
		if !placed && pos <= offset+len(r) {
			r = r[:pos-offset] + sym + r[pos-offset:]
			placed = true
		}
		offset += len(runs[i])
		view[i] = r
	}
	return Result{plain[:pos] + sym + plain[pos:], strings.Join(view, " ")}, nil
}

// Strength reports the guessing space in bits.
type Strength struct {
	Known float64 // attacker knows this algorithm and its options
	Naive float64 // attacker brute-forces every printable ASCII string of this length
}

// Estimate computes Strength for o and a password of the given length.
// Known counts every password Generate can return for o: run characters with
// no neighbour repeats and all classes present, run lengths when varied, and
// the symbol's choice and position.
func Estimate(o Options, length int) Strength {
	sizes := make([]float64, len(classes))
	for i, cl := range classes {
		sizes[i] = float64(len(cl))
	}
	full := 1<<len(classes) - 1
	// dp[mask][last] = sequences whose used classes are mask and whose last
	// run is in class last. Counts in float64 stay exact enough for bits.
	dp := make([][]float64, full+1)
	for m := range dp {
		dp[m] = make([]float64, len(classes))
	}
	for c := range classes {
		dp[1<<c][c] = sizes[c]
	}
	for step := 1; step < o.Runs; step++ {
		next := make([][]float64, full+1)
		for m := range next {
			next[m] = make([]float64, len(classes))
		}
		for m := range dp {
			for last, v := range dp[m] {
				if v == 0 {
					continue
				}
				for c := range classes {
					choices := sizes[c]
					if c == last {
						choices--
					}
					next[m|1<<c][c] += v * choices
				}
			}
		}
		dp = next
	}
	total := 0.0
	for _, v := range dp[full] {
		total += v
	}
	bits := math.Log2(total)
	if o.Jitter > 0 {
		bits += float64(o.Runs) * math.Log2(float64(2*o.Jitter+1))
	}
	if o.Symbols != "" {
		// With jitter the insertion points depend on run lengths; use this
		// password's length as the estimate.
		plainLen := length - 1
		bits += math.Log2(float64(len(o.Symbols))) + math.Log2(float64(plainLen+1))
	}
	return Strength{Known: bits, Naive: float64(length) * math.Log2(94)}
}
