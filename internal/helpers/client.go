package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"terraform-provider-parallels/internal/clientmodels"

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
	ctx context.Context
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

func NewHttpCaller(ctx context.Context) *HttpCaller {
	return &HttpCaller{
		ctx: ctx,
	}
}

func (c *HttpCaller) GetDataFromClient(url string, headers *map[string]string, auth HttpCallerAuth, destination interface{}) (*HttpCallerResponse, error) {
	return c.RequestDataToClient(HttpCallerVerbGet, url, headers, nil, auth, destination)
}

func (c *HttpCaller) PostDataToClient(url string, headers *map[string]string, data interface{}, auth HttpCallerAuth, destination interface{}) (*HttpCallerResponse, error) {
	return c.RequestDataToClient(HttpCallerVerbPost, url, headers, data, auth, destination)
}

func (c *HttpCaller) PutDataToClient(url string, headers *map[string]string, data interface{}, auth HttpCallerAuth, destination interface{}) (*HttpCallerResponse, error) {
	return c.RequestDataToClient(HttpCallerVerbPut, url, headers, data, auth, destination)
}

func (c *HttpCaller) DeleteDataFromClient(url string, headers *map[string]string, auth HttpCallerAuth, destination interface{}) (*HttpCallerResponse, error) {
	return c.RequestDataToClient(HttpCallerVerbDelete, url, headers, nil, auth, destination)
}

func (c *HttpCaller) RequestDataToClient(verb HttpCallerVerb, url string, headers *map[string]string, data interface{}, auth HttpCallerAuth, destination interface{}) (*HttpCallerResponse, error) {
	tflog.Info(c.ctx, fmt.Sprintf("%v data from %s", verb, url))
	var err error
	clientResponse := HttpCallerResponse{
		StatusCode: 0,
		Data:       nil,
	}

	if destination != nil {
		var destType = reflect.TypeOf(destination)
		if destType.Kind() != reflect.Ptr {
			return &clientResponse, errors.New("dest must be a pointer type")
		}
	}

	if url == "" {
		return &clientResponse, errors.New("url cannot be empty")
	}

	client := http.DefaultClient
	var req *http.Request

	if data != nil {
		reqBody, err := json.Marshal(data)
		if err != nil {
			return &clientResponse, fmt.Errorf("error marshalling data, err: %v", err)
		}
		req, err = http.NewRequest(verb.String(), url, bytes.NewBuffer(reqBody))
	} else {
		req, err = http.NewRequest(verb.String(), url, nil)
	}

	if err != nil {
		return &clientResponse, fmt.Errorf("error creating request, err: %v", err)
	}
	if req == nil {
		return &clientResponse, fmt.Errorf("request is nil")
	}

	tflog.Info(c.ctx, fmt.Sprintf("Request: %v", req))

	if auth.BearerToken != "" {
		tflog.Info(c.ctx, fmt.Sprintf("Setting Authorization header to Bearer %s", auth.BearerToken))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", auth.BearerToken))
	} else if auth.ApiKey != "" {
		req.Header.Set("X-Api-Key", auth.ApiKey)
	}

	req.Header.Set("Content-Type", "application/json")
	if headers != nil && len(*headers) > 0 {
		for k, v := range *headers {
			req.Header.Set("X-Filter", fmt.Sprintf("%s=%s", k, v))
		}
	}

	response, err := client.Do(req)
	clientResponse.StatusCode = response.StatusCode

	if err != nil {
		return &clientResponse, fmt.Errorf("error %s data on %s, err: %v", verb, url, err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var errMsg clientmodels.APIErrorResponse
		body, bodyErr := io.ReadAll(response.Body)
		if bodyErr == nil {
			if err := json.Unmarshal(body, &errMsg); err == nil {
				clientResponse.ApiError = &errMsg
			}
		}

		return &clientResponse, fmt.Errorf("error %s data from %s, status code: %d", verb, url, response.StatusCode)
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

func (c *HttpCaller) GetJwtToken(baseUrl, username, password string) (string, error) {
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
	tflog.Info(c.ctx, "Getting token from %s"+baseUrl+"/api/v1/auth/token with username"+username+" and password"+password+"")

	var tokenResponse clientmodels.TokenLoginResponse
	if _, err := c.PostDataToClient(baseUrl+"/api/v1/auth/token", nil, tokenRequest, HttpCallerAuth{}, &tokenResponse); err != nil {
		return "", err
	}
	return tokenResponse.Token, nil
}

func (c *HttpCaller) GetFileFromUrl(fileUrl string, destinationPath string) error {
	// Create the file in the tmp folder
	file, err := os.Create(destinationPath)
	if err != nil {
		return err
	}

	defer file.Close()

	// Download the file from the URL
	resp, err := http.Get(fileUrl)
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
	if strings.HasPrefix(url, "/") {
		url = url[1:]
	}
	if strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}

	return url
}
