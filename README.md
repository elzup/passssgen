# passssgen

A CLI that generates passwords you can still type by hand. Half joke, fully usable.

The name reads `pa ssss gen`: a 4-character run, `ssss`, sits in the middle.

A password is a row of runs (one character repeated) plus a single symbol. Strength is reported honestly, in bits, against **an attacker who knows this algorithm**.

```console
$ passssgen
ffffyyyy2222EEEE-9999
  ffff yyyy 2222 EEEE- 9999
  21 chars, known 33.0 bits, naive 137.6 bits

$ passssgen -r 13
DDDD7777aaaahhhhLLLLRRRREEEESSSSRRRR?vvvvffff9999KKKK
  DDDD 7777 aaaa hhhh LLLL RRRR EEEE SSSS RRRR? vvvv ffff 9999 KKKK
  53 chars, known 81.4 bits, naive 347.4 bits
```

Or, in block mode, one block per character class so a phone keyboard rarely changes layer:

```console
$ passssgen -m block
YMMUDK@33xwabdu
  YMMUDK @ 33 xwabdu
  15 chars, known 63.6 bits, naive 98.3 bits
```

stdout gets only the password. The view and strength go to stderr, so `passssgen | pbcopy` copies just the password.

## Rules

| Item | Rule |
| --- | --- |
| Runs | Each run repeats one character `-b` times (default 4). Neighbouring runs always differ |
| Classes | Lowercase, uppercase and digits all appear, plus one symbol (none with `-s ""`) |
| Look-alikes | `0 O o 1 l I` are never used |
| Symbol | One of `! @ . - $ ?`, between two runs or at either end, so it never splits a run (`-a` allows any position). Only symbols most sites accept, so you don't get stuck at a password form |
| Randomness | `crypto/rand`. Candidates that break a rule are thrown away and redrawn, so there is no bias |

## Strength

| Label | Meaning |
| --- | --- |
| `known` | Guessing space for an attacker who knows passssgen, its flags and the password length. **This is the real strength.** The length is treated as known because it leaks, e.g. through the dots in a masked field |
| `naive` | An attacker brute-forcing the 94 printable ASCII characters (no space) at the same length. For reference only |

Length does not make it strong; **the number of runs** does.

| Options | Length | known |
| --- | --- | --- |
| default (`-r 5`) | 21 | ~33 bits: online logins with rate limiting only |
| `-r 13` | 53 | ~81 bits: on par with 12 random characters including symbols (79 bits) |
| `-j 1` | ~21 | ~38 bits on average (min ~33): run lengths vary from 3 to 5 |
| `-r 11 -j 1` | ~45 | ~83 bits on average (min ~73): 8 characters shorter than fixed `-r 13` |
| `-a` | 21 | ~35 bits: the symbol may split a run, +1.8 bits at the default size |

With `-j`, strength varies per password. Once the length is known, only run lengths that add up to it remain. At extreme lengths, such as every run at its shortest or longest, few combinations remain and the password is weaker. Check `known` in the output and generate again if it falls short.

## Block mode

`-m block` drops the runs. It writes random uppercase letters, the symbol, digits, then lowercase letters, always in that order. Typing it on a phone takes about three keyboard switches (caps lock on, the number layer, back to letters), however long the blocks are. Mixed runs switch every few characters instead.

| Options | Length | known | Example |
| --- | --- | --- | --- |
| `-m block` (`-u 6 -d 2 -l 6`) | 15 | ~64 bits: enough for Wi-Fi (WPA2) | `YMMUDK@33xwabdu` |
| `-m block -u 8 -d 3 -l 8` | 20 | ~85 bits: on par with 13 random characters | `JRQEUSEH-925buyzmifr` |
| `-m block -u 4 -d 1 -l 3` | 9 | ~38 bits: online only | `FCVD!5vzv` |

Every character is random and the layout is fixed, so `known` is exact. Compared with runs mode, you trade the repeated characters for far fewer switches and a much shorter password at the same strength.

## Keep quiet about using it

Don't say publicly, or next to an account, that you use passssgen. An attacker who knows it only has to search the `known` space, which is far smaller than `naive`. Keeping quiet can push an attacker back toward `naive`, but treat that as a bonus, never as protection. If one of your passwords ever leaks in plain text (a phishing page, a site that stores passwords badly), its shape (runs, or the block layout) gives away the generator, and your other accounts then face the `known` search. A good password stays safe even when the attacker knows how it was made. Pick options so that `known` alone is strong enough for the account.

## Usage

```
passssgen [-m runs] [-r runs] [-b len] [-j jitter] [-a] [-s symbols] [-c count] [-q]
passssgen -m block [-u upper] [-d digits] [-l lower] [-s symbols] [-c count] [-q]
```

| Flag | Default | Description |
| --- | --- | --- |
| `-m` | `runs` | `runs` or `block` |
| `-r` | 5 | Number of runs in one password |
| `-b` | 4 | Characters per run (the centre when `-j` is set) |
| `-j` | 0 | How far run lengths may vary. `-b 4 -j 1` gives 3 to 5. Adds strength, but you have to count characters from the run view |
| `-s` | `!@.-$?` | Symbol candidates. Distinct ASCII punctuation only (no letters, digits, spaces or non-ASCII). `""` for no symbol |
| `-a` | off | Let the symbol land inside a run too, not only between runs |
| `-u` | 6 | Block mode: uppercase letters |
| `-d` | 2 | Block mode: digits |
| `-l` | 6 | Block mode: lowercase letters |
| `-c` | 1 | Number of passwords to print, at least 1 |
| `-q` | off | Hide the view and strength on stderr |

## Install

```sh
go install github.com/elzup/passssgen@latest
```

Or grab a binary from the releases, built by GitHub Actions on `v*` tags.
