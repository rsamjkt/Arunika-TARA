package vclaim

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

type envelope struct {
	MetaData metaData        `json:"metaData"`
	Response json.RawMessage `json:"response"`
}

type metaData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// doGet melakukan GET request, validasi metaData, dan mendekripsi response.
func (c *Client) doGet(ctx context.Context, path string, target any) error {
	ts := c.now().Unix()
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeaders(c.headers(ts)).
		Get(path)
	if err != nil {
		return fmt.Errorf("HTTP GET %s: %w", path, err)
	}
	if resp.IsError() {
		body := resp.Body()
		if len(body) > 200 {
			body = body[:200]
		}
		return fmt.Errorf("HTTP GET %s status %d: %s", path, resp.StatusCode(), body)
	}
	return c.parseEnvelope(resp.Body(), target, ts)
}

// doPost melakukan POST JSON, validasi metaData, dan mendekripsi response.
func (c *Client) doPost(ctx context.Context, path string, body any, target any) error {
	ts := c.now().Unix()
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeaders(c.headers(ts)).
		SetBody(body).
		Post(path)
	if err != nil {
		return fmt.Errorf("HTTP POST %s: %w", path, err)
	}
	if resp.IsError() {
		rb := resp.Body()
		if len(rb) > 200 {
			rb = rb[:200]
		}
		return fmt.Errorf("HTTP POST %s status %d: %s", path, resp.StatusCode(), rb)
	}
	return c.parseEnvelope(resp.Body(), target, ts)
}

// parseEnvelope membaca raw response body, validasi metaData,
// decrypt+decompress response field, lalu unmarshal ke target.
func (c *Client) parseEnvelope(body []byte, target any, ts int64) error {
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}

	if err := mapErrorCode(env.MetaData.Code, env.MetaData.Message); err != nil {
		return err
	}

	if target == nil {
		return nil
	}
	if len(env.Response) == 0 || string(env.Response) == "null" {
		return nil
	}

	// Response berupa JSON string (encrypted+compressed) atau object/array (legacy/mock).
	var asString string
	if err := json.Unmarshal(env.Response, &asString); err == nil {
		plain, err := c.decrypt(asString, ts)
		if err != nil {
			return fmt.Errorf("decrypt response: %w", err)
		}
		if err := json.Unmarshal(plain, target); err != nil {
			return fmt.Errorf("unmarshal plaintext (%s): %w",
				strconv.Quote(string(plain)), err)
		}
		return nil
	}

	// Bukan string — unmarshal langsung (mock/legacy path).
	if err := json.Unmarshal(env.Response, target); err != nil {
		return fmt.Errorf("unmarshal response object: %w", err)
	}
	return nil
}

func urlDate(t timeFormatter) string {
	return t.Format("2006-01-02")
}

type timeFormatter interface {
	Format(layout string) string
}
