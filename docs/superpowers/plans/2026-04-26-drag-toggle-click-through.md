# Drag Toggle Click-Through Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Windows tray menu switch that turns pet dragging on or off, with mouse click-through while dragging is off.

**Architecture:** Keep Go as the source of truth for drag state. Add persisted `dragEnabled` config, guard backend drag start, emit frontend drag-state events, and isolate Windows tray/click-through APIs in platform files.

**Tech Stack:** Go 1.23, Wails v2.12, Win32 APIs through `golang.org/x/sys/windows`, vanilla JS/Vite frontend.

---

### File Structure

- Modify `D:\daidai\pet-desktop\app.go`: config field, drag state methods, startup integration, frontend event emission, tray lifecycle.
- Create `D:\daidai\pet-desktop\app_test.go`: focused unit tests for config defaulting, drag toggling, and drag start guard.
- Create `D:\daidai\pet-desktop\window_windows.go`: Windows click-through style helper.
- Create `D:\daidai\pet-desktop\window_other.go`: non-Windows no-op helper.
- Create `D:\daidai\pet-desktop\tray_windows.go`: minimal Windows notification-area icon and popup menu.
- Create `D:\daidai\pet-desktop\tray_other.go`: non-Windows no-op tray helper.
- Modify `D:\daidai\pet-desktop\frontend\src\main.js`: local drag-enabled flag and backend event listener.
- Modify `D:\daidai\pet-desktop\frontend\src\style.css`: disabled cursor state.
- Modify generated bindings only if the Wails generator updates them during verification.

### Task 1: Config and Backend Drag State

- [x] **Step 1: Write failing tests**

Create `D:\daidai\pet-desktop\app_test.go` with tests for:

```go
package main

import "testing"

func TestNewDefaultConfigEnablesDrag(t *testing.T) {
	cfg := newDefaultConfig()
	if !cfg.DragEnabled {
		t.Fatal("expected drag to be enabled by default")
	}
}

func TestSetDragEnabledUpdatesConfig(t *testing.T) {
	app := NewApp()
	app.cfg = newDefaultConfig()

	app.SetDragEnabled(false)
	if app.dragEnabled() {
		t.Fatal("expected drag to be disabled")
	}

	app.SetDragEnabled(true)
	if !app.dragEnabled() {
		t.Fatal("expected drag to be enabled")
	}
}

func TestSetDragStartIgnoredWhenDragDisabled(t *testing.T) {
	app := NewApp()
	app.cfg = newDefaultConfig()
	app.SetDragEnabled(false)

	app.SetDragStart()

	if app.isDragActive() {
		t.Fatal("expected drag start to be ignored when drag is disabled")
	}
}
```

- [x] **Step 2: Verify red**

Run `go test -count=1 ./...`.

Expected: FAIL because `newDefaultConfig`, `Config.DragEnabled`, `SetDragEnabled`, and `dragEnabled` do not exist.

- [x] **Step 3: Implement minimal backend state**

In `app.go`, add `DragEnabled bool` to `Config`, add `newDefaultConfig()`, default loading through it, add `dragEnabled()`, `SetDragEnabled(bool)`, `ToggleDragEnabled()`, and guard `SetDragStart()`.

- [x] **Step 4: Verify green**

Run `go test -count=1 ./...`.

Expected: PASS.

### Task 2: Window Click-Through and Drag Stop

- [x] **Step 1: Write failing test**

Extend `D:\daidai\pet-desktop\app_test.go` with:

```go
func TestDisablingDragStopsActiveDrag(t *testing.T) {
	app := NewApp()
	app.cfg = newDefaultConfig()
	app.mu.Lock()
	app.isDragging = true
	app.dragSeq = 7
	app.dragStop = make(chan struct{})
	app.mu.Unlock()

	app.SetDragEnabled(false)

	if app.isDragActive() {
		t.Fatal("expected active drag to stop when drag is disabled")
	}
}
```

- [x] **Step 2: Verify red**

Run `go test -count=1 ./...`.

Expected: FAIL because disabling drag does not yet cancel active drag.

- [x] **Step 3: Implement click-through hooks**

Add `setWindowClickThrough(ctx context.Context, enabled bool) error` in `window_windows.go` using `GetWindowLongPtrW`, `SetWindowLongPtrW`, and `WS_EX_TRANSPARENT`. Add `window_other.go` no-op. Update `SetDragEnabled(false)` to stop active drag before applying click-through.

- [x] **Step 4: Verify green**

Run `go test -count=1 ./...`.

Expected: PASS.

### Task 3: Tray Menu

- [x] **Step 1: Write tray integration smoke test**

Extend `D:\daidai\pet-desktop\app_test.go` with:

```go
func TestToggleDragEnabledFlipsState(t *testing.T) {
	app := NewApp()
	app.cfg = newDefaultConfig()

	app.ToggleDragEnabled()
	if app.dragEnabled() {
		t.Fatal("expected drag to be disabled after first toggle")
	}

	app.ToggleDragEnabled()
	if !app.dragEnabled() {
		t.Fatal("expected drag to be enabled after second toggle")
	}
}
```

- [x] **Step 2: Verify red**

Run `go test -count=1 ./...`.

Expected: FAIL until `ToggleDragEnabled` is complete.

- [x] **Step 3: Implement minimal Windows tray adapter**

Create `tray_windows.go` with a hidden window, `Shell_NotifyIconW`, and a popup menu containing `Disable Drag` or `Enable Drag`. Command handler calls `App.ToggleDragEnabled()`. Create `tray_other.go` no-op. In startup, call `newTrayMenu(a)` and in close, dispose it.

- [x] **Step 4: Verify green**

Run `go test -count=1 ./...`.

Expected: PASS.

### Task 4: Frontend State Sync

- [x] **Step 1: Update frontend**

In `frontend\src\main.js`, add `dragEnabled = true`, listen for `updateDragState`, skip pointer handlers when disabled, and update a `drag-disabled` CSS class.

In `frontend\src\style.css`, add cursor styling for disabled drag.

- [x] **Step 2: Verify frontend build**

Run `npm run build` from `D:\daidai\pet-desktop\frontend`.

Expected: PASS and produce `frontend\dist`.

### Task 5: Final Verification

- [x] **Step 1: Format**

Run `gofmt -w app.go app_test.go window_windows.go window_other.go tray_windows.go tray_other.go`.

- [x] **Step 2: Backend tests**

Run `go test -count=1 ./...`.

Expected: PASS.

- [x] **Step 3: Frontend build**

Run `npm run build` in `D:\daidai\pet-desktop\frontend`.

Expected: PASS.

- [x] **Step 4: Review diff**

Run `git diff -- app.go app_test.go window_windows.go window_other.go tray_windows.go tray_other.go frontend/src/main.js frontend/src/style.css`.

Expected: diff only covers drag toggle, click-through, tray, tests, and frontend sync.
