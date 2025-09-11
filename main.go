package main

import (
	"fmt"
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"huego/internal/api"
	"huego/internal/bridge"
	"huego/internal/config"
	"image/color"
	"log"
	"os"
	"sort"
)

var lightToggles = map[string]*widget.Clickable{}

func main() {
	go func() {
		w := new(app.Window)
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window) error {
	var ops op.Ops
	theme := material.NewTheme()

	cfg, err := config.LoadConfig()
	if err != nil || cfg.Username == "" {
		// Discover + authenticate on first run
		bridges, err := bridge.DiscoverBridges()
		if err != nil || len(bridges) == 0 {
			return errOr("No bridges found", err)
		}
		auth, err := bridge.Authenticate(bridges[0].InternalIP)
		if err != nil {
			return errOr("Authentication failed", err)
		}
		cfg = config.Config{BridgeIP: bridges[0].InternalIP, Username: auth.Username}
		if err := config.SaveConfig(cfg); err != nil {
			log.Println("Warning: failed to save config:", err)
		}
	}

	br := api.Bridge{IP: cfg.BridgeIP, Username: cfg.Username}
	lights, err := api.GetLights(br)
	if err != nil {
		return err
	}

	// Ensure a stable Clickable per light ID.
	for id := range lights {
		if _, ok := lightToggles[id]; !ok {
			lightToggles[id] = new(widget.Clickable)
		}
	}

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			layoutList(gtx, theme, br, lights)
			e.Frame(gtx.Ops)
		}
	}
}

func layoutList(gtx layout.Context, th *material.Theme, br api.Bridge, lights map[string]api.LightV1) layout.Dimensions {
	// 1) Build a deterministic order
	type row struct{ id, name string }
	rows := make([]row, 0, len(lights))
	for id, l := range lights {
		name := l.Name
		if name == "" {
			name = id
		}
		rows = append(rows, row{id: id, name: name})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		func() []layout.FlexChild {
			children := make([]layout.FlexChild, 0, len(rows))
			for _, r := range rows {
				id := r.id
				light := lights[id] // snapshot

				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(th, lightToggles[id], light.Name)
					if light.State.On {
						btn.Background = color.NRGBA{R: 80, G: 200, B: 100, A: 255}
					} else {
						btn.Background = color.NRGBA{R: 180, G: 180, B: 180, A: 255}
					}

					// 2) Use your SetLightState on click
					for lightToggles[id].Clicked(gtx) {
						newOn := !light.State.On

						// pick a sensible brightness when turning on
						bri := light.State.Bri
						if bri <= 0 {
							bri = 254 // Hue v1 valid range is 1..254
						}

						if err := api.SetLightState(br, id, newOn, bri); err != nil {
							fmt.Println("toggle failed:", err)
							// do NOT mutate local state on failure
						} else {
							// reflect new state locally so UI updates immediately
							light.State.On = newOn
							lights[id] = light
						}
					}

					return btn.Layout(gtx)
				}))
			}
			return children
		}()...,
	)
}
func errOr(msg string, err error) error {
	if err != nil {
		return err
	}
	return fmt.Errorf(msg)
}
