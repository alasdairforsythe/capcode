#pragma once

#include <cstdint>
#include <span>
#include <vector>

namespace capcode {

using Bytes = std::vector<std::uint8_t>;

constexpr char32_t CharacterToken = 'C';
constexpr char32_t WordToken = 'W';
constexpr char32_t DeleteToken = 'D';
constexpr std::uint8_t NoCapcodeDeleteToken = 0x7F;
constexpr std::uint8_t NoCapcodeSubstitute = 0x14;
constexpr char32_t Apostrophe = '\'';
constexpr char32_t Apostrophe2 = 0x2019U;
constexpr char32_t RuneError = 0xFFFDU;

Bytes encode(std::span<const std::uint8_t> data);
Bytes decode(Bytes data);
Bytes decode_copy(std::span<const std::uint8_t> data);

Bytes no_capcode_encode(std::span<const std::uint8_t> data);
Bytes no_capcode_decode(Bytes data);
Bytes no_capcode_decode_copy(std::span<const std::uint8_t> data);

class Decoder {
 public:
  Bytes decode(std::span<const std::uint8_t> data);
  Bytes no_capcode_decode(std::span<const std::uint8_t> data);
  void reset();

 private:
  bool in_word_ = false;
  bool in_char_ = false;
  bool delete_ = false;
  bool ignore_ = false;
};

}  // namespace capcode
