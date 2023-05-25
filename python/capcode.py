import unicodedata

characterToken = 'C'
wordToken = 'W'
beginToken = 'B'
endToken = 'E'
apostrophe = '\''
apostrophe2 = '’'

def is_modifier(r):
    return 'M' in unicodedata.category(r)

def encode(data):
    buf = []
    cap_start_pos = 0
    cap_end_pos = 0
    second_cap_start_pos = 0
    last_word_cap_end_pos = 0
    n_words = 0
    in_caps = False
    single_letter = False
    in_word = False

    for r in data:
        if in_caps:
            if r.isalpha():
                if r.isupper():
                    if not in_word:
                        in_word = True
                        if n_words == 0:
                            second_cap_start_pos = len(buf)
                        last_word_cap_end_pos = cap_end_pos
                        n_words += 1
                    buf.append(r.lower())
                    cap_end_pos = len(buf)
                    single_letter = False
                else:
                    if single_letter and in_word:
                        buf[cap_start_pos] = characterToken
                    else:
                        if n_words == 0:
                            if not in_word:
                                buf[cap_start_pos] = wordToken
                            else:
                                buf[cap_start_pos] = characterToken
                                i2 = cap_start_pos + 1
                                while i2 < cap_end_pos:
                                    r2 = buf[i2]
                                    if r2.isalpha():
                                        buf.insert(i2, characterToken)
                                        cap_end_pos += 1
                                        i2 += 1
                                    i2 += 1
                        elif n_words == 1:
                            buf[cap_start_pos] = wordToken
                            if not in_word:
                                buf.insert(second_cap_start_pos, wordToken)
                            else:
                                i2 = second_cap_start_pos
                                while i2 < cap_end_pos:
                                    r2 = buf[i2]
                                    if r2.isalpha():
                                        buf.insert(i2, characterToken)
                                        cap_end_pos += 1
                                        i2 += 1
                                    i2 += 1
                        elif n_words == 2:
                            if not in_word:
                                buf.insert(cap_end_pos, endToken)
                            else:
                                buf[cap_start_pos] = wordToken
                                buf.insert(second_cap_start_pos, wordToken)
                                cap_end_pos += 1
                                i2 = last_word_cap_end_pos + 1
                                while i2 < cap_end_pos:
                                    r2 = buf[i2]
                                    if r2.isalpha():
                                        buf.insert(i2, characterToken)
                                        cap_end_pos += 1
                                        i2 += 1
                                    i2 += 1
                        else:
                            if not in_word:
                                buf.insert(cap_end_pos, endToken)
                            else:
                                buf.insert(last_word_cap_end_pos, endToken)
                                cap_end_pos += 1
                                i2 = last_word_cap_end_pos + 1
                                while i2 < cap_end_pos:
                                    r2 = buf[i2]
                                    if r2.isalpha():
                                        buf.insert(i2, characterToken)
                                        cap_end_pos += 1
                                        i2 += 1
                                    i2 += 1
                    buf.append(r)
                    in_caps = False
                    cap_start_pos = len(buf)
            else:
                buf.append(r)
                if is_modifier(r):
                    cap_end_pos = len(buf)
                elif r != apostrophe and r != apostrophe2 and not r.isdigit():
                    in_word = False
        else:
            if r.isupper():
                cap_start_pos = len(buf)
                buf.append(beginToken)
                buf.append(r.lower())
                cap_end_pos = len(buf)
                n_words = 0
                in_caps = True
                in_word = True
                single_letter = True
            else:
                buf.append(r)
                cap_start_pos = len(buf)

    if in_caps:
        if n_words == 0:
            buf[cap_start_pos] = wordToken
        elif n_words == 1:
            buf[cap_start_pos] = wordToken
            buf.insert(second_cap_start_pos, wordToken)
        else:
            buf.insert(cap_end_pos, endToken)

    return ''.join(buf)

def decode(data):
    destination = ""
    in_caps = False
    char_up = False
    word_up = False
    for r in data:
        if r == characterToken:
            char_up = True
        elif r == wordToken:
            word_up = True
        elif r == beginToken:
            in_caps = True
        elif r == endToken:
            in_caps = False
        else:
            if char_up:
                destination += r.upper()
                char_up = False
            elif word_up:
                if r.isalpha():
                    destination += r.upper()
                else:
                    if not (r.isdigit() or r == apostrophe or r == apostrophe2 or is_modifier(r)):
                        word_up = False
                    destination += r
            elif in_caps:
                destination += r.upper()
            else:
                destination += r
    return destination


class Decoder:
    def __init__(self):
        self.in_caps = False
        self.char_up = False
        self.word_up = False

    def decode(self, data):
        destination = ""
        for r in data:
            if r == characterToken:
                self.char_up = True
            elif r == wordToken:
                self.word_up = True
            elif r == beginToken:
                self.in_caps = True
            elif r == endToken:
                self.in_caps = False
            else:
                if self.char_up:
                    destination += r.upper()
                    self.char_up = False
                elif self.word_up:
                    if r.isalpha():
                        destination += r.upper()
                    else:
                        if not (r.isdigit() or r == apostrophe or r == apostrophe2 or is_modifier(r)):
                            self.word_up = False
                        destination += r
                elif self.in_caps:
                    destination += r.upper()
                else:
                    destination += r
        return destination

#def main():
#    string_to_encode = "THIS a ISÁ d A Test! TeST. THE Á QUICK BROWN FOXa  ÁÁ s Áccord s Á ÁÁÁjumped. ǅ TEST TEST TEST Á"
#    print("Original String: ", string_to_encode)
#    string_to_encode = unicodedata.normalize('NFC', string_to_encode)
#    print("NFC String:      ", string_to_encode)
#
#    encoded_string = encode(string_to_encode)
#    print("Encoded String: ", encoded_string)
#    encoded_string = unicodedata.normalize('NFD', encoded_string)
#    print("NFD String:      ", encoded_string)
#
#    decoder = Decoder()
#    decoded_string = decoder.decode(encoded_string)
#    print("Decoded String:  ", decoded_string)
#    print("NFC String:      ", unicodedata.normalize('NFC', decoded_string))
#
#if __name__ == "__main__":
#    main()