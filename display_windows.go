//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
	shcore                  = windows.NewLazySystemDLL("shcore.dll")
	procGetDpiForMonitor    = shcore.NewProc("GetDpiForMonitor")
)

const monitorDPITypeEffective = 0

type winRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type winMonitorInfo struct {
	Size    uint32
	Monitor winRect
	Work    winRect
	Flags   uint32
}

type displayLayoutContainer struct {
	displays displayLayout
}

func readDisplayLayout() displayLayout {
	container := &displayLayoutContainer{}
	procEnumDisplayMonitors.Call(0, 0, displayMonitorEnumProcPtr, uintptr(unsafe.Pointer(container)))
	return container.displays
}

var displayMonitorEnumProcPtr = syscall.NewCallback(displayMonitorEnumProc)

func displayMonitorEnumProc(hMonitor, hdcMonitor, rectPtr, data uintptr) uintptr {
	container := (*displayLayoutContainer)(unsafe.Pointer(data))
	dpiX, dpiY := getMonitorDPI(hMonitor)
	var info winMonitorInfo
	info.Size = uint32(unsafe.Sizeof(info))
	ret, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		if rectPtr == 0 {
			return 1
		}
		rect := (*winRect)(unsafe.Pointer(rectPtr))
		container.displays = append(container.displays, displayRect{
			Left:   int(rect.Left),
			Top:    int(rect.Top),
			Right:  int(rect.Right),
			Bottom: int(rect.Bottom),
			DPIX:   dpiX,
			DPIY:   dpiY,
		})
		return 1
	}

	container.displays = append(container.displays, displayRect{
		Left:   int(info.Monitor.Left),
		Top:    int(info.Monitor.Top),
		Right:  int(info.Monitor.Right),
		Bottom: int(info.Monitor.Bottom),
		DPIX:   dpiX,
		DPIY:   dpiY,
	})
	return 1
}

func getMonitorDPI(hMonitor uintptr) (uint, uint) {
	var dpiX, dpiY uint32
	ret, _, _ := procGetDpiForMonitor.Call(
		hMonitor,
		monitorDPITypeEffective,
		uintptr(unsafe.Pointer(&dpiX)),
		uintptr(unsafe.Pointer(&dpiY)),
	)
	if ret != 0 || dpiX == 0 || dpiY == 0 {
		return 96, 96
	}
	return uint(dpiX), uint(dpiY)
}
