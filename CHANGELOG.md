# Changelog

## Version 0.2 (In Development)

### Breaking Changes
- Migrated from `golang.org/x/net/context` to standard library `context` package
- Updated `basic_server` example to use modern `NewServer()` and `Handle()` API

### Features
- Added Go modules support (go.mod)
- Migrated from Travis CI to GitHub Actions for CI/CD
- Added support for Go 1.21+

### Bug Fixes
- Fixed incorrect type assertions in `message.go` - now properly uses type variable `t` instead of `arg`
- Fixed string reading in OSC message parsing - now correctly uses returned byte count from `readPaddedString()`

### Improvements
- Updated all import paths from `github.com/hypebeast/go-osc` to `github.com/kward/go-osc`
- Removed dependency on `golang.org/x/net` package
- Modernized CI/CD pipeline with GitHub Actions
- Updated examples to use current best practices

## Version 0.1

Initial fork from https://github.com/hypebeast/go-osc