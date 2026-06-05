# capcode-cpp

C++20 port of the Go runtime package `github.com/alasdairforsythe/capcode`.

This library contains the runtime Capcode encoder/decoder used by TokenMonster vocabularies. The implementation is a direct C++ translation of the Go encode/decode paths, including the NoCapcode delete-token variant and streaming decoder state.

## Requirements

- C++20 compiler
- CMake 3.20+
- ICU development libraries discoverable through `pkg-config` (`icu-uc`)

On Debian/Ubuntu:

```sh
sudo apt-get install cmake pkg-config libicu-dev
```

## Build

```sh
cmake -S . -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -j
ctest --test-dir build --output-on-failure
```

## Install

```sh
cmake --install build --prefix /usr/local
```

Then consume it from CMake:

```cmake
find_package(capcode_cpp CONFIG REQUIRED)
target_link_libraries(your_target PRIVATE capcode::capcode)
```

## API

```cpp
#include <capcode/capcode.hpp>

#include <cstdint>
#include <span>
#include <string>
#include <vector>

std::span<const std::uint8_t> bytes(std::string_view s) {
  return {reinterpret_cast<const std::uint8_t*>(s.data()), s.size()};
}

auto encoded = capcode::encode(bytes("Hello NASA"));
auto decoded = capcode::decode(encoded);

capcode::Decoder stream;
auto part1 = stream.decode(bytes("W hello"));
auto part2 = stream.decode(bytes(" world"));
```

Public functions:

- `capcode::encode(data)` mirrors Go `Encode`.
- `capcode::decode(bytes)` mirrors Go `Decode`, taking ownership of the input buffer and returning the decoded buffer.
- `capcode::decode_copy(data)` is a copy-in convenience wrapper.
- `capcode::no_capcode_encode(data)` mirrors Go `NoCapcodeEncode`.
- `capcode::no_capcode_decode(bytes)` mirrors Go `NoCapcodeDecode`.
- `capcode::no_capcode_decode_copy(data)` is a copy-in convenience wrapper.
- `capcode::Decoder` mirrors the Go streaming `Decoder` state for Capcode and NoCapcode decoding.

## License

MIT, matching the upstream project.
