package main

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"slices"
	"strings"
	"unicode"
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
	// Anywhere lets the symbol land inside a run too. By default it only goes
	// between runs or at either end, so no run is split while typing.
	Anywhere bool
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
	// Symbols are picked and counted per byte, so each must be one distinct
	// ASCII character that cannot merge into a run.
	for i, r := range o.Symbols {
		punct := unicode.IsPunct(r) || unicode.IsSymbol(r)
		if r > unicode.MaxASCII || !punct || strings.ContainsRune(o.Symbols[:i], r) {
			return fmt.Errorf("symbols must be distinct ASCII punctuation, got %q", o.Symbols)
		}
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
				if strings.ContainsRune(cl, rune(c)) {
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
		if slices.Contains(used, false) {
			continue
		}
		return insertSymbol(runs, o)
	}
}

func insertSymbol(runs []string, o Options) (Result, error) {
	plain := strings.Join(runs, "")
	if o.Symbols == "" {
		return Result{plain, strings.Join(runs, " ")}, nil
	}
	k, err := randInt(len(o.Symbols))
	if err != nil {
		return Result{}, err
	}
	pos, err := randInt(symbolSlots(o, len(plain)))
	if err != nil {
		return Result{}, err
	}
	if !o.Anywhere {
		// pos is a boundary index; turn it into the offset after pos runs.
		pos = len(strings.Join(runs[:pos], ""))
	}
	sym := string(o.Symbols[k])
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

// symbolSlots is how many places the symbol can go in a password whose runs
// add up to plainLen characters.
func symbolSlots(o Options, plainLen int) int {
	if o.Anywhere {
		return plainLen + 1
	}
	return o.Runs + 1
}

// Strength reports the guessing space in bits.
type Strength struct {
	Known float64 // attacker knows this algorithm, its options and the length
	Naive float64 // attacker brute-forces the 94 printable ASCII characters (no space) at this length
}

// Estimate computes Strength for o and a password of the given length.
// Known counts every password of that length Generate can return for o: run
// characters with no neighbour repeats and all classes present, run lengths
// that add up to the length, and the symbol's choice and position. The length
// is treated as known because it shows in masked fields and is easy to count.
func Estimate(o Options, length int) Strength {
	sizes := make([]float64, len(classes))
	for i, cl := range classes {
		sizes[i] = float64(len(cl))
	}
	full := 1<<len(classes) - 1
	// dp[mask][last] = sequences whose used classes are mask and whose last
	// run is in class last. Counts in float64 stay exact enough for bits.
	table := func() [][]float64 {
		t := make([][]float64, full+1)
		for m := range t {
			t[m] = make([]float64, len(classes))
		}
		return t
	}
	dp := table()
	for c := range classes {
		dp[1<<c][c] = sizes[c]
	}
	for range o.Runs - 1 {
		next := table()
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
	plainLen := length
	if o.Symbols != "" {
		plainLen--
	}
	bits := math.Log2(total) + math.Log2(lengthSplits(o, plainLen))
	if o.Symbols != "" {
		bits += math.Log2(float64(len(o.Symbols))) + math.Log2(float64(symbolSlots(o, plainLen)))
	}
	return Strength{Known: bits, Naive: float64(length) * math.Log2(94)}
}

// lengthSplits counts the run lengths in RunLen-Jitter..RunLen+Jitter that add
// up to total. Every split is equally likely, so it is the guessing space for
// run lengths once the length is known.
func lengthSplits(o Options, total int) float64 {
	ways := make([]float64, total+1)
	ways[0] = 1
	for range o.Runs {
		next := make([]float64, total+1)
		for sum, v := range ways {
			for n := o.RunLen - o.Jitter; n <= min(o.RunLen+o.Jitter, total-sum); n++ {
				next[sum+n] += v
			}
		}
		ways = next
	}
	return ways[total]
}
