package common

import (
	"errors"
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
		return nil, errors.New("invalid connection string format")
	}

	credentials := connStr[:atIndex]
	host := connStr[atIndex+1:]
	if credentials == "" {
		return nil, errors.New("credentials cannot be empty")
	}

	// Split host and parameters
	hostParts := strings.SplitN(host, "?", 2)
	host = hostParts[0]
	if host == "" {
		return nil, errors.New("host cannot be empty")
	}

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

	var authentication authenticator.Authentication
	// DevOps uses credentials without ':' as an API key. Keep accepting the
	// provider's legacy api_key:<key> format as well.
	username, password, hasPassword := strings.Cut(credentials, ":")
	switch {
	case !hasPassword:
		authentication = authenticator.Authentication{
			ApiKey: types.StringValue(credentials),
		}
	case strings.EqualFold(username, "api_key"):
		if password == "" {
			return nil, errors.New("API key cannot be empty")
		}
		authentication = authenticator.Authentication{
			ApiKey: types.StringValue(password),
		}
	default:
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

// FormatHostConnectionString serializes a parsed connection using DevOps's
// catalog syntax. Use the same host and credentials for lookup and remote pull.
func FormatHostConnectionString(config apiclient.HostConfig) string {
	credentials := config.Authorization.ApiKey.ValueString()
	if credentials == "" {
		credentials = config.Authorization.Username.ValueString() + ":" + config.Authorization.Password.ValueString()
	}
	return "host=" + credentials + "@" + config.Host
}
