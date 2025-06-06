# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- **Provider Error Handling**: Provider functions can now return an error as the second return value
  - Supports both `func() T` and `func() (T, error)` provider signatures
  - Supports both `func(deps...) T` and `func(deps...) (T, error)` provider signatures with dependencies
  - When error is not `nil`, provider invocation fails and error is propagated with context
  - When error is `nil`, the first return value is used as the provided instance
  - Error messages include provider type and location information for better debugging
- **Enhanced Error Context**: All provider-related errors now include detailed context information
  - Provider type information
  - Error source location
  - Dependency chain information
- **Comprehensive Interface Documentation**: Added detailed type support comments to all public and internal interfaces
  - Provider interface methods now include detailed type support information
  - Injector interface methods document supported injection patterns
  - Container interface methods specify supported types and error conditions
  - Public API functions include comprehensive usage examples and type constraints

### Technical Details
- Modified `FuncProvider` struct to include `hasError` field
- Updated `NewFuncProvider` to validate error return types
- Enhanced `Invoke` method to handle error checking and propagation
- Maintained backward compatibility with existing provider functions

### Documentation
- Enhanced interface documentation in `dixinternal/interfaces.go` with detailed type support information
- Improved public API documentation in `dix.go` with comprehensive examples and usage patterns
- Added detailed comments for Provider, Injector, Container, and Dependency interfaces
- Documented supported and unsupported types for better developer experience

### Examples
- Added comprehensive provider error handling examples in `example/provider-error/`
- Added correct type usage examples in `example/provider-error-correct/`

### Supported Types (Clarification)
Provider functions support the following output types:
- Pointer types: `*T`
- Interface types: `interface{}`
- Struct types: `struct{}`
- Map types: `map[K]V`
- Slice types: `[]T`
- Function types: `func(...) ...`

**Note**: Basic types (`string`, `int`, `bool`, etc.) are **not** supported as provider outputs. Use pointer types instead (e.g., `*string`, `*int`).

## [Previous Versions] 