package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

const (
	dragInterval = 8 * time.Millisecond
)

const (
	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCXVirtualScreen = 78
	smCYVirtualScreen = 79
	vkLButton         = 0x01
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
)

type Config struct {
	Position       Position `json:"position"`
	Visible        bool     `json:"visible"`
	ScalePercent   int      `json:"scalePercent"`
	StepSize       int      `json:"stepSize"`
	WalkIntervalMs int      `json:"walkIntervalMs"`
	DragEnabled    bool     `json:"dragEnabled"`
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Direction bool

const (
	Left  Direction = false
	Right Direction = true
)

type App struct {
	ctx         context.Context
	cfg         *Config
	movement    *Movement
	cfgPath     string
	tray        *trayController
	mu          sync.RWMutex
	isDragging  bool
	dragStop    chan struct{}
	dragSeq     uint64
	dragOffsetX int
	dragOffsetY int
}

type winPoint struct {
	X int32
	Y int32
}

func NewApp() *App {
	rand.Seed(time.Now().UnixNano())
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cfg = a.loadConfig()

	width, height := a.petSize()
	screenX, _, screenW, _ := getVirtualScreenBounds()
	minX := screenX
	maxX := screenX + screenW - width
	x, y := clampWindowPositionForSize(a.cfg.Position.X, a.cfg.Position.Y, width, height)
	a.setCurrentPosition(x, y)

	log.Printf("Initializing movement at (%d, %d), screen x range: %d..%d", x, y, minX, maxX)
	a.movement = NewMovement(x, y, minX, maxX, a.cfg.StepSize, time.Duration(a.cfg.WalkIntervalMs)*time.Millisecond)
	a.movement.SetCallback(func(x, y int, direction Direction, walking bool) {
		if a.isDragActive() {
			return
		}

		a.setCurrentPosition(x, y)
		a.moveWindow(x, y)
		a.emitAnimationState(direction, walking)
	})

	a.applyWindowSize()
	a.moveWindow(x, y)
	if err := setWindowClickThrough(a.ctx, !a.currentDragEnabled()); err != nil {
		log.Printf("startup: failed to apply click-through state: %v", err)
	}
	a.emitDragState()

	if a.currentVisible() {
		a.resumeMovementIfVisible()
	} else if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}

	tray, err := newTrayController(a)
	if err != nil {
		log.Printf("tray startup failed: %v", err)
	} else {
		a.setTray(tray)
	}
}

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

func (a *App) saveConfig() {
	a.mu.RLock()
	if a.cfg == nil {
		a.mu.RUnlock()
		return
	}
	cfg := *a.cfg
	path := a.cfgPath
	a.mu.RUnlock()

	if path == "" {
		return
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}

	content, err := json.Marshal(cfg)
	if err != nil {
		return
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		log.Printf("saveConfig: write temp config failed: %v", err)
		return
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(path)
		if renameErr := os.Rename(tmpPath, path); renameErr != nil {
			log.Printf("saveConfig: replace config failed: %v", renameErr)
			_ = os.Remove(tmpPath)
		}
	}
}

func getCursorPosition() (int, int, bool) {
	var point winPoint
	ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	if ret == 0 {
		return 0, 0, false
	}
	return int(point.X), int(point.Y), true
}

func isLeftMouseButtonDown() bool {
	state, _, _ := procGetAsyncKeyState.Call(vkLButton)
	return state&0x8000 != 0
}

func getVirtualScreenBounds() (int, int, int, int) {
	x, _, _ := procGetSystemMetrics.Call(smXVirtualScreen)
	y, _, _ := procGetSystemMetrics.Call(smYVirtualScreen)
	w, _, _ := procGetSystemMetrics.Call(smCXVirtualScreen)
	h, _, _ := procGetSystemMetrics.Call(smCYVirtualScreen)
	return int(int32(x)), int(int32(y)), int(w), int(h)
}

func clampWindowPositionForSize(x, y int, width, height int) (int, int) {
	screenX, screenY, screenW, screenH := getVirtualScreenBounds()
	maxX := screenX + screenW - width
	maxY := screenY + screenH - height

	if maxX < screenX {
		maxX = screenX
	}
	if maxY < screenY {
		maxY = screenY
	}

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

func (a *App) petSize() (int, int) {
	return petSizeForScale(a.currentScalePercent())
}

func (a *App) clampWindowPosition(x, y int) (int, int) {
	width, height := a.petSize()
	return clampWindowPositionForSize(x, y, width, height)
}

func (a *App) SetDragStart() {
	if !a.currentDragEnabled() {
		return
	}

	cursorX, cursorY, ok := getCursorPosition()
	if !ok {
		log.Println("SetDragStart: cannot read cursor position")
		return
	}

	x, y := a.currentPosition()

	a.mu.Lock()
	a.isDragging = true
	a.dragSeq++
	a.dragOffsetX = cursorX - x
	a.dragOffsetY = cursorY - y
	stop := make(chan struct{})
	a.dragStop = stop
	seq := a.dragSeq
	a.mu.Unlock()

	if a.movement != nil {
		a.movement.Stop()
	}

	go a.runDragLoop(seq, stop)
}

func (a *App) SetDragEnd() {
	a.completeDragFromCursor()
}

func (a *App) runDragLoop(seq uint64, stop <-chan struct{}) {
	ticker := time.NewTicker(dragInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if !isLeftMouseButtonDown() {
				a.completeDragFromCursor()
				return
			}

			cursorX, cursorY, ok := getCursorPosition()
			if !ok {
				continue
			}

			a.mu.RLock()
			if !a.isDragging || a.dragSeq != seq {
				a.mu.RUnlock()
				return
			}
			offsetX := a.dragOffsetX
			offsetY := a.dragOffsetY
			a.mu.RUnlock()

			x, y := a.clampWindowPosition(cursorX-offsetX, cursorY-offsetY)
			a.setCurrentPosition(x, y)
			a.moveWindow(x, y)
		}
	}
}

func (a *App) completeDragFromCursor() {
	cursorX, cursorY, ok := getCursorPosition()
	if !ok {
		return
	}

	a.mu.RLock()
	if !a.isDragging {
		a.mu.RUnlock()
		return
	}
	seq := a.dragSeq
	offsetX := a.dragOffsetX
	offsetY := a.dragOffsetY
	a.mu.RUnlock()

	x, y := a.clampWindowPosition(cursorX-offsetX, cursorY-offsetY)
	a.completeDrag(seq, x, y)
}

func (a *App) completeDrag(seq uint64, x int, y int) {
	a.mu.Lock()
	if !a.isDragging || a.dragSeq != seq {
		a.mu.Unlock()
		return
	}
	stop := a.dragStop
	a.dragStop = nil
	a.isDragging = false
	a.dragSeq++
	a.mu.Unlock()

	if stop != nil {
		close(stop)
	}

	a.setCurrentPosition(x, y)
	if a.movement != nil {
		a.movement.SetPosition(x, y)
	}

	a.moveWindow(x, y)
	a.emitDragEnded()
	a.saveConfig()

	if a.currentVisible() {
		a.resumeMovementIfVisible()
	}
}

func (a *App) onBeforeClose(ctx context.Context) bool {
	a.saveConfig()
	if tray := a.takeTray(); tray != nil {
		tray.Destroy()
	}
	return false
}

func (a *App) currentPosition() (int, int) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.cfg == nil {
		return 0, 0
	}
	return a.cfg.Position.X, a.cfg.Position.Y
}

func (a *App) setCurrentPosition(x, y int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cfg == nil {
		a.cfg = newDefaultConfig()
	}
	a.cfg.Position.X = x
	a.cfg.Position.Y = y
}

func (a *App) isDragActive() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isDragging
}

func (a *App) moveWindow(x, y int) {
	if a.ctx == nil {
		log.Println("moveWindow: a.ctx is nil, cannot move window")
		return
	}

	runtime.WindowSetPosition(a.ctx, x, y)
}

func (a *App) applyWindowSize() {
	if a.ctx == nil {
		return
	}

	width, height := a.petSize()
	runtime.WindowSetSize(a.ctx, width, height)
	a.emitPetSettings()
}

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
	a.updateMovementSettings()
	if a.movement != nil {
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

func (a *App) showPet() {
	a.applyWindowSize()
	x, y := a.currentPosition()
	x, y = a.clampWindowPosition(x, y)
	a.setCurrentPosition(x, y)
	a.updateMovementSettings()
	if a.movement != nil {
		a.movement.SetPosition(x, y)
	}
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
	}
	a.moveWindow(x, y)
	a.resumeMovementIfVisible()
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

func (a *App) currentDragEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.cfg == nil {
		return defaultDragEnabled
	}
	return a.cfg.DragEnabled
}

func (a *App) SetDragEnabled(enabled bool) {
	if !enabled {
		a.stopActiveDrag()
	}

	a.mu.Lock()
	if a.cfg == nil {
		a.cfg = newDefaultConfig()
	}
	a.cfg.DragEnabled = enabled
	a.mu.Unlock()

	if err := setWindowClickThrough(a.ctx, !enabled); err != nil {
		log.Printf("SetDragEnabled: failed to update click-through state: %v", err)
	}
	a.saveConfig()
	a.emitDragState()
	a.refreshTrayMenu()
}

func (a *App) ToggleDragEnabled() {
	a.SetDragEnabled(!a.currentDragEnabled())
}

func (a *App) GetDragEnabled() bool {
	return a.currentDragEnabled()
}

func (a *App) stopActiveDrag() {
	a.mu.Lock()
	if !a.isDragging {
		a.mu.Unlock()
		return
	}
	stop := a.dragStop
	a.dragStop = nil
	a.isDragging = false
	a.dragSeq++
	a.mu.Unlock()

	if stop != nil {
		close(stop)
	}

	a.emitDragEnded()
	a.resumeMovementIfVisible()
}

func (a *App) movementBounds() (int, int) {
	petW, _ := a.petSize()
	screenX, _, screenW, _ := getVirtualScreenBounds()
	maxX := screenX + screenW - petW
	if maxX < screenX {
		maxX = screenX
	}
	return screenX, maxX
}

func (a *App) updateMovementSettings() {
	if a.movement == nil {
		return
	}

	minX, maxX := a.movementBounds()
	a.movement.UpdateSettings(a.currentStepSize(), a.currentWalkInterval(), minX, maxX)
}

func (a *App) resumeMovementIfVisible() {
	a.mu.RLock()
	movement := a.movement
	visible := a.cfg == nil || a.cfg.Visible
	dragging := a.isDragging
	a.mu.RUnlock()

	if movement == nil || !visible || dragging {
		return
	}

	movement.Resume()
}

func (a *App) refreshTrayMenu() {
	if tray := a.currentTray(); tray != nil {
		tray.Refresh()
	}
}

func (a *App) Quit() {
	a.saveConfig()
	if tray := a.takeTray(); tray != nil {
		tray.Destroy()
	}
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

func (a *App) currentTray() *trayController {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.tray
}

func (a *App) setTray(tray *trayController) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tray = tray
}

func (a *App) takeTray() *trayController {
	a.mu.Lock()
	defer a.mu.Unlock()
	tray := a.tray
	a.tray = nil
	return tray
}

func (a *App) emitDragEnded() {
	if a.ctx == nil {
		return
	}

	runtime.EventsEmit(a.ctx, "dragEnded")
}

func (a *App) emitDragState() {
	if a.ctx == nil {
		return
	}

	runtime.EventsEmit(a.ctx, "updateDragState", map[string]interface{}{
		"enabled": a.currentDragEnabled(),
	})
}

func (a *App) emitAnimationState(direction Direction, walking bool) {
	if a.ctx == nil {
		return
	}

	dirStr := "r"
	if direction == Left {
		dirStr = "l"
	}

	stateStr := "stand"
	if walking {
		stateStr = "walk"
	}

	runtime.EventsEmit(a.ctx, "updatePositionState", map[string]interface{}{
		"state": stateStr,
		"dir":   dirStr,
	})
}

func (a *App) GetPetSettings() map[string]int {
	width, height := a.petSize()
	a.mu.RLock()
	scalePercent := defaultScalePercent
	if a.cfg != nil {
		scalePercent = a.cfg.ScalePercent
	}
	a.mu.RUnlock()

	return map[string]int{
		"width":        width,
		"height":       height,
		"scalePercent": scalePercent,
	}
}

func (a *App) emitPetSettings() {
	if a.ctx == nil {
		return
	}

	runtime.EventsEmit(a.ctx, "updatePetSettings", a.GetPetSettings())
}
