// passssgen generates passwords that are easy to type by hand: runs of one
// repeated character, every character class, and a single common symbol.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

var version = "dev"

const usage = `usage: passssgen [flags]

  -m mode     runs (default) or block

runs mode: runs of one repeated character
  -r runs     number of runs (default 5)
  -b len      characters per run (default 4)
  -j n        let each run length wobble by up to n around len (default 0)
  -a          let the symbol land inside a run, not only between runs

block mode: uppercase, symbol, digits, lowercase (e.g. BMMST.58rtvjw)
  -u n        uppercase characters (default 5)
  -d n        digits (default 2)
  -l n        lowercase characters (default 5)

both modes
  -s symbols  candidates for the single symbol (default "!@.-$?", "" for none)
              distinct ASCII punctuation only
  -c count    number of passwords, >= 1 (default 1)
  -q          hide the view and strength on stderr
  -version    print version

stderr shows the runs or blocks separated by spaces and the strength in bits:
  known  attacker knows passssgen, its flags and the length (the honest number)
  naive  attacker brute-forces the 94 printable ASCII characters (no space)
         at the same length
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("passssgen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	o := Options{}
	fs.IntVar(&o.Runs, "r", 5, "")
	fs.IntVar(&o.RunLen, "b", 4, "")
	fs.IntVar(&o.Jitter, "j", 0, "")
	fs.StringVar(&o.Symbols, "s", DefaultSymbols, "")
	fs.BoolVar(&o.Anywhere, "a", false, "")
	mode := fs.String("m", "runs", "")
	fs.IntVar(&o.Upper, "u", 5, "")
	fs.IntVar(&o.Digits, "d", 2, "")
	fs.IntVar(&o.Lower, "l", 5, "")
	count := fs.Int("c", 1, "")
	quiet := fs.Bool("q", false, "")
	showVersion := fs.Bool("version", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version)
		return 0
	}
	switch *mode {
	case "runs":
	case "block":
		o.Block = true
	default:
		fmt.Fprintf(stderr, "passssgen: mode must be runs or block, got %q\n", *mode)
		return 2
	}
	if *count < 1 {
		fmt.Fprintln(stderr, "passssgen: count must be >= 1")
		return 2
	}
	for range *count {
		res, err := Generate(o)
		if err != nil {
			fmt.Fprintln(stderr, "passssgen:", err)
			return 1
		}
		fmt.Fprintln(stdout, res.Password)
		if *quiet {
			continue
		}
		s := Estimate(o, len(res.Password))
		fmt.Fprintf(stderr, "  %s\n  %d chars, known %.1f bits, naive %.1f bits\n",
			res.View, len(res.Password), s.Known, s.Naive)
	}
	return 0
}
