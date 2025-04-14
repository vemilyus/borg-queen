// Copyright (C) 2025 Alex Katlein
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/config"
	"github.com/vemilyus/borg-queen/credentials/internal/model"
	"io"
	"net/http"
	"time"
)

type HttpClient struct {
	config  *config.Config
	timeout time.Duration
}

func New(config *config.Config) *HttpClient {
	return &HttpClient{
		config:  config,
		timeout: 1 * time.Second,
	}
}

func (hc *HttpClient) buildUrl(path model.Path) string {
	result := ""

	if hc.config.UseTls {
		result += "https://"
	} else {
		//goland:noinspection HttpUrlsUsage
		result += "http://"
	}

	result += hc.config.StoreHost

	if hc.config.StorePort != nil {
		result += fmt.Sprintf(":%d", *hc.config.StorePort)
	} else if hc.config.UseTls {
		result += ":443"
	} else {
		result += ":80"
	}

	result += path.String()

	return result
}

func (hc *HttpClient) request(ctx context.Context, method string, path model.Path, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		defer memguard.WipeBytes(bodyBytes)

		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, hc.buildUrl(path), bodyReader)
	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(req)
}

func handleResponse(resp *http.Response, result any) error {
	defer func() { _ = resp.Body.Close() }()

	buf := bytes.NewBuffer(make([]byte, 0, resp.ContentLength))
	defer memguard.WipeBytes(buf.Bytes())

	_, err := buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusBadRequest {
		errResp := model.ErrorResponse{}
		err = json.Unmarshal(buf.Bytes(), &errResp)

		return errors.New(errResp.Message)
	} else if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return errors.New(fmt.Sprintf("unexpected response: %d -> %s", resp.StatusCode, string(buf.Bytes())))
	}

	if result != nil {
		return json.Unmarshal(buf.Bytes(), result)
	}

	return nil
}

func (hc *HttpClient) Get(path model.Path, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	resp, err := hc.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return handleResponse(resp, result)
}

func (hc *HttpClient) Post(path model.Path, body any, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	resp, err := hc.request(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return handleResponse(resp, result)
}

func (hc *HttpClient) Delete(path model.Path, body any, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	resp, err := hc.request(ctx, http.MethodDelete, path, body)
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return handleResponse(resp, result)
}
