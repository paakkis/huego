package main

import (
	"errors"
	"fmt"
	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/colorpicker"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"huego/internal/api"
	"huego/internal/bridge"
	"huego/internal/config"
	"image/color"
	"log"
	"math"
	"os"
	"sort"
	"strings"
)

var (
	lightSliders       = map[string]*widget.Float{}
	lightPickerStates  = map[string]*colorpicker.State{}
	lightPickerButtons = map[string]*widget.Clickable{}
	lightPickerOpen    = map[string]bool{}
	lightsList         widget.List
	sliderInit         = map[string]bool{}
	title              = "Huego"
	MAX_WIDTH          = unit.Dp(400)
	MAX_HEIGHT         = unit.Dp(425)
	bg                 = color.NRGBA{R: 18, G: 18, B: 22, A: 255}
	lav                = color.NRGBA{R: 0xD3, G: 0xD3, B: 0xFF, A: 0xFF}
	white              = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	icon               *widget.Icon
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
		startTray(w)
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
	theme.Palette = material.Palette{
		Bg:         bg,
		Fg:         lav,
		ContrastBg: lav,
		ContrastFg: bg,
	}
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	if lightsList.Axis == 0 {
		lightsList.Axis = layout.Vertical
	}
	theme.Face = "Go Mono"
	theme.FingerSize = 1
	ic, err := widget.NewIcon(icons.ImageColorLens)
	if err != nil {
		log.Fatal(err)
	}
	icon = ic

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
		if _, ok := lightPickerStates[id]; !ok {
			state := new(colorpicker.State)
			state.SetColor(white)
			lightPickerStates[id] = state
		}
		if _, ok := lightPickerButtons[id]; !ok {
			lightPickerButtons[id] = new(widget.Clickable)
		}
	}
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx C) D {
					paint.Fill(gtx.Ops, bg)
					return layout.Dimensions{Size: gtx.Constraints.Max}
				}),
				layout.Stacked(func(gtx C) D {
					return layoutList(gtx, theme, br, lights)
				}),
			)
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

	list := material.List(th, &lightsList)
	return list.Layout(gtx, len(rows), func(gtx layout.Context, index int) layout.Dimensions {
		id := rows[index].id
		light := lights[id]

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
			briDisplay = int(math.Round(float64(light.State.Bri-1) / 253.0 * 100.0))
		}
		nameW := gtx.Dp(unit.Dp(140))
		block := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
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
					layout.Rigid(func(gtx C) D {
						if !supportsXY(light) {
							return D{}
						}
						btn := lightPickerButtons[id]
						if btn.Clicked(gtx) {
							lightPickerOpen[id] = !lightPickerOpen[id]
							if lightPickerOpen[id] {
								state := lightPickerStates[id]
								if light.State.XY != [2]float64{} {
									yLuma := float64(light.State.Bri) / 254.0
									if !light.State.On {
										yLuma = 0
									}
									state.SetColor(api.XYToRGB(light.State.XY, yLuma))
								}
							}
						}
						return layout.UniformInset(unit.Dp(6)).Layout(gtx,
							material.IconButton(th, btn, icon, "Color").Layout,
						)
					}),
				)
			}),
			layout.Rigid(func(gtx C) D {
				if !supportsXY(light) || !lightPickerOpen[id] {
					return D{}
				}
				state := lightPickerStates[id]
				return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
					return layout.W.Layout(gtx, func(gtx C) D {
						gtx.Constraints.Max.X = gtx.Dp(unit.Dp(260))
						return colorpicker.PickerStyle{
							Label:         "Color",
							Theme:         th,
							State:         state,
							MonospaceFace: "Go Mono",
						}.Layout(gtx)
					})
				})
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
			if err := api.SetLightState(br, id, newOn, bri, light.State.XY); err != nil {
				fmt.Println("slider update failed:", err)
			} else {
				light.State.On = newOn
				if newOn {
					light.State.Bri = bri
				}
				lights[id] = light
			}
		}
		if supportsXY(light) && lightPickerOpen[id] {
			color := lightPickerStates[id].Color()
			xy := api.GetRGBtoXY(color)
			if err := api.SetLightState(br, id, light.State.On, light.State.Bri, xy); err != nil {
				fmt.Println("color update failed:", err)
			} else {
				light.State.XY = xy
				lights[id] = light
			}
		}
		return block
	})
}

func supportsXY(light api.LightV1) bool {
	if light.State.XY != [2]float64{} {
		return true
	}
	return strings.Contains(strings.ToLower(light.Type), "color")
}

func errOr(msg string, err error) error {
	if err != nil {
		return err
	}
	return errors.New(msg)
}
