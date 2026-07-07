#include <capcode/capcode.hpp>

#include <cassert>
#include <span>
#include <string>
#include <vector>

namespace {

std::span<const std::uint8_t> bytes(std::string_view s) {
  return {reinterpret_cast<const std::uint8_t*>(s.data()), s.size()};
}

std::string str(const std::vector<std::uint8_t>& b) {
  return {reinterpret_cast<const char*>(b.data()), b.size()};
}

}  // namespace

int main() {
  auto encoded = capcode::encode(bytes("Hello NASA 2026"));
  assert(str(capcode::decode(encoded)) == "Hello NASA 2026");

  auto no_cap = capcode::no_capcode_encode(bytes("abcDEF 123"));
  assert(str(capcode::no_capcode_decode(no_cap)) == "abcDEF 123");

  // Streaming decode. "W " is the encoded form of "space + start of an
  // uppercase word", so the leading space is part of the output. Word state is
  // terminated by the space that opens the second chunk, so " world" stays
  // lowercase. These values match the Go reference (capcode.Decoder.Decode).
  capcode::Decoder decoder;
  auto first = decoder.decode(bytes("W hello"));
  auto second = decoder.decode(bytes(" world"));
  assert(str(first) == " HELLO");
  assert(str(second) == " world");

  decoder.reset();
  auto no_first = decoder.no_capcode_decode(std::vector<std::uint8_t>{'a', capcode::NoCapcodeDeleteToken});
  auto no_second = decoder.no_capcode_decode(bytes(" b"));
  assert(str(no_first) == "a");
  assert(str(no_second) == "b");

  // Real-text round trip: encode then decode must reconstruct the input
  // exactly. The sample mixes sentence case, ALL-CAPS acronyms, the literal
  // token letters C/W/D as ordinary capitals, digits, apostrophes and some
  // multibyte UTF-8, and is repeated so the encoder's buffer growth path runs.
  const std::string sample =
      "Nominated for Best Documentary at 2004's Academy Awards, My Architect "
      "follows Nathaniel Kahn. NASA launched a DVD from California to "
      "Washington; it's O'Brien's naive, cafe-going resume. HTTP APIs return "
      "JSON \xE2\x80\x94 3.14 CO2 \xC3\xA9 \xE6\x97\xA5\xE6\x9C\xAC 42%.\n";
  std::string big;
  for (int k = 0; k < 5000; ++k) big += sample;
  auto big_in = bytes(big);

  assert(str(capcode::decode(capcode::encode(big_in))) == big);
  assert(str(capcode::no_capcode_decode(capcode::no_capcode_encode(big_in))) == big);
}
