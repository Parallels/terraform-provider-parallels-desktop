package telemetry

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"runtime"
)

// LicenseGroupType is the Amplitude group used to roll events up by license,
// so license-level usage can be analysed alongside per-machine usage.
const LicenseGroupType = "license"

type TelemetryItem struct {
	UserID     string
	DeviceId   string
	Type       string
	Groups     map[string][]string
	Properties map[string]interface{}
	Options    map[string]interface{}
}

// NewTelemetryItem builds a telemetry event. The unique identity (Amplitude
// user_id/device_id) is the machine fingerprint, which counts one unique user
// per machine regardless of who runs terraform or which license is used.
//
// The license key is only ever sent hashed (license_hash property and the
// "license" group) for license-level analytics; an absent/empty license is
// simply omitted instead of producing a hash of the empty string.
func NewTelemetryItem(ctx context.Context, license string, eventType TelemetryEvent, mode TelemetryEventMode, properties, options map[string]interface{}) TelemetryItem {
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
	if VERSION != "" {
		item.Properties["version"] = VERSION
	}

	// Per-machine unique anonymous identity for Amplitude user_id/device_id.
	item.UserID = MachineFingerprint()
	item.DeviceId = item.UserID

	// License hash kept for license-level (group) analytics.
	if license != "" {
		hash := sha256.New()
		hash.Write([]byte(license))
		hashedLicense := base64.StdEncoding.EncodeToString(hash.Sum(nil))

		item.Properties["license_hash"] = hashedLicense
		item.Groups = map[string][]string{LicenseGroupType: {hashedLicense}}
	}

	return item
}
