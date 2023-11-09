package helpers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"terraform-provider-parallels-desktop/internal/constants"
)

func Sha256Hash(input string) string {
	hashedPassword := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hashedPassword[:])
}

func Base64Encode(input string) string {
	return base64.StdEncoding.EncodeToString([]byte(input))
}

func Base64Decode(input string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func GetHostUrl(host string) string {
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}

	host = strings.TrimSuffix(host, "/")

	if !strings.Contains(host, ":") {
		host = host + ":" + constants.DefaultApiPort
	}
	return host
}

func GetHostApiBaseUrl(host string) string {
	return strings.TrimSuffix(GetHostUrl(host)+constants.API_PREFIX, "/")
}
