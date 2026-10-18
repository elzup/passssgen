// hwpass generates passwords that survive being handwritten, read aloud, or
// typed on a TV remote: no ambiguous characters, no case, grouped by 4.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

var version = "dev"

const usage = `usage:
  hwpass [-g groups] [-s size] [-n count] [-q]   generate passwords
  hwpass normalize [-s size] <text>...            fix a hand-copied password
  hwpass version
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "normalize":
			return runNormalize(args[1:], stdout, stderr)
		case "version":
			fmt.Fprintln(stdout, version)
			return 0
		}
	}
	return runGenerate(args, stdout, stderr)
}

func runGenerate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("hwpass", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	groups := fs.Int("g", 4, "number of groups")
	size := fs.Int("s", 4, "characters per group")
	count := fs.Int("n", 1, "number of passwords")
	quiet := fs.Bool("q", false, "do not print entropy to stderr")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	for i := 0; i < *count; i++ {
		pw, err := Generate(*groups, *size)
		if err != nil {
			fmt.Fprintln(stderr, "hwpass:", err)
			return 1
		}
		fmt.Fprintln(stdout, pw)
	}
	if !*quiet {
		fmt.Fprintf(stderr, "%d bits\n", *groups**size*BitsPerChar)
	}
	return 0
}

func runNormalize(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("normalize", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	size := fs.Int("s", 4, "characters per group")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return 2
	}
	out, err := Normalize(strings.Join(fs.Args(), " "), *size)
	if err != nil {
		fmt.Fprintln(stderr, "hwpass:", err)
		return 1
	}
	fmt.Fprintln(stdout, out)
	return 0
}
