//go:build windows

package main

import (
	"fmt"
	rt "runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmApp               = 0x8000
	wmCommand           = 0x0111
	wmClose             = 0x0010
	wmDestroy           = 0x0002
	wmNull              = 0x0000
	wmRButtonUp         = 0x0205
	wmLButtonDblClk     = 0x0203
	csHRedraw           = 0x0002
	csVRedraw           = 0x0001
	cwUseDefault        = 0x80000000
	idiApplication      = 32512
	idcArrow            = 32512
	imageIcon           = 1
	lrDefaultSize       = 0x0040
	lrShared            = 0x8000
	nifMessage          = 0x00000001
	nifIcon             = 0x00000002
	nifTip              = 0x00000004
	nimAdd              = 0x00000000
	nimDelete           = 0x00000002
	tpmLeftAlign        = 0x0000
	tpmBottomAlign      = 0x0020
	tpmRightButton      = 0x0002
	mfString            = 0x0000
	mfSeparator         = 0x0800
	mfChecked           = 0x0008
	mfUnchecked         = 0x0000
	trayIconID          = 1
	trayCallbackMsg     = wmApp + 1
	menuCommandShowHide = 100
	menuCommandQuit     = 101
	menuCommandSizeBase = 200
	menuCommandStepBase = 300
	menuCommandWalkBase = 400
)

type trayController struct {
	app         *App
	hwnd        windows.Handle
	threadID    uint32
	refreshFunc func()
	destroyFunc func()
	destroyOnce sync.Once
	done        chan struct{}
}

type point struct {
	X int32
	Y int32
}

type msg struct {
	HWnd     windows.Handle
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type notifyIconData struct {
	CbSize            uint32
	HWnd              windows.Handle
	UID               uint32
	UFlags            uint32
	UCallbackMessage  uint32
	HIcon             windows.Handle
	SzTip             [128]uint16
	DwState           uint32
	DwStateMask       uint32
	SzInfo            [256]uint16
	UTimeoutOrVersion uint32
	SzInfoTitle       [64]uint16
	DwInfoFlags       uint32
	GuidItem          windows.GUID
	HBalloonIcon      windows.Handle
}

var (
	shell32                 = windows.NewLazySystemDLL("shell32.dll")
	procShellNotifyIconW    = shell32.NewProc("Shell_NotifyIconW")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procGetCursorPosUser32  = user32.NewProc("GetCursorPos")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procGetModuleHandleW    = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetModuleHandleW")
	procGetCurrentThreadID  = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetCurrentThreadId")
	procLoadCursorW         = user32.NewProc("LoadCursorW")
	procLoadImageW          = user32.NewProc("LoadImageW")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
)

var (
	trayClassName     = syscall.StringToUTF16Ptr("PetDesktopTrayWindow")
	trayWindowProcPtr = syscall.NewCallback(trayWindowProc)
	trayRegisterClass sync.Once
	trayRegisterErr   error
	trayControllers   sync.Map
)

func newTrayController(app *App) (*trayController, error) {
	controller := &trayController{
		app:  app,
		done: make(chan struct{}),
	}
	initResult := make(chan error, 1)

	go controller.run(initResult)

	if err := <-initResult; err != nil {
		return nil, err
	}
	return controller, nil
}

func (t *trayController) Refresh() {
	if t == nil {
		return
	}
	if t.refreshFunc != nil {
		t.refreshFunc()
	}
}

func (t *trayController) Destroy() {
	if t == nil {
		return
	}
	if t.destroyFunc != nil {
		t.destroyFunc()
		return
	}

	t.destroyOnce.Do(func() {
		if t.threadID != 0 && currentThreadID() == t.threadID {
			t.destroyOnThread()
			return
		}
		if t.hwnd != 0 {
			postMessage(t.hwnd, wmClose, 0, 0)
		}
	})
}

func (t *trayController) run(initResult chan<- error) {
	rt.LockOSThread()
	defer rt.UnlockOSThread()

	t.threadID = currentThreadID()

	if err := registerTrayWindowClass(); err != nil {
		initResult <- err
		close(t.done)
		return
	}

	hwnd, err := createTrayWindow()
	if err != nil {
		initResult <- err
		close(t.done)
		return
	}

	t.hwnd = hwnd
	trayControllers.Store(hwnd, t)

	if err := t.addTrayIcon(); err != nil {
		trayControllers.Delete(hwnd)
		destroyWindow(hwnd)
		initResult <- err
		close(t.done)
		return
	}

	initResult <- nil

	var message msg
	for {
		result, _, _ := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&message)),
			0,
			0,
			0,
		)
		if int32(result) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}

	trayControllers.Delete(hwnd)
	close(t.done)
}

func (t *trayController) addTrayIcon() error {
	icon, err := loadTrayIcon()
	if err != nil {
		return err
	}

	nid := t.notifyIconData()
	nid.HIcon = icon
	copyUTF16(nid.SzTip[:], "Pet Desktop")

	ok, _, callErr := procShellNotifyIconW.Call(
		uintptr(nimAdd),
		uintptr(unsafe.Pointer(&nid)),
	)
	if ok == 0 {
		return fmt.Errorf("Shell_NotifyIconW(NIM_ADD): %w", callErr)
	}
	return nil
}

func (t *trayController) destroyOnThread() {
	if t.hwnd != 0 {
		nid := t.notifyIconData()
		procShellNotifyIconW.Call(
			uintptr(nimDelete),
			uintptr(unsafe.Pointer(&nid)),
		)
		destroyWindow(t.hwnd)
		t.hwnd = 0
	}
}

func (t *trayController) notifyIconData() notifyIconData {
	return notifyIconData{
		CbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:             t.hwnd,
		UID:              trayIconID,
		UFlags:           nifMessage | nifIcon | nifTip,
		UCallbackMessage: trayCallbackMsg,
	}
}

func (t *trayController) showMenu() {
	if t.app == nil || t.hwnd == 0 {
		return
	}

	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	visible := t.app.currentVisible()
	scalePercent := t.app.currentScalePercent()
	stepSize := t.app.currentStepSize()
	walkInterval := int(t.app.currentWalkInterval() / 1_000_000)

	showHideLabel := "Hide"
	if !visible {
		showHideLabel = "Show"
	}

	appendMenuString(menu, menuCommandShowHide, showHideLabel, false)
	appendMenuSeparator(menu)

	for index, preset := range scalePresets {
		appendMenuString(menu, uintptr(menuCommandSizeBase+index), fmt.Sprintf("Size %d%%", preset), preset == scalePercent)
	}
	appendMenuSeparator(menu)

	for index, preset := range stepSizePresets {
		appendMenuString(menu, uintptr(menuCommandStepBase+index), fmt.Sprintf("Step %d", preset), preset == stepSize)
	}
	appendMenuSeparator(menu)

	for index, preset := range walkIntervalPresets {
		appendMenuString(menu, uintptr(menuCommandWalkBase+index), fmt.Sprintf("Interval %dms", preset), preset == walkInterval)
	}
	appendMenuSeparator(menu)
	appendMenuString(menu, menuCommandQuit, "Quit", false)

	var pt point
	ret, _, _ := procGetCursorPosUser32.Call(uintptr(unsafe.Pointer(&pt)))
	if ret == 0 {
		return
	}

	procSetForegroundWindow.Call(uintptr(t.hwnd))
	procTrackPopupMenu.Call(
		menu,
		uintptr(tpmLeftAlign|tpmBottomAlign|tpmRightButton),
		uintptr(int32(pt.X)),
		uintptr(int32(pt.Y)),
		0,
		uintptr(t.hwnd),
		0,
	)
	postMessage(t.hwnd, wmNull, 0, 0)
}

func (t *trayController) handleCommand(commandID uintptr) {
	switch {
	case commandID == menuCommandShowHide:
		t.app.SetVisible(!t.app.currentVisible())
	case commandID == menuCommandQuit:
		t.app.Quit()
	case commandID >= menuCommandSizeBase && commandID < menuCommandSizeBase+uintptr(len(scalePresets)):
		t.app.SetScalePercent(scalePresets[commandID-menuCommandSizeBase])
	case commandID >= menuCommandStepBase && commandID < menuCommandStepBase+uintptr(len(stepSizePresets)):
		t.app.SetStepSize(stepSizePresets[commandID-menuCommandStepBase])
	case commandID >= menuCommandWalkBase && commandID < menuCommandWalkBase+uintptr(len(walkIntervalPresets)):
		t.app.SetWalkIntervalMs(walkIntervalPresets[commandID-menuCommandWalkBase])
	}
}

func trayWindowProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	controllerValue, _ := trayControllers.Load(windows.Handle(hwnd))
	controller, _ := controllerValue.(*trayController)

	switch message {
	case trayCallbackMsg:
		if controller != nil && (lParam == wmRButtonUp || lParam == wmLButtonDblClk) {
			controller.showMenu()
			return 0
		}
	case wmCommand:
		if controller != nil {
			controller.handleCommand(wParam & 0xffff)
			return 0
		}
	case wmClose:
		if controller != nil {
			controller.destroyOnThread()
			return 0
		}
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return ret
}

func registerTrayWindowClass() error {
	trayRegisterClass.Do(func() {
		instance, _, err := procGetModuleHandleW.Call(0)
		if instance == 0 {
			trayRegisterErr = fmt.Errorf("GetModuleHandleW: %w", err)
			return
		}

		cursor, _, _ := procLoadCursorW.Call(0, uintptr(idcArrow))
		icon, iconErr := loadTrayIcon()
		if iconErr != nil {
			trayRegisterErr = iconErr
			return
		}

		class := wndClassEx{
			CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
			Style:         csHRedraw | csVRedraw,
			LpfnWndProc:   trayWindowProcPtr,
			HInstance:     windows.Handle(instance),
			HIcon:         icon,
			HCursor:       windows.Handle(cursor),
			LpszClassName: trayClassName,
			HIconSm:       icon,
		}

		atom, _, regErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class)))
		if atom == 0 {
			const errorClassAlreadyExists syscall.Errno = 1410
			if regErr != errorClassAlreadyExists {
				trayRegisterErr = fmt.Errorf("RegisterClassExW: %w", regErr)
			}
		}
	})

	return trayRegisterErr
}

func createTrayWindow() (windows.Handle, error) {
	instance, _, err := procGetModuleHandleW.Call(0)
	if instance == 0 {
		return 0, fmt.Errorf("GetModuleHandleW: %w", err)
	}

	hwnd, _, createErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(trayClassName)),
		uintptr(unsafe.Pointer(trayClassName)),
		0,
		uintptr(cwUseDefault),
		uintptr(cwUseDefault),
		0,
		0,
		0,
		0,
		instance,
		0,
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowExW: %w", createErr)
	}

	return windows.Handle(hwnd), nil
}

func destroyWindow(hwnd windows.Handle) {
	if hwnd == 0 {
		return
	}
	procDestroyWindow.Call(uintptr(hwnd))
}

func appendMenuString(menu uintptr, commandID uintptr, label string, checked bool) {
	text, _ := windows.UTF16PtrFromString(label)
	flags := uintptr(mfString | mfUnchecked)
	if checked {
		flags = uintptr(mfString | mfChecked)
	}
	procAppendMenuW.Call(menu, flags, commandID, uintptr(unsafe.Pointer(text)))
}

func appendMenuSeparator(menu uintptr) {
	procAppendMenuW.Call(menu, uintptr(mfSeparator), 0, 0)
}

func loadTrayIcon() (windows.Handle, error) {
	icon, _, err := procLoadImageW.Call(
		0,
		uintptr(idiApplication),
		uintptr(imageIcon),
		0,
		0,
		uintptr(lrDefaultSize|lrShared),
	)
	if icon == 0 {
		return 0, fmt.Errorf("LoadImageW: %w", err)
	}
	return windows.Handle(icon), nil
}

func postMessage(hwnd windows.Handle, message uint32, wParam, lParam uintptr) {
	if hwnd == 0 {
		return
	}
	procPostMessageW.Call(uintptr(hwnd), uintptr(message), wParam, lParam)
}

func currentThreadID() uint32 {
	threadID, _, _ := procGetCurrentThreadID.Call()
	return uint32(threadID)
}

func copyUTF16(dst []uint16, src string) {
	encoded, _ := windows.UTF16FromString(src)
	copy(dst, encoded)
}
