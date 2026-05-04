//go:build windows

package main

import (
	"bytes"
	"fmt"
	"image"
	"syscall"
	"unsafe"
)

var (
	user32ProcOpenClipboard  = user32.NewProc("OpenClipboard")
	user32ProcCloseClipboard = user32.NewProc("CloseClipboard")
	user32ProcEmptyClipboard = user32.NewProc("EmptyClipboard")
	user32ProcSetClipboard   = user32.NewProc("SetClipboardData")
	kernel32ProcGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	kernel32ProcGlobalLock   = kernel32.NewProc("GlobalLock")
	kernel32ProcGlobalUnlock = kernel32.NewProc("GlobalUnlock")
	kernel32ProcGlobalFree   = kernel32.NewProc("GlobalFree")
)

const (
	cfDIB        = 8
	gmemMoveable = 0x0002
	biRGB        = 0
)

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

func writeStickerImageToClipboard(pngBytes []byte) error {
	img, _, err := image.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return fmt.Errorf("decode png for windows clipboard: %w", err)
	}
	dib, err := imageToDIB(img)
	if err != nil {
		return err
	}
	return writeDIBToClipboard(dib)
}

func imageToDIB(img image.Image) ([]byte, error) {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid image size %dx%d", w, h)
	}
	pixelBytes := w * h * 4
	header := bitmapInfoHeader{
		Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:       int32(w),
		Height:      int32(h), // bottom-up DIB
		Planes:      1,
		BitCount:    32,
		Compression: biRGB,
		SizeImage:   uint32(pixelBytes),
	}
	dib := make([]byte, int(header.Size)+pixelBytes)
	*(*bitmapInfoHeader)(unsafe.Pointer(&dib[0])) = header

	offset := int(header.Size)
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			r16, g16, b16, a16 := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			r := byte(r16 >> 8)
			g := byte(g16 >> 8)
			bl := byte(b16 >> 8)
			a := byte(a16 >> 8)
			dib[offset+0] = bl
			dib[offset+1] = g
			dib[offset+2] = r
			dib[offset+3] = a
			offset += 4
		}
	}
	return dib, nil
}

func writeDIBToClipboard(dib []byte) error {
	if len(dib) == 0 {
		return fmt.Errorf("empty dib data")
	}
	for attempt := 1; attempt <= 8; attempt++ {
		if err := tryWriteDIBToClipboard(dib); err == nil {
			return nil
		} else if attempt == 8 {
			return err
		}
	}
	return fmt.Errorf("unreachable clipboard write")
}

func tryWriteDIBToClipboard(dib []byte) error {
	openRet, _, openErr := user32ProcOpenClipboard.Call(0)
	if openRet == 0 {
		if openErr != nil && openErr != syscall.Errno(0) {
			return openErr
		}
		return fmt.Errorf("OpenClipboard failed")
	}
	defer user32ProcCloseClipboard.Call()

	emptyRet, _, emptyErr := user32ProcEmptyClipboard.Call()
	if emptyRet == 0 {
		if emptyErr != nil && emptyErr != syscall.Errno(0) {
			return emptyErr
		}
		return fmt.Errorf("EmptyClipboard failed")
	}

	mem, _, allocErr := kernel32ProcGlobalAlloc.Call(gmemMoveable, uintptr(len(dib)))
	if mem == 0 {
		if allocErr != nil && allocErr != syscall.Errno(0) {
			return allocErr
		}
		return fmt.Errorf("GlobalAlloc failed")
	}

	ptr, _, lockErr := kernel32ProcGlobalLock.Call(mem)
	if ptr == 0 {
		kernel32ProcGlobalFree.Call(mem)
		if lockErr != nil && lockErr != syscall.Errno(0) {
			return lockErr
		}
		return fmt.Errorf("GlobalLock failed")
	}

	dst := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(dib))
	copy(dst, dib)
	kernel32ProcGlobalUnlock.Call(mem)

	setRet, _, setErr := user32ProcSetClipboard.Call(cfDIB, mem)
	if setRet == 0 {
		kernel32ProcGlobalFree.Call(mem)
		if setErr != nil && setErr != syscall.Errno(0) {
			return setErr
		}
		return fmt.Errorf("SetClipboardData(CF_DIB) failed")
	}
	return nil
}
