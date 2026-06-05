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

  capcode::Decoder decoder;
  auto first = decoder.decode(bytes("W hello"));
  auto second = decoder.decode(bytes(" world"));
  assert(str(first) == "HELLO");
  assert(str(second) == " WORLD");

  decoder.reset();
  auto no_first = decoder.no_capcode_decode(std::vector<std::uint8_t>{'a', capcode::NoCapcodeDeleteToken});
  auto no_second = decoder.no_capcode_decode(bytes(" b"));
  assert(str(no_first) == "a");
  assert(str(no_second) == "b");
}
