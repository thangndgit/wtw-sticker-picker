//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static bool getCursorPosition(int* x, int* y) {
    NSPoint p = [NSEvent mouseLocation];
    NSScreen* screen = [NSScreen mainScreen];
    if (screen == nil) {
        return false;
    }
    CGFloat h = [screen frame].size.height;
    *x = (int)p.x;
    *y = (int)(h - p.y);
    return true;
}
*/
import "C"

func currentCursorPosition() (int, int, bool) {
	var x C.int
	var y C.int
	ok := bool(C.getCursorPosition(&x, &y))
	return int(x), int(y), ok
}
