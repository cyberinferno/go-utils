package utils

import (
	"bytes"
	"io"
	"net/http"
)

// SendDiscordNotification sends a message to a Discord channel via its webhook URL.
// The request is sent asynchronously; errors are ignored. The webhook URL must be
// valid and the Discord API must be reachable.
//
// Parameters:
//   - webhook: The Discord webhook URL to POST to
//   - content: The message content to send (used as the "content" field in the JSON body)
func SendDiscordNotification(webhook string, content string) {
	data := []byte(`{"content": "` + content + `"}`)
	req, err := http.NewRequest("POST", webhook, bytes.NewBuffer(data))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
}
