# Android appearance QA — 2026-09-13

final result: passed

Scope: native Compose implementation of the two selected day/night references. This is a visual adaptation, not Apple's proprietary material renderer.

Evidence: `ui-day.png` and `ui-night.png`, rendered by AppearanceTest on the Android emulator. Both references and both updated captures were inspected together. Reference aspect ratio is approximately 390:844; emulator content is 1080:2400, so absolute screenshot coordinates are not compared. Test composer contains “晚餐” to verify input survives configuration changes; reference composer is empty.

Initial issue: screenshot harness used Surface's default surface color rather than the app background. Fixed the harness to use the same background as App, then recaptured both modes. White groups now separate from gray background; dark cards separate from the charcoal canvas.

Verified: day grouped receipt, night amount hierarchy, readable secondary text, accessible settings/send descriptions, no clipped content in captured viewport, source summary and comments retained. Theme configuration changes preserve displayed message and input. Existing login, 2FA, save, editor, history management and storage tests pass.

Remaining P3 differences: Android system font/icons differ from iOS; night rendering uses subtle tonal surfaces instead of optical refraction; timestamps retain full year for historical clarity. Small-screen/font-scale extremes and physical-device rendering were not visually verified.

Build, unit tests and lint pass (existing lint warnings remain). Six emulator instrumentation tests pass. First flow run encountered pre-existing emulator profile data; rerunning on cleared dedicated emulator app storage passed.
