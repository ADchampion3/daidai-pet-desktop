//go:build windows

package main

import (
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	trayCallbackMessage = 0x0400 + 1
	trayIconID          = 1
	trayIDToggleDrag    = 1001

	wmDestroy   = 0x0002
	wmRButtonUp = 0x0205

	nimAdd    = 0x00000000
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	idiApplication = 32512

	mfString       = 0x00000000
	tpmRightAlign  = 0x0008
	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100
)

var (
	trayShell32             = windows.NewLazySystemDLL("shell32.dll")
	trayKernel32            = windows.NewLazySystemDLL("kernel32.dll")
	procShellNotifyIconW    = trayShell32.NewProc("Shell_NotifyIconW")
	procGetModuleHandleW    = trayKernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procLoadIconW           = user32.NewProc("LoadIconW")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetCursorPosTray    = user32.NewProc("GetCursorPos")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	trayWindowClassName, _  = windows.UTF16PtrFromString("DaidaiPetTrayWindow")
	trayWindowTitle, _      = windows.UTF16PtrFromString("Daidai Pet Tray")
	trayInstances           = map[uintptr]*trayMenu{}
	trayInstancesMu         sync.Mutex
)

type trayMenu struct {
	app   *App
	hwnd  uintptr
	ready chan error
}

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     uintptr
}

type trayWndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type trayMsg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      winPoint
}

func newTrayMenu(app *App) (*trayMenu, error) {
	t := &trayMenu{app: app, ready: make(chan error, 1)}
	go t.run()
	if err := <-t.ready; err != nil {
		return nil, err
	}
	return t, nil
}

func (t *trayMenu) Dispose() {
	if t == nil || t.hwnd == 0 {
		return
	}
	t.deleteIcon()
	procDestroyWindow.Call(t.hwnd)
}

func trayDragLabel(enabled bool) string {
	if enabled {
		return "Disable Drag"
	}
	return "Enable Drag"
}

func (t *trayMenu) run() {
	instance, _, _ := procGetModuleHandleW.Call(0)
	class := trayWndClassEx{
		CbSize:        uint32(unsafe.Sizeof(trayWndClassEx{})),
		LpfnWndProc:   windows.NewCallback(trayWindowProc),
		HInstance:     instance,
		LpszClassName: trayWindowClassName,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class)))

	hwnd, _, callErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(trayWindowClassName)),
		uintptr(unsafe.Pointer(trayWindowTitle)),
		0,
		0, 0, 0, 0,
		0, 0,
		instance,
		0,
	)
	if hwnd == 0 {
		t.ready <- fmt.Errorf("CreateWindowExW: %w", callErr)
		return
	}

	t.hwnd = hwnd
	trayInstancesMu.Lock()
	trayInstances[hwnd] = t
	trayInstancesMu.Unlock()

	if err := t.addIcon(); err != nil {
		t.ready <- err
		return
	}
	t.ready <- nil

	var msg trayMsg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func trayWindowProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	switch msg {
	case trayCallbackMessage:
		if lparam == wmRButtonUp {
			if t := trayForWindow(hwnd); t != nil {
				t.showMenu()
			}
			return 0
		}
	case wmDestroy:
		trayInstancesMu.Lock()
		delete(trayInstances, hwnd)
		trayInstancesMu.Unlock()
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return ret
}

func trayForWindow(hwnd uintptr) *trayMenu {
	trayInstancesMu.Lock()
	defer trayInstancesMu.Unlock()
	return trayInstances[hwnd]
}

func (t *trayMenu) addIcon() error {
	icon, _, _ := procLoadIconW.Call(0, uintptr(idiApplication))
	nid := t.notifyIconData()
	nid.HIcon = icon
	copy(nid.SzTip[:], windows.StringToUTF16("Daidai Pet"))

	ret, _, callErr := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		return fmt.Errorf("Shell_NotifyIconW add: %w", callErr)
	}
	return nil
}

func (t *trayMenu) deleteIcon() {
	nid := t.notifyIconData()
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func (t *trayMenu) notifyIconData() notifyIconData {
	return notifyIconData{
		CbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:             t.hwnd,
		UID:              trayIconID,
		UFlags:           nifMessage | nifIcon | nifTip,
		UCallbackMessage: trayCallbackMessage,
	}
}

func (t *trayMenu) showMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	label, _ := windows.UTF16PtrFromString(trayDragLabel(t.app.dragEnabled()))
	procAppendMenuW.Call(menu, mfString, trayIDToggleDrag, uintptr(unsafe.Pointer(label)))

	var point winPoint
	procGetCursorPosTray.Call(uintptr(unsafe.Pointer(&point)))
	procSetForegroundWindow.Call(t.hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(
		menu,
		tpmRightAlign|tpmRightButton|tpmReturnCmd,
		uintptr(point.X),
		uintptr(point.Y),
		0,
		t.hwnd,
		0,
	)

	if cmd == trayIDToggleDrag {
		t.app.ToggleDragEnabled()
	}
}
