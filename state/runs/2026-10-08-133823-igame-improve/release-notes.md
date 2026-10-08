Offline image: `igame:v0.7.31`

## What's Changed
* auto-improve: test: cover encryption key contract through config loading by @hkjang in https://github.com/hkjang/igame/pull/38

This patch adds 25 parser cases and 30 config-loading cases for the existing
32-byte encryption-key contract, encodings, UTF-8 byte length, whitespace
handling, and invalid-input rejection without leaking the key. Product behavior
and database schema are unchanged. Both the old and new tests passed on their
first runs; this is coverage improvement, not a reproduced bug fix or Red→Green.

**Full Changelog**: https://github.com/hkjang/igame/compare/v0.7.30...v0.7.31
