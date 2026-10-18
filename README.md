# hwpass

紙に手書きする・口頭で伝える・リモコンで打つ、でも写し間違えないパスワードを出す CLI。

```console
$ hwpass
7KQM-2HXD-9RTF-4WBN
80 bits

$ hwpass normalize "7kqm 2hxd 9rtf 4wbn"
7KQM-2HXD-9RTF-4WBN
```

## 規則

| 項目 | 内容 |
| --- | --- |
| 文字 | Crockford Base32 の 32 文字 `0-9 A-Z` から `I L O U` を除く |
| 大文字小文字 | 区別しない (出力は大文字) |
| 区切り | 4 文字ごとに `-` |
| 強度 | 1 文字 5 bit。既定 16 文字 = 80 bit |
| 乱数 | `crypto/rand` |
| `normalize` | 小文字・空白を許し、`O` → `0`、`I` `L` → `1` に読み替えて正規化 |

## 使い方

```
hwpass [-g groups] [-s size] [-n count] [-q]
hwpass normalize [-s size] <text>...
hwpass version
```

- `-g` グループ数 (既定 4)、`-s` 1 グループの文字数 (既定 4)、`-n` 生成数、`-q` 強度表示を消す
- 強度は stderr に出るので `hwpass | pbcopy` でもパスワードだけ渡る

## インストール

```sh
go install github.com/elzup/hwpass@latest
```

またはリリースページからバイナリを取得 (`v*` タグで GitHub Actions がビルド)。

## 制約

「記号必須」「大文字小文字の混在必須」のサイトには、そのままでは通らない。
