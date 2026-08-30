# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
- Add timestamps to process output (fgrosse/prox#5)

## [1.1.0] - 2024-07-13
- Improve file name matching when determining parser for value of --procfile flag (fgrosse/prox#8)

## [1.0.0] - 2023-03-10
- v1 release

## [0.5.0] - 2018-12-09
### Fixed
- Implement quoting for environment variables (e.g. to preserve spaces at the end)
- Go module support

## [0.4.0] - 2018-11-11
### Fixed
- Correctly parse spaces, flags, escapes, etc in process command lines.

### Changed
- Add more information to README.md

## [0.3.0] - 2018-04-02
### Added
- Validate all processes and prevent duplicate process names.

## [0.2.0] - 2018-04-01
### Added
- Auto-detect log format of processes (i.e. JSON).
