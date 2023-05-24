package capcode

/*

	capcode

	For encoding uppercasing and titlecasing into lowercasing.

	- Parsed as UTF-8 glyphs
	- WordSeparator is any glyph that is not a letter, number or one of two apostrophes '’
	- CapitalWord is a word where every letter is uppercase and it's bounded by a WordSeparator on both sides, or end of text
	
	Decoding:
		The C characterToken makes the following 1 UTF8 glyph uppercase
		The T titleToken makes the following UTF8 glyph titlecase (for special glphs that have distinct uppercase & titlecase)
		The W wordToken makes all characters following this uppercase until a WordSeparator reached
		The S startToken makes all glyphs uppercase until the next E endToken

	Encoding:
		Any titlecase glyph is to be lowercased and proceeded by T titleToken (for special glphs that have distinct uppercase & titlecase)
		3 or more CapitalWords in sequence are lowercased and begin with S startToken and end with E endToken, e.g. THE QUICK BROWN -> Sthe quick brownE
		1 or 2 CapitalWords in sequence are each proceeded by W wordToken, e.g. THE QUICK -> Wthe Wquick
		If 2 or more letters at the end of a word are uppercased, and its followed by 2 or more CapitalWords, insert S startToken just before the 2 or more letters, E endToken after the CapitalWords and lowercase all in between, e.g. THE QUICK BROWN -> Sthe quick brownE
		If 1 or more letters at the end of a word are uppercased, the uppercased letters are lowercased and proceeded by W wordTOken, e.g. teST -> teWst, tesT -> tesWt
		Any other uppercase characters within a word are lowercased and proceeded by the C characterToken, e.g. Test -> Ctest, tESt -> tCeCst

	Notes:
		Titlecase glyphs are always proceeded by T titleToken, and are otherwise unrelated to the rules for the uppercase
		C characterToken never occurs before the last character in a word, in that case W wordToken is used
		E EndToken never occurs in the middle of a word, while s StartToken may occur in the middle of a word

*/

import (
	"unicode"
	"unicode/utf8"
	"io"
	"os"
	"sync"
)

const (
	characterToken = 'C'
	wordToken      = 'W'
	titleToken     = 'T'
	startToken     = 'S'
	endToken       = 'E'
	apostrophe	   = '\''
	apostrophe2    = '’'
	bufferLen      = 20480
	glyphMaxLen	   = 4
	bufferReserve  = 32
)

var pool = sync.Pool{
    New: func() interface{} {
        return make([]byte, bufferLen)
    },
}

type Encoder struct {
	buf     []byte
	pos, capStartPos, secondCapStartPos, capEndPos, nWords, lastWordCapEndPos int
	inCaps, singleLetter, inWord bool
}

type Writer struct {
	w	io.Writer
	e	Encoder
	cursor int
	closed bool
}

func EncodeFile(from string, to string) error {
	// Open the file for reading
	r, err := os.Open(from)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer w.Close()

	buf := pool.Get().([]byte)
	defer pool.Put(buf)

	encoder := NewWriter(w)
	defer encoder.Close()
	var i int
	var err2 error
	for err != io.EOF {
		i, err = r.Read(buf)
		_, err2 = encoder.Write(buf[:i])
		if err2 != nil {
			return err2
		}
	}
	if err == io.EOF {
		return nil
	}
	return err
}

func DecodeFile(from string, to string) error {
	// Open the file for reading
	r, err := os.Open(from)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer w.Close()

	buf := pool.Get().([]byte)
	defer pool.Put(buf)

	decoder := NewReader(r)
	var i int
	var err2 error
	for err != io.EOF {
		i, err = decoder.Read(buf)
		_, err2 = w.Write(buf[:i])
		if err2 != nil {
			return err2
		}
	}
	if err == io.EOF {
		return nil
	}
	return err
}

func NewWriter(f io.Writer) *Writer {
	return &Writer{w: f, e: Encoder{buf: pool.Get().([]byte)}}
}

// Close will force a flush even if its inside a sequence of capitals, it will still be valid but the sequence will begin with startToken instead of another
func (w *Writer) Close() (err error) {
	if w.closed {
		return nil
	}
	w.closed = true
	w.e.end()
	if w.e.pos - w.cursor > 0 {
		_, err = w.w.Write(w.e.buf[w.cursor:w.e.pos])
		w.cursor = w.e.pos
	}
	if len(w.e.buf) == bufferLen {
		pool.Put(w.e.buf)
		w.e.buf = nil
	}
	return
}

// Will only do a soft flush up to a safe point if thinks it might be inside a sequence of capitals, this is because sometimes it needs to go back to change a value
func (w *Writer) Flush() (err error) {
	if w.e.capStartPos - w.cursor > 0 {
		_, err = w.w.Write(w.e.buf[w.cursor:w.e.capStartPos])
		w.cursor = w.e.capStartPos
	}
	return
}

func (w *Writer) Write(data []byte) (int, error) {
	var i, at, pos int
	var eof bool
	for {
		if i, eof = w.e.encode(data[at:]); eof {
			return at, nil // the data is now in the buffer
		} else {
			pos = w.e.capStartPos
			if i == 0 { // if nothing was written, do a hard flush instead of a soft flush
				w.e.end()
				pos = w.e.pos
			}
			_, err := w.w.Write(w.e.buf[w.cursor:pos])
			w.cursor = 0
			copy(w.e.buf, w.e.buf[pos:w.e.pos])
			w.e.pos -= pos
			w.e.capStartPos = 0
			w.e.secondCapStartPos -= pos
			w.e.lastWordCapEndPos -= pos
			w.e.capEndPos -= pos
			at += i
			if err != nil {
				return at, err
			}
		}
	}
	return at, nil
}

func Encode(data []byte) []byte {
	l := len(data) + (len(data)/4) + bufferReserve
	e := Encoder{buf: make([]byte, l)}
	var i, at int
	var eof bool
	for {
		if i, eof = e.encode(data[at:]); eof {
			e.end()
			return e.buf[0:e.pos]
		} else { // if there's not enough space in the buffer, grow it
			newbuf := make([]byte, len(e.buf) + (len(e.buf)/2))
			copy(newbuf, e.buf[0:e.pos])
			e.buf = newbuf
			at += i
		}
	}
}

func (e *Encoder) end() { // this may use 1 character but there is always 1 character reserved so it doesn't check
	if e.inCaps {
		e.inWord = false
		//e.capEndPos = e.pos // it ends at the end
		var r2 rune
		var n2 int
		if e.singleLetter && e.inWord { // only 1 letter is capitalized
			e.buf[e.capStartPos] = characterToken
		} else { // >1 capitals in the run
			switch e.nWords {
				case 0:
					if !e.inWord { // it's a single capital word, followed by space and then lowercase letter
						e.buf[e.capStartPos] = wordToken
					} else { // it's 2 or more capital letter immediately, followed by a lowercase, e.g. TEst
						// go back and put C in front of all of the letters
						e.buf[e.capStartPos] = characterToken
						r2, n2 = utf8.DecodeRune(e.buf[e.capStartPos:])
						for i2:=e.capStartPos+n2+1; i2<e.capEndPos; i2+=n2 {
							r2, n2 = utf8.DecodeRune(e.buf[i2:])
							if unicode.IsLetter(r2) {
								copy(e.buf[i2+1:e.pos+1], e.buf[i2:e.pos])
								e.buf[i2] = characterToken
								e.pos++
								e.capEndPos++
								i2++
								if e.pos >= len(e.buf) {
									// no choice but to grow the buffer because we need to lookback
									newbuf := make([]byte, len(e.buf) + (len(e.buf) / 4))
									copy(newbuf, e.buf)
									e.buf = newbuf
								}
							}
						}
					}
				case 1: // the first word is all in caps
					e.buf[e.capStartPos] = wordToken // replace the startToken with wordToken on the first word
					if !e.inWord { // There are two capital words in a row, then space and then lowercase letters
						copy(e.buf[e.secondCapStartPos+1:e.pos+1], e.buf[e.secondCapStartPos:e.pos]) // make room for the wordToken in front of the second word
						e.buf[e.secondCapStartPos] = wordToken // inject the wordToken in front of the second word
						e.pos++
					} else { // There's one word all in caps, and then another word beginning with caps, but not all caps
						// The second word should have all uppercase letter marked with characterToken
						for i2:=e.secondCapStartPos; i2<e.capEndPos; i2+=n2 {
							r2, n2 = utf8.DecodeRune(e.buf[i2:])
							if unicode.IsLetter(r2) {
								copy(e.buf[i2+1:e.pos+1], e.buf[i2:e.pos])
								e.buf[i2] = characterToken
								e.pos++
								e.capEndPos++
								i2++
								if e.pos >= len(e.buf) {
									// no choice but to grow the buffer because we need to lookback
									newbuf := make([]byte, len(e.buf) + (len(e.buf) / 4))
									copy(newbuf, e.buf)
									e.buf = newbuf
								}
							}
						}
					}
				case 2:
					if !e.inWord { // 3 words in a row, all capitals
						copy(e.buf[e.capEndPos+1:e.pos+1], e.buf[e.capEndPos:e.pos]) // make room for the endToken after the last seen capital letter
						e.buf[e.capEndPos] = endToken // inject the endToken
						e.pos++
					 } else { // 2 capital words in a row, then a word beginning with capitals but not all capitals
						e.buf[e.capStartPos] = wordToken // replace the startToken with wordToken on the first word
						copy(e.buf[e.secondCapStartPos+1:e.pos+1], e.buf[e.secondCapStartPos:e.pos]) // make room for the wordToken in front of the second word
						e.buf[e.secondCapStartPos] = wordToken // inject the wordToken in front of the second word
						e.pos++
						e.capEndPos++
						for i2:=e.lastWordCapEndPos+1; i2<e.capEndPos; i2+=n2 {
							r2, n2 = utf8.DecodeRune(e.buf[i2:])
							if unicode.IsLetter(r2) {
								copy(e.buf[i2+1:e.pos+1], e.buf[i2:e.pos])
								e.buf[i2] = characterToken
								e.pos++
								e.capEndPos++
								i2++
								if e.pos >= len(e.buf) {
									// no choice but to grow the buffer because we need to lookback
									newbuf := make([]byte, len(e.buf) + (len(e.buf) / 4))
									copy(newbuf, e.buf)
									e.buf = newbuf
								}
							}
						}
					}
				default: // there are 3 or more words all in caps
					if !e.inWord {
						copy(e.buf[e.capEndPos+1:e.pos+1], e.buf[e.capEndPos:e.pos]) // make room for the endToken after the last seen capital letter
						e.buf[e.capEndPos] = endToken // inject the endToken
						e.pos++
					} else { // the last word begins with capitals but contains non-capitals
						copy(e.buf[e.lastWordCapEndPos+1:e.pos+1], e.buf[e.lastWordCapEndPos:e.pos]) // make room for the endToken after the last seen capital letter in the previous word
						e.buf[e.lastWordCapEndPos] = endToken // inject the endToken
						e.pos++
						e.capEndPos++
						// Put a characterToken in front of every capital from then until now
						for i2:=e.lastWordCapEndPos+1; i2<e.capEndPos; i2+=n2 {
							r2, n2 = utf8.DecodeRune(e.buf[i2:])
							if unicode.IsLetter(r2) {
								copy(e.buf[i2+1:e.pos+1], e.buf[i2:e.pos])
								e.buf[i2] = characterToken
								e.pos++
								e.capEndPos++
								i2++
								if e.pos >= len(e.buf) {
									// no choice but to grow the buffer because we need to lookback
									newbuf := make([]byte, len(e.buf) + (len(e.buf) / 4))
									copy(newbuf, e.buf)
									e.buf = newbuf
								}
							}
						}
					}
			}

			e.inCaps = false
		}
	}
}

func (e *Encoder) encode(data []byte) (int, bool) {
	var r, r2 rune
	var i, i2, n, n2 int
	// These are copied to move them onto the stack, which may or may not have happened without doing this depending on the optimizer
	var pos int = e.pos
	var capStartPos int = e.capStartPos // this is both the beginning of the current capital streak and also the position safe to flush to
	var capEndPos int = e.capEndPos
	var secondCapStartPos int = e.secondCapStartPos
	var lastWordCapEndPos int = e.lastWordCapEndPos
	var nWords int = e.nWords
	var inCaps bool = e.inCaps
	var singleLetter bool = e.singleLetter
	var inWord bool = e.inWord
	var buf []byte = e.buf
	var dangerZone int = len(buf) - bufferReserve // reserve buffer space for modifications

	for i=0; i < len(data); i += n {
		r, n = utf8.DecodeRune(data[i:]) // get the next rune

		// Check there is enough space in the buffer
		if pos + n >= dangerZone {
			e.pos = pos
			e.capStartPos = capStartPos
			e.secondCapStartPos = secondCapStartPos
			e.lastWordCapEndPos = lastWordCapEndPos
			e.capEndPos = capEndPos
			e.nWords = nWords
			e.inCaps = inCaps
			e.singleLetter = singleLetter
			e.inWord = inWord
			e.buf = buf
			return i, false
		}

		if inCaps {
			if unicode.IsLetter(r) {
				if unicode.IsUpper(r) {
					if !inWord { // this is the first letter of a new word in a sequence of capitals
						inWord = true
						if nWords == 0 { // this is the first letter of the 2nd word
							secondCapStartPos = pos
						}
						lastWordCapEndPos = capEndPos // the last seen capital letter from the previous word
						nWords++
					}
					pos += utf8.EncodeRune(buf[pos:], unicode.ToLower(r))
					capEndPos = pos
					singleLetter = false
				} else { // a non-capital letter in a run of capitals
					// Close capitals run
					if singleLetter && inWord { // only 1 letter is capitalized
						buf[capStartPos] = characterToken
					} else { // >1 capitals in the run
						switch nWords {
							case 0:
								if !inWord { // it's a single capital word, followed by space and then lowercase letter
									buf[capStartPos] = wordToken
								} else { // it's 2 or more capital letters immediately, followed by a lowercase, e.g. TEst
									// go back and put C in front of all of the letters
									buf[capStartPos] = characterToken
									for i2=capStartPos+n2+1; i2<capEndPos; i2+=n2 {
										r2, n2 = utf8.DecodeRune(buf[i2:])
										if unicode.IsLetter(r2) {
											copy(buf[i2+1:pos+1], buf[i2:pos])
											buf[i2] = characterToken
											pos++
											capEndPos++
											i2++
											if pos >= len(buf) {
												// no choice but to grow the buffer because we need to lookback
												newbuf := make([]byte, len(buf) * 2)
												copy(newbuf, buf)
												buf = newbuf
												e.buf = newbuf
												dangerZone = len(buf) - bufferReserve
											}
										}
									}
								}
							case 1: // the first word is all in caps
								buf[capStartPos] = wordToken // replace the startToken with wordToken on the first word
								if !inWord { // There are two capital words in a row, then space and then lowercase letters
									copy(buf[secondCapStartPos+1:pos+1], buf[secondCapStartPos:pos]) // make room for the wordToken in front of the second word
									buf[secondCapStartPos] = wordToken // inject the wordToken in front of the second word
									pos++
								} else { // There's one word all in caps, and then another word beginning with caps, but not all caps
									// The second word should have all uppercase letters marked with characterToken
									for i2=secondCapStartPos; i2<capEndPos; i2+=n2 {
										r2, n2 = utf8.DecodeRune(buf[i2:])
										if unicode.IsLetter(r2) {
											copy(buf[i2+1:pos+1], buf[i2:pos])
											buf[i2] = characterToken
											pos++
											capEndPos++
											i2++
											if pos >= len(buf) {
												// no choice but to grow the buffer because we need to lookback
												newbuf := make([]byte, len(buf) * 2)
												copy(newbuf, buf)
												buf = newbuf
												e.buf = newbuf
												dangerZone = len(buf) - bufferReserve
											}
										}
									}
								}
							case 2:
								if !inWord { // 3 words in a row, all capitals
									copy(buf[capEndPos+1:pos+1], buf[capEndPos:pos]) // make room for the endToken after the last seen capital letter
									buf[capEndPos] = endToken // inject the endToken
									pos++
								 } else { // 2 capital words in a row, then a word beginning with capitals but not all capitals
									buf[capStartPos] = wordToken // replace the startToken with wordToken on the first word
									copy(buf[secondCapStartPos+1:pos+1], buf[secondCapStartPos:pos]) // make room for the wordToken in front of the second word
									buf[secondCapStartPos] = wordToken // inject the wordToken in front of the second word
									pos++
									capEndPos++
									for i2=lastWordCapEndPos+1; i2<capEndPos; i2+=n2 {
										r2, n2 = utf8.DecodeRune(buf[i2:])
										if unicode.IsLetter(r2) {
											copy(buf[i2+1:pos+1], buf[i2:pos])
											buf[i2] = characterToken
											pos++
											capEndPos++
											i2++
											if pos >= len(buf) {
												// no choice but to grow the buffer because we need to lookback
												newbuf := make([]byte, len(buf) * 2)
												copy(newbuf, buf)
												buf = newbuf
												e.buf = newbuf
												dangerZone = len(buf) - bufferReserve
											}
										}
									}
								}
							default: // there are at least 3 words all in caps
								if !inWord {
									copy(buf[capEndPos+1:pos+1], buf[capEndPos:pos]) // make room for the endToken after the last seen capital letter
									buf[capEndPos] = endToken // inject the endToken
									pos++
								} else { // the last word begins with capitals but contains non-capitals
									copy(buf[lastWordCapEndPos+1:pos+1], buf[lastWordCapEndPos:pos]) // make room for the endToken after the last seen capital letter in the previous word
									buf[lastWordCapEndPos] = endToken // inject the endToken
									pos++
									capEndPos++
									// Put a characterToken in front of every capital from then until now
									for i2=lastWordCapEndPos+1; i2<capEndPos; i2+=n2 {
										r2, n2 = utf8.DecodeRune(buf[i2:])
										if unicode.IsLetter(r2) {
											copy(buf[i2+1:pos+1], buf[i2:pos])
											buf[i2] = characterToken
											pos++
											capEndPos++
											i2++
											if pos >= len(buf) {
												// no choice but to grow the buffer because we need to lookback
												newbuf := make([]byte, len(buf) * 2)
												copy(newbuf, buf)
												buf = newbuf
												e.buf = newbuf
												dangerZone = len(buf) - bufferReserve
											}
										}
									}
								}
						}
					}
					pos += utf8.EncodeRune(buf[pos:], r) // write the current lowercase letter
					inCaps = false
					capStartPos = pos // the current safe flush position
				}
			} else { // its not a letter
				if r != apostrophe && r != apostrophe2 && !unicode.IsNumber(r) { // words may contain apostrophe or numbers
					inWord = false
				}
				pos += utf8.EncodeRune(buf[pos:], r) // write the non-letter as it is
			}
		} else {
			if unicode.IsUpper(r) { // Begin run of capitals
				capStartPos = pos
				buf[capStartPos] = startToken // this is necessary in case the buffer ends whilst still inCaps
				pos += utf8.EncodeRune(buf[pos+1:], unicode.ToLower(r)) + 1
				capEndPos = pos
				n2 = n
				singleLetter = true
				inCaps = true
				inWord = true
				nWords = 0
			} else {
				if unicode.IsTitle(r) {
					buf[pos] = titleToken
					pos += utf8.EncodeRune(buf[pos+1:], unicode.ToLower(r)) + 1
				} else {
					pos += utf8.EncodeRune(buf[pos:], r)
				}
				capStartPos = pos // the current safe flush position
			}
		}
	}
	e.pos = pos
	e.capStartPos = capStartPos
	e.secondCapStartPos = secondCapStartPos
	e.lastWordCapEndPos = lastWordCapEndPos
	e.capEndPos = capEndPos
	e.nWords = nWords
	e.inCaps = inCaps
	e.singleLetter = singleLetter
	e.inWord = inWord
	e.buf = buf
	return i, true // all of data was written
}

// Checks the last byte is not a token
func isToken(chr byte) bool {
	switch chr {
		case characterToken:
		case wordToken:
		case titleToken:
		case startToken:
		case endToken:
			return true
		default:
			return false
	}
}

// Decodes the bytes into the same slice
func Decode(data []byte) []byte {
	return DecodeFrom(data, data)
}

func DecodeFrom(destination []byte, source []byte) []byte {
	var i, n, pos, l int
	var r rune
	var inCaps bool

	// If the last character is a token, ignore it
	switch source[len(source) - 1] {
		case characterToken:
		case wordToken:
		case titleToken:
		case startToken:
		case endToken:
			l = len(source) - 1
		default:
			l = len(source)
	}

	for ; i < l; i += n {
		r, n = utf8.DecodeRune(source[i:]) // get the next rune
		switch r {
			case 'T':
				i++
				r, n = utf8.DecodeRune(source[i:])
				pos += utf8.EncodeRune(destination[pos:], unicode.ToTitle(r))
			case 'C':
				i++
				r, n = utf8.DecodeRune(source[i:])
				pos += utf8.EncodeRune(destination[pos:], unicode.ToUpper(r))
			case 'W':
				for i+=n; i<l; i+=n {
					r, n = utf8.DecodeRune(source[i:])
					if unicode.IsLetter(r) {
						pos += utf8.EncodeRune(destination[pos:], unicode.ToUpper(r))
						break
					} else {
						pos += utf8.EncodeRune(destination[pos:], r)
						if !(unicode.IsNumber(r) || r == apostrophe || r == apostrophe2) {
							break
						}
					}
				}
			case 'B':
				inCaps = true
			case 'E':
				inCaps = false
			default:
				if !inCaps { // prefer this branch
					pos += utf8.EncodeRune(destination[pos:], r)
				} else {
					pos += utf8.EncodeRune(destination[pos:], unicode.ToUpper(r))
				}
		}
	}

	return destination[:pos]
}

type Reader struct {
	r io.Reader
	inCaps bool
}

func NewReader(f io.Reader) *Reader {
	return &Reader{r: f}
}

// Populate slice of bytes
func (d *Reader) Read(data []byte) (int, error) {

	var i, n, l, pos, dangerZone int
	var r rune
	var err error
	inCaps := d.inCaps

	for {
		n, err = d.r.Read(data[l:])  // Because the decoded cannot be longer than the encoded, I'm using the output slice as the buffer
		l += n
		if i == l {
			d.inCaps = inCaps
			return pos, err
		}
		
		if err == io.EOF {
			dangerZone = l
		} else {
			dangerZone = l-glyphMaxLen-1  // must have enough for the entire next rune, otherwise we might cut it in half
		}

		for ; i < dangerZone; i += n {
			r, n = utf8.DecodeRune(data[i:]) // get the next rune
			switch r {
				case 'T':
					i++
					r, n = utf8.DecodeRune(data[i:])
					pos += utf8.EncodeRune(data[pos:], unicode.ToTitle(r))
				case 'C':
					i++
					r, n = utf8.DecodeRune(data[i:])
					pos += utf8.EncodeRune(data[pos:], unicode.ToUpper(r))
				case 'W':
					for i+=n; i<len(data); i+=n {
						r, n = utf8.DecodeRune(data[i:])
						if unicode.IsLetter(r) {
							pos += utf8.EncodeRune(data[pos:], unicode.ToUpper(r))
							break
						} else {
							pos += utf8.EncodeRune(data[pos:], r)
							if !(unicode.IsNumber(r) || r == apostrophe || r == apostrophe2) {
								break
							}
						}
					}
				case 'B':
					inCaps = true
				case 'E':
					inCaps = false
				default:
					if !inCaps { // prefer this branch
						pos += utf8.EncodeRune(data[pos:], r)
					} else {
						pos += utf8.EncodeRune(data[pos:], unicode.ToUpper(r))
					}
			}
		}
	}
}