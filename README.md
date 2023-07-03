# capcode
Lossless normalization of text for the purpose of tokenization.

### Fully UTF-8 Compliant

- Supports Unicode 13.0.0 (newer version supported depending on the implementation)
- Works correctly with any UTF-8 encoding scheme (NFC, NFD, etc.)

### Features

- No information is lost
- The encoded text can be decoded exactly back to the original
- Safe: an LLM trained on this will still understand about uppercasing
