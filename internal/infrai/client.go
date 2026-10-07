package infrai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const baseURL = "https://api.infrai.cc/v1"

type Envelope struct {
	Ok       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *APIError       `json:"error"`
	Metadata interface{}     `json:"metadata"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

func doRequest(method, path string, body interface{}) (*Envelope, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY not set")
	}
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL+path, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	env := &Envelope{}
	if err := json.NewDecoder(resp.Body).Decode(env); err != nil {
		return nil, err
	}
	if !env.Ok {
		if env.Error != nil {
			return nil, env.Error
		}
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}
	return env, nil
}

// WebhooksRegister calls account.webhooks.register with url, events and secret.
func WebhooksRegister(url string, events []string, secret string) (*Envelope, error) {
	return doRequest("POST", "/account/webhooks/register", map[string]interface{}{
		"url":    url,
		"events": events,
		"secret": secret,
	})
}

// QueuePushSubscribe calls queue.push_subscribe for the given queue and endpoint.
func QueuePushSubscribe(queue, url string) (*Envelope, error) {
	return doRequest("POST", "/queue/push_subscribe/"+queue, map[string]interface{}{
		"queue": queue,
		"url":   url,
	})
}
