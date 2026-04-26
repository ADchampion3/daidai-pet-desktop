# Drag Toggle Click-Through Design

## Goal

Add a Windows tray menu switch that enables or disables desktop pet dragging. When dragging is disabled, mouse clicks on the pet window must pass through to the window underneath.

## Scope

In scope:

- A tray menu item for toggling drag behavior.
- Persisted drag enabled state in the existing config file.
- Runtime updates without restarting the app.
- Windows click-through behavior for the pet window while drag is disabled.
- Frontend cursor and drag gesture state synchronized with the backend.

Out of scope:

- A separate settings window.
- Hotkeys.
- Cross-platform click-through support beyond keeping non-Windows builds harmless.
- Changing movement, animation, size, or artwork behavior.

## Behavior

Dragging is enabled by default to preserve the current experience.

When dragging is enabled:

- Left-clicking the pet starts the existing backend-controlled drag loop.
- The pet cursor communicates that it can be moved.
- The pet window receives mouse input normally.

When dragging is disabled:

- The tray menu shows an action to enable dragging again.
- The window uses Windows extended style `WS_EX_TRANSPARENT`, so mouse input passes through the transparent Wails window to the next window underneath.
- The frontend treats drag gestures as disabled and does not call `SetDragStart`.
- If the state changes during a drag, the app ends the drag cleanly before applying click-through.

## Architecture

The Go backend remains the source of truth for drag state.

Main changes:

- Extend `Config` with `DragEnabled bool`.
- Normalize missing config values so existing users get `DragEnabled: true`.
- Add `App.SetDragEnabled(enabled bool)` and `App.ToggleDragEnabled()` for tray callbacks.
- Guard `SetDragStart()` so disabled drag cannot start even if the frontend calls it.
- Add a small window-style adapter that applies or removes `WS_EX_TRANSPARENT` on Windows.
- Emit a frontend event after startup and after toggles so the UI can update cursor and event behavior.

The Windows-specific click-through code should stay behind a narrow function, for example `setWindowClickThrough(ctx, enabled) error`, with a no-op implementation for non-Windows builds if platform files are introduced.

## Tray Menu

The tray menu gets one drag item:

```text
Disable Drag
```

or, when disabled:

```text
Enable Drag
```

If the project already has a tray adapter at implementation time, this item should be added there. If the current codebase still has no tray implementation, the implementation should add the smallest Windows tray adapter needed for this toggle and keep it isolated from `App` behind callbacks.

## Data Flow

Startup:

1. Load config.
2. Normalize `DragEnabled` to `true` when absent.
3. Apply click-through based on `DragEnabled`.
4. Emit frontend drag state.
5. Create or update tray menu state.

Tray toggle:

1. Tray command calls `App.ToggleDragEnabled()`.
2. `App` updates config and saves it.
3. `App` stops any active drag.
4. `App` applies or removes click-through.
5. `App` emits frontend drag state.

Frontend pointer input:

1. On left mouse down or single touch start, check local `dragEnabled`.
2. If disabled, do nothing.
3. If enabled, call `SetDragStart()` as today.

## Error Handling

If applying the Windows style fails, log the error and leave the config value unchanged only if the user action cannot be applied. Startup should log failures and continue with the best available behavior.

If the frontend misses the initial event, drag still remains correct because `SetDragStart()` is guarded in Go.

## Testing

Unit tests should cover:

- Config normalization defaults `DragEnabled` to `true`.
- Toggling drag updates app state and config.
- `SetDragStart()` does not activate dragging when disabled.
- Completing or cancelling an active drag during disable does not panic.

Build verification should include:

- `go test ./...`
- Frontend build if Wails-generated frontend bindings change.

Manual Windows verification:

- With drag enabled, pet can be dragged.
- With drag disabled, clicking the pet area activates or clicks the window underneath.
- The tray label changes between enable and disable states.
- Restart preserves the selected drag state.
