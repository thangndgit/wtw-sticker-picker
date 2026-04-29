package main

import "github.com/wailsapp/wails/v3/pkg/application"

const (
	popupWindowName = "popup"
	popupWidth      = 420
	popupHeight     = 280
	popupMargin     = 8
)

func showPopupNearCursor(app *application.App, windowName string) {
	window, ok := app.Window.GetByName(windowName)
	if !ok {
		return
	}

	cursorX, cursorY, hasCursor := currentCursorPosition()
	if !hasCursor {
		primary := app.Screen.GetPrimary()
		if primary != nil {
			cursorX = primary.WorkArea.X + (primary.WorkArea.Width / 2)
			cursorY = primary.WorkArea.Y + (primary.WorkArea.Height / 2)
		}
	}

	cursorPoint := application.Point{X: cursorX, Y: cursorY}
	screen := app.Screen.ScreenNearestDipPoint(cursorPoint)
	if screen == nil {
		screen = app.Screen.GetPrimary()
		if screen == nil {
			window.Show().Focus()
			return
		}
	}

	targetX := cursorPoint.X - (popupWidth / 2)
	targetY := cursorPoint.Y - popupHeight

	minX := screen.WorkArea.X + popupMargin
	maxX := screen.WorkArea.X + screen.WorkArea.Width - popupWidth - popupMargin
	minY := screen.WorkArea.Y + popupMargin
	maxY := screen.WorkArea.Y + screen.WorkArea.Height - popupHeight - popupMargin

	if maxX < minX {
		maxX = minX
	}
	if maxY < minY {
		maxY = minY
	}

	if targetX < minX {
		targetX = minX
	}
	if targetX > maxX {
		targetX = maxX
	}
	if targetY < minY {
		targetY = minY
	}
	if targetY > maxY {
		targetY = maxY
	}

	window.SetPosition(targetX, targetY)
	window.Show()
	window.Focus()
}
