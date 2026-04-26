package vclaim

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// envelope adalah wrapper standard untuk semua response VClaim v2.0.
//
//	{ "metaData": { "code": "200", "message": "OK" },
//	  "response": "<base64 AES-256-CBC ciphertext>" }
//
// Field response sengaja json.RawMessage — bisa berupa string
// (encrypted, kasus normal v2.0) atau object/null (kasus tertentu).
type envelope struct {
	MetaData metaData        `json:"metaData"`
	Response json.RawMessage `json:"response"`
}

type metaData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// doGet melakukan GET request, validasi metaData, dan mendekripsi
// response field ke target. target harus pointer ke struct/map yang
// JSON-tag nya match dengan struktur plaintext setelah decrypt.
//
// Return error wrapped dengan VClaimError + sentinel domain (jika ada)
// saat metaData.code menandakan error.
func (c *Client) doGet(ctx context.Context, path string, target any) error {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeaders(c.headers()).
		Get(path)
	if err != nil {
		return fmt.Errorf("HTTP GET %s: %w", path, err)
	}
	if resp.IsError() {
		return fmt.Errorf("HTTP GET %s status %d", path, resp.StatusCode())
	}
	return c.parseEnvelope(resp.Body(), target)
}

// doPost melakukan POST JSON, validasi metaData, dan mendekripsi response.
func (c *Client) doPost(ctx context.Context, path string, body any, target any) error {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeaders(c.headers()).
		SetBody(body).
		Post(path)
	if err != nil {
		return fmt.Errorf("HTTP POST %s: %w", path, err)
	}
	if resp.IsError() {
		return fmt.Errorf("HTTP POST %s status %d", path, resp.StatusCode())
	}
	return c.parseEnvelope(resp.Body(), target)
}

// parseEnvelope membaca raw response body, validasi metaData,
// decrypt response field, lalu unmarshal ke target.
//
// Behaviour khusus: kalau response bukan JSON string (mis. object),
// langsung di-unmarshal tanpa decrypt — ini fallback untuk response
// yang tidak dienkripsi (mis. mock test atau endpoint legacy).
func (c *Client) parseEnvelope(body []byte, target any) error {
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}

	if err := mapErrorCode(env.MetaData.Code, env.MetaData.Message); err != nil {
		return err
	}

	// target opsional — beberapa endpoint hanya peduli sukses/gagal
	// (mis. health check) tanpa butuh body.
	if target == nil {
		return nil
	}
	if len(env.Response) == 0 || string(env.Response) == "null" {
		return nil
	}

	// Cek apakah response berupa JSON string (encrypted) atau object/array.
	var asString string
	if err := json.Unmarshal(env.Response, &asString); err == nil {
		plain, err := c.decrypt(asString)
		if err != nil {
			return fmt.Errorf("decrypt response: %w", err)
		}
		if err := json.Unmarshal(plain, target); err != nil {
			return fmt.Errorf("unmarshal plaintext (%s): %w",
				strconv.Quote(string(plain)), err)
		}
		return nil
	}

	// Bukan string — unmarshal langsung (path fallback).
	if err := json.Unmarshal(env.Response, target); err != nil {
		return fmt.Errorf("unmarshal response object: %w", err)
	}
	return nil
}

// urlDate memformat time.Time ke "yyyy-MM-dd" yang dipakai BPJS URL path.
// Output strict ISO date — hari ada di zone WIB karena BPJS server di Jakarta.
func urlDate(t timeFormatter) string {
	return t.Format("2006-01-02")
}

// timeFormatter di-extract sebagai interface kecil supaya method endpoint
// gampang ditest dengan stub time.
type timeFormatter interface {
	Format(layout string) string
}
