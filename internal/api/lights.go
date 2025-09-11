package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Bridge struct {
	IP       string
	Username string
}

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

type LightV1 struct {
	State struct {
		On  bool `json:"on"`
		Bri int  `json:"bri"`
	} `json:"state"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type resourceResponse struct {
	Data []Light `json:"data"`
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

type LightState struct {
	On      *OnState      `json:"on,omitempty"`
	Dimming *DimmingState `json:"dimming,omitempty"`
}

type OnState struct {
	On bool `json:"on"`
}

type DimmingState struct {
	Brightness float64 `json:"brightness"` // range 1.0–100.0
}

func SetLightState(bridge Bridge, lightID string, on bool, bri int) error {
	url := fmt.Sprintf("http://%s/api/%s/lights/%s/state", bridge.IP, bridge.Username, lightID)

	state := map[string]interface{}{
		"on":  on,
		"bri": bri, // 1 to 254
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
