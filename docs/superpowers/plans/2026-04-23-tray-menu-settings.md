# Tray Menu Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Windows system tray menu that can hide/show the pet and adjust display scale, walking step size, and walking interval with persisted settings.

**Architecture:** Keep Go as the only source of truth for settings, movement, visibility, and bounds. Add focused settings helpers, make movement configurable, add a Windows tray adapter behind a narrow `App` interface, and keep the frontend limited to sprite scale rendering.

**Tech Stack:** Go 1.23, Wails v2.12, Windows Win32 APIs through `golang.org/x/sys/windows`, Vanilla JS/Vite frontend.

---

## File Structure

- Modify `D:\daidai\pet-desktop\app.go`: integrate settings, window sizing, visibility actions, movement updates, and tray callbacks.
- Create `D:\daidai\pet-desktop\settings.go`: defaults, preset validation, config normalization, scaled pet size calculation.
- Create `D:\daidai\pet-desktop\settings_test.go`: unit tests for config normalization and size calculation.
- Create `D:\daidai\pet-desktop\movement.go`: move `Movement` out of `app.go` and make movement parameters configurable.
- Create `D:\daidai\pet-desktop\movement_test.go`: unit tests for movement settings updates and bounds.
- Create `D:\daidai\pet-desktop\tray_windows.go`: Windows tray icon and menu adapter.
- Create `D:\daidai\pet-desktop\tray_other.go`: non-Windows no-op adapter so `go test ./...` stays portable.
- Modify `D:\daidai\pet-desktop\main.go`: call tray cleanup on shutdown/close only through `App`.
- Modify `D:\daidai\pet-desktop\frontend\src\main.js`: listen for `updatePetSettings` and resize the pet element.
- Modify `D:\daidai\pet-desktop\frontend\src\style.css`: make the pet element size controllable by CSS variables.
- Modify `D:\daidai\pet-desktop\frontend\wailsjs\go\main\App.js`: keep generated bindings aligned if exported Go methods change.
- Modify `D:\daidai\pet-desktop\frontend\wailsjs\go\main\App.d.ts`: keep generated TypeScript declarations aligned if exported Go methods change.

This workspace is not currently a git repository. Commit steps below are replaced with explicit verification snapshots.

---

### Task 1: Settings Model And Tests

**Files:**

- Create: `D:\daidai\pet-desktop\settings.go`
- Create: `D:\daidai\pet-desktop\settings_test.go`
- Modify: `D:\daidai\pet-desktop\app.go`

- [ ] **Step 1: Write failing settings tests**

Create `D:\daidai\pet-desktop\settings_test.go`:

```go
package main

import "testing"

func TestNormalizeConfigAddsDefaults(t *testing.T) {
	cfg := newDefaultConfig()
	cfg.Position = Position{X: 7, Y: 9}

	NormalizeConfig(cfg)

	if cfg.Position.X != 7 || cfg.Position.Y != 9 {
		t.Fatalf("position changed: %+v", cfg.Position)
	}
	if !cfg.Visible {
		t.Fatal("expected visible default true")
	}
	if cfg.ScalePercent != defaultScalePercent {
		t.Fatalf("scale = %d", cfg.ScalePercent)
	}
	if cfg.StepSize != defaultStepSize {
		t.Fatalf("step = %d", cfg.StepSize)
	}
	if cfg.WalkIntervalMs != defaultWalkIntervalMs {
		t.Fatalf("interval = %d", cfg.WalkIntervalMs)
	}
}

func TestNormalizeConfigRejectsInvalidPresetValues(t *testing.T) {
	cfg := newDefaultConfig()
	cfg.Position = Position{X: 1, Y: 2}
	cfg.ScalePercent = 99
	cfg.StepSize = 999
	cfg.WalkIntervalMs = 42

	NormalizeConfig(cfg)

	if cfg.ScalePercent != defaultScalePercent {
		t.Fatalf("scale = %d", cfg.ScalePercent)
	}
	if cfg.StepSize != defaultStepSize {
		t.Fatalf("step = %d", cfg.StepSize)
	}
	if cfg.WalkIntervalMs != defaultWalkIntervalMs {
		t.Fatalf("interval = %d", cfg.WalkIntervalMs)
	}
}

func TestPetSizeForScale(t *testing.T) {
	w, h := petSizeForScale(120)

	if w != 120 {
		t.Fatalf("width = %d", w)
	}
	if h != 184 {
		t.Fatalf("height = %d", h)
	}
}
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
go test ./...
```

Expected: fail with undefined `NormalizeConfig`, `defaultScalePercent`, `defaultStepSize`, `defaultWalkIntervalMs`, or `petSizeForScale`.

- [ ] **Step 3: Add settings helpers**

Create `D:\daidai\pet-desktop\settings.go`:

```go
package main

const (
	basePetWidth  = 100
	basePetHeight = 153
)

const (
	defaultVisible        = true
	defaultScalePercent   = 100
	defaultStepSize       = 20
	defaultWalkIntervalMs = 150
)

var (
	scalePresets        = []int{80, 100, 120, 150}
	stepSizePresets     = []int{10, 20, 30, 40}
	walkIntervalPresets = []int{100, 150, 300, 600}
)

func NormalizeConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	if !containsPreset(scalePresets, cfg.ScalePercent) {
		cfg.ScalePercent = defaultScalePercent
	}
	if !containsPreset(stepSizePresets, cfg.StepSize) {
		cfg.StepSize = defaultStepSize
	}
	if !containsPreset(walkIntervalPresets, cfg.WalkIntervalMs) {
		cfg.WalkIntervalMs = defaultWalkIntervalMs
	}
}

func newDefaultConfig() *Config {
	return &Config{
		Position:       Position{X: 100, Y: 100},
		Visible:        defaultVisible,
		ScalePercent:   defaultScalePercent,
		StepSize:       defaultStepSize,
		WalkIntervalMs: defaultWalkIntervalMs,
	}
}

func containsPreset(values []int, value int) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func petSizeForScale(scalePercent int) (int, int) {
	return basePetWidth * scalePercent / 100, basePetHeight * scalePercent / 100
}
```

- [ ] **Step 4: Extend `Config` and load defaults**

Modify `D:\daidai\pet-desktop\app.go`:

```go
type Config struct {
	Position       Position `json:"position"`
	Visible        bool     `json:"visible"`
	ScalePercent   int      `json:"scalePercent"`
	StepSize       int      `json:"stepSize"`
	WalkIntervalMs int      `json:"walkIntervalMs"`
}
```

Replace `loadConfig` default creation:

```go
func (a *App) loadConfig() *Config {
	cfg := newDefaultConfig()
	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}
	a.cfgPath = filepath.Join(home, ".pet-desktop", "config.json")
	data, err := os.ReadFile(a.cfgPath)
	if err != nil {
		return cfg
	}
	if len(data) > 0 {
		_ = json.Unmarshal(data, cfg)
	}
	NormalizeConfig(cfg)
	return cfg
}
```

Replace `setCurrentPosition` nil initialization:

```go
if a.cfg == nil {
	a.cfg = newDefaultConfig()
}
```

- [ ] **Step 5: Run tests and verify pass**

Run:

```powershell
go test ./...
```

Expected: pass.

- [ ] **Step 6: Snapshot changed files**

Run:

```powershell
Get-ChildItem settings.go,settings_test.go,app.go | Select-Object Name,Length,LastWriteTime
```

Expected: all three files exist or are updated. No commit is made because the workspace is not a git repository.

---

### Task 2: Configurable Movement

**Files:**

- Create: `D:\daidai\pet-desktop\movement.go`
- Create: `D:\daidai\pet-desktop\movement_test.go`
- Modify: `D:\daidai\pet-desktop\app.go`

- [ ] **Step 1: Write movement tests**

Create `D:\daidai\pet-desktop\movement_test.go`:

```go
package main

import (
	"testing"
	"time"
)

func TestMovementUsesConfigurableStepSize(t *testing.T) {
	m := NewMovement(50, 20, 0, 200, 30, 150*time.Millisecond)
	m.direction = Right
	m.running = true

	m.doMove()

	if m.x != 80 {
		t.Fatalf("x = %d", m.x)
	}
	m.Stop()
}

func TestMovementUpdateSettings(t *testing.T) {
	m := NewMovement(50, 20, 0, 200, 20, 150*time.Millisecond)

	m.UpdateSettings(10, 300*time.Millisecond, 5, 95)

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stepSize != 10 {
		t.Fatalf("step = %d", m.stepSize)
	}
	if m.moveInterval != 300*time.Millisecond {
		t.Fatalf("interval = %s", m.moveInterval)
	}
	if m.minX != 5 || m.maxX != 95 {
		t.Fatalf("bounds = %d..%d", m.minX, m.maxX)
	}
}
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
go test ./...
```

Expected: fail because `NewMovement` does not accept configurable step/interval and `UpdateSettings` does not exist.

- [ ] **Step 3: Move and replace movement implementation**

Create `D:\daidai\pet-desktop\movement.go` with the full `Movement` type and methods. Remove the old `Movement` type and its methods from `app.go`.

```go
package main

import (
	"sync"
	"time"
)

const standDuration = 300 * time.Millisecond

type Movement struct {
	mu           sync.Mutex
	x, y         int
	direction    Direction
	minX         int
	maxX         int
	stepSize     int
	moveInterval time.Duration
	moveTimer    *time.Timer
	onUpdate     func(x, y int, direction Direction, walking bool)
	running      bool
}

func NewMovement(x, y, minX, maxX, stepSize int, moveInterval time.Duration) *Movement {
	m := &Movement{
		x:            x,
		y:            y,
		direction:    Right,
		minX:         minX,
		maxX:         maxX,
		stepSize:     stepSize,
		moveInterval: moveInterval,
	}
	if randIntn(2) == 0 {
		m.direction = Left
	}
	return m
}

var randIntn = func(n int) int {
	return randSource.Intn(n)
}

func (m *Movement) SetCallback(cb func(x, y int, direction Direction, walking bool)) {
	m.onUpdate = cb
}

func (m *Movement) Start() {
	m.mu.Lock()
	m.running = true
	if m.moveTimer != nil {
		m.moveTimer.Stop()
		m.moveTimer = nil
	}
	m.mu.Unlock()

	m.doMove()
}

func (m *Movement) Resume() {
	m.mu.Lock()
	m.running = true
	if m.moveTimer != nil {
		m.moveTimer.Stop()
		m.moveTimer = nil
	}
	m.mu.Unlock()

	m.scheduleNextWalk()
}

func (m *Movement) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.running = false
	if m.moveTimer != nil {
		m.moveTimer.Stop()
		m.moveTimer = nil
	}
}

func (m *Movement) SetPosition(x, y int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.x = x
	m.y = y
}

func (m *Movement) UpdateSettings(stepSize int, moveInterval time.Duration, minX int, maxX int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stepSize = stepSize
	m.moveInterval = moveInterval
	m.minX = minX
	m.maxX = maxX
	if m.x < m.minX {
		m.x = m.minX
	} else if m.x > m.maxX {
		m.x = m.maxX
	}
}

func (m *Movement) doMove() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}

	if m.direction == Right {
		m.x += m.stepSize
	} else {
		m.x -= m.stepSize
	}

	bounced := false
	if m.x < m.minX {
		m.x = m.minX
		m.direction = Right
		bounced = true
	} else if m.x > m.maxX {
		m.x = m.maxX
		m.direction = Left
		bounced = true
	}

	m.notifyUpdateLocked(true)
	m.mu.Unlock()

	if bounced {
		m.scheduleBounce()
		return
	}

	m.scheduleStand()
}

func (m *Movement) scheduleStand() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}
	if m.moveTimer != nil {
		m.moveTimer.Stop()
	}
	m.moveTimer = time.AfterFunc(standDuration, m.doStand)
}

func (m *Movement) doStand() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.notifyUpdateLocked(false)
	m.mu.Unlock()

	m.scheduleNextWalk()
}

func (m *Movement) scheduleNextWalk() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}
	if m.moveTimer != nil {
		m.moveTimer.Stop()
	}
	m.moveTimer = time.AfterFunc(m.moveInterval, m.doMove)
}

func (m *Movement) scheduleBounce() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}
	if m.moveTimer != nil {
		m.moveTimer.Stop()
	}
	m.moveTimer = time.AfterFunc(standDuration, m.doStand)
}

func (m *Movement) notifyUpdateLocked(walking bool) {
	if m.onUpdate != nil {
		m.onUpdate(m.x, m.y, m.direction, walking)
	}
}
```

In `app.go`, add a package-level random source near the existing Win32 vars:

```go
var randSource = rand.New(rand.NewSource(time.Now().UnixNano()))
```

Remove `rand.Seed(time.Now().UnixNano())` from `NewApp`.

- [ ] **Step 4: Update movement construction**

In `startup`, compute current size and movement settings:

```go
petW, petH := a.petSize()
screenX, _, screenW, _ := getVirtualScreenBounds()
minX := screenX
maxX := screenX + screenW - petW
x, y := clampWindowPositionForSize(a.cfg.Position.X, a.cfg.Position.Y, petW, petH)
a.setCurrentPosition(x, y)

a.movement = NewMovement(
	x,
	y,
	minX,
	maxX,
	a.cfg.StepSize,
	time.Duration(a.cfg.WalkIntervalMs)*time.Millisecond,
)
```

- [ ] **Step 5: Run tests and verify pass**

Run:

```powershell
go test ./...
```

Expected: pass.

- [ ] **Step 6: Snapshot changed files**

Run:

```powershell
Get-ChildItem movement.go,movement_test.go,app.go | Select-Object Name,Length,LastWriteTime
```

Expected: movement code is now isolated in `movement.go`.

---

### Task 3: Scaled Window And Frontend Contract

**Files:**

- Modify: `D:\daidai\pet-desktop\app.go`
- Modify: `D:\daidai\pet-desktop\frontend\src\main.js`
- Modify: `D:\daidai\pet-desktop\frontend\src\style.css`

- [ ] **Step 1: Add backend size helpers**

In `app.go`, add:

```go
func (a *App) petSize() (int, int) {
	a.mu.RLock()
	scale := defaultScalePercent
	if a.cfg != nil {
		scale = a.cfg.ScalePercent
	}
	a.mu.RUnlock()

	return petSizeForScale(scale)
}

func clampWindowPositionForSize(x, y int, width, height int) (int, int) {
	screenX, screenY, screenW, screenH := getVirtualScreenBounds()
	maxX := screenX + screenW - width
	maxY := screenY + screenH - height

	if x < screenX {
		x = screenX
	} else if x > maxX {
		x = maxX
	}

	if y < screenY {
		y = screenY
	} else if y > maxY {
		y = maxY
	}

	return x, y
}
```

Modify `D:\daidai\pet-desktop\main.go` to use the base dimensions from `settings.go`:

```go
Width:            basePetWidth,
Height:           basePetHeight,
```

Remove the old `petWidth` and `petHeight` constants from `app.go`; all runtime bounds must use `a.petSize()` or `petSizeForScale(...)`.

Replace `clampWindowPosition` body:

```go
func (a *App) clampWindowPosition(x, y int) (int, int) {
	petW, petH := a.petSize()
	return clampWindowPositionForSize(x, y, petW, petH)
}
```

Update drag code to call `a.clampWindowPosition(...)` instead of `clampWindowPosition(...)`.

- [ ] **Step 2: Add window size and frontend event emitters**

In `app.go`, add:

```go
func (a *App) applyWindowSize() {
	if a.ctx == nil {
		return
	}
	w, h := a.petSize()
	runtime.WindowSetSize(a.ctx, w, h)
	a.emitPetSettings()
}

func (a *App) emitPetSettings() {
	if a.ctx == nil {
		return
	}
	w, h := a.petSize()
	scale := defaultScalePercent
	a.mu.RLock()
	if a.cfg != nil {
		scale = a.cfg.ScalePercent
	}
	a.mu.RUnlock()

	runtime.EventsEmit(a.ctx, "updatePetSettings", map[string]interface{}{
		"width":        w,
		"height":       h,
		"scalePercent": scale,
	})
}
```

Call `a.applyWindowSize()` during startup before `a.moveWindow(...)`.

- [ ] **Step 3: Update frontend CSS variables**

Modify `D:\daidai\pet-desktop\frontend\src\style.css` so the pet element is controlled by CSS custom properties:

```css
:root {
  --pet-width: 100px;
  --pet-height: 153px;
}

html,
body {
  margin: 0;
  padding: 0;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: transparent;
}

#app {
  width: var(--pet-width);
  height: var(--pet-height);
  overflow: hidden;
}

#pet {
  width: var(--pet-width);
  height: var(--pet-height);
  display: block;
  user-select: none;
  -webkit-user-drag: none;
  cursor: grab;
}

#pet.dragging {
  cursor: grabbing;
}
```

If the existing file contains additional required image-rendering rules, keep them and only replace fixed width/height values with the variables above.

- [ ] **Step 4: Update frontend event listener**

In `D:\daidai\pet-desktop\frontend\src\main.js`, add after runtime readiness:

```js
window.runtime.EventsOn('updatePetSettings', (data) => {
  if (!data || !Number.isFinite(data.width) || !Number.isFinite(data.height)) {
    return
  }

  document.documentElement.style.setProperty('--pet-width', `${data.width}px`)
  document.documentElement.style.setProperty('--pet-height', `${data.height}px`)
})
```

- [ ] **Step 5: Build frontend**

Run:

```powershell
npm run build
```

Expected: Vite build succeeds and `frontend/dist` updates.

- [ ] **Step 6: Run Go tests**

Run:

```powershell
go test ./...
```

Expected: pass.

---

### Task 4: Visibility And Runtime Setting Methods

**Files:**

- Modify: `D:\daidai\pet-desktop\app.go`
- Modify: `D:\daidai\pet-desktop\frontend\wailsjs\go\main\App.js`
- Modify: `D:\daidai\pet-desktop\frontend\wailsjs\go\main\App.d.ts`

- [ ] **Step 1: Add setting methods on `App`**

In `app.go`, add:

```go
func (a *App) SetVisible(visible bool) {
	a.mu.Lock()
	if a.cfg == nil {
		a.cfg = newDefaultConfig()
	}
	a.cfg.Visible = visible
	a.mu.Unlock()

	if visible {
		a.showPet()
	} else {
		a.hidePet()
	}
	a.saveConfig()
	a.refreshTrayMenu()
}

func (a *App) SetScalePercent(scalePercent int) {
	if !containsPreset(scalePresets, scalePercent) {
		return
	}

	a.mu.Lock()
	if a.cfg == nil {
		a.cfg = newDefaultConfig()
	}
	a.cfg.ScalePercent = scalePercent
	a.mu.Unlock()

	a.applyWindowSize()
	x, y := a.currentPosition()
	x, y = a.clampWindowPosition(x, y)
	a.setCurrentPosition(x, y)
	if a.movement != nil {
		minX, maxX := a.movementBounds()
		a.movement.UpdateSettings(a.currentStepSize(), a.currentWalkInterval(), minX, maxX)
		a.movement.SetPosition(x, y)
	}
	a.moveWindow(x, y)
	a.saveConfig()
	a.refreshTrayMenu()
}

func (a *App) SetStepSize(stepSize int) {
	if !containsPreset(stepSizePresets, stepSize) {
		return
	}

	a.mu.Lock()
	if a.cfg == nil {
		a.cfg = newDefaultConfig()
	}
	a.cfg.StepSize = stepSize
	a.mu.Unlock()

	a.updateMovementSettings()
	a.saveConfig()
	a.refreshTrayMenu()
}

func (a *App) SetWalkIntervalMs(walkIntervalMs int) {
	if !containsPreset(walkIntervalPresets, walkIntervalMs) {
		return
	}

	a.mu.Lock()
	if a.cfg == nil {
		a.cfg = newDefaultConfig()
	}
	a.cfg.WalkIntervalMs = walkIntervalMs
	a.mu.Unlock()

	a.updateMovementSettings()
	a.saveConfig()
	a.refreshTrayMenu()
}
```

- [ ] **Step 2: Add helper methods used by setters**

In `app.go`, add:

```go
func (a *App) showPet() {
	if a.ctx == nil {
		return
	}
	a.applyWindowSize()
	x, y := a.currentPosition()
	x, y = a.clampWindowPosition(x, y)
	a.setCurrentPosition(x, y)
	runtime.WindowShow(a.ctx)
	a.moveWindow(x, y)
	if a.movement != nil && !a.isDragActive() {
		a.movement.Resume()
	}
}

func (a *App) hidePet() {
	if a.movement != nil {
		a.movement.Stop()
	}
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}
}

func (a *App) currentStepSize() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.cfg == nil {
		return defaultStepSize
	}
	return a.cfg.StepSize
}

func (a *App) currentScalePercent() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.cfg == nil {
		return defaultScalePercent
	}
	return a.cfg.ScalePercent
}

func (a *App) currentWalkInterval() time.Duration {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.cfg == nil {
		return time.Duration(defaultWalkIntervalMs) * time.Millisecond
	}
	return time.Duration(a.cfg.WalkIntervalMs) * time.Millisecond
}

func (a *App) currentVisible() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.cfg == nil {
		return defaultVisible
	}
	return a.cfg.Visible
}

func (a *App) movementBounds() (int, int) {
	petW, _ := a.petSize()
	screenX, _, screenW, _ := getVirtualScreenBounds()
	return screenX, screenX + screenW - petW
}

func (a *App) updateMovementSettings() {
	if a.movement == nil {
		return
	}
	minX, maxX := a.movementBounds()
	a.movement.UpdateSettings(a.currentStepSize(), a.currentWalkInterval(), minX, maxX)
}
```

- [ ] **Step 3: Respect visibility in startup and drag completion**

In startup, start movement only when visible:

```go
if a.cfg.Visible {
	go func() {
		time.Sleep(2 * time.Second)
		if a.currentVisible() && a.movement != nil {
			a.movement.Start()
		}
	}()
} else if a.ctx != nil {
	runtime.WindowHide(a.ctx)
}
```

In `completeDrag`, resume only when visible:

```go
if a.movement != nil && a.currentVisible() {
	a.movement.Resume()
}
```

- [ ] **Step 4: Update Wails generated stubs manually**

Modify `D:\daidai\pet-desktop\frontend\wailsjs\go\main\App.js`:

```js
// @ts-check
// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

export function SetDragEnd() {
  return window['go']['main']['App']['SetDragEnd']();
}

export function SetDragStart() {
  return window['go']['main']['App']['SetDragStart']();
}

export function SetScalePercent(arg1) {
  return window['go']['main']['App']['SetScalePercent'](arg1);
}

export function SetStepSize(arg1) {
  return window['go']['main']['App']['SetStepSize'](arg1);
}

export function SetVisible(arg1) {
  return window['go']['main']['App']['SetVisible'](arg1);
}

export function SetWalkIntervalMs(arg1) {
  return window['go']['main']['App']['SetWalkIntervalMs'](arg1);
}
```

Modify `D:\daidai\pet-desktop\frontend\wailsjs\go\main\App.d.ts`:

```ts
// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

export function SetDragEnd():Promise<void>;
export function SetDragStart():Promise<void>;
export function SetScalePercent(arg1:number):Promise<void>;
export function SetStepSize(arg1:number):Promise<void>;
export function SetVisible(arg1:boolean):Promise<void>;
export function SetWalkIntervalMs(arg1:number):Promise<void>;
```

- [ ] **Step 5: Run checks**

Run:

```powershell
go test ./...
npm run build
```

Expected: both pass.

---

### Task 5: Windows Tray Adapter

**Files:**

- Create: `D:\daidai\pet-desktop\tray_windows.go`
- Create: `D:\daidai\pet-desktop\tray_other.go`
- Modify: `D:\daidai\pet-desktop\app.go`

- [ ] **Step 1: Add tray fields to `App`**

In `app.go`, extend `App`:

```go
type App struct {
	ctx         context.Context
	cfg         *Config
	movement    *Movement
	tray        *trayController
	cfgPath     string
	mu          sync.RWMutex
	isDragging  bool
	dragStop    chan struct{}
	dragSeq     uint64
	dragOffsetX int
	dragOffsetY int
}
```

- [ ] **Step 2: Create non-Windows no-op tray**

Create `D:\daidai\pet-desktop\tray_other.go`:

```go
//go:build !windows

package main

type trayController struct{}

func newTrayController(app *App) (*trayController, error) {
	return &trayController{}, nil
}

func (t *trayController) Refresh() {}

func (t *trayController) Destroy() {}
```

- [ ] **Step 3: Create Windows tray adapter**

Create `D:\daidai\pet-desktop\tray_windows.go`:

```go
//go:build windows

package main

import (
	"log"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmAppTray       = 0x8000 + 1
	wmCommand       = 0x0111
	wmDestroy       = 0x0002
	wmRButtonUp     = 0x0205
	wmLButtonDblClk = 0x0203

	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	imageIcon       = 1
	lrShared        = 0x00008000
	idiApplication  = 32512
	mfString        = 0x00000000
	mfSeparator     = 0x00000800
	mfChecked       = 0x00000008
	mfUnchecked     = 0x00000000
	tpmRightButton  = 0x0002
)

const (
	trayIDToggleVisible = 1001
	trayIDScale80       = 1101
	trayIDScale100      = 1102
	trayIDScale120      = 1103
	trayIDScale150      = 1104
	trayIDStep10        = 1201
	trayIDStep20        = 1202
	trayIDStep30        = 1203
	trayIDStep40        = 1204
	trayIDInterval100   = 1301
	trayIDInterval150   = 1302
	trayIDInterval300   = 1303
	trayIDInterval600   = 1304
	trayIDQuit          = 1401
)

var (
	trayUser32                  = windows.NewLazySystemDLL("user32.dll")
	trayShell32                 = windows.NewLazySystemDLL("shell32.dll")
	procRegisterClassExW        = trayUser32.NewProc("RegisterClassExW")
	procCreateWindowExW         = trayUser32.NewProc("CreateWindowExW")
	procDefWindowProcW          = trayUser32.NewProc("DefWindowProcW")
	procDestroyWindow           = trayUser32.NewProc("DestroyWindow")
	procCreatePopupMenu         = trayUser32.NewProc("CreatePopupMenu")
	procAppendMenuW             = trayUser32.NewProc("AppendMenuW")
	procTrackPopupMenu          = trayUser32.NewProc("TrackPopupMenu")
	procDestroyMenu             = trayUser32.NewProc("DestroyMenu")
	procGetCursorPosTray        = trayUser32.NewProc("GetCursorPos")
	procPostQuitMessage         = trayUser32.NewProc("PostQuitMessage")
	procGetMessageW             = trayUser32.NewProc("GetMessageW")
	procTranslateMessage        = trayUser32.NewProc("TranslateMessage")
	procDispatchMessageW        = trayUser32.NewProc("DispatchMessageW")
	procLoadIconW               = trayUser32.NewProc("LoadIconW")
	procSetForegroundWindow     = trayUser32.NewProc("SetForegroundWindow")
	procShellNotifyIconW        = trayShell32.NewProc("Shell_NotifyIconW")
	trayWindowClassName, _      = windows.UTF16PtrFromString("DaidaiPetTrayWindow")
	trayControllersByHWND       = map[uintptr]*trayController{}
	trayControllersByHWNDLocker = make(chan struct{}, 1)
)

type trayController struct {
	app  *App
	hwnd uintptr
	done chan struct{}
}

type trayPoint struct {
	X int32
	Y int32
}

type trayMsg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      trayPoint
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type notifyIconData struct {
	Size             uint32
	HWnd             uintptr
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             uintptr
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GuidItem         windows.GUID
	BalloonIcon      uintptr
}

func newTrayController(app *App) (*trayController, error) {
	t := &trayController{app: app, done: make(chan struct{})}
	ready := make(chan error, 1)
	go t.run(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return t, nil
}

func (t *trayController) run(ready chan<- error) {
	runtime.LockOSThread()
	defer close(t.done)

	if err := registerTrayWindowClass(); err != nil {
		ready <- err
		return
	}

	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(trayWindowClassName)),
		uintptr(unsafe.Pointer(trayWindowClassName)),
		0,
		0, 0, 0, 0,
		0, 0, 0, 0,
	)
	if hwnd == 0 {
		ready <- err
		return
	}
	t.hwnd = hwnd
	withTrayMap(func() {
		trayControllersByHWND[hwnd] = t
	})

	if !t.addIcon() {
		ready <- syscall.EINVAL
		return
	}
	ready <- nil

	var msg trayMsg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	t.deleteIcon()
	withTrayMap(func() {
		delete(trayControllersByHWND, hwnd)
	})
}

func registerTrayWindowClass() error {
	icon, _, _ := procLoadIconW.Call(0, uintptr(idiApplication))
	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   syscall.NewCallback(trayWndProc),
		Icon:      icon,
		IconSm:    icon,
		ClassName: trayWindowClassName,
	}
	ret, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 && err != syscall.ERROR_CLASS_ALREADY_EXISTS {
		return err
	}
	return nil
}

func trayWndProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	var t *trayController
	withTrayMap(func() {
		t = trayControllersByHWND[hwnd]
	})

	switch msg {
	case wmAppTray:
		if t != nil && (lParam == wmRButtonUp || lParam == wmLButtonDblClk) {
			t.showMenu()
			return 0
		}
	case wmCommand:
		if t != nil {
			t.handleCommand(int(wParam & 0xffff))
			return 0
		}
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func withTrayMap(fn func()) {
	trayControllersByHWNDLocker <- struct{}{}
	defer func() { <-trayControllersByHWNDLocker }()
	fn()
}

func (t *trayController) addIcon() bool {
	icon, _, _ := procLoadIconW.Call(0, uintptr(idiApplication))
	nid := t.notifyIconData(icon)
	ret, _, _ := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	return ret != 0
}

func (t *trayController) deleteIcon() {
	nid := t.notifyIconData(0)
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func (t *trayController) notifyIconData(icon uintptr) notifyIconData {
	nid := notifyIconData{
		Size:            uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:            t.hwnd,
		ID:              1,
		Flags:           nifMessage | nifIcon | nifTip,
		CallbackMessage: wmAppTray,
		Icon:            icon,
	}
	copy(nid.Tip[:], windows.StringToUTF16("Daidai Pet"))
	return nid
}

func (t *trayController) showMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	visible := t.app.currentVisible()
	scale := t.app.currentScalePercent()
	step := t.app.currentStepSize()
	interval := int(t.app.currentWalkInterval() / time.Millisecond)

	appendMenuItem(menu, trayIDToggleVisible, labelForVisible(visible), false)
	appendSeparator(menu)
	appendMenuItem(menu, trayIDScale80, "Size 80%", scale == 80)
	appendMenuItem(menu, trayIDScale100, "Size 100%", scale == 100)
	appendMenuItem(menu, trayIDScale120, "Size 120%", scale == 120)
	appendMenuItem(menu, trayIDScale150, "Size 150%", scale == 150)
	appendSeparator(menu)
	appendMenuItem(menu, trayIDStep10, "Step 10 px", step == 10)
	appendMenuItem(menu, trayIDStep20, "Step 20 px", step == 20)
	appendMenuItem(menu, trayIDStep30, "Step 30 px", step == 30)
	appendMenuItem(menu, trayIDStep40, "Step 40 px", step == 40)
	appendSeparator(menu)
	appendMenuItem(menu, trayIDInterval100, "Fast 100 ms", interval == 100)
	appendMenuItem(menu, trayIDInterval150, "Default 150 ms", interval == 150)
	appendMenuItem(menu, trayIDInterval300, "Slow 300 ms", interval == 300)
	appendMenuItem(menu, trayIDInterval600, "Idle 600 ms", interval == 600)
	appendSeparator(menu)
	appendMenuItem(menu, trayIDQuit, "Quit", false)

	var point trayPoint
	procGetCursorPosTray.Call(uintptr(unsafe.Pointer(&point)))
	procSetForegroundWindow.Call(t.hwnd)
	procTrackPopupMenu.Call(menu, tpmRightButton, uintptr(point.X), uintptr(point.Y), 0, t.hwnd, 0)
}

func appendMenuItem(menu uintptr, id int, label string, checked bool) {
	flags := uintptr(mfString | mfUnchecked)
	if checked {
		flags = mfString | mfChecked
	}
	text, _ := windows.UTF16PtrFromString(label)
	procAppendMenuW.Call(menu, flags, uintptr(id), uintptr(unsafe.Pointer(text)))
}

func appendSeparator(menu uintptr) {
	procAppendMenuW.Call(menu, mfSeparator, 0, 0)
}

func labelForVisible(visible bool) string {
	if visible {
		return "Hide"
	}
	return "Show"
}

func (t *trayController) handleCommand(id int) {
	switch id {
	case trayIDToggleVisible:
		t.app.SetVisible(!t.app.currentVisible())
	case trayIDScale80:
		t.app.SetScalePercent(80)
	case trayIDScale100:
		t.app.SetScalePercent(100)
	case trayIDScale120:
		t.app.SetScalePercent(120)
	case trayIDScale150:
		t.app.SetScalePercent(150)
	case trayIDStep10:
		t.app.SetStepSize(10)
	case trayIDStep20:
		t.app.SetStepSize(20)
	case trayIDStep30:
		t.app.SetStepSize(30)
	case trayIDStep40:
		t.app.SetStepSize(40)
	case trayIDInterval100:
		t.app.SetWalkIntervalMs(100)
	case trayIDInterval150:
		t.app.SetWalkIntervalMs(150)
	case trayIDInterval300:
		t.app.SetWalkIntervalMs(300)
	case trayIDInterval600:
		t.app.SetWalkIntervalMs(600)
	case trayIDQuit:
		t.app.Quit()
	}
}

func (t *trayController) Refresh() {}

func (t *trayController) Destroy() {
	if t == nil || t.hwnd == 0 {
		return
	}
	procDestroyWindow.Call(t.hwnd)
	<-t.done
}
```

- [ ] **Step 4: Wire tray lifecycle in `App`**

In `startup`, after initial window setup:

```go
tray, err := newTrayController(a)
if err != nil {
	log.Printf("startup: failed to create tray menu: %v", err)
} else {
	a.tray = tray
}
```

Add:

```go
func (a *App) refreshTrayMenu() {
	if a.tray != nil {
		a.tray.Refresh()
	}
}

func (a *App) Quit() {
	a.saveConfig()
	if a.tray != nil {
		a.tray.Destroy()
		a.tray = nil
	}
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}
```

Update `onBeforeClose`:

```go
func (a *App) onBeforeClose(ctx context.Context) bool {
	a.saveConfig()
	if a.tray != nil {
		a.tray.Destroy()
		a.tray = nil
	}
	return false
}
```

- [ ] **Step 5: Run Go tests**

Run:

```powershell
go test ./...
```

Expected: pass on Windows.

---

### Task 6: Final Integration And Verification

**Files:**

- Modify as needed: `D:\daidai\pet-desktop\app.go`
- Modify as needed: `D:\daidai\pet-desktop\tray_windows.go`
- Modify as needed: `D:\daidai\pet-desktop\frontend\src\main.js`
- Modify as needed: `D:\daidai\pet-desktop\frontend\src\style.css`

- [ ] **Step 1: Format Go code**

Run:

```powershell
gofmt -w app.go settings.go settings_test.go movement.go movement_test.go tray_windows.go tray_other.go
```

Expected: no output.

- [ ] **Step 2: Run Go tests**

Run:

```powershell
go test ./...
```

Expected: pass.

- [ ] **Step 3: Build frontend**

Run:

```powershell
npm run build
```

Expected: Vite build succeeds.

- [ ] **Step 4: Search for stale hardcoded dimensions and old movement constants**

Run:

```powershell
rg -n "petWidth|petHeight|stepSize\\s*=|moveInterval\\s*=|syncPetPosition|GetScreenSize" app.go movement.go settings.go frontend/src frontend/wailsjs/go/main
```

Expected: only intentional references remain. `petWidth` and `petHeight` should not remain as app-wide constants; base dimensions should be `basePetWidth` and `basePetHeight`.

- [ ] **Step 5: Manual tray verification**

Run the app normally and verify:

```text
1. Tray icon appears in the Windows notification area.
2. Right-click opens the menu.
3. Hide hides the pet.
4. Show restores the pet from the tray.
5. Size 80/100/120/150 changes the visible pet and window bounds.
6. Step 10/20/30/40 changes horizontal movement distance.
7. Interval 100/150/300/600 changes walking cadence.
8. Drag remains cursor-synchronous and resumes movement after release.
9. Restart preserves selected settings.
```

- [ ] **Step 6: Final snapshot**

Run:

```powershell
Get-ChildItem app.go,settings.go,movement.go,tray_windows.go,tray_other.go,frontend\\src\\main.js,frontend\\src\\style.css | Select-Object Name,Length,LastWriteTime
```

Expected: all implementation files are present and recently updated.

---

## Plan Self-Review

Spec coverage:

- Tray menu: Task 5.
- Hide/show: Task 4 and Task 5.
- Scale presets: Task 1, Task 3, Task 4, Task 5.
- Step presets: Task 1, Task 2, Task 4, Task 5.
- Walk interval presets: Task 1, Task 2, Task 4, Task 5.
- Persistence: Task 1 and Task 4.
- Runtime updates: Task 3 and Task 4.
- Frontend-only rendering: Task 3.
- Virtual screen bounds using scaled size: Task 3 and Task 4.
- Verification: Task 6.

Placeholder scan:

- No placeholder or deferred implementation notes are required for execution.

Type consistency:

- Config fields are `Visible`, `ScalePercent`, `StepSize`, and `WalkIntervalMs`.
- Frontend event is `updatePetSettings`.
- Public setters are `SetVisible`, `SetScalePercent`, `SetStepSize`, and `SetWalkIntervalMs`.
- Tray refresh method is intentionally no-op for the Win32 dynamic popup implementation because menu state is rebuilt on every open.
