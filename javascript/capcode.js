const characterToken = 'C';
const wordToken = 'W';
const titleToken = 'T';
const startToken = 'S';
const endToken = 'E';
const apostrophe = '\'';
const apostrophe2 = '’';

function isUpper(r) {
  return /\p{Lu}/u.test(r);
}

function isLower(r) {
  return /\p{Ll}/u.test(r);
}

function isLetter(r) {
  return /\p{L}/u.test(r);
}

function isNumber(r) {
  return /\p{Nd}/u.test(r);
}

function isTitle(r) {
  return /\p{Lt}/u.test(r);
}

function toTitleCase(glyph) {
  const titlecaseGlyph = String.fromCodePoint(glyph.codePointAt(0) - 1);
  if (isTitle(titlecaseGlyph)) {
    return titlecaseGlyph;
  }
  return glyph.toUpperCase();
}

function encode(data) {
  let buf = new Array(Math.ceil(data.length + (data.length / 4) + 8));
  let pos = 0;
  let capStartPos = 0;
  let capEndPos = 0;
  let secondCapStartPos = 0;
  let lastWordCapEndPos = 0;
  let nWords = 0;
  let inCaps = false;
  let singleLetter = false;
  let inWord = false;
  let i = 0;

  for (let r of data) {

    if (inCaps) {
      if (isLetter(r)) {
        if (isUpper(r)) {
          if (!inWord) {
            inWord = true;
            if (nWords === 0) {
              secondCapStartPos = pos;
            }
            lastWordCapEndPos = capEndPos;
            nWords++;
          }
          buf[pos++] = r.toLowerCase();
          capEndPos = pos;
          singleLetter = false;
        } else {
          if (singleLetter && inWord) {
            buf[capStartPos] = characterToken;
          } else {
            switch (nWords) {
              case 0:
                if (!inWord) {
                  buf[capStartPos] = wordToken;
                } else {
                  buf[capStartPos] = characterToken;
                  for (let i2 = capStartPos + 1; i2 < capEndPos; i2++) {
                    let r2 = buf[i2];
                    if (isLetter(r2)) {
                        for (let j = pos; j > i2; j--) {
                            buf[j] = buf[j - 1];
                        }
                        buf[i2] = characterToken;
                        pos++;
                        capEndPos++;
                        i2++;
                    }
                  }
                }
                break;
              case 1:
                buf[capStartPos] = wordToken;
                if (!inWord) {
                  buf.splice(secondCapStartPos, 0, wordToken);
                  pos++;
                } else {
                  for (let i2 = secondCapStartPos; i2 < capEndPos; i2++) {
                    let r2 = buf[i2];
                    if (isLetter(r2)) {
                        for (let j = pos; j > i2; j--) {
                            buf[j] = buf[j - 1];
                        }
                        buf[i2] = characterToken;
                        pos++;
                        capEndPos++;
                        i2++;
                    }
                  }
                }
                break;
              case 2:
                if (!inWord) {
                  buf.splice(capEndPos, 0, endToken);
                  pos++;
                } else {
                  buf[capStartPos] = wordToken;
                  buf.splice(secondCapStartPos, 0, wordToken);
                  pos++;
                  capEndPos++;
                  for (let i2 = lastWordCapEndPos + 1; i2 < capEndPos; i2++) {
                    let r2 = buf[i2];
                    if (isLetter(r2)) {
                        for (let j = pos; j > i2; j--) {
                            buf[j] = buf[j - 1];
                        }
                        buf[i2] = characterToken;
                        pos++;
                        capEndPos++;
                        i2++;
                    }
                  }
                }
                break;
              default:
                if (!inWord) {
                  buf.splice(capEndPos, 0, endToken);
                  pos++;
                } else {
                  buf.splice(lastWordCapEndPos, 0, endToken);
                  pos++;
                  capEndPos++;
                  for (let i2 = lastWordCapEndPos + 1; i2 < capEndPos; i2++) {
                    let r2 = buf[i2];
                    if (isLetter(r2)) {
                        for (let j = pos; j > i2; j--) {
                            buf[j] = buf[j - 1];
                        }
                        buf[i2] = characterToken;
                        pos++;
                        capEndPos++;
                        i2++;
                    }
                  }
                }
            }
          }
          buf[pos++] = r;
          inCaps = false;
          capStartPos = pos;
        }
      } else {
        if (r !== apostrophe && r !== apostrophe2 && !isNumber(r)) {
          inWord = false;
        }
        buf[pos++] = r;
      }
    } else {
      if (isUpper(r)) {
        capStartPos = pos;
        buf[pos++] = startToken;
        buf[pos++] = r.toLowerCase();
        capEndPos = pos;
        nWords = 0;
        inCaps = true;
        inWord = true;
        singleLetter = true;
      } else {
        if (isTitle(r)) {
          buf[pos++] = titleToken;
          buf[pos++] = r.toLowerCase();
        } else {
          buf[pos++] = r;
        }
        capStartPos = pos;
      }
    }

    i++
  }

  if (inCaps) {
    inWord = false
    //capEndPos = pos
    if (singleLetter && inWord) {
      buf[capStartPos] = characterToken;
    } else {
      switch (nWords) {
        case 0:
          if (!inWord) {
            buf[capStartPos] = wordToken;
          } else {
            buf[capStartPos] = characterToken;
            for (let i2 = capStartPos + 1; i2 < capEndPos; i2++) {
              let r2 = buf[i2];
              if (isLetter(r2)) {
                  for (let j = pos; j > i2; j--) {
                      buf[j] = buf[j - 1];
                  }
                  buf[i2] = characterToken;
                  pos++;
                  capEndPos++;
                  i2++;
              }
            }
          }
          break;
        case 1:
          buf[capStartPos] = wordToken;
          if (!inWord) {
            buf.splice(secondCapStartPos, 0, wordToken);
            pos++;
          } else {
            for (let i2 = secondCapStartPos + 1; i2 < capEndPos; i2++) {
              let r2 = buf[i2];
              if (isLetter(r2)) {
                  for (let j = pos; j > i2; j--) {
                      buf[j] = buf[j - 1];
                  }
                  buf[i2] = characterToken;
                  pos++;
                  capEndPos++;
                  i2++;
              }
            }
          }
          break;
        case 2:
          if (!inWord) {
            buf.splice(capEndPos, 0, endToken);
            pos++;
          } else {
            buf[capStartPos] = wordToken;
            buf.splice(secondCapStartPos, 0, wordToken);
            pos++;
            capEndPos++;
            for (let i2 = lastWordCapEndPos + 1; i2 < capEndPos; i2++) {
              let r2 = buf[i2];
              if (isLetter(r2)) {
                  for (let j = pos; j > i2; j--) {
                      buf[j] = buf[j - 1];
                  }
                  buf[i2] = characterToken;
                  pos++;
                  capEndPos++;
                  i2++;
              }
            }
          }
          break;
        default:
          if (!inWord) {
            buf.splice(capEndPos, 0, endToken);
            pos++;
          } else {
            buf.splice(lastWordCapEndPos, 0, endToken);
            pos++;
            capEndPos++;
            for (let i2 = lastWordCapEndPos + 1; i2 < capEndPos; i2++) {
              let r2 = buf[i2];
              if (isLetter(r2)) {
                  for (let j = pos; j > i2; j--) {
                      buf[j] = buf[j - 1];
                  }
                  buf[i2] = characterToken;
                  pos++;
                  capEndPos++;
                  i2++;
              }
            }
          }
      }
    }

  }

  return buf.slice(0, pos).join('');
}

function decode(data) {
    let destination = "";  
    let inCaps = false;
    let charUp = false;
    let wordUp = false;
    let titleUp = false;
    for (let r of data) {
        switch (r) {
            case titleToken: // there's no easy way to do this in Javascript so its just being uppercased for now
            titleUp = true;
            break;
            case characterToken:
            charUp = true;
            break;
            case wordToken:
            wordUp = true;
            break;
            case startToken:
            inCaps = true;
            break;
            case endToken:
            inCaps = false;
            break;
            default:
                if (charUp) {
                    destination += r.toUpperCase();
                    charUp = false;
                  } else if (wordUp) {
                    if (isLetter(r)) {
                        destination += r.toUpperCase();
                    } else {
                        if (!(isNumber(r) || r == apostrophe || r == apostrophe2)) {
                            wordUp = false
                        }
                        destination += r;
                    }
                  } else if (inCaps) {
                    destination += r.toUpperCase();
                  } else if (titleUp) {
                    destination += toTitleCase(r);
                  } else {
                    destination += r;
                  }
      }
    }
    return destination;
  }

  class Decoder {
    constructor() {
      this.inCaps = false;
      this.charUp = false;
      this.wordUp = false;
      this.titleUp = false;
    }
  
    decode(data) {
      let destination = "";
      for (let r of data) {
        switch (r) {
          case titleToken: // there's no easy way to do this in Javascript so its just being uppercased for now
            this.titleUp = true;
            break;
          case characterToken:
            this.charUp = true;
            break;
          case wordToken:
            this.wordUp = true;
            break;
          case startToken:
            this.inCaps = true;
            break;
          case endToken:
            this.inCaps = false;
            break;
          default:
            if (this.charUp) {
              destination += r.toUpperCase();
              this.charUp = false;
            } else if (this.wordUp) {
              if (isLetter(r)) {
                destination += r.toUpperCase();
              } else {
                if (!(isNumber(r) || r == apostrophe || r == apostrophe2)) {
                  this.wordUp = false;
                }
                destination += r;
              }
            } else if (this.inCaps) {
              destination += r.toUpperCase();
            } else if (this.titleUp) {
              destination += toTitleCase(r);
            } else {
              destination += r;
            }
        }
      }
      return destination;
    }
  }