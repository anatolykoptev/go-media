# Changelog

## [0.3.7](https://github.com/anatolykoptev/go-media/compare/v0.3.6...v0.3.7) (2026-07-25)


### Fixed

* **dash:** select only H.264 video representations for Telegram playback ([#31](https://github.com/anatolykoptev/go-media/issues/31)) ([50a7c96](https://github.com/anatolykoptev/go-media/commit/50a7c960f63f05bd2e9ea0cf2d81a463a0435f5f))

## [0.3.6](https://github.com/anatolykoptev/go-media/compare/v0.3.5...v0.3.6) (2026-07-25)


### Fixed

* **dash:** classify bare contentType + prefer FBContentLength ([#29](https://github.com/anatolykoptev/go-media/issues/29)) ([4ebb159](https://github.com/anatolykoptev/go-media/commit/4ebb15982a1a8968fde93bd1c5d78a0941656b24))

## [0.3.5](https://github.com/anatolykoptev/go-media/compare/v0.3.4...v0.3.5) (2026-07-25)


### Added

* **instagram:** DASH manifest parsing + budget-aware quality selection (unlocks 1080p) ([#27](https://github.com/anatolykoptev/go-media/issues/27)) ([a97cbcd](https://github.com/anatolykoptev/go-media/commit/a97cbcd2b900d672f3ac6f042f41aad83abdb4fe))

## [0.3.4](https://github.com/anatolykoptev/go-media/compare/v0.3.3...v0.3.4) (2026-07-24)


### Added

* **stats:** map view/comment/share engagement counts (Instagram + YouTube) ([fabf77f](https://github.com/anatolykoptev/go-media/commit/fabf77f30512178f5c334002eedf0f2137eeebef))


### Changed

* remove duplicate transcribe/openai — use transcribe/gostt ([#25](https://github.com/anatolykoptev/go-media/issues/25)) ([6b6bc40](https://github.com/anatolykoptev/go-media/commit/6b6bc40e0920e36a46c5228fb2f57a8779797c3b))


### Dependencies

* bump go-stt to v0.3.0 ([#23](https://github.com/anatolykoptev/go-media/issues/23)) ([3b74382](https://github.com/anatolykoptev/go-media/commit/3b743823af07d9eefb85603bff21d5925266641d))

## [0.3.3](https://github.com/anatolykoptev/go-media/compare/v0.3.2...v0.3.3) (2026-07-17)


### Added

* clip padding + audio fade at cut boundaries ([#20](https://github.com/anatolykoptev/go-media/issues/20)) ([2c4d899](https://github.com/anatolykoptev/go-media/commit/2c4d899ce702213515132b5e81fc563855dde936))

## [0.3.2](https://github.com/anatolykoptev/go-media/compare/v0.3.1...v0.3.2) (2026-07-17)


### Added

* add video clip extraction from transcription chunks ([#15](https://github.com/anatolykoptev/go-media/issues/15)) ([e63c4d1](https://github.com/anatolykoptev/go-media/commit/e63c4d11e7c27c57be4139b040b30ba9341f6de3))

## [0.3.1](https://github.com/anatolykoptev/go-media/compare/v0.3.0...v0.3.1) (2026-07-17)


### Fixed

* repo-review-council bugs [#1](https://github.com/anatolykoptev/go-media/issues/1)-[#12](https://github.com/anatolykoptev/go-media/issues/12) + CI setup ([#13](https://github.com/anatolykoptev/go-media/issues/13)) ([aa02404](https://github.com/anatolykoptev/go-media/commit/aa02404a0dbbfc862a2b246d745924d19ac94f9f))
