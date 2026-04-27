package main

import (
	"sync"
	"time"
)

type displayRect struct {
	Left   int
	Top    int
	Right  int
	Bottom int
	DPIX   uint
	DPIY   uint
}

type displayLayout []displayRect

type movementBoundsProvider func(x, y int) (int, int, int, int)

var displayLayoutProvider = readDisplayLayout

var (
	layoutCacheMu    sync.RWMutex
	layoutCache      displayLayout
	layoutCacheStamp time.Time
)

const layoutCacheTTL = 2 * time.Second

func invalidateLayoutCache() {
	layoutCacheMu.Lock()
	layoutCache = nil
	layoutCacheStamp = time.Time{}
	layoutCacheMu.Unlock()
}

func getDisplayLayout() displayLayout {
	layoutCacheMu.RLock()
	if time.Since(layoutCacheStamp) < layoutCacheTTL && len(layoutCache) > 0 {
		cached := layoutCache
		layoutCacheMu.RUnlock()
		return cached
	}
	layoutCacheMu.RUnlock()

	fresh := displayLayoutProvider()
	if len(fresh) == 0 {
		x, y, w, h := getVirtualScreenBounds()
		fresh = displayLayout{{Left: x, Top: y, Right: x + w, Bottom: y + h}}
	}

	layoutCacheMu.Lock()
	layoutCache = fresh
	layoutCacheStamp = time.Now()
	layoutCacheMu.Unlock()

	return fresh
}

func (l displayLayout) bounds() displayRect {
	if len(l) == 0 {
		return displayRect{}
	}
	bounds := l[0]
	for _, display := range l[1:] {
		if display.Left < bounds.Left {
			bounds.Left = display.Left
		}
		if display.Top < bounds.Top {
			bounds.Top = display.Top
		}
		if display.Right > bounds.Right {
			bounds.Right = display.Right
		}
		if display.Bottom > bounds.Bottom {
			bounds.Bottom = display.Bottom
		}
	}
	return bounds
}

func (l displayLayout) clampWindowPosition(x, y, width, height int) (int, int) {
	if len(l) == 0 {
		return x, y
	}
	for _, display := range l {
		if display.containsWindow(x, y, width, height) {
			return x, y
		}
	}

	centerX := x + width/2
	centerY := y + height/2
	nearest := l[0]
	nearestDistance := nearest.distanceSquaredToPoint(centerX, centerY)
	for _, display := range l[1:] {
		if distance := display.distanceSquaredToPoint(centerX, centerY); distance < nearestDistance {
			nearest = display
			nearestDistance = distance
		}
	}
	return nearest.clampWindowPosition(x, y, width, height)
}

func (l displayLayout) clampWindowPositionToDisplay(index int, x, y, width, height int) (int, int) {
	if len(l) == 0 {
		return x, y
	}
	display, ok := l.display(index)
	if !ok {
		display = l[0]
	}
	return display.clampWindowPosition(x, y, width, height)
}

func (l displayLayout) movementBounds(x, y, width, height int) (int, int, int, int) {
	if len(l) == 0 {
		return x, x, y, y
	}

	var intervals []displayInterval
	for _, display := range l {
		if !display.canFitWindowAtY(y, height) {
			continue
		}
		maxX := display.Right - width
		if maxX < display.Left {
			maxX = display.Left
		}
		intervals = append(intervals, displayInterval{Min: display.Left, Max: maxX})
	}
	if len(intervals) == 0 {
		clampedX, clampedY := l.clampWindowPosition(x, y, width, height)
		return l.movementBounds(clampedX, clampedY, width, height)
	}

	intervals = mergeDisplayIntervals(intervals, width)
	chosen := intervals[0]
	bestDistance := chosen.distanceToX(x)
	for _, interval := range intervals[1:] {
		if distance := interval.distanceToX(x); distance < bestDistance {
			chosen = interval
			bestDistance = distance
		}
	}
	bounds := l.bounds()
	maxY := bounds.Bottom - height
	if maxY < bounds.Top {
		maxY = bounds.Top
	}
	return chosen.Min, chosen.Max, bounds.Top, maxY
}

func (l displayLayout) movementBoundsForDisplay(index int, x, y, width, height int) (int, int, int, int) {
	if len(l) == 0 {
		return x, x, y, y
	}
	display, ok := l.display(index)
	if !ok {
		display = l[0]
	}

	width, height = display.windowPhysicalSize(width, height)
	maxX := display.Right - width
	if maxX < display.Left {
		maxX = display.Left
	}
	maxY := display.Bottom - height
	if maxY < display.Top {
		maxY = display.Top
	}
	return display.Left, maxX, display.Top, maxY
}

func (l displayLayout) display(index int) (displayRect, bool) {
	if index < 0 || index >= len(l) {
		return displayRect{}, false
	}
	return l[index], true
}

func (l displayLayout) windowPhysicalSizeForDisplay(index int, width, height int) (int, int) {
	if len(l) == 0 {
		return width, height
	}
	display, ok := l.display(index)
	if !ok {
		display = l[0]
	}
	return display.windowPhysicalSize(width, height)
}

func (l displayLayout) windowPhysicalSize(x, y, width, height int) (int, int) {
	if len(l) == 0 {
		return width, height
	}
	display := l.displayForWindowPosition(x, y, width, height)
	return display.windowPhysicalSize(width, height)
}

func (l displayLayout) displayForWindowPosition(x, y, width, height int) displayRect {
	best := l[0]
	bestArea := best.intersectionArea(x, y, width, height)
	for _, display := range l[1:] {
		if area := display.intersectionArea(x, y, width, height); area > bestArea {
			best = display
			bestArea = area
		}
	}
	if bestArea > 0 {
		return best
	}

	centerX := x + width/2
	centerY := y + height/2
	bestDistance := best.distanceSquaredToPoint(centerX, centerY)
	for _, display := range l[1:] {
		if distance := display.distanceSquaredToPoint(centerX, centerY); distance < bestDistance {
			best = display
			bestDistance = distance
		}
	}
	return best
}

func (r displayRect) containsWindow(x, y, width, height int) bool {
	return x >= r.Left && y >= r.Top && x+width <= r.Right && y+height <= r.Bottom
}

func (r displayRect) windowPhysicalSize(width, height int) (int, int) {
	return scaleWithDPI(width, r.DPIX), scaleWithDPI(height, r.DPIY)
}

func (r displayRect) intersectionArea(x, y, width, height int) int {
	left := max(r.Left, x)
	top := max(r.Top, y)
	right := min(r.Right, x+width)
	bottom := min(r.Bottom, y+height)
	if right <= left || bottom <= top {
		return 0
	}
	return (right - left) * (bottom - top)
}

func (r displayRect) centerWindowPosition(width, height int) (int, int) {
	x := r.Left + (r.Right-r.Left-width)/2
	y := r.Top + (r.Bottom-r.Top-height)/2
	return r.clampWindowPosition(x, y, width, height)
}

func (r displayRect) canFitWindowAtY(y, height int) bool {
	return y >= r.Top && y+height <= r.Bottom
}

func (r displayRect) clampWindowPosition(x, y, width, height int) (int, int) {
	maxX := r.Right - width
	maxY := r.Bottom - height
	if maxX < r.Left {
		maxX = r.Left
	}
	if maxY < r.Top {
		maxY = r.Top
	}
	return clampInt(x, r.Left, maxX), clampInt(y, r.Top, maxY)
}

func (r displayRect) distanceSquaredToPoint(x, y int) int {
	dx := 0
	if x < r.Left {
		dx = r.Left - x
	} else if x > r.Right {
		dx = x - r.Right
	}

	dy := 0
	if y < r.Top {
		dy = r.Top - y
	} else if y > r.Bottom {
		dy = y - r.Bottom
	}
	return dx*dx + dy*dy
}

type displayInterval struct {
	Min int
	Max int
}

func (i displayInterval) distanceToX(x int) int {
	if x < i.Min {
		return i.Min - x
	}
	if x > i.Max {
		return x - i.Max
	}
	return 0
}

func mergeDisplayIntervals(intervals []displayInterval, windowWidth int) []displayInterval {
	for i := 1; i < len(intervals); i++ {
		current := intervals[i]
		j := i - 1
		for ; j >= 0 && intervals[j].Min > current.Min; j-- {
			intervals[j+1] = intervals[j]
		}
		intervals[j+1] = current
	}

	merged := intervals[:1]
	for _, interval := range intervals[1:] {
		last := &merged[len(merged)-1]
		if interval.Min <= last.Max+windowWidth {
			if interval.Max > last.Max {
				last.Max = interval.Max
			}
			continue
		}
		merged = append(merged, interval)
	}
	return merged
}

func scaleWithDPI(pixels int, dpi uint) int {
	if dpi == 0 {
		dpi = 96
	}
	return pixels * int(dpi) / 96
}

func clampInt(value, lo, hi int) int {
	return max(lo, min(hi, value))
}
