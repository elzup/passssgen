// hwpass generates passwords that are easy to type by hand: runs of one
// repeated character, every character class, and a single common symbol.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

var version = "dev"

const usage = `usage: hwpass [flags]

  -r runs     number of runs (default 5)
  -b len      characters per run (default 4)
  -v          vary run length randomly in 1..len (stronger, but you count)
  -s symbols  candidates for the single symbol (default "!@.-$?", "" for none)
  -c count    number of passwords (default 1)
  -q          print only the password
  -version    print version

stderr shows the runs separated by spaces and the strength in bits:
  known  attacker knows hwpass and its flags (the honest number)
  naive  attacker brute-forces all printable ASCII of the same length
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("hwpass", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	o := Options{}
	fs.IntVar(&o.Runs, "r", 5, "")
	fs.IntVar(&o.RunLen, "b", 4, "")
	fs.BoolVar(&o.Vary, "v", false, "")
	fs.StringVar(&o.Symbols, "s", DefaultSymbols, "")
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
	for i := 0; i < *count; i++ {
		res, err := Generate(o)
		if err != nil {
			fmt.Fprintln(stderr, "hwpass:", err)
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
