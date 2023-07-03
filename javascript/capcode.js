const characterToken = 'C';
const wordToken = 'W';
const deleteToken = 'D';
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

function isModifier(r) {
  return /\p{M}/u.test(r);
}

function capcode_encode(data) {
  let buf = new Array(Math.ceil(data.length + (data.length / 2) + 8));
  let pos = 0;
  let gobackPos = 0;
  let wordTokenPos = 0;
  let rlast = '.';
  let rlast2 = '.';
  let inWord = false;
  let multiLetter = false;

  for (let r of data) {

    if (inWord) {
      if (isUpper(r)) {
        if (!(isLetter(rlast) || rlast == apostrophe || rlast == apostrophe2 || isModifier(rlast))) {
          buf[pos++] = deleteToken;
          buf[pos++] = ' ';
        }
        multiLetter = true;
        buf[pos++] = r.toLowerCase();
      } else {
        if (isLower(r)) {
          inWord = false;
          buf[wordTokenPos] = characterToken;
          if (multiLetter) {
            for (let i2 = gobackPos; i2 < pos; i2++) {
              if (buf[i2] == deleteToken && buf[i2+1] == ' ') {
                if (isLower(buf[i2 + 2])) {
                  for (let j = pos+1; j > i2 + 2; j--) {
                    buf[j] = buf[j - 1];
                  }
                  buf[i2] = deleteToken;
                  buf[i2+1] = characterToken;
                  buf[i2+2] = ' ';
                  pos++;
                  i2++
                }
                i2 += 2;
              } else {
                if (isLower(buf[i2])) {
                  for (let j = pos+3; j > i2; j--) {
                    buf[j] = buf[j - 3];
                  }
                  buf[i2] = deleteToken;
                  buf[i2+1] = characterToken;
                  buf[i2+2] = ' ';
                  pos += 3;
                  i2 += 3;
                }
              }
            }
          }
          if (!(isLetter(rlast) || rlast == apostrophe || rlast == apostrophe2 || isModifier(rlast))) {
            buf[pos++] = deleteToken;
            buf[pos++] = ' ';
          }
        } else {
          if (isNumber(r)) {
            if (!isNumber(rlast)) {
              buf[pos++] = deleteToken;
              buf[pos++] = ' '
            }
          } else if (!(r == apostrophe || r == apostrophe2 || isModifier(r))) {
            inWord = false;
          }
        }
        buf[pos++] = r
      }
    } else {
      if (isLower(r)) {
        if (!(rlast == ' ' || isLetter(rlast) || (isLetter(rlast2) && (rlast == apostrophe || rlast == apostrophe2)) || isModifier(rlast))) {
          buf[pos++] = deleteToken;
          buf[pos++] = ' ';
        }
        buf[pos++] = r;
      } else if (isUpper(r)) {
        if (rlast == ' ') {
          wordTokenPos = pos - 1;
          buf[wordTokenPos] = wordToken;
          buf[pos++] = ' ';
        } else {
          buf[pos++] = deleteToken;
          wordTokenPos = pos;
          buf[pos++] = wordToken;
          buf[pos++] = ' '
        }
        buf[pos++] = r.toLowerCase();
        gobackPos = pos;
        multiLetter = false;
        inWord = true;
      } else if (isNumber(r)) {
        if (!(rlast == ' ' || isNumber(rlast))) {
          buf[pos++] = deleteToken;
          buf[pos++] = ' ';
        }
        buf[pos++] = r;
      } else {
        buf[pos++] = r;
      }
    }
    rlast2 = rlast;
    rlast = r;
  }

  return buf.slice(0, pos).join('');
}

class CapcodeDecoder {
    constructor() {
      this.inWord = false;
      this.inChar = false;
      this.delete = false;
      this.ignore = false;
    }
  
    decode(data) {
      let destination = "";
      for (let r of data) {
        switch (r) {
          case characterToken:
            this.inChar = true;
            this.inWord = false;
            continue;
          case wordToken:
            this.inWord = true;
            this.inChar = false;
            this.ignore = true;
            continue;
          case deleteToken:
            this.delete = true;
            continue;
          case ' ':
            if (this.delete) {
                this.delete = false;
            } else {
                destination += ' ';
                if (!this.ignore) {
                    this.inWord = false;
                }
            }
            break;
          default:
            if (this.delete) {
                this.delete = false;
            } else if (this.inChar) {
                this.inChar = false;
                destination += r.toUpperCase();
            } else if (this.inWord) {
                if (isLower(r) || isUpper(r)) {
                    destination += r.toUpperCase();
                } else {
                    destination += r;
                    if (!(isNumber(r) || r == apostrophe || r == apostrophe2 || isModifier(r))) {
                      this.inWord = false;
                    }
                }
            } else {
                destination += r;
            }
        }
        this.ignore = false;
      }

      return destination;
    }
}
