# capcode
Lossless normalization of uppercase characters.
Currently only in Go, but in "executables" directory is a command line tool. You can compile it as follows:
```
git clone https://github.com/alasdairforsythe/capcode/executables
go mod init
go mod tidy
go build capcode.go
```

## Examples:
```
The QUICK BROWN FOX Jumped over the LAZY dog. CamelCase. THANK YOU!
```
```
Cthe Bquick brown foxE Cjumped over the Wlazy dog. CcamelCase. Wthank Wyou!
```

## Features

- UTF-8 compliant: supports uppercase and titlecase glpyhs
- No information is lost
- The encoded text can be decoded exactly back to the original
- Extremely fast: no regular expressions, only 1 loop of the text
- Safe: an LLM trained on this will still understand about uppercasing

## Formula

Definitions:
- WordSeparator is any glyph that is not a letter, number or one of two apostrophes '’
- CapitalWord is a word where every letter is uppercase and it's bounded by a WordSeparator on both sides

Decoding:
- The C characterToken makes the following 1 UTF8 glyph uppercase
- The T titleToken makes the following UTF8 glyph titlecase (for special glphs that have distinct uppercase & titlecase)
- The W wordToken makes all characters following this uppercase until a WordSeparator reached
- The S startToken makes all glyphs uppercase until the next E endToken

Encoding:
- Any titlecase glyph is to be lowercased and proceeded by T titleToken (for special glphs that have distinct uppercase & titlecase)
- 3 or more CapitalWords in sequence are lowercased and begin with S startToken and end with E endToken, e.g. The Quick Brown -> Sthe quick brownE
- 1 or 2 CapitalWords in sequence are each proceeded by W wordToken, e.g. The Quick -> Wthe Wquick
- If 2 or more letters at the end of a word are uppercased, and its followed by 2 or more CapitalWords, insert S startToken just before the 2 or more letters, E endToken after the CapitalWords and lowercase all in between
- If 1 or more letters at the end of a word are uppercased, the uppercased letters are lowercased and proceeded by W wordTOken, e.g. teST -> teWst, tesT -> tesWt
- Any other uppercase characters within a word are lowercased and proceeded by the C characterToken, e.g. Test -> Ctest, tESt -> tCeCst

Notes:
- Titlecase glyphs are always proceeded by T titleToken, and are otherwise unrelated to the rules for the uppercase
- C characterToken never occurs before the last character in a word, in that case W wordToken is used
- E EndToken never occurs in the middle of a word, while s StartToken may occur in the middle of a word