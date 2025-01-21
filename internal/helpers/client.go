package helpers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"terraform-provider-parallels-desktop/internal/clientmodels"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type HttpCallerVerb string

const (
	HttpCallerVerbGet    HttpCallerVerb = "GET"
	HttpCallerVerbPost   HttpCallerVerb = "POST"
	HttpCallerVerbPut    HttpCallerVerb = "PUT"
	HttpCallerVerbDelete HttpCallerVerb = "DELETE"
)

func (v HttpCallerVerb) String() string {
	return string(v)
}

type HttpCaller struct {
	disableTlsVerification bool
}

type HttpCallerAuth struct {
	BearerToken string
	ApiKey      string
}

type HttpCallerResponse struct {
	StatusCode int
	Data       interface{}
	ApiError   *clientmodels.APIErrorResponse
}

func NewHttpCaller(ctx context.Context, disableTlsVerification bool) *HttpCaller {
	return &HttpCaller{
		disableTlsVerification: disableTlsVerification,
	}
}

func (c *HttpCaller) GetDataFromClient(ctx context.Context, url string, headers *map[string]string, auth *HttpCallerAuth, destination interface{}) (*HttpCallerResponse, error) {
	return c.RequestDataToClient(ctx, HttpCallerVerbGet, url, headers, nil, auth, destination)
}

func (c *HttpCaller) PostDataToClient(ctx context.Context, url string, headers *map[string]string, data interface{}, auth *HttpCallerAuth, destination interface{}) (*HttpCallerResponse, error) {
	return c.RequestDataToClient(ctx, HttpCallerVerbPost, url, headers, data, auth, destination)
}

func (c *HttpCaller) PutDataToClient(ctx context.Context, url string, headers *map[string]string, data interface{}, auth *HttpCallerAuth, destination interface{}) (*HttpCallerResponse, error) {
	return c.RequestDataToClient(ctx, HttpCallerVerbPut, url, headers, data, auth, destination)
}

func (c *HttpCaller) DeleteDataFromClient(ctx context.Context, url string, headers *map[string]string, auth *HttpCallerAuth, destination interface{}) (*HttpCallerResponse, error) {
	return c.RequestDataToClient(ctx, HttpCallerVerbDelete, url, headers, nil, auth, destination)
}

func (c *HttpCaller) RequestDataToClient(ctx context.Context, verb HttpCallerVerb, url string, headers *map[string]string, data interface{}, auth *HttpCallerAuth, destination interface{}) (*HttpCallerResponse, error) {
	tflog.Info(ctx, fmt.Sprintf("%v data from %s", verb, url))
	var err error
	clientResponse := HttpCallerResponse{
		StatusCode: 0,
		Data:       nil,
	}

	if destination != nil {
		destType := reflect.TypeOf(destination)
		if destType.Kind() != reflect.Ptr {
			return &clientResponse, errors.New("dest must be a pointer type")
		}
	}

	if url == "" {
		return &clientResponse, errors.New("url cannot be empty")
	}

	client := http.DefaultClient
	if c.disableTlsVerification {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			// Set a timeout for the client for 15 seconds to avoid hanging
			Timeout: 60 * time.Second,
		}
	}
	var req *http.Request

	if data != nil {
		reqBody, err := json.MarshalIndent(data, "", "  ")
		tflog.Info(ctx, fmt.Sprintf("Request body: %s", reqBody))
		if err != nil {
			return &clientResponse, fmt.Errorf("error marshalling data, err: %v", err)
		}
		req, err = http.NewRequest(verb.String(), url, bytes.NewBuffer(reqBody))
		if err != nil {
			return &clientResponse, fmt.Errorf("error creating request, err: %v", err)
		}
	} else {
		req, err = http.NewRequest(verb.String(), url, nil)
		if err != nil {
			return &clientResponse, fmt.Errorf("error creating request, err: %v", err)
		}
	}

	if req == nil {
		return &clientResponse, errors.New("request is nil")
	}

	if auth != nil {
		if auth.BearerToken != "" {
			tflog.Info(ctx, "Setting Authorization header to Bearer "+auth.BearerToken)
			req.Header.Set("Authorization", "Bearer "+auth.BearerToken)
		} else if auth.ApiKey != "" {
			req.Header.Set("X-Api-Key", auth.ApiKey)
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-No-Cache", "true")
	if headers != nil && len(*headers) > 0 {
		for k, v := range *headers {
			req.Header.Set(k, v)
		}
	}

	response, err := client.Do(req)
	if err != nil {
		return &clientResponse, fmt.Errorf("error %s data on %s, err: %v", verb, url, err)
	}
	defer response.Body.Close()

	clientResponse.StatusCode = response.StatusCode
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var errMsg clientmodels.APIErrorResponse
		body, bodyErr := io.ReadAll(response.Body)
		if bodyErr == nil {
			if err := json.Unmarshal(body, &errMsg); err == nil {
				clientResponse.ApiError = &errMsg
			}
		}
		// set a clientResponse.ApiError if it is nil
		if clientResponse.ApiError == nil {
			clientResponse.ApiError = &clientmodels.APIErrorResponse{
				Code: int64(response.StatusCode),
			}
		}

		if clientResponse.ApiError.Message != "" {
			return &clientResponse, fmt.Errorf("error on %s data from %s, err: %v message: %v", verb, url, clientResponse.ApiError.Code, clientResponse.ApiError.Message)
		} else {
			return &clientResponse, fmt.Errorf("error on %s data from %s, status code: %d", verb, url, response.StatusCode)
		}
	}

	if destination != nil {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return &clientResponse, fmt.Errorf("error reading response body from %s, err: %v", url, err)
		}

		err = json.Unmarshal(body, destination)
		if err != nil {
			return &clientResponse, fmt.Errorf("error unmarshalling body from %s, err: %v ", url, err)
		}

		clientResponse.Data = destination
	}

	return &clientResponse, nil
}

func (c *HttpCaller) GetJwtToken(ctx context.Context, baseUrl, username, password string) (string, error) {
	if username == "" {
		return "", errors.New("username cannot be empty")
	}

	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	tokenRequest := clientmodels.TokenLoginRequest{
		Email:    username,
		Password: password,
	}

	tflog.Info(ctx, "Getting token from %s"+baseUrl+"/api/v1/auth/token with username"+username+" and password"+password+"")

	var tokenResponse clientmodels.TokenLoginResponse
	if _, err := c.PostDataToClient(ctx, baseUrl+"/api/v1/auth/token", nil, tokenRequest, nil, &tokenResponse); err != nil {
		return "", err
	}
	return tokenResponse.Token, nil
}

func (c *HttpCaller) GetFileFromUrl(ctx context.Context, fileUrl string, destinationPath string) error {
	// Validate and clean the destination path
	cleanPath := filepath.Clean(destinationPath)
	if strings.Contains(cleanPath, "..") {
		return errors.New("invalid destination path: path traversal attempt detected")
	}

	// Create the file in the tmp folder
	file, err := os.OpenFile(cleanPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	defer file.Close()

	// Download the file from the URL using a client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileUrl, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the file to disk
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func CleanUrlSuffixAndPrefix(url string) string {
	url = strings.TrimPrefix(url, "/")
	url = strings.TrimSuffix(url, "/")
	return url
}
