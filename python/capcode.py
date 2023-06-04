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
    destination = []
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
                destination.append(r.upper())
                char_up = False
            elif word_up:
                if r.isalpha():
                    destination.append(r.upper())
                else:
                    if not (r.isdigit() or r == apostrophe or r == apostrophe2 or is_modifier(r)):
                        word_up = False
                    destination.append(r)
            elif in_caps:
                destination.append(r.upper())
            else:
                destination.append(r)
    return ''.join(destination)

class Decoder:
    def __init__(self):
        self.in_caps = False
        self.char_up = False
        self.word_up = False

    def decode(self, data):
        destination = []
        in_caps = self.in_caps
        char_up = self.char_up
        word_up = self.word_up
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
                    destination.append(r.upper())
                    char_up = False
                elif word_up:
                    if r.isalpha():
                        destination.append(r.upper())
                    else:
                        if not (r.isdigit() or r == apostrophe or r == apostrophe2 or is_modifier(r)):
                            word_up = False
                        destination.append(r)
                elif in_caps:
                    destination.append(r.upper())
                else:
                    destination.append(r)
        self.in_caps = in_caps
        self.char_up = char_up
        self.word_up = word_up
        return ''.join(destination)
