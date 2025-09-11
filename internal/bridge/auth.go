package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AuthRequest struct {
	Devicetype        string `json:"devicetype"`
	GenerateClientKey bool   `json:"generateclientkey"`
}

type AuthSuccess struct {
	Username  string `json:"username"`
	ClientKey string `json:"clientkey"`
}

type AuthResponse struct {
	Success *AuthSuccess           `json:"success,omitempty"`
	Error   map[string]interface{} `json:"error,omitempty"`
}

func Authenticate(bridgeIP string) (*AuthSuccess, error) {
	url := fmt.Sprintf("http://%s/api", bridgeIP)

	reqBody := AuthRequest{
		Devicetype:        "huegoapp#golightcontroller",
		GenerateClientKey: true,
	}

	jsonBody, _ := json.Marshal(reqBody)

	fmt.Println("waiting for bridge button press...")

	for i := 0; i < 30; i++ {
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var result []AuthResponse
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &result)

		if len(result) > 0 && result[0].Success != nil {
			return result[0].Success, nil
		}
		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("authentication timeout")
}
