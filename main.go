package main

import (
	"fmt"
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"huego/internal/api"
	"huego/internal/bridge"
	"huego/internal/config"
	"log"
	"math"
	"os"
	"sort"
)

var (
	lightSliders = map[string]*widget.Float{}
	sliderInit   = map[string]bool{}
	title        = "Huegio"
	MAX_WIDTH    = unit.Dp(400)
	MAX_HEIGHT   = unit.Dp(300)
)

type (
	D = layout.Dimensions
	C = layout.Context
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title(title))
		w.Option(app.Size(MAX_WIDTH, MAX_HEIGHT))
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

	for id := range lights {
		if _, ok := lightSliders[id]; !ok {
			lightSliders[id] = new(widget.Float)
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
				light := lights[id]

				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					slider := lightSliders[id]
					if !sliderInit[id] {
						if light.State.On && light.State.Bri > 0 {
							slider.Value = float32(light.State.Bri-1) / 253.0
						} else {
							slider.Value = 0
						}
						sliderInit[id] = true
					}

					briDisplay := 0
					if slider.Value > 0 {
						briDisplay = int(math.Round(float64(slider.Value)*253.0)) + 1
					}
					nameW := gtx.Dp(unit.Dp(140))
					dims := layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							gtx.Constraints.Min.X = nameW
							gtx.Constraints.Max.X = nameW
							return layout.UniformInset(unit.Dp(8)).Layout(gtx,
								material.Body1(th, light.Name).Layout,
							)
						}),
						layout.Flexed(1, material.Slider(th, slider).Layout),
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(8)).Layout(gtx,
								material.Body1(th, fmt.Sprintf("%d", briDisplay)).Layout,
							)
						}),
					)

					if slider.Dragging() {
						newOn := slider.Value > 0
						bri := light.State.Bri
						if newOn {
							bri = int(math.Round(float64(slider.Value)*253.0)) + 1
						} else if bri <= 0 {
							bri = 1
						}
						if err := api.SetLightState(br, id, newOn, bri); err != nil {
							fmt.Println("slider update failed:", err)
						} else {
							light.State.On = newOn
							if newOn {
								light.State.Bri = bri
							}
							lights[id] = light
						}
					}

					return dims
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
