//go:build windows

package main

import (
	"errors"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	keybd_event "github.com/micmonay/keybd_event"
)

var (
	user32ProcGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	user32ProcSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	user32ProcBringWindowToTop    = user32.NewProc("BringWindowToTop")
	user32ProcShowWindow          = user32.NewProc("ShowWindow")
	user32ProcIsIconic            = user32.NewProc("IsIconic")
	user32ProcAttachThreadInput   = user32.NewProc("AttachThreadInput")
	user32ProcGetWindowThreadPID  = user32.NewProc("GetWindowThreadProcessId")
	user32ProcGetGUIThreadInfo    = user32.NewProc("GetGUIThreadInfo")
	user32ProcSetFocus            = user32.NewProc("SetFocus")
	user32ProcPostMessageW        = user32.NewProc("PostMessageW")
	user32ProcSendInput           = user32.NewProc("SendInput")
	kernel32                      = syscall.NewLazyDLL("kernel32.dll")
	kernel32ProcGetCurrentThread  = kernel32.NewProc("GetCurrentThreadId")
	lastForegroundWindow          uintptr
	lastFocusedWindow             uintptr
)

const (
	inputKeyboard  = 1
	keyeventfKeyup = 0x0002
	vkControl      = 0x11
	vkShift        = 0x10
	vkInsert       = 0x2D
	vkMenu         = 0x12
	vkLWin         = 0x5B
	vkRWin         = 0x5C
	swRestore      = 9
	wmPaste        = 0x0302
)

type keyboardInput struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type input struct {
	Type uint32
	Ki   keyboardInput
}

type guiThreadInfo struct {
	CbSize        uint32
	Flags         uint32
	HwndActive    uintptr
	HwndFocus     uintptr
	HwndCapture   uintptr
	HwndMenuOwner uintptr
	HwndMoveSize  uintptr
	HwndCaret     uintptr
	RcCaretLeft   int32
	RcCaretTop    int32
	RcCaretRight  int32
	RcCaretBottom int32
}

func releaseStickyModifiers() error {
	seq := []input{
		{Type: inputKeyboard, Ki: keyboardInput{WVk: vkLWin, DwFlags: keyeventfKeyup}},
		{Type: inputKeyboard, Ki: keyboardInput{WVk: vkRWin, DwFlags: keyeventfKeyup}},
		{Type: inputKeyboard, Ki: keyboardInput{WVk: vkMenu, DwFlags: keyeventfKeyup}},
		{Type: inputKeyboard, Ki: keyboardInput{WVk: vkControl, DwFlags: keyeventfKeyup}},
	}
	size := uintptr(unsafe.Sizeof(seq[0]))
	sent, _, _ := user32ProcSendInput.Call(uintptr(len(seq)), uintptr(unsafe.Pointer(&seq[0])), size)
	if sent == 0 {
		err := syscall.GetLastError()
		if err == syscall.Errno(0) {
			return errors.New("SendInput modifier release returned 0")
		}
		return err
	}
	return nil
}

func focusWindow(hwnd uintptr) error {
	targetThread := windowThreadID(hwnd)
	currentThread := currentThreadID()
	attached := false
	if targetThread != 0 && currentThread != 0 && targetThread != currentThread {
		ok, _, _ := user32ProcAttachThreadInput.Call(currentThread, targetThread, 1)
		if ok != 0 {
			attached = true
		}
	}
	if attached {
		defer func() {
			_, _, _ = user32ProcAttachThreadInput.Call(currentThread, targetThread, 0)
		}()
	}

	var lastErr error
	for attempt := 1; attempt <= 6; attempt++ {
		if isMinimized(hwnd) {
			_, _, _ = user32ProcShowWindow.Call(hwnd, swRestore)
		}
		_, _, _ = user32ProcBringWindowToTop.Call(hwnd)
		ret, _, err := user32ProcSetForegroundWindow.Call(hwnd)
		if ret == 0 && err != nil && err != syscall.Errno(0) {
			lastErr = err
		}
		time.Sleep(50 * time.Millisecond)
		fg, _, _ := user32ProcGetForegroundWindow.Call()
		if fg == hwnd {
			if lastFocusedWindow != 0 {
				_, _, _ = user32ProcSetFocus.Call(lastFocusedWindow)
			}
			return nil
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("unable to foreground captured window")
}

func isMinimized(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	ret, _, _ := user32ProcIsIconic.Call(hwnd)
	return ret != 0
}

func windowThreadID(hwnd uintptr) uintptr {
	if hwnd == 0 {
		return 0
	}
	tid, _, _ := user32ProcGetWindowThreadPID.Call(hwnd, 0)
	return tid
}

func currentThreadID() uintptr {
	tid, _, _ := kernel32ProcGetCurrentThread.Call()
	return tid
}

func capturePasteTarget() {
	hwnd, _, _ := user32ProcGetForegroundWindow.Call()
	if hwnd == 0 {
		return
	}
	lastForegroundWindow = hwnd
	lastFocusedWindow = focusedWindowForTarget(hwnd)
}

func focusedWindowForTarget(hwnd uintptr) uintptr {
	threadID := windowThreadID(hwnd)
	if threadID == 0 {
		return 0
	}
	info := guiThreadInfo{CbSize: uint32(unsafe.Sizeof(guiThreadInfo{}))}
	ret, _, _ := user32ProcGetGUIThreadInfo.Call(threadID, uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return 0
	}
	return info.HwndFocus
}

func pasteIntoCapturedTarget() error {
	if lastForegroundWindow != 0 {
		_ = focusWindow(lastForegroundWindow)
		time.Sleep(120 * time.Millisecond)
	}
	if lastFocusedWindow != 0 {
		_, _, _ = user32ProcPostMessageW.Call(lastFocusedWindow, wmPaste, 0, 0)
		// Allow the control to process WM_PASTE before keyboard fallback.
		time.Sleep(45 * time.Millisecond)
	}
	if err := sendPasteByExternalKeyboardLib(); err != nil {
		return err
	}
	return nil
}

func sendPasteByExternalKeyboardLib() error {
	_ = releaseStickyModifiers()

	var lastErr error
	for attempt := 1; attempt <= 4; attempt++ {
		if err := sendCtrlVViaKeybdEvent(); err == nil {
			_ = sendShiftInsertViaKeybdEvent()
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(60 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("keybd_event paste failed")
	}
	return lastErr
}

func sendCtrlVViaKeybdEvent() error {
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		return fmt.Errorf("init keybd_event: %w", err)
	}
	kb.SetKeys(keybd_event.VK_V)
	kb.HasCTRL(true)
	if err := kb.Launching(); err != nil {
		return fmt.Errorf("launch Ctrl+V: %w", err)
	}
	return nil
}

func sendShiftInsertViaKeybdEvent() error {
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		return err
	}
	kb.SetKeys(keybd_event.VK_INSERT)
	kb.HasSHIFT(true)
	return kb.Launching()
}

func pasteRawStickerIntoCapturedTarget(ext string, raw []byte) error {
	return errors.New("pasting raw sticker files is not supported on windows yet")
}
