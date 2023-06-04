package capcode

/*

	capcode

	For encoding uppercasing into lowercasing.

	- Parsed as UTF-8 glyphs
	- WordSeparator is any glyph that is not a letter, number, modifier or one of two apostrophes '’
	- CapitalWord is a word where every letter is uppercase and it's bounded by a WordSeparator on both sides, or end of text
	
	Decoding:
		The C characterToken makes the following 1 UTF8 glyph uppercase
		The W wordToken makes all characters following this uppercase until a WordSeparator reached
		The B beginToken makes all glyphs uppercase until the next E endToken

	Encoding:
		3 or more CapitalWords in sequence are lowercased and begin with S beginToken and end with E endToken, e.g. THE QUICK BROWN -> Sthe quick brownE
		1 or 2 CapitalWords in sequence are each proceeded by W wordToken, e.g. THE QUICK -> Wthe Wquick
		If 2 or more letters at the end of a word are uppercased, and its followed by 2 or more CapitalWords, insert S beginToken just before the 2 or more letters, E endToken after the CapitalWords and lowercase all in between, e.g. THE QUICK BROWN -> Sthe quick brownE
		If 1 or more letters at the end of a word are uppercased, the uppercased letters are lowercased and proceeded by W wordTOken, e.g. teST -> teWst, tesT -> tesWt
		Any other uppercase characters within a word are lowercased and proceeded by the C characterToken, e.g. Test -> Ctest, tESt -> tCeCst

	Notes:
		Titlecase glyphs (for special glphs that have distinct uppercase & titlecase) are left unchanged
		C characterToken never occurs before the last character in a word, in that case W wordToken is used
		E EndToken never occurs in the middle of a word, while s beginToken may occur in the middle of a word

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
	beginToken     = 'B'
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

func isModifier(r rune) bool {
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r)
}

// Returns the number of bytes at the end of the slice of bytes that are part of an incomplete UTF-8 sequence
func IncompleteUTF8Bytes(bytes []byte) int {
    bytesLen := len(bytes)
    // Single byte or empty string
	if bytesLen == 0 {
		return 0
	}
    if  bytes[bytesLen - 1] & 0b10000000 == 0 {
        return 0
    }
    // Find the start of the last character sequence
    seqStart := bytesLen - 1
    for seqStart >= 0 && (bytes[seqStart] & 0b11000000) == 0b10000000 {
        seqStart--
    }
    // If no sequence start found, all bytes are continuation bytes and thus are all incomplete
    if seqStart == -1 {
        return bytesLen
    }
    // Determine expected sequence length from leading byte
    seqLen := 0
    for (bytes[seqStart] & (0b10000000 >> seqLen)) != 0 {
        seqLen++
    }
    // If sequence length is larger than the remaining bytes, it's incomplete
    if bytesLen - seqStart < seqLen {
        return seqLen - (bytesLen - seqStart)
    }
    return 0
}

func IncompleteUTF16Bytes(bytes []byte) int {
	bytesLen := len(bytes)
	if bytesLen == 0 {
		return 0
	}
	if bytesLen % 2 != 0 {
		var lastThreeBytes uint16
		if bytesLen >= 3 {
			lastThreeBytes = binary.LittleEndian.Uint16(bytes[bytesLen-3 : bytesLen-1])
			if lastThreeBytes >= 0xD800 && lastThreeBytes <= 0xDBFF {
				return 3
			}
		}
		return 1
	}
	// Check if last 16-bit unit is a high surrogate
	lastTwoBytes := binary.LittleEndian.Uint16(bytes[bytesLen-2 : bytesLen])
	if lastTwoBytes >= 0xD800 && lastTwoBytes <= 0xDBFF && bytesLen < 4 {
		return 2 // High surrogate without a following low surrogate
	}
	return 0
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

// Close will force a flush even if its inside a sequence of capitals, it will still be valid but the sequence will begin with beginToken instead of another
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
		switch e.nWords {
			case 0: // it's a single capital word
				e.buf[e.capStartPos] = wordToken
			case 1: // There are two capital words in a row
				e.buf[e.capStartPos] = wordToken // replace the beginToken with wordToken on the first word
				copy(e.buf[e.secondCapStartPos+1:e.pos+1], e.buf[e.secondCapStartPos:e.pos]) // make room for the wordToken in front of the second word
				e.buf[e.secondCapStartPos] = wordToken // inject the wordToken in front of the second word
				e.pos++
			default: // there are 3 or more words all in caps
				copy(e.buf[e.capEndPos+1:e.pos+1], e.buf[e.capEndPos:e.pos]) // make room for the endToken after the last seen capital letter
				e.buf[e.capEndPos] = endToken // inject the endToken
				e.pos++
		}
		e.inCaps = false
		e.inWord = false
	}
}

func (e *Encoder) encode(data []byte) (int, bool) {
	var r, r2 rune
	var i, i2, n, n2 int
	// These are copied to move them onto the stack
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
								buf[capStartPos] = wordToken // replace the beginToken with wordToken on the first word
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
									buf[capStartPos] = wordToken // replace the beginToken with wordToken on the first word
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
				pos += utf8.EncodeRune(buf[pos:], r) // write the non-letter as it is
				if isModifier(r) {
					capEndPos = pos
				} else if r != apostrophe && r != apostrophe2 && !unicode.IsNumber(r) { // words may contain apostrophes, numbers or modifiers
					inWord = false
				}
			}
		} else {
			if unicode.IsUpper(r) { // Begin run of capitals
				capStartPos = pos
				buf[capStartPos] = beginToken // this is necessary in case the buffer ends whilst still inCaps
				pos += utf8.EncodeRune(buf[pos+1:], unicode.ToLower(r)) + 1
				capEndPos = pos
				n2 = n
				singleLetter = true
				inCaps = true
				inWord = true
				nWords = 0
			} else {
				pos += utf8.EncodeRune(buf[pos:], r)
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
		case beginToken:
		case endToken:
			l = len(source) - 1
		default:
			l = len(source)
	}

	for ; i < l; i += n {
		r, n = utf8.DecodeRune(source[i:]) // get the next rune
		switch r {
			case characterToken:
				i++
				r, n = utf8.DecodeRune(source[i:])
				pos += utf8.EncodeRune(destination[pos:], unicode.ToUpper(r))
			case wordToken:
				for i+=n; i<l; i+=n {
					r, n = utf8.DecodeRune(source[i:])
					if unicode.IsLetter(r) {
						pos += utf8.EncodeRune(destination[pos:], unicode.ToUpper(r))
						break
					} else {
						pos += utf8.EncodeRune(destination[pos:], r)
						if !(unicode.IsNumber(r) || r == apostrophe || r == apostrophe2 || isModifier(r)) {
							break
						}
					}
				}
			case beginToken:
				inCaps = true
			case endToken:
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

type Decoder struct {
	inCaps bool
	charUp bool
	wordUp bool
}

// Decodes the bytes into the same slice
func (d *Decoder) Decode(data []byte) []byte {
	return d.DecodeFrom(data, data)
}

func (d *Decoder) DecodeFrom(destination []byte, source []byte) []byte {
	var i, n, pos, l int
	var r rune

	inCaps := d.inCaps
	charUp := d.charUp
	wordUp := d.wordUp

	for ; i < l; i += n {
		r, n = utf8.DecodeRune(source[i:]) // get the next rune
		switch r {
			case characterToken:
				charUp = true
			case wordToken:
				wordUp = true
			case beginToken:
				inCaps = true
			case endToken:
				inCaps = false
			default:
				switch {
					case charUp:
						pos += utf8.EncodeRune(destination[pos:], unicode.ToUpper(r))
						charUp = false
					case wordUp:
						if unicode.IsLetter(r) {
							pos += utf8.EncodeRune(destination[pos:], unicode.ToUpper(r))
						} else {
							pos += utf8.EncodeRune(data[pos:], r)
							if !(unicode.IsNumber(r) || r == apostrophe || r == apostrophe2 || isModifier(r)) {
								wordUp = false
							}
						}
					case inCaps:
						pos += utf8.EncodeRune(destination[pos:], unicode.ToUpper(r))
					default:
						pos += utf8.EncodeRune(destination[pos:], r)
				}
		}
	}

	d.inCaps = inCaps
	d.charUp = charUp
	d.wordUp = wordUp

	return destination[:pos]
}

type Reader struct {
	r io.Reader
	d Decoder
}

func NewReader(f io.Reader) *Reader {
	return &Reader{r: f}
}

// Populate slice of bytes
func (d *Reader) Read(data []byte) (int, error) {
	l := len(data) - glyphMaxLen
	if l <= 0 {
		return 0, errors.New(`Buffer too small`)
	}
	n, err := d.r.Read(data[:l])
	if err == io.EOF {
		newar := d.d.Decode(data[:n])
		return len(newar), err
	}

	i := incompleteUTF8(data[:n])
	if i == 0 {
		newar := d.d.Decode(data[:n])
		return len(newar), err
	} else {
		for {
			n2, err = d.r.Read(data[n:n+1])
			n += n2
			i = IncompleteUTF8Bytes(data[:n])
			if n == len(data) || err == io.EOF || i == 0 || n2 == 0 {
				newar := d.d.Decode(data[:n])
				return len(newar), err
			}
		}
	}
}
