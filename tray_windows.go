//go:build windows

package main

import (
	"gioui.org/app"
	"gioui.org/io/system"

	"fyne.io/systray"
)

func startTray(w *app.Window) {
	go systray.Run(func() { trayReady(w) }, func() {})
}

func trayReady(w *app.Window) {
	systray.SetTitle(title)
	systray.SetTooltip(title)

	showItem := systray.AddMenuItem("Show", "Bring the window to front")
	minItem := systray.AddMenuItem("Minimize", "Minimize the window")
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("Quit", "Quit the app")

	go func() {
		for {
			select {
			case <-showItem.ClickedCh:
				w.Perform(system.ActionRaise)
			case <-minItem.ClickedCh:
				w.Perform(system.ActionMinimize)
			case <-quitItem.ClickedCh:
				systray.Quit()
				w.Perform(system.ActionClose)
				return
			}
		}
	}()

}
