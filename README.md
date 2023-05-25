# capcode
Lossless normalization of uppercase characters.
Native implementations are available in Python, Javascript & Go. There is also a command-line tool in the "executables" directory. You can compile it as follows:
```
git clone https://github.com/alasdairforsythe/capcode
cd capcode/executables
go mod init capcode
go mod tidy
go build capcode.go
```

### Examples:
```
The QUICK BROWN FOX Jumped over the LAZY dog. NextOne. THANK YOU!
```
```
Cthe Bquick brown foxE Cjumped over the Wlazy dog. CnextCone. Wthank Wyou!
```

### Fully UTF-8 Compliant

- Supports Unicode 13.0.0 (newer version supported depending on the implementation)
- Works correctly with any UTF-8 encoding scheme (NFC, NFD, etc.)

### Features

- No information is lost
- The encoded text can be decoded exactly back to the original
- Extremely fast: no regular expressions, only 1 loop of the text
- Safe: an LLM trained on this will still understand about uppercasing

### To Do

☐ Optimized pure C implementation

☐ Make a Python module wrapping the C implementation

☐ Push it to PyPI

### Formula

Definitions:
- WordSeparator is any glyph that is not a letter, number or one of two apostrophes '’
- CapitalWord is a word where every letter is uppercase and it's bounded by a WordSeparator on both sides, or end of text

Decoding:
- The C characterToken makes the following 1 UTF8 glyph uppercase
- The W wordToken makes all characters following this uppercase until a WordSeparator reached
- The B beginToken makes all glyphs uppercase until the next E endToken

Encoding:
- 3 or more CapitalWords in sequence are lowercased and begin with B beginToken and end with E endToken, e.g. THE QUICK BROWN -> Sthe quick brownE
- 1 or 2 CapitalWords in sequence are each proceeded by W wordToken, e.g. THE QUICK -> Wthe Wquick
- If 2 or more letters at the end of a word are uppercased, and its followed by 2 or more CapitalWords, insert B beginToken just before the 2 or more letters, E endToken after the CapitalWords and lowercase all in between, e.g. tHE QUICK BROWN -> tShe quick brownE
- If 1 or more letters at the end of a word are uppercased, the uppercased letters are lowercased and proceeded by W wordToken, e.g. teST -> teWst, tesT -> tesWt
- Any other uppercase characters within a word are lowercased and proceeded by the C characterToken, e.g. Test -> Ctest, tESt -> tCeCst

Notes:
- Titlecase glyphs (for special glphs that have distinct uppercase & titlecase) are left unchanged
- C characterToken never occurs before the last character in a word, in that case W wordToken is used (W uppercases all characters from here until the end of the word)
- E endToken never occurs in the middle of a word, while B beginToken may occur in the middle of a word
