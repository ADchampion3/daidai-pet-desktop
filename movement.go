package main

import (
	"math/rand"
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
