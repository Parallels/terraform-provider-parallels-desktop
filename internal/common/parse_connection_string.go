package common

import (
	"fmt"
	"strings"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ParseHostConnectionString(connStr string) (*apiclient.HostConfig, error) {
	// Remove "host=" prefix if present
	connStr = strings.TrimPrefix(connStr, "host=")

	// Split at '@' to separate credentials and host
	atIndex := strings.LastIndex(connStr, "@")
	if atIndex == -1 {
		return nil, fmt.Errorf("invalid connection string format")
	}

	credentials := connStr[:atIndex]
	host := connStr[atIndex+1:]

	// Split host and parameters
	hostParts := strings.SplitN(host, "?", 2)
	host = hostParts[0]

	var params map[string]string
	if len(hostParts) == 2 {
		params = make(map[string]string)
		paramStr := hostParts[1]
		paramPairs := strings.Split(paramStr, "&")
		for _, pair := range paramPairs {
			keyValue := strings.SplitN(pair, "=", 2)
			if len(keyValue) == 2 {
				params[keyValue[0]] = keyValue[1]
			}
		}
	}
	host = strings.ToLower(host)
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}

	// Split credentials at ':' to get username and password
	credIndex := strings.Index(credentials, ":")
	if credIndex == -1 {
		return nil, fmt.Errorf("invalid credentials format")
	}
	username := credentials[:credIndex]
	password := credentials[credIndex+1:]

	var authentication authenticator.Authentication
	if strings.EqualFold(username, "api_key") {
		authentication = authenticator.Authentication{
			ApiKey: types.StringValue(password),
		}
	} else {
		authentication = authenticator.Authentication{
			Username: types.StringValue(username),
			Password: types.StringValue(password),
		}
	}

	// Populate the HostConfig
	hostConfig := &apiclient.HostConfig{
		Host:          host,
		Authorization: &authentication,
	}

	return hostConfig, nil
}
