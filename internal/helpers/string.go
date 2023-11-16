package helpers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math"
	"strconv"
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

func ConvertByteToGigabyte(bytes float64) float64 {
	gb := float64(bytes) / 1024 / 1024 / 1024
	return math.Round(gb*100) / 100
}

func ConvertByteToMegabyte(bytes float64) float64 {
	mb := float64(bytes) / 1024 / 1024
	return math.Round(mb*100) / 100
}

func GetSizeByteFromString(s string) (float64, error) {
	s = strings.ToLower(s)
	if strings.Contains(s, "gb") || strings.Contains(s, "gi") {
		s = strings.ReplaceAll(s, "gb", "")
		s = strings.ReplaceAll(s, "gi", "")
		s = strings.TrimSpace(s)
		size, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return -1, err
		}
		return size * 1024 * 1024 * 1024, nil
	}
	if strings.Contains(s, "mb") || strings.Contains(s, "mi") {
		s = strings.ReplaceAll(s, "mb", "")
		s = strings.ReplaceAll(s, "mi", "")
		s = strings.TrimSpace(s)
		size, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return -1, err
		}
		return size * 1024 * 1024, nil
	}
	if strings.Contains(s, "kb") || strings.Contains(s, "ki") {
		s = strings.ReplaceAll(s, "kb", "")
		s = strings.ReplaceAll(s, "ki", "")
		s = strings.TrimSpace(s)
		size, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return -1, err
		}
		return size * 1024, nil
	}

	return -1, errors.New("invalid size")
}
