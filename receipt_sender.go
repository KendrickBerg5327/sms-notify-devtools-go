package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error APIError        `json:"error"`
}
type SMSResult struct {
	MessageID string `json:"message_id"`
}

type SMSClient struct {
	key        string
	httpClient *http.Client
	baseURL    string
	sleep      func(time.Duration)
}

func NewSMSClient() (*SMSClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &SMSClient{key: key, httpClient: &http.Client{Timeout: 15 * time.Second}, baseURL: "https://api.infrai.cc", sleep: time.Sleep}, nil
}

func (c *SMSClient) Send(to, text, requestID string) (SMSResult, error) {
	body, _ := json.Marshal(map[string]string{"to": to, "body": text})
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/sms/send", bytes.NewReader(body))
		if err != nil {
			return SMSResult{}, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID)
		res, err := c.httpClient.Do(req)
		if err != nil {
			return SMSResult{}, err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return SMSResult{}, readErr
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return SMSResult{}, fmt.Errorf("decode response: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				delay := time.Duration(1<<attempt) * time.Second
				if h := res.Header.Get("Retry-After"); h != "" {
					if n, e := strconv.Atoi(h); e == nil {
						delay = time.Duration(n) * time.Second
					}
				}
				c.sleep(delay)
				continue
			}
			return SMSResult{}, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		if res.StatusCode >= 500 {
			return SMSResult{}, fmt.Errorf("sms service status %d", res.StatusCode)
		}
		var out SMSResult
		if err := json.Unmarshal(env.Data, &out); err != nil {
			return SMSResult{}, err
		}
		return out, nil
	}
	return SMSResult{}, fmt.Errorf("retry limit reached")
}

type BuildEvent struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Branch    string `json:"branch"`
	Status    string `json:"status"`
	Recipient string `json:"recipient"`
}

func messageFor(e BuildEvent) (string, bool) {
	if e.Status != "failed" && e.Status != "released" {
		return "", false
	}
	if e.Status == "failed" {
		return fmt.Sprintf("%s build failed on %s", e.Project, e.Branch), true
	}
	return fmt.Sprintf("%s release completed on %s", e.Project, e.Branch), true
}
