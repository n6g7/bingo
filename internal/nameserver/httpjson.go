package nameserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func jsonEncode(value any) (io.ReadCloser, error) {
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(value)
	return io.NopCloser(buf), err
}

func jsonDecode(value io.ReadCloser, output any) error {
	defer value.Close()
	return json.NewDecoder(value).Decode(output)
}

type JsonClient struct {
	http.Client
	CsrfToken string
}

func (jc *JsonClient) DoJSON(req *http.Request, requestBody any, responseBody any) error {
	// Marshall request body if provided
	if requestBody != nil {
		encodedBody, err := jsonEncode(requestBody)
		if err != nil {
			return fmt.Errorf("failed to encode request body: %w", err)
		}
		req.Body = encodedBody
		req.Header.Set("Content-Type", "application/json")
	}

	// Request json response
	req.Header.Set("Accept", "application/json")

	if jc.CsrfToken != "" {
		req.Header.Set("X-CSRF-TOKEN", jc.CsrfToken)
	}

	// Send request
	resp, err := jc.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status %s (failed to read body)", resp.Status)
		} else {
			return fmt.Errorf("status %s: %s", resp.Status, body)
		}
	}

	// Unmarshall response body if an output struct is provided
	if responseBody != nil {
		if err := jsonDecode(resp.Body, responseBody); err != nil {
			return fmt.Errorf("failed to decode response body: %w", err)
		}
	}

	return nil
}

func (jc *JsonClient) GetJSON(url string, responseBody any) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	return jc.DoJSON(req, nil, responseBody)
}

func (jc *JsonClient) PostJSON(url string, requestBody any, responseBody any) error {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	return jc.DoJSON(req, requestBody, responseBody)
}

func (jc *JsonClient) PutJSON(url string, requestBody any, responseBody any) error {
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	return jc.DoJSON(req, requestBody, responseBody)
}

func (jc *JsonClient) DeleteJSON(url string, requestBody any, responseBody any) error {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	return jc.DoJSON(req, requestBody, responseBody)
}
