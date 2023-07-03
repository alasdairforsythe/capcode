package capcode

import (
	"unicode"
	"unicode/utf8"
)

const (
	CharacterToken = 'C'
	WordToken      = 'W'
	DeleteToken	   = 'D'
	NoCapcodeDeleteToken = '\x7F'
	NoCapcodeSubstitute = '\x14'
	Apostrophe	   = '\''
	Apostrophe2    = '’'
	RuneError      = '\uFFFD'
	bufferReserve  = 7
)

func isModifier(r rune) bool {
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r)
}

func grow(buf []byte) []byte {
	newbuf := make([]byte, len(buf) + (len(buf) / 3) + bufferReserve)
	copy(newbuf, buf)
	return newbuf
}

func Encode(data []byte) []byte {
	var r, r2, rlast, rlast2 rune
	var i, i2, n, n2, pos, wordTokenPos int
	var inWord, multiLetter bool
	buf := make([]byte, len(data)+(len(data)/2)+bufferReserve)
	dangerZone := len(buf) - bufferReserve

	for i = 0; i < len(data); i += n {
		r, n = utf8.DecodeRune(data[i:]) // get the next rune

		// Check there is enough space in the buffer
		if pos >= dangerZone {
			buf = grow(buf)
			dangerZone = len(buf) - bufferReserve
		}

		if inWord {
			if unicode.IsUpper(r) {
				if !(unicode.IsLetter(rlast) || rlast == Apostrophe || rlast == Apostrophe2 || isModifier(rlast)) {
					buf[pos] = DeleteToken
					buf[pos+1] = ' '
					pos += 2
				}
				multiLetter = true
				pos += utf8.EncodeRune(buf[pos:], unicode.ToLower(r))
			} else {
				if unicode.IsLower(r) {
					inWord = false
					buf[wordTokenPos] = CharacterToken
					if multiLetter {
						// Go back and put CharacterToken in front of them all
						for i2 = n2; i2 < pos; i2 += n2 {
							if buf[i2] == DeleteToken && buf[i2+1] == ' ' {
								// there is a DeleteToken here already, so just add the 1 character token, if it's a letter next
								r2, n2 = utf8.DecodeRune(buf[i2+2:])
								if unicode.IsLower(r2) {
									if pos >= dangerZone {
										buf = grow(buf)
										dangerZone = len(buf) - bufferReserve
									}
									copy(buf[i2+3:pos+1], buf[i2+2:pos])
									buf[i2] = DeleteToken
									buf[i2+1] = CharacterToken
									buf[i2+2] = ' '
									pos++ // we only added 1
									i2++
								}
								i2 += 2
							} else {
								r2, n2 = utf8.DecodeRune(buf[i2:])
								if unicode.IsLower(r2) {
									if pos >= dangerZone {
										buf = grow(buf)
										dangerZone = len(buf) - bufferReserve
									}
									copy(buf[i2+3:pos+3], buf[i2:pos])
									buf[i2] = DeleteToken
									buf[i2+1] = CharacterToken
									buf[i2+2] = ' '
									pos += 3
									i2 += 3
								}
							}
						}
					}
					if !(unicode.IsLetter(rlast) || rlast == Apostrophe || rlast == Apostrophe2 || isModifier(rlast)) {
						buf[pos] = DeleteToken
						buf[pos+1] = ' '
						pos += 2
					}
				} else {
					if unicode.IsNumber(r) {
						if !unicode.IsNumber(rlast) {
							buf[pos] = DeleteToken
							buf[pos+1] = ' '
							pos += 2
						}
					} else if !(r == Apostrophe || r == Apostrophe2 || isModifier(r)) {
						inWord = false
					}
				}
				//pos += utf8.EncodeRune(buf[pos:], r)
				switch n {
				case 1:
					buf[pos] = data[i]
					pos++
				case 2:
					buf[pos] = data[i]
					buf[pos+1] = data[i+1]
					pos += 2
				case 3:
					buf[pos] = data[i]
					buf[pos+1] = data[i+1]
					buf[pos+2] = data[i+2]
					pos += 3
				case 4:
					buf[pos] = data[i]
					buf[pos+1] = data[i+1]
					buf[pos+2] = data[i+2]
					buf[pos+3] = data[i+3]
					pos += 4
				}
			}
		} else {
			if unicode.IsLower(r) {
				if !(rlast == ' ' || unicode.IsLower(rlast) || (unicode.IsLetter(rlast2) && (rlast == Apostrophe || rlast == Apostrophe2)) || isModifier(rlast)) {
					buf[pos] = DeleteToken
					buf[pos+1] = ' '
					pos += 2
				}
				//pos += utf8.EncodeRune(buf[pos:], r)
				switch n {
				case 1:
					buf[pos] = data[i]
					pos++
				case 2:
					buf[pos] = data[i]
					buf[pos+1] = data[i+1]
					pos += 2
				case 3:
					buf[pos] = data[i]
					buf[pos+1] = data[i+1]
					buf[pos+2] = data[i+2]
					pos += 3
				case 4:
					buf[pos] = data[i]
					buf[pos+1] = data[i+1]
					buf[pos+2] = data[i+2]
					buf[pos+3] = data[i+3]
					pos += 4
				}
			} else if unicode.IsUpper(r) { // Begin run of capitals
				if rlast == ' ' {
					wordTokenPos = pos - 1
					buf[pos-1] = WordToken
					buf[pos] = ' '
					pos++
				} else {
					wordTokenPos = pos + 1
					buf[pos] = DeleteToken
					buf[pos+1] = WordToken
					buf[pos+2] = ' '
					pos += 3
				}
				pos += utf8.EncodeRune(buf[pos:], unicode.ToLower(r))
				n2 = pos
				multiLetter = false
				inWord = true
			} else if unicode.IsNumber(r) {
				if !(rlast == ' ' || unicode.IsNumber(rlast)) {
					buf[pos] = DeleteToken
					buf[pos+1] = ' '
					pos += 2
				}
				//pos += utf8.EncodeRune(buf[pos:], r)
				switch n {
				case 1:
					buf[pos] = data[i]
					pos++
				case 2:
					buf[pos] = data[i]
					buf[pos+1] = data[i+1]
					pos += 2
				case 3:
					buf[pos] = data[i]
					buf[pos+1] = data[i+1]
					buf[pos+2] = data[i+2]
					pos += 3
				case 4:
					buf[pos] = data[i]
					buf[pos+1] = data[i+1]
					buf[pos+2] = data[i+2]
					buf[pos+3] = data[i+3]
					pos += 4
				}
			} else {
				//pos += utf8.EncodeRune(buf[pos:], r)
				switch n {
				case 1:
					buf[pos] = data[i]
					pos++
				case 2:
					buf[pos] = data[i]
					buf[pos+1] = data[i+1]
					pos += 2
				case 3:
					buf[pos] = data[i]
					buf[pos+1] = data[i+1]
					buf[pos+2] = data[i+2]
					pos += 3
				case 4:
					buf[pos] = data[i]
					buf[pos+1] = data[i+1]
					buf[pos+2] = data[i+2]
					buf[pos+3] = data[i+3]
					pos += 4
				}
			}
		}

		rlast2 = rlast
		rlast = r
	}
	return buf[0:pos]
}

type Decoder struct {
	inWord bool
	inChar bool
	delete bool
	ignore bool
}

func (d *Decoder) Decode(b []byte) []byte {
	return d.DecodeFrom(b, b)
}

func (d *Decoder) DecodeFrom(dst []byte, src []byte) []byte {
	var r rune
	var i, n, pos int
	for ; i < len(src); i += n {
		r, n = utf8.DecodeRune(src[i:]) // get the next rune
		switch r {
			case CharacterToken:
				d.inChar = true
				d.inWord = false
				continue
			case WordToken:
				d.inWord = true
				d.inChar = false
				d.ignore = true
				continue
			case DeleteToken:
				d.delete = true
				continue
			case ' ':
				if d.delete {
					d.delete = false
				} else {
					dst[pos] = ' '
					pos++
					if !d.ignore {
						d.inWord = false
					}
				}
			default:
				switch {
					case d.delete:
						d.delete = false
					case d.inChar:
						d.inChar = false
						if r == RuneError {
							switch n {
								case 1:
									dst[pos] = src[i]
									pos++
								case 2:
									dst[pos] = src[i]
									dst[pos+1] = src[i+1]
									pos += 2
								case 3:
									dst[pos] = src[i]
									dst[pos+1] = src[i+1]
									dst[pos+2] = src[i+2]
									pos += 3
								case 4:
									dst[pos] = src[i]
									dst[pos+1] = src[i+1]
									dst[pos+2] = src[i+2]
									dst[pos+3] = src[i+3]
									pos += 4
							}
						} else {
							pos += utf8.EncodeRune(dst[pos:], unicode.ToUpper(r))
						}
					case d.inWord:
						if unicode.IsLower(r) || unicode.IsUpper(r) { // Chinese characters are neither upper nor lower, but are letters
							pos += utf8.EncodeRune(dst[pos:], unicode.ToUpper(r))
						} else {
							switch n {
								case 1:
									dst[pos] = src[i]
									pos++
								case 2:
									dst[pos] = src[i]
									dst[pos+1] = src[i+1]
									pos += 2
								case 3:
									dst[pos] = src[i]
									dst[pos+1] = src[i+1]
									dst[pos+2] = src[i+2]
									pos += 3
								case 4:
									dst[pos] = src[i]
									dst[pos+1] = src[i+1]
									dst[pos+2] = src[i+2]
									dst[pos+3] = src[i+3]
									pos += 4
							}
							if !(unicode.IsNumber(r) || r == Apostrophe || r == Apostrophe2 || isModifier(r)) {
								d.inWord = false
							}
						}
					default:
						switch n {
						case 1:
							dst[pos] = src[i]
							pos++
						case 2:
							dst[pos] = src[i]
							dst[pos+1] = src[i+1]
							pos += 2
						case 3:
							dst[pos] = src[i]
							dst[pos+1] = src[i+1]
							dst[pos+2] = src[i+2]
							pos += 3
						case 4:
							dst[pos] = src[i]
							dst[pos+1] = src[i+1]
							dst[pos+2] = src[i+2]
							dst[pos+3] = src[i+3]
							pos += 4
					}
				}
		}
		d.ignore = false
	}
	return dst[0:pos]
}

func Decode(b []byte) []byte {
	var r rune
	var i, n, pos int
	var inChar, inWord, delete, ignore bool
	for ; i < len(b); i += n {
		r, n = utf8.DecodeRune(b[i:]) // get the next rune
		switch r {
			case CharacterToken:
				inChar = true
				inWord = false
				continue
			case WordToken:
				inWord = true
				inChar = false
				ignore = true
				continue
			case DeleteToken:
				delete = true
				continue
			case ' ':
				if delete {
					delete = false
				} else {
					b[pos] = ' '
					pos++
					if !ignore {
						inWord = false
					}
				}
			default:
				switch {
					case delete:
						delete = false
					case inChar:
						inChar = false
						if r == RuneError {
							switch n {
								case 1:
									b[pos] = b[i]
									pos++
								case 2:
									b[pos] = b[i]
									b[pos+1] = b[i+1]
									pos += 2
								case 3:
									b[pos] = b[i]
									b[pos+1] = b[i+1]
									b[pos+2] = b[i+2]
									pos += 3
								case 4:
									b[pos] = b[i]
									b[pos+1] = b[i+1]
									b[pos+2] = b[i+2]
									b[pos+3] = b[i+3]
									pos += 4
							}
						} else {
							pos += utf8.EncodeRune(b[pos:], unicode.ToUpper(r))
						}
					case inWord:
						if unicode.IsLower(r) || unicode.IsUpper(r) { // Chinese characters are neither upper nor lower, but are letters
							pos += utf8.EncodeRune(b[pos:], unicode.ToUpper(r))
						} else {
							switch n {
								case 1:
									b[pos] = b[i]
									pos++
								case 2:
									b[pos] = b[i]
									b[pos+1] = b[i+1]
									pos += 2
								case 3:
									b[pos] = b[i]
									b[pos+1] = b[i+1]
									b[pos+2] = b[i+2]
									pos += 3
								case 4:
									b[pos] = b[i]
									b[pos+1] = b[i+1]
									b[pos+2] = b[i+2]
									b[pos+3] = b[i+3]
									pos += 4
							}
							if !(unicode.IsNumber(r) || r == Apostrophe || r == Apostrophe2 || isModifier(r)) {
								inWord = false
							}
						}
					default:
						switch n {
						case 1:
							b[pos] = b[i]
							pos++
						case 2:
							b[pos] = b[i]
							b[pos+1] = b[i+1]
							pos += 2
						case 3:
							b[pos] = b[i]
							b[pos+1] = b[i+1]
							b[pos+2] = b[i+2]
							pos += 3
						case 4:
							b[pos] = b[i]
							b[pos+1] = b[i+1]
							b[pos+2] = b[i+2]
							b[pos+3] = b[i+3]
							pos += 4
					}
				}
		}
		ignore = false
	}
	return b[0:pos]
}

// The NoCapcode versions do only the D DeleteToken

func NoCapcodeEncode(data []byte) []byte {
	var r, rlast, rlast2 rune
	var i, n, pos int
	buf := make([]byte, len(data)+(len(data)/2)+bufferReserve)
	dangerZone := len(buf) - bufferReserve

	for i = 0; i < len(data); i += n {
		r, n = utf8.DecodeRune(data[i:]) // get the next rune

		// Check there is enough space in the buffer
		if pos >= dangerZone {
			buf = grow(buf)
			dangerZone = len(buf) - bufferReserve
		}

		if unicode.IsLetter(r) {
			if !(rlast == ' ' || unicode.IsLetter(rlast) || (unicode.IsLetter(rlast2) && (rlast == Apostrophe || rlast == Apostrophe2)) || isModifier(rlast)) {
				buf[pos] = NoCapcodeDeleteToken
				buf[pos+1] = ' '
				pos += 2
			}
		} else if unicode.IsNumber(r) {
			if !(rlast == ' ' || unicode.IsNumber(rlast)) {
				buf[pos] = NoCapcodeDeleteToken
				buf[pos+1] = ' '
				pos += 2
			}
		}
		if r == NoCapcodeDeleteToken {
			// in this case the token we're using for delete was already in the text
			// safest thing to do is replace it with something else
			// otherwise mayhem with ensue
			// let's replace it with ASCII 20 (DEVICE CONTROL 4)
			buf[pos] = NoCapcodeSubstitute
			pos++
		} else {
			switch n {
			case 1:
				buf[pos] = data[i]
				pos++
			case 2:
				buf[pos] = data[i]
				buf[pos+1] = data[i+1]
				pos += 2
			case 3:
				buf[pos] = data[i]
				buf[pos+1] = data[i+1]
				buf[pos+2] = data[i+2]
				pos += 3
			case 4:
				buf[pos] = data[i]
				buf[pos+1] = data[i+1]
				buf[pos+2] = data[i+2]
				buf[pos+3] = data[i+3]
				pos += 4
			}
		}

		rlast2 = rlast
		rlast = r
	}
	return buf[0:pos]
}

func NoCapcodeDecode(b []byte) []byte {
	var i, pos int
	for ; i < len(b); i++ {
		if b[i] == NoCapcodeDeleteToken {
			i++
		} else {
			b[pos] = b[i]
			pos++
		}
	}
	return b[0:pos]
}

func NoCapcodeDecodeFrom(dst []byte, src []byte) []byte {
	var i, pos int
	for ; i < len(src); i++ {
		if src[i] == NoCapcodeDeleteToken {
			i++
		} else {
			dst[pos] = src[i]
			pos++
		}
	}
	return dst[0:pos]
}

func (d *Decoder) NoCapcodeDecode(b []byte) []byte {
	var i, pos int
	for ; i < len(b); i++ {
		if b[i] == NoCapcodeDeleteToken {
			d.delete = true
		} else {
			if d.delete {
				d.delete = false
			} else {
				b[pos] = b[i]
				pos++
			}
		}
	}
	return b[0:pos]
}

func (d *Decoder) NoCapcodeDecodeFrom(dst []byte, src []byte) []byte {
	var i, pos int
	for ; i < len(src); i++ {
		if src[i] == NoCapcodeDeleteToken {
			d.delete = true
		} else {
			if d.delete {
				d.delete = false
			} else {
				dst[pos] = src[i]
				pos++
			}
		}
	}
	return dst[0:pos]
}
