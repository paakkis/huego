package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"math"
	"net/http"
)

type Bridge struct {
	IP       string
	Username string
}

// This is the newer Philips Hue light data's structure
// Older data types are declared in the LightV1 struct
type Light struct {
	ID       string `json:"id"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	On struct {
		On bool `json:"on"`
	} `json:"on"`
	Dimming struct {
		Brightness float64 `json:"brightness"`
	} `json:"dimming"`
}

// Older Philiphs Hue light struct
type LightV1 struct {
	State struct {
		On  bool       `json:"on"`
		Bri int        `json:"bri"`
		XY  [2]float64 `json:"xy"`
	} `json:"state"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	ProductName string `json:"productname"`
}

// Get all lights from the bridge
func GetLights(bridge Bridge) (map[string]LightV1, error) {
	url := fmt.Sprintf("http://%s/api/%s/lights", bridge.IP, bridge.Username)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var lights map[string]LightV1
	err = json.Unmarshal(body, &lights)
	return lights, err
}

func SetLightState(bridge Bridge, lightID string, on bool, bri int, xy [2]float64) error {
	url := fmt.Sprintf("http://%s/api/%s/lights/%s/state", bridge.IP, bridge.Username, lightID)

	state := map[string]any{
		"on":  on,
		"bri": bri, // 1 to 254
		"xy":  xy,  // color of the light
	}

	body, err := json.Marshal(state)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Hue V1 API error: %s", respBody)
	}

	return nil
}

// Philips Hue does not define their color as RGB
// Instead they use an XY value
// https://github.com/PhilipsHue/PhilipsHueSDK-iOS-OSX/commit/f41091cf671e13fe8c32fcced12604cd31cceaf3
// https://stackoverflow.com/questions/22564187/rgb-to-philips-hue-hsb
func GetRGBtoXY(c color.Color) [2]float64 {
	normalizedToOne := [3]float64{}
	r, g, b, _ := c.RGBA()
	normalizedToOne[0] = float64(r) / 65535.0
	normalizedToOne[1] = float64(g) / 65535.0
	normalizedToOne[2] = float64(b) / 65535.0

	var red, green, blue float64

	if normalizedToOne[0] > 0.04045 {
		red = math.Pow((normalizedToOne[0]+0.055)/(1.0+0.055), 2.4)
	} else {
		red = normalizedToOne[0] / 12.92
	}

	if normalizedToOne[1] > 0.04045 {
		green = math.Pow((normalizedToOne[1]+0.055)/(1.0+0.055), 2.4)
	} else {
		green = normalizedToOne[1] / 12.92
	}

	if normalizedToOne[2] > 0.04045 {
		blue = math.Pow((normalizedToOne[2]+0.055)/(1.0+0.055), 2.4)
	} else {
		blue = normalizedToOne[2] / 12.92
	}

	X := red*0.649926 + green*0.103455 + blue*0.197109
	Y := red*0.234327 + green*0.743075 + blue*0.022598
	Z := red*0.0000000 + green*0.053077 + blue*1.035763

	denom := X + Y + Z
	if denom == 0 {
		return [2]float64{0, 0}
	}

	x := X / denom
	y := Y / denom

	return [2]float64{x, y}
}

func XYToRGB(xy [2]float64, yLuma float64) color.NRGBA {
	x := xy[0]
	y := xy[1]
	if y == 0 {
		return color.NRGBA{A: 255}
	}

	// Reconstruct XYZ
	X := (x / y) * yLuma
	Y := yLuma
	Z := ((1 - x - y) / y) * yLuma

	// Inverse matrix (Hue wide-gamut)
	r := 1.612*X - 0.203*Y - 0.302*Z
	g := -0.509*X + 1.412*Y + 0.066*Z
	b := 0.026*X - 0.072*Y + 0.962*Z

	// Clamp negatives
	if r < 0 {
		r = 0
	}
	if g < 0 {
		g = 0
	}
	if b < 0 {
		b = 0
	}

	// Gamma correction
	r = gammaEncode(r)
	g = gammaEncode(g)
	b = gammaEncode(b)

	// Clamp to [0,1]
	if r > 1 {
		r = 1
	}
	if g > 1 {
		g = 1
	}
	if b > 1 {
		b = 1
	}

	return color.NRGBA{
		R: uint8(r*255 + 0.5),
		G: uint8(g*255 + 0.5),
		B: uint8(b*255 + 0.5),
		A: 255,
	}
}

func gammaEncode(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1.0/2.4) - 0.055
}
