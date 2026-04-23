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
	petWidth  = 100
	petHeight = 153
)

const (
	moveInterval  = 150 * time.Millisecond
	standDuration = 300 * time.Millisecond
	dragInterval  = 8 * time.Millisecond
	stepSize      = 20
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
	Position Position `json:"position"`
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

type Movement struct {
	mu        sync.Mutex
	x, y      int
	direction Direction
	minX      int
	maxX      int
	moveTimer *time.Timer
	onUpdate  func(x, y int, direction Direction, walking bool)
	running   bool
}

func NewMovement(x, y, minX, maxX int) *Movement {
	m := &Movement{
		x:         x,
		y:         y,
		direction: Right,
		minX:      minX,
		maxX:      maxX,
	}
	if rand.Intn(2) == 0 {
		m.direction = Left
	}
	return m
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

func (m *Movement) doMove() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}

	if m.direction == Right {
		m.x += stepSize
	} else {
		m.x -= stepSize
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
	m.moveTimer = time.AfterFunc(moveInterval, m.doMove)
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

func NewApp() *App {
	rand.Seed(time.Now().UnixNano())
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cfg = a.loadConfig()

	screenX, _, screenW, _ := getVirtualScreenBounds()
	minX := screenX
	maxX := screenX + screenW - petWidth

	log.Printf("Initializing movement at (%d, %d), screen x range: %d..%d", a.cfg.Position.X, a.cfg.Position.Y, minX, maxX)
	a.movement = NewMovement(a.cfg.Position.X, a.cfg.Position.Y, minX, maxX)
	a.movement.SetCallback(func(x, y int, direction Direction, walking bool) {
		if a.isDragActive() {
			return
		}

		a.setCurrentPosition(x, y)
		a.moveWindow(x, y)
		a.emitAnimationState(direction, walking)
	})

	a.setCurrentPosition(a.cfg.Position.X, a.cfg.Position.Y)
	a.moveWindow(a.cfg.Position.X, a.cfg.Position.Y)

	go func() {
		time.Sleep(2 * time.Second)
		a.movement.Start()
	}()
}

func (a *App) loadConfig() *Config {
	cfg := &Config{Position: Position{X: 100, Y: 100}}
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

	_ = os.WriteFile(path, content, 0644)
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

func clampWindowPosition(x, y int) (int, int) {
	screenX, screenY, screenW, screenH := getVirtualScreenBounds()
	maxX := screenX + screenW - petWidth
	maxY := screenY + screenH - petHeight

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

func (a *App) SetDragStart() {
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

			x, y := clampWindowPosition(cursorX-offsetX, cursorY-offsetY)
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

	x, y := clampWindowPosition(cursorX-offsetX, cursorY-offsetY)
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

	if a.movement != nil {
		a.movement.Resume()
	}
}

func (a *App) onBeforeClose(ctx context.Context) bool {
	a.saveConfig()
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
		a.cfg = &Config{}
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

func (a *App) emitDragEnded() {
	if a.ctx == nil {
		return
	}

	runtime.EventsEmit(a.ctx, "dragEnded")
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
