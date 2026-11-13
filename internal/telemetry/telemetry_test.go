package telemetry

import (
	"context"
	"crypto/sha256"
	"sync"

	"encoding/base64"
	"testing"
	"time"

	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	mockamp "terraform-provider-parallels-desktop/internal/telemetry/test"
)

// withTestService points the package singleton at a mock-backed service for
// the duration of fn (TrackOperation/TrackSession resolve it via Get).
func withTestService(t *testing.T, fn func()) []*types.Event {
	t.Helper()

	mock := mockamp.NewMockAmplitudeClient()
	var captured []*types.Event
	mock.On("Track", func(args ...interface{}) {
		ev := args[0].(types.Event)
		captured = append(captured, &ev)
	})
	mock.On("Flush", func(args ...interface{}) {})

	previous := globalTelemetryService
	globalTelemetryService = &TelemetryService{
		client:          mock,
		EnableTelemetry: true,
		CallBackChan:    make(chan types.ExecuteResult, 32),
	}
	defer func() { globalTelemetryService = previous }()

	fn()
	return captured
}

func TestMachineFingerprintStableAndSized(t *testing.T) {
	first := MachineFingerprint()
	second := MachineFingerprint()

	if first != second {
		t.Fatalf("fingerprint must be stable per process: %q != %q", first, second)
	}
	// SHA-256 (32 bytes) as unpadded base64url is 43 chars.
	if len(first) != 43 {
		t.Fatalf("unexpected fingerprint length %d for %q", len(first), first)
	}
}

func TestNewTelemetryItemUsesFingerprintIdentity(t *testing.T) {
	license := "AAAAA-BBBBB-CCCCC-DDDDD-EEEEE"
	wantHash := sha256B64(license)

	item := NewTelemetryItem(context.Background(), license, EventDeploy, ModeCreate, nil, nil)

	if item.UserID != MachineFingerprint() {
		t.Errorf("UserID = %q, want machine fingerprint %q", item.UserID, MachineFingerprint())
	}
	if item.DeviceId != item.UserID {
		t.Errorf("DeviceId = %q, want same as UserID %q", item.DeviceId, item.UserID)
	}
	if item.DeviceId == "service" {
		t.Error("DeviceId must be a per-machine value, not the old constant")
	}

	if got := item.Properties["license_hash"]; got != wantHash {
		t.Errorf("license_hash = %v, want %v", got, wantHash)
	}
	if groups := item.Groups[LicenseGroupType]; len(groups) != 1 || groups[0] != wantHash {
		t.Errorf("Groups[%s] = %v, want [%s]", LicenseGroupType, groups, wantHash)
	}
	// The license itself must never appear in plaintext.
	for key, v := range item.Properties {
		if s, ok := v.(string); ok && s == license {
			t.Errorf("property %q leaks the plaintext license", key)
		}
	}

	if _, ok := item.Properties["os"]; !ok {
		t.Error("missing os property")
	}
	if _, ok := item.Properties["architecture"]; !ok {
		t.Error("missing architecture property")
	}
}

func TestNewTelemetryItemEmptyLicenseProducesNoSuperUser(t *testing.T) {
	item := NewTelemetryItem(context.Background(), "", EventDeploy, ModeRead, nil, nil)

	if _, ok := item.Properties["license_hash"]; ok {
		t.Error("empty license must not produce a license_hash (SHA256(\"\") mega-user)")
	}
	if len(item.Groups) != 0 {
		t.Errorf("empty license must not produce groups, got %v", item.Groups)
	}
	// Identity still works without a license.
	if item.UserID != MachineFingerprint() {
		t.Errorf("UserID = %q, want machine fingerprint", item.UserID)
	}
}

func TestTrackEventSendsIdentityAndFlushes(t *testing.T) {
	mock := mockamp.NewMockAmplitudeClient()

	var tracked types.Event
	flushed := 0
	mock.On("Track", func(args ...interface{}) {
		tracked = args[0].(types.Event)
	})
	mock.On("Flush", func(args ...interface{}) {
		flushed++
	})

	svc := &TelemetryService{
		client:          mock,
		EnableTelemetry: true,
		CallBackChan:    make(chan types.ExecuteResult, 32),
	}

	item := NewTelemetryItem(context.Background(), "LICENSE-KEY", EventCloneVm, ModeCreate, nil, nil)
	svc.TrackEvent(context.Background(), item)

	if tracked.EventType != item.Type {
		t.Errorf("EventType = %q, want %q", tracked.EventType, item.Type)
	}
	if tracked.UserID != MachineFingerprint() || tracked.DeviceID != MachineFingerprint() {
		t.Errorf("event identity = (%q, %q), want machine fingerprint on both", tracked.UserID, tracked.DeviceID)
	}
	if got := tracked.Groups[LicenseGroupType]; len(got) != 1 {
		t.Errorf("event Groups[%s] = %v, want one hashed license", LicenseGroupType, got)
	}
	if flushed == 0 {
		t.Error("TrackEvent must flush so short-lived provider processes deliver events")
	}
}

func TestCallbackNeverBlocks(t *testing.T) {
	// Unbuffered channel with no reader: a blocking send would hang forever.
	svc := &TelemetryService{
		EnableTelemetry: true,
		CallBackChan:    make(chan types.ExecuteResult),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 3 {
			svc.Callback(context.Background(), types.ExecuteResult{Code: 200})
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Callback blocked on an unread channel")
	}
}

func TestNewDisablesWithoutApiKey(t *testing.T) {
	svc := New(context.Background()) // AMPLITUDE_API_KEY is empty in tests
	if svc.EnableTelemetry {
		t.Fatal("telemetry must be disabled when no API key is injected")
	}
	// TrackEvent on a disabled service must be a no-op, not a nil-client panic.
	svc.TrackEvent(context.Background(), TelemetryItem{Type: "x"})
}

func TestTrackOperationReportsSanitizedOutcome(t *testing.T) {
	clean := withTestService(t, func() {
		TrackOperation(context.Background(), "lic", EventCloneVm, ModeCreate, &diag.Diagnostics{}, time.Now())
	})
	if len(clean) != 1 {
		t.Fatalf("expected 1 event, got %d", len(clean))
	}
	props := clean[0].EventProperties
	if props["outcome"] != "success" || props["error_count"] != 0 {
		t.Errorf("clean op = outcome:%v errors:%v, want success:0", props["outcome"], props["error_count"])
	}
	if _, ok := props["duration_ms"]; !ok {
		t.Error("missing duration_ms property")
	}

	var diags diag.Diagnostics
	diags.AddError("boom", "secret host 10.0.0.1")
	diags.AddWarning("warn", "not an error")
	failed := withTestService(t, func() {
		TrackOperation(context.Background(), "lic", EventCloneVm, ModeCreate, &diags, time.Now())
	})
	props = failed[0].EventProperties
	if props["outcome"] != "error" || props["error_count"] != 1 {
		t.Errorf("failed op = outcome:%v errors:%v, want error:1 (warnings must not count)", props["outcome"], props["error_count"])
	}
	for k, v := range props {
		if s, ok := v.(string); ok && (s == "boom" || s == "secret host 10.0.0.1") {
			t.Errorf("property %q leaks diag text: %q", k, s)
		}
	}
}

func TestTrackSessionFiresOnceWithVersionAndCI(t *testing.T) {
	t.Setenv("TF_IN_AUTOMATION", "1")
	sessionOnce = &sync.Once{}
	defer func() { sessionOnce = &sync.Once{} }()

	captured := withTestService(t, func() {
		TrackSession(context.Background(), "lic", "1.9.8")
		TrackSession(context.Background(), "lic", "1.9.8")
	})
	if len(captured) != 1 {
		t.Fatalf("session must fire exactly once per process, got %d events", len(captured))
	}
	if captured[0].EventType != "PD-TERRAFORM-PROVIDER::PROVIDER::CONFIGURE" {
		t.Errorf("EventType = %q", captured[0].EventType)
	}
	if captured[0].EventProperties["terraform_version"] != "1.9.8" {
		t.Errorf("terraform_version = %v", captured[0].EventProperties["terraform_version"])
	}
	if captured[0].EventProperties["is_ci"] != true {
		t.Errorf("is_ci = %v, want true with TF_IN_AUTOMATION", captured[0].EventProperties["is_ci"])
	}
}

func sha256B64(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
