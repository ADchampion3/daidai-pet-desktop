package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func withDisplayLayoutForTest(t *testing.T, layout displayLayout) {
	t.Helper()
	previous := displayLayoutProvider
	displayLayoutProvider = func() displayLayout {
		return layout
	}
	t.Cleanup(func() {
		displayLayoutProvider = previous
	})
}

func TestCurrentScalePercentUsesConfiguredValue(t *testing.T) {
	app := &App{cfg: newDefaultConfig()}
	app.cfg.ScalePercent = 120

	if got := app.currentScalePercent(); got != 120 {
		t.Fatalf("currentScalePercent() = %d", got)
	}
}

func TestCurrentScalePercentUsesDefaultWhenConfigMissing(t *testing.T) {
	app := &App{}

	if got := app.currentScalePercent(); got != defaultScalePercent {
		t.Fatalf("currentScalePercent() = %d", got)
	}
}

func TestCurrentWalkIntervalUsesConfiguredMilliseconds(t *testing.T) {
	app := &App{cfg: newDefaultConfig()}
	app.cfg.WalkIntervalMs = 300

	if got := app.currentWalkInterval(); got != 300*time.Millisecond {
		t.Fatalf("currentWalkInterval() = %s", got)
	}
}

func TestMovementBoundsUseScaledWidth(t *testing.T) {
	withDisplayLayoutForTest(t, displayLayout{
		{Left: -1280, Top: 0, Right: 0, Bottom: 1024},
		{Left: 0, Top: 0, Right: 1920, Bottom: 1080},
	})
	app := &App{cfg: newDefaultConfig()}
	app.cfg.ScalePercent = 150
	app.cfg.DisplayIndex = 1

	minX, maxX := app.movementBounds()
	expectedWidth, _ := petSizeForScale(150)

	if minX != 0 {
		t.Fatalf("minX = %d", minX)
	}
	if maxX != 1920-expectedWidth {
		t.Fatalf("maxX = %d", maxX)
	}
}

func TestClampWindowPositionAllowsNegativeMonitorCoordinates(t *testing.T) {
	withDisplayLayoutForTest(t, displayLayout{
		{Left: -1280, Top: 0, Right: 0, Bottom: 1024},
		{Left: 0, Top: 0, Right: 1920, Bottom: 1080},
	})

	x, y := clampWindowPositionForSize(-900, 100, 60, 92)

	if x != -900 || y != 100 {
		t.Fatalf("position = %d,%d", x, y)
	}
}

func TestClampWindowPositionAvoidsVirtualScreenGaps(t *testing.T) {
	withDisplayLayoutForTest(t, displayLayout{
		{Left: 0, Top: 0, Right: 1920, Bottom: 1080},
		{Left: 1920, Top: 300, Right: 3200, Bottom: 1324},
	})

	x, y := clampWindowPositionForSize(2100, 100, 60, 92)

	if x != 2100 || y != 300 {
		t.Fatalf("position = %d,%d", x, y)
	}
}

func TestClampWindowPositionKeepsWindowInsideNearestDisplayAfterScale(t *testing.T) {
	withDisplayLayoutForTest(t, displayLayout{
		{Left: 0, Top: 0, Right: 200, Bottom: 200},
		{Left: 300, Top: 0, Right: 500, Bottom: 200},
	})

	x, y := clampWindowPositionForSize(470, 170, 90, 138)

	if x != 410 || y != 62 {
		t.Fatalf("position = %d,%d", x, y)
	}
}

func TestMovementBoundsUseCurrentYVisibleInterval(t *testing.T) {
	withDisplayLayoutForTest(t, displayLayout{
		{Left: 0, Top: 0, Right: 1920, Bottom: 1080},
		{Left: 1920, Top: 300, Right: 3200, Bottom: 1324},
	})
	app := &App{cfg: newDefaultConfig()}

	app.cfg.Position = Position{X: 2000, Y: 100}
	minX, maxX := app.movementBounds()
	if minX != 0 || maxX != 1920-basePetWidth {
		t.Fatalf("top-row bounds = %d..%d", minX, maxX)
	}

	app.cfg.DisplayIndex = 1
	app.cfg.Position = Position{X: 2000, Y: 400}
	minX, maxX = app.movementBounds()
	if minX != 1920 || maxX != 3200-basePetWidth {
		t.Fatalf("selected-display bounds = %d..%d", minX, maxX)
	}
}

func TestSetDisplayIndexMovesPetToSelectedDisplay(t *testing.T) {
	withDisplayLayoutForTest(t, displayLayout{
		{Left: 0, Top: 0, Right: 1920, Bottom: 1080},
		{Left: 1920, Top: 300, Right: 3200, Bottom: 1324},
	})
	app := &App{cfg: newDefaultConfig()}
	app.cfg.Position = Position{X: 100, Y: 100}

	app.SetDisplayIndex(1)

	if app.cfg.DisplayIndex != 1 {
		t.Fatalf("display index = %d", app.cfg.DisplayIndex)
	}
	if app.cfg.Position.X < 1920 || app.cfg.Position.X+basePetWidth > 3200 {
		t.Fatalf("x not on selected display: %d", app.cfg.Position.X)
	}
	if app.cfg.Position.Y < 300 || app.cfg.Position.Y+basePetHeight > 1324 {
		t.Fatalf("y not on selected display: %d", app.cfg.Position.Y)
	}
}

func TestSetDisplayIndexRejectsMissingDisplay(t *testing.T) {
	withDisplayLayoutForTest(t, displayLayout{
		{Left: 0, Top: 0, Right: 1920, Bottom: 1080},
	})
	app := &App{cfg: newDefaultConfig()}
	app.cfg.DisplayIndex = 0

	app.SetDisplayIndex(4)

	if app.cfg.DisplayIndex != 0 {
		t.Fatalf("display index = %d", app.cfg.DisplayIndex)
	}
}

func TestResumeMovementIfVisibleResumesWhenVisibleAndNotDragging(t *testing.T) {
	app := &App{
		cfg:      newDefaultConfig(),
		movement: NewMovement(100, 100, 0, 600, defaultStepSize, time.Duration(defaultWalkIntervalMs)*time.Millisecond),
	}

	app.resumeMovementIfVisible()

	app.movement.mu.Lock()
	defer app.movement.mu.Unlock()

	if !app.movement.running {
		t.Fatal("movement.running = false")
	}
	if app.movement.moveTimer == nil {
		t.Fatal("movement.moveTimer = nil")
	}
}

func TestResumeMovementIfVisibleDoesNotResumeWhileDragging(t *testing.T) {
	app := &App{
		cfg:        newDefaultConfig(),
		movement:   NewMovement(100, 100, 0, 600, defaultStepSize, time.Duration(defaultWalkIntervalMs)*time.Millisecond),
		isDragging: true,
	}

	app.resumeMovementIfVisible()

	app.movement.mu.Lock()
	defer app.movement.mu.Unlock()

	if app.movement.running {
		t.Fatal("movement.running = true")
	}
	if app.movement.moveTimer != nil {
		t.Fatal("movement.moveTimer should be nil while dragging")
	}
}

func TestSetVisibleFalseStopsMovementWithoutContext(t *testing.T) {
	app := &App{
		cfg:      newDefaultConfig(),
		movement: NewMovement(100, 100, 0, 600, defaultStepSize, time.Duration(defaultWalkIntervalMs)*time.Millisecond),
	}
	app.movement.Resume()

	app.SetVisible(false)

	if app.currentVisible() {
		t.Fatal("currentVisible() = true")
	}

	app.movement.mu.Lock()
	running := app.movement.running
	moveTimer := app.movement.moveTimer
	app.movement.mu.Unlock()

	if running {
		t.Fatal("movement.running = true")
	}
	if moveTimer != nil {
		t.Fatal("movement.moveTimer should be nil after hide")
	}
}

func TestSetDragEnabledFalseStopsActiveDrag(t *testing.T) {
	app := &App{
		cfg:      newDefaultConfig(),
		movement: NewMovement(100, 100, 0, 600, defaultStepSize, time.Duration(defaultWalkIntervalMs)*time.Millisecond),
	}
	app.mu.Lock()
	app.isDragging = true
	app.dragStop = make(chan struct{})
	app.dragSeq = 3
	app.mu.Unlock()

	app.SetDragEnabled(false)

	if app.currentDragEnabled() {
		t.Fatal("currentDragEnabled() = true")
	}
	if app.isDragActive() {
		t.Fatal("isDragActive() = true")
	}
}

func TestToggleDragEnabledFlipsState(t *testing.T) {
	app := &App{cfg: newDefaultConfig()}

	app.ToggleDragEnabled()
	if app.currentDragEnabled() {
		t.Fatal("currentDragEnabled() = true after first toggle")
	}

	app.ToggleDragEnabled()
	if !app.currentDragEnabled() {
		t.Fatal("currentDragEnabled() = false after second toggle")
	}
}

func TestUpdateMovementSettingsUsesCurrentConfig(t *testing.T) {
	app := &App{
		cfg:      newDefaultConfig(),
		movement: NewMovement(10000, 100, 0, 50, defaultStepSize, time.Duration(defaultWalkIntervalMs)*time.Millisecond),
	}
	app.cfg.ScalePercent = 150
	app.cfg.StepSize = 30
	app.cfg.WalkIntervalMs = 600

	app.updateMovementSettings()

	expectedMinX, expectedMaxX := app.movementBounds()

	app.movement.mu.Lock()
	defer app.movement.mu.Unlock()

	if app.movement.stepSize != 30 {
		t.Fatalf("stepSize = %d", app.movement.stepSize)
	}
	if app.movement.moveInterval != 600*time.Millisecond {
		t.Fatalf("moveInterval = %s", app.movement.moveInterval)
	}
	if app.movement.minX != expectedMinX {
		t.Fatalf("minX = %d", app.movement.minX)
	}
	if app.movement.maxX != expectedMaxX {
		t.Fatalf("maxX = %d", app.movement.maxX)
	}
	if app.movement.x != expectedMaxX {
		t.Fatalf("x = %d", app.movement.x)
	}
}

func TestGetPetSettingsUsesConfiguredScale(t *testing.T) {
	app := &App{cfg: newDefaultConfig()}
	app.cfg.ScalePercent = 150

	settings := app.GetPetSettings()

	if settings["width"] != 90 {
		t.Fatalf("width = %d", settings["width"])
	}
	if settings["height"] != 138 {
		t.Fatalf("height = %d", settings["height"])
	}
	if settings["scalePercent"] != 150 {
		t.Fatalf("scalePercent = %d", settings["scalePercent"])
	}
}

func TestRefreshTrayMenuCallsTrayRefresh(t *testing.T) {
	refreshed := 0
	app := &App{
		tray: &trayController{
			refreshFunc: func() {
				refreshed++
			},
		},
	}

	app.refreshTrayMenu()

	if refreshed != 1 {
		t.Fatalf("refresh count = %d", refreshed)
	}
}

func TestOnBeforeCloseDestroysTray(t *testing.T) {
	destroyed := 0
	app := &App{
		cfg:     newDefaultConfig(),
		cfgPath: filepath.Join(t.TempDir(), "config.json"),
		tray: &trayController{
			destroyFunc: func() {
				destroyed++
			},
		},
	}

	if got := app.onBeforeClose(context.Background()); got {
		t.Fatal("onBeforeClose() = true")
	}

	if destroyed != 1 {
		t.Fatalf("destroy count = %d", destroyed)
	}
}

func TestQuitSavesConfigAndDestroysTray(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")
	destroyed := 0
	app := &App{
		cfg:     newDefaultConfig(),
		cfgPath: cfgPath,
		tray: &trayController{
			destroyFunc: func() {
				destroyed++
			},
		},
	}
	app.cfg.Visible = false

	app.Quit()

	if destroyed != 1 {
		t.Fatalf("destroy count = %d", destroyed)
	}
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("config not written: %v", err)
	}
}

func TestSaveConfigWritesNonEmptyJSON(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	app := &App{
		cfg:     newDefaultConfig(),
		cfgPath: cfgPath,
	}
	app.cfg.Position = Position{X: 4688, Y: 85}
	app.cfg.DragEnabled = false

	app.saveConfig()

	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("config file is empty")
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if cfg.Position.X != 4688 || cfg.Position.Y != 85 {
		t.Fatalf("position = %+v", cfg.Position)
	}
	if cfg.DragEnabled {
		t.Fatal("dragEnabled = true")
	}
}
