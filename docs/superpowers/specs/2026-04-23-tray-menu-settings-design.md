# Tray Menu Settings Design

## Goal

Add a Windows system tray menu for the desktop pet. The menu must let the user hide or show the pet, adjust the rendered pet size, adjust walking step size, and adjust walking interval. Settings must persist across restarts.

The tray menu is the Windows notification area menu, not the Wails application menu attached to `options.App.Menu`.

## Scope

In scope:

- Windows tray icon with a right-click menu.
- Hide/show toggle.
- Pet display scale presets: 80%, 100%, 120%, 150%.
- Walking step presets: 10px, 20px, 30px, 40px.
- Walking interval presets: 100ms, 150ms, 300ms, 600ms.
- Persisted config values in the existing user config file.
- Runtime updates without requiring app restart.
- Frontend resize notification so the sprite matches the backend window size.

Out of scope:

- Arbitrary numeric input controls.
- A separate settings window.
- Cross-platform tray behavior beyond keeping the implementation isolated enough to replace later.
- Reworking animation artwork.

## Architecture

The Go backend remains the single source of truth. The frontend only renders the pet and applies scale updates emitted by Go.

Main units:

- `Config`: stores position, visibility, scale, step size, and walking interval.
- `Movement`: owns automatic walking state and uses configurable step and interval values.
- `App`: coordinates config loading, window movement, window visibility, tray callbacks, and persistence.
- Windows tray adapter: owns tray icon/menu creation and forwards menu commands to `App`.
- Frontend: listens for a settings event and applies the requested visual size.

Wails `menu.Menu` and `menu.MenuItem` are still useful as the shape for menu state, but the current public `options.App` API is for application menus. If Wails exposes a usable tray registration path in this project, the tray adapter can use it. Otherwise, the adapter should be implemented with Windows tray APIs behind a narrow interface.

## Menu Structure

```text
Daiai Pet
- Show / Hide
- Size
  - 80%
  - 100%
  - 120%
  - 150%
- Step Size
  - 10 px
  - 20 px
  - 30 px
  - 40 px
- Walk Interval
  - Fast 100 ms
  - Default 150 ms
  - Slow 300 ms
  - Idle 600 ms
- Quit
```

Only one value in each preset group may be active. Menu state should be refreshed after every successful setting change.

## Config

Extend the existing config file with these fields:

```json
{
  "position": { "x": 100, "y": 100 },
  "visible": true,
  "scalePercent": 100,
  "stepSize": 20,
  "walkIntervalMs": 150
}
```

Default values:

- `visible`: `true`
- `scalePercent`: `100`
- `stepSize`: `20`
- `walkIntervalMs`: `150`

Invalid or missing values must be normalized to defaults. Existing config files that only contain `position` must continue to load.

## Data Flow

Startup:

1. Load config.
2. Normalize config.
3. Compute current pet width and height from scale.
4. Clamp saved position to the virtual screen using the scaled size.
5. Create movement with scaled bounds and configured movement parameters.
6. Move and size the window.
7. Emit frontend settings.
8. Create tray menu.
9. Start movement only when `visible` is true.

Menu click:

1. Tray adapter invokes an `App` method.
2. `App` updates config under lock.
3. `App` applies the change to window or movement state.
4. `App` saves config.
5. `App` refreshes tray checked/radio state.
6. `App` emits frontend updates when visual size changes.

## Behavior

Hide:

- Stop automatic movement.
- Save current position.
- Set `visible=false`.
- Call `runtime.WindowHide`.
- Keep the tray icon active.

Show:

- Set `visible=true`.
- Clamp current position using current scaled size.
- Apply current window size.
- Call `runtime.WindowShow`.
- Move to the clamped position.
- Resume automatic movement.

Size change:

- Update `scalePercent`.
- Recompute pet width and height.
- Update window size.
- Clamp current position using the new dimensions.
- Update movement bounds.
- Emit frontend scale/size update.
- If hidden, save the setting but do not show the pet.

Step size change:

- Update `stepSize`.
- Update movement settings.
- The next automatic move uses the new step.

Walk interval change:

- Update `walkIntervalMs`.
- Update movement settings.
- The next scheduled walk uses the new interval.

Drag interaction:

- Existing drag behavior remains backend-controlled.
- If settings change during a drag, do not force movement to resume.
- On drag completion, save the final position and resume with current settings if visible.

Quit:

- Save config.
- Destroy or unregister the tray icon when applicable.
- Quit the Wails runtime.

## Window Bounds

All bounds must use the Windows virtual screen metrics already used by drag logic. Bounds must use the current scaled pet width and height, not hardcoded `petWidth` and `petHeight`.

This prevents the old class of bugs where frontend and backend disagree about screen or pet dimensions.

## Frontend Contract

Add one runtime event named `updatePetSettings` with:

```json
{
  "width": 100,
  "height": 153,
  "scalePercent": 100
}
```

The frontend applies these values to the pet element. It must not compute or persist position.

## Error Handling

- Config load failure falls back to defaults.
- Config save failure should be logged and should not crash the app.
- Tray creation failure should be logged. The app can still run, but hidden mode should not be allowed if there is no recovery path.
- Window operations should guard against a nil Wails context.
- Unsupported preset values from stale config should normalize to defaults.

## Verification

Automated checks:

- `go test ./...`
- `npm run build`

Manual checks:

- Tray icon appears after app startup.
- Right-click tray menu opens.
- Hide hides the pet and keeps tray available.
- Show restores the pet.
- Size presets update window size and sprite size.
- Size changes keep the pet inside virtual screen bounds.
- Step size changes visibly affect walking distance.
- Walk interval changes visibly affect movement cadence.
- Drag still follows cursor and resumes movement after release.
- Restart preserves the last selected settings.

## Implementation Notes

Prefer small, replaceable boundaries:

- Keep tray-specific Windows details outside `Movement`.
- Keep frontend changes limited to render scale.
- Keep config normalization in one function so old config files stay compatible.
- Avoid reintroducing frontend position synchronization.
