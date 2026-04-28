package main

import (
	"math/rand"
	"sync"
	"time"
)

const standDuration = 300 * time.Millisecond

const (
	minVerticalBounceOffset = 30
	maxVerticalBounceOffset = 100
)

type Movement struct {
	mu           sync.Mutex
	x, y         int
	direction    Direction
	minX         int
	maxX         int
	minY         int
	maxY         int
	stepSize     int
	moveInterval time.Duration
	bounds       movementBoundsProvider
	moveTimer    *time.Timer
	onUpdate     func(x, y int, direction Direction, walking bool)
	running      bool
}

func NewMovement(x, y, minX, maxX, minY, maxY, stepSize int, moveInterval time.Duration) *Movement {
	m := &Movement{
		x:            x,
		y:            y,
		direction:    Right,
		minX:         minX,
		maxX:         maxX,
		minY:         minY,
		maxY:         maxY,
		stepSize:     stepSize,
		moveInterval: moveInterval,
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
	m.refreshBoundsLocked()
}

func (m *Movement) UpdateSettings(stepSize int, moveInterval time.Duration, minX int, maxX int, minY int, maxY int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stepSize = stepSize
	m.moveInterval = moveInterval
	m.minX = minX
	m.maxX = maxX
	m.minY = minY
	m.maxY = maxY
	m.refreshBoundsLocked()

	m.clampPositionLocked()
}

func (m *Movement) SetBoundsProvider(bounds movementBoundsProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.bounds = bounds
	m.refreshBoundsLocked()
}

func (m *Movement) doMove() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.refreshBoundsLocked()

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
	if bounced {
		m.applyVerticalBounceLocked()
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

func (m *Movement) refreshBoundsLocked() {
	if m.bounds == nil {
		return
	}
	m.minX, m.maxX, m.minY, m.maxY = m.bounds(m.x, m.y)
	m.clampPositionLocked()
}

func (m *Movement) clampPositionLocked() {
	if m.x < m.minX {
		m.x = m.minX
	} else if m.x > m.maxX {
		m.x = m.maxX
	}
	if m.y < m.minY {
		m.y = m.minY
	} else if m.y > m.maxY {
		m.y = m.maxY
	}
}

func (m *Movement) applyVerticalBounceLocked() {
	delta := minVerticalBounceOffset + rand.Intn(maxVerticalBounceOffset-minVerticalBounceOffset+1)
	if rand.Intn(2) == 0 {
		delta = -delta
	}
	m.y += delta
	m.clampPositionLocked()
}
