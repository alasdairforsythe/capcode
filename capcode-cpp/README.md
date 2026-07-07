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
cmake -S . -B build -DCMAKE_BUILD_TYPE=Release -DCMAKE_CXX_FLAGS_RELEASE="-O2 -DNDEBUG"
cmake --build build -j
ctest --test-dir build --output-on-failure
```

### Optimization level

Build with `-O2`. Measured with g++ 13.3 on real text, the encode hot path is
about 9% faster at `-O2` than at the `-O3` that `CMAKE_BUILD_TYPE=Release`
selects by default (`-O3` over-optimizes the branchy encode loop); decode is
within noise. Plain `cmake -DCMAKE_BUILD_TYPE=Release` still works and is
correct, just slightly slower on encode.

The SIMD run-copy fast paths use only SSE2, which is part of the x86-64 baseline
(no `-march`, no runtime CPU check, and a scalar fallback compiles on other
architectures). `-mavx2`, `-march=native`, and hand-written AVX2/AVX-512 were
measured and did **not** help — capcode's runs are short (word length), so the
wider vectors add per-operation cost without covering more useful bytes. Stick
with the portable default.

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
