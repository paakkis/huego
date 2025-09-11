package bridge

import (
	"encoding/json"
	"io"
	"net/http"
)

type HueBridge struct {
	ID         string `json:"id"`
	InternalIP string `json:"internalipaddress"`
}

func DiscoverBridges() ([]HueBridge, error) {
	resp, err := http.Get("https://discovery.meethue.com")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var bridges []HueBridge
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &bridges)
	return bridges, err
}
