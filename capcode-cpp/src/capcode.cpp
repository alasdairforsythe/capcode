#include <capcode/capcode.hpp>

#include <unicode/uchar.h>

#include <algorithm>

namespace capcode {
namespace {

constexpr int bufferReserve = 7;

struct Rune {
  char32_t value = RuneError;
  int size = 0;
};

Rune decode_utf8(std::span<const std::uint8_t> bytes) {
  if (bytes.empty()) {
    return {RuneError, 0};
  }
  const auto b0 = bytes[0];
  if (b0 < 0x80) {
    return {b0, 1};
  }
  auto invalid = Rune{RuneError, 1};
  if (b0 < 0xC2) {
    return invalid;
  }
  if (b0 < 0xE0) {
    if (bytes.size() < 2 || (bytes[1] & 0xC0) != 0x80) return invalid;
    return {static_cast<char32_t>(((b0 & 0x1F) << 6) | (bytes[1] & 0x3F)), 2};
  }
  if (b0 < 0xF0) {
    if (bytes.size() < 3 || (bytes[1] & 0xC0) != 0x80 || (bytes[2] & 0xC0) != 0x80) {
      return invalid;
    }
    if (b0 == 0xE0 && bytes[1] < 0xA0) return invalid;
    if (b0 == 0xED && bytes[1] >= 0xA0) return invalid;
    return {static_cast<char32_t>(((b0 & 0x0F) << 12) | ((bytes[1] & 0x3F) << 6) |
                                  (bytes[2] & 0x3F)),
            3};
  }
  if (b0 < 0xF5) {
    if (bytes.size() < 4 || (bytes[1] & 0xC0) != 0x80 || (bytes[2] & 0xC0) != 0x80 ||
        (bytes[3] & 0xC0) != 0x80) {
      return invalid;
    }
    if (b0 == 0xF0 && bytes[1] < 0x90) return invalid;
    if (b0 == 0xF4 && bytes[1] >= 0x90) return invalid;
    return {static_cast<char32_t>(((b0 & 0x07) << 18) | ((bytes[1] & 0x3F) << 12) |
                                  ((bytes[2] & 0x3F) << 6) | (bytes[3] & 0x3F)),
            4};
  }
  return invalid;
}

bool is_modifier(char32_t r) {
  auto t = u_charType(static_cast<UChar32>(r));
  return t == U_NON_SPACING_MARK || t == U_COMBINING_SPACING_MARK || t == U_ENCLOSING_MARK;
}

bool unicode_letter(char32_t r) {
  if (r < 128) return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z');
  return u_isUAlphabetic(static_cast<UChar32>(r));
}

bool is_number(char32_t r) {
  if (r < 128) return r >= '0' && r <= '9';
  auto t = u_charType(static_cast<UChar32>(r));
  return t == U_DECIMAL_DIGIT_NUMBER || t == U_LETTER_NUMBER || t == U_OTHER_NUMBER;
}

bool is_lower(char32_t r) {
  if (r < 128) return r >= 'a' && r <= 'z';
  return u_islower(static_cast<UChar32>(r));
}

bool is_upper(char32_t r) {
  if (r < 128) return r >= 'A' && r <= 'Z';
  return u_isupper(static_cast<UChar32>(r));
}

char32_t to_lower(char32_t r) {
  if (r >= 'A' && r <= 'Z') return r + 32;
  return static_cast<char32_t>(u_tolower(static_cast<UChar32>(r)));
}

char32_t to_upper(char32_t r) {
  if (r >= 'a' && r <= 'z') return r - 32;
  return static_cast<char32_t>(u_toupper(static_cast<UChar32>(r)));
}

void grow(Bytes& buf) { buf.resize(buf.size() + (buf.size() / 3) + bufferReserve); }

int write_utf8_at(Bytes& buf, std::size_t pos, char32_t r) {
  if (r <= 0x7F) {
    buf[pos] = static_cast<std::uint8_t>(r);
    return 1;
  }
  if (r <= 0x7FF) {
    buf[pos] = static_cast<std::uint8_t>(0xC0 | (r >> 6));
    buf[pos + 1] = static_cast<std::uint8_t>(0x80 | (r & 0x3F));
    return 2;
  }
  if (r <= 0xFFFF) {
    buf[pos] = static_cast<std::uint8_t>(0xE0 | (r >> 12));
    buf[pos + 1] = static_cast<std::uint8_t>(0x80 | ((r >> 6) & 0x3F));
    buf[pos + 2] = static_cast<std::uint8_t>(0x80 | (r & 0x3F));
    return 3;
  }
  buf[pos] = static_cast<std::uint8_t>(0xF0 | (r >> 18));
  buf[pos + 1] = static_cast<std::uint8_t>(0x80 | ((r >> 12) & 0x3F));
  buf[pos + 2] = static_cast<std::uint8_t>(0x80 | ((r >> 6) & 0x3F));
  buf[pos + 3] = static_cast<std::uint8_t>(0x80 | (r & 0x3F));
  return 4;
}

void copy_original_at(Bytes& buf, std::size_t& pos, std::span<const std::uint8_t> data,
                      std::size_t i, int n) {
  switch (n) {
    case 1:
      buf[pos] = data[i];
      ++pos;
      break;
    case 2:
      buf[pos] = data[i];
      buf[pos + 1] = data[i + 1];
      pos += 2;
      break;
    case 3:
      buf[pos] = data[i];
      buf[pos + 1] = data[i + 1];
      buf[pos + 2] = data[i + 2];
      pos += 3;
      break;
    case 4:
      buf[pos] = data[i];
      buf[pos + 1] = data[i + 1];
      buf[pos + 2] = data[i + 2];
      buf[pos + 3] = data[i + 3];
      pos += 4;
      break;
  }
}

}  // namespace

Bytes encode(std::span<const std::uint8_t> data) {
  char32_t r = 0, r2 = 0, rlast = 0, rlast2 = 0;
  std::size_t i = 0;
  int n = 0;
  std::size_t i2 = 0;
  int n2 = 0;
  std::size_t pos = 0;
  std::size_t word_token_pos = 0;
  bool in_word = false;
  bool multi_letter = false;
  Bytes buf(data.size() + (data.size() / 2) + bufferReserve);
  std::size_t danger_zone = buf.size() - bufferReserve;

  while (i < data.size()) {
    auto dec = decode_utf8(data.subspan(i));
    r = dec.value;
    n = dec.size <= 0 ? 1 : dec.size;

    if (pos >= danger_zone) {
      grow(buf);
      danger_zone = buf.size() - bufferReserve;
    }

    if (in_word) {
      if (is_upper(r)) {
        if (!(unicode_letter(rlast) || rlast == Apostrophe || rlast == Apostrophe2 ||
              is_modifier(rlast))) {
          buf[pos] = static_cast<std::uint8_t>(DeleteToken);
          buf[pos + 1] = ' ';
          pos += 2;
        }
        multi_letter = true;
        pos += static_cast<std::size_t>(write_utf8_at(buf, pos, to_lower(r)));
      } else {
        if (is_lower(r)) {
          in_word = false;
          buf[word_token_pos] = static_cast<std::uint8_t>(CharacterToken);
          if (multi_letter) {
            for (i2 = static_cast<std::size_t>(n2); i2 < pos;
                 i2 += static_cast<std::size_t>(n2)) {
              if (buf[i2] == DeleteToken && buf[i2 + 1] == ' ') {
                auto r2d = decode_utf8(std::span<const std::uint8_t>(buf).subspan(i2 + 2));
                r2 = r2d.value;
                n2 = r2d.size <= 0 ? 1 : r2d.size;
                if (is_lower(r2)) {
                  if (pos >= danger_zone) {
                    grow(buf);
                    danger_zone = buf.size() - bufferReserve;
                  }
                  std::copy_backward(buf.begin() + static_cast<std::ptrdiff_t>(i2 + 2),
                                     buf.begin() + static_cast<std::ptrdiff_t>(pos),
                                     buf.begin() + static_cast<std::ptrdiff_t>(pos + 1));
                  buf[i2] = static_cast<std::uint8_t>(DeleteToken);
                  buf[i2 + 1] = static_cast<std::uint8_t>(CharacterToken);
                  buf[i2 + 2] = ' ';
                  ++pos;
                  ++i2;
                }
                i2 += 2;
              } else {
                auto r2d = decode_utf8(std::span<const std::uint8_t>(buf).subspan(i2));
                r2 = r2d.value;
                n2 = r2d.size <= 0 ? 1 : r2d.size;
                if (is_lower(r2)) {
                  if (pos >= danger_zone) {
                    grow(buf);
                    danger_zone = buf.size() - bufferReserve;
                  }
                  std::copy_backward(buf.begin() + static_cast<std::ptrdiff_t>(i2),
                                     buf.begin() + static_cast<std::ptrdiff_t>(pos),
                                     buf.begin() + static_cast<std::ptrdiff_t>(pos + 3));
                  buf[i2] = static_cast<std::uint8_t>(DeleteToken);
                  buf[i2 + 1] = static_cast<std::uint8_t>(CharacterToken);
                  buf[i2 + 2] = ' ';
                  pos += 3;
                  i2 += 3;
                }
              }
            }
          }
          if (!(unicode_letter(rlast) || rlast == Apostrophe || rlast == Apostrophe2 ||
                is_modifier(rlast))) {
            buf[pos] = static_cast<std::uint8_t>(DeleteToken);
            buf[pos + 1] = ' ';
            pos += 2;
          }
        } else {
          if (is_number(r)) {
            if (!is_number(rlast)) {
              buf[pos] = static_cast<std::uint8_t>(DeleteToken);
              buf[pos + 1] = ' ';
              pos += 2;
            }
          } else if (!(r == Apostrophe || r == Apostrophe2 || is_modifier(r))) {
            in_word = false;
          }
        }
        copy_original_at(buf, pos, data, i, n);
      }
    } else {
      if (is_lower(r)) {
        if (!(rlast == ' ' || is_lower(rlast) ||
              (unicode_letter(rlast2) && (rlast == Apostrophe || rlast == Apostrophe2)) ||
              is_modifier(rlast))) {
          buf[pos] = static_cast<std::uint8_t>(DeleteToken);
          buf[pos + 1] = ' ';
          pos += 2;
        }
        copy_original_at(buf, pos, data, i, n);
      } else if (is_upper(r)) {
        if (rlast == ' ') {
          word_token_pos = pos - 1;
          buf[pos - 1] = static_cast<std::uint8_t>(WordToken);
          buf[pos] = ' ';
          ++pos;
        } else {
          word_token_pos = pos + 1;
          buf[pos] = static_cast<std::uint8_t>(DeleteToken);
          buf[pos + 1] = static_cast<std::uint8_t>(WordToken);
          buf[pos + 2] = ' ';
          pos += 3;
        }
        pos += static_cast<std::size_t>(write_utf8_at(buf, pos, to_lower(r)));
        n2 = static_cast<int>(pos);
        multi_letter = false;
        in_word = true;
      } else if (is_number(r)) {
        if (!(rlast == ' ' || is_number(rlast))) {
          buf[pos] = static_cast<std::uint8_t>(DeleteToken);
          buf[pos + 1] = ' ';
          pos += 2;
        }
        copy_original_at(buf, pos, data, i, n);
      } else {
        copy_original_at(buf, pos, data, i, n);
      }
    }

    rlast2 = rlast;
    rlast = r;
    i += static_cast<std::size_t>(n);
  }
  buf.resize(pos);
  return buf;
}

Bytes decode(Bytes b) {
  char32_t r = 0;
  std::size_t pos = 0;
  bool in_char = false;
  bool in_word = false;
  bool delete_next = false;
  bool ignore = false;
  for (std::size_t i = 0; i < b.size();) {
    auto dec = decode_utf8(std::span<const std::uint8_t>(b).subspan(i));
    r = dec.value;
    int n = dec.size <= 0 ? 1 : dec.size;
    switch (r) {
      case CharacterToken:
        in_char = true;
        in_word = false;
        i += static_cast<std::size_t>(n);
        continue;
      case WordToken:
        in_word = true;
        in_char = false;
        ignore = true;
        i += static_cast<std::size_t>(n);
        continue;
      case DeleteToken:
        delete_next = true;
        i += static_cast<std::size_t>(n);
        continue;
      case ' ':
        if (delete_next) {
          delete_next = false;
        } else {
          b[pos] = ' ';
          ++pos;
          if (!ignore) in_word = false;
        }
        break;
      default:
        if (delete_next) {
          delete_next = false;
        } else if (in_char) {
          in_char = false;
          if (r == RuneError) {
            copy_original_at(b, pos, b, i, n);
          } else {
            pos += static_cast<std::size_t>(write_utf8_at(b, pos, to_upper(r)));
          }
        } else if (in_word) {
          if (is_lower(r) || is_upper(r)) {
            pos += static_cast<std::size_t>(write_utf8_at(b, pos, to_upper(r)));
          } else {
            copy_original_at(b, pos, b, i, n);
            if (!(is_number(r) || r == Apostrophe || r == Apostrophe2 || is_modifier(r))) {
              in_word = false;
            }
          }
        } else {
          copy_original_at(b, pos, b, i, n);
        }
        break;
    }
    ignore = false;
    i += static_cast<std::size_t>(n);
  }
  b.resize(pos);
  return b;
}

Bytes decode_copy(std::span<const std::uint8_t> data) { return decode(Bytes(data.begin(), data.end())); }

Bytes no_capcode_encode(std::span<const std::uint8_t> data) {
  char32_t r = 0, rlast = 0, rlast2 = 0;
  std::size_t pos = 0;
  Bytes buf(data.size() + (data.size() / 2) + bufferReserve);
  std::size_t danger_zone = buf.size() - bufferReserve;
  for (std::size_t i = 0; i < data.size();) {
    auto dec = decode_utf8(data.subspan(i));
    r = dec.value;
    int n = dec.size <= 0 ? 1 : dec.size;
    if (pos >= danger_zone) {
      grow(buf);
      danger_zone = buf.size() - bufferReserve;
    }
    if (unicode_letter(r)) {
      if (!(rlast == ' ' || unicode_letter(rlast) ||
            (unicode_letter(rlast2) && (rlast == Apostrophe || rlast == Apostrophe2)) ||
            is_modifier(rlast))) {
        buf[pos] = NoCapcodeDeleteToken;
        buf[pos + 1] = ' ';
        pos += 2;
      }
    } else if (is_number(r)) {
      if (!(rlast == ' ' || is_number(rlast))) {
        buf[pos] = NoCapcodeDeleteToken;
        buf[pos + 1] = ' ';
        pos += 2;
      }
    }
    if (r == NoCapcodeDeleteToken) {
      buf[pos] = NoCapcodeSubstitute;
      ++pos;
    } else {
      copy_original_at(buf, pos, data, i, n);
    }
    rlast2 = rlast;
    rlast = r;
    i += static_cast<std::size_t>(n);
  }
  buf.resize(pos);
  return buf;
}

Bytes no_capcode_decode(Bytes b) {
  std::size_t pos = 0;
  for (std::size_t i = 0; i < b.size(); ++i) {
    if (b[i] == NoCapcodeDeleteToken) {
      ++i;
    } else {
      b[pos] = b[i];
      ++pos;
    }
  }
  b.resize(pos);
  return b;
}

Bytes no_capcode_decode_copy(std::span<const std::uint8_t> data) {
  return no_capcode_decode(Bytes(data.begin(), data.end()));
}

Bytes Decoder::decode(std::span<const std::uint8_t> src) {
  Bytes dst(src.size() + bufferReserve);
  std::size_t pos = 0;
  for (std::size_t i = 0; i < src.size();) {
    auto dec = decode_utf8(src.subspan(i));
    auto r = dec.value;
    int n = dec.size <= 0 ? 1 : dec.size;
    switch (r) {
      case CharacterToken:
        in_char_ = true;
        in_word_ = false;
        i += static_cast<std::size_t>(n);
        continue;
      case WordToken:
        in_word_ = true;
        in_char_ = false;
        ignore_ = true;
        i += static_cast<std::size_t>(n);
        continue;
      case DeleteToken:
        delete_ = true;
        i += static_cast<std::size_t>(n);
        continue;
      case ' ':
        if (delete_) {
          delete_ = false;
        } else {
          dst[pos] = ' ';
          ++pos;
          if (!ignore_) in_word_ = false;
        }
        break;
      default:
        if (delete_) {
          delete_ = false;
        } else if (in_char_) {
          in_char_ = false;
          if (r == RuneError) {
            copy_original_at(dst, pos, src, i, n);
          } else {
            pos += static_cast<std::size_t>(write_utf8_at(dst, pos, to_upper(r)));
          }
        } else if (in_word_) {
          if (is_lower(r) || is_upper(r)) {
            pos += static_cast<std::size_t>(write_utf8_at(dst, pos, to_upper(r)));
          } else {
            copy_original_at(dst, pos, src, i, n);
            if (!(is_number(r) || r == Apostrophe || r == Apostrophe2 || is_modifier(r))) {
              in_word_ = false;
            }
          }
        } else {
          copy_original_at(dst, pos, src, i, n);
        }
        break;
    }
    ignore_ = false;
    i += static_cast<std::size_t>(n);
  }
  dst.resize(pos);
  return dst;
}

Bytes Decoder::no_capcode_decode(std::span<const std::uint8_t> src) {
  Bytes dst(src.size());
  std::size_t pos = 0;
  for (std::size_t i = 0; i < src.size(); ++i) {
    if (src[i] == NoCapcodeDeleteToken) {
      delete_ = true;
    } else {
      if (delete_) {
        delete_ = false;
      } else {
        dst[pos] = src[i];
        ++pos;
      }
    }
  }
  dst.resize(pos);
  return dst;
}

void Decoder::reset() {
  in_word_ = false;
  in_char_ = false;
  delete_ = false;
  ignore_ = false;
}

}  // namespace capcode
