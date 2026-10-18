package main

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// Alphabet is Crockford Base32: digits + uppercase letters without I, L, O, U.
// 32 symbols = 5 bits each, so a random byte masked with 31 stays unbiased.
const Alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// BitsPerChar is the entropy each generated character carries.
const BitsPerChar = 5

// Generate returns groups*size random characters joined by "-".
func Generate(groups, size int) (string, error) {
	if groups < 1 || size < 1 {
		return "", fmt.Errorf("groups and size must be >= 1")
	}
	buf := make([]byte, groups*size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	parts := make([]string, groups)
	for g := range parts {
		chunk := make([]byte, size)
		for i := range chunk {
			chunk[i] = Alphabet[buf[g*size+i]&31]
		}
		parts[g] = string(chunk)
	}
	return strings.Join(parts, "-"), nil
}

// Normalize turns a hand-copied password back into its canonical form.
// Case is ignored, O reads as 0, I and L read as 1, and spaces / hyphens are
// dropped before regrouping by size.
func Normalize(input string, size int) (string, error) {
	if size < 1 {
		return "", fmt.Errorf("size must be >= 1")
	}
	var b strings.Builder
	for _, r := range strings.ToUpper(input) {
		switch {
		case r == ' ' || r == '-' || r == '\t':
			continue
		case r == 'O':
			r = '0'
		case r == 'I' || r == 'L':
			r = '1'
		case !strings.ContainsRune(Alphabet, r):
			return "", fmt.Errorf("invalid character %q", r)
		}
		b.WriteRune(r)
	}
	s := b.String()
	if s == "" {
		return "", fmt.Errorf("empty input")
	}
	var parts []string
	for len(s) > size {
		parts = append(parts, s[:size])
		s = s[size:]
	}
	return strings.Join(append(parts, s), "-"), nil
}
