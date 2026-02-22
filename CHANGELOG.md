# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com),
and this project adheres to [Semantic Versioning](https://semver.org).

## [Unreleased]

### Fixed

- Remove committed binary and add to .gitignore

## [v0.1.0] - 2026-02-22

### Added

- Terminal fireplace with ASCII art flames and ember bed
- `--color` flag and cap fireplace height
- Fireplace frame: brick walls, mantel, and hearth floor
- Cross-platform terminal size via `golang.org/x/term`
- GoReleaser config and release workflow
- Makefile with standard dev commands
- .gitignore for build artifacts

### Changed

- Rewrite to lo-fi ASCII fire aesthetic
- Full-screen true-color fire with dynamic flames and ember bed
- Improve flame rendering: distinct tongues, swaying, height variation
- Narrow flames to hearth width
- Clearer log shapes with stacked layout and ember glow
- Improve hearth base: hot coals, crossed logs, stone floor
- Re-record demo with color mode and fireplace frame

[Unreleased]: https://github.com/maxbeizer/gh-hearth/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/maxbeizer/gh-hearth/releases/tag/v0.1.0
