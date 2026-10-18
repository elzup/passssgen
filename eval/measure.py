# 強度の実測: 1 つ目のファイルで重複テスト (推定 bit)、2 つ目以降を zxcvbn で評価する
# usage: pip install zxcvbn; passssgen -q -c 1000000 > a.txt; python3 eval/measure.py a.txt b.txt ...
import sys, statistics
from math import log2
from collections import Counter
from zxcvbn import zxcvbn
d = open(sys.argv[1]).read().split()
m = len(d)
dups = sum(c*(c-1)//2 for c in Counter(d).values())
print("collision test: samples", m, "pairs colliding", dups,
      "-> estimated bits %.1f" % log2(m*m/2/dups) if dups else "")
for f in sys.argv[2:]:
    pw = open(f).read().split()[:1000]
    bits = [log2(zxcvbn(p)["guesses"]) for p in pw]
    scores = Counter(zxcvbn(p)["score"] for p in pw[:300])
    print(f, "zxcvbn bits median %.1f min %.1f max %.1f" % (statistics.median(bits), min(bits), max(bits)), "scores", dict(scores))
