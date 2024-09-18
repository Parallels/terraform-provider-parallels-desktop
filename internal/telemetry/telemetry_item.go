package telemetry

import (
	"context"
	"crypto"
	"encoding/base64"
	"fmt"
	"runtime"
)

type TelemetryItem struct {
	UserID     string
	DeviceId   string
	Type       string
	Properties map[string]interface{}
	Options    map[string]interface{}
}

func NewTelemetryItem(ctx context.Context, userId string, eventType TelemetryEvent, mode TelemetryEventMode, properties, options map[string]interface{}) TelemetryItem {
	item := TelemetryItem{
		Type:       fmt.Sprintf("%s::%s", string(eventType), string(mode)),
		Properties: properties,
		Options:    options,
	}
	if item.Properties == nil {
		item.Properties = make(map[string]interface{})
	}
	if item.Options == nil {
		item.Options = make(map[string]interface{})
	}

	// Adding default properties
	item.Properties["os"] = runtime.GOOS
	item.Properties["architecture"] = runtime.GOARCH

	hash := crypto.SHA256.New()
	hash.Write([]byte(userId))
	hashedUserId := base64.StdEncoding.EncodeToString(hash.Sum(nil))

	if len(hashedUserId) > 10 {
		item.Properties["user_id"] = hashedUserId
	}

	return item
}
