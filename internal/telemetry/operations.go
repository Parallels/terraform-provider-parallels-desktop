package telemetry

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// TrackOperation records one completed terraform operation (resource CRUD or
// datasource read). Call it through defer so the event is emitted on every
// exit path, including error returns and panics:
//
//	defer telemetry.TrackOperation(ctx, license, telemetry.EventCloneVm, telemetry.ModeCreate, &resp.Diagnostics, time.Now())()
//
// Exactly one event is sent per operation. It carries only sanitized
// quality signals — outcome, error count and duration — never diag text,
// which may contain hostnames, IPs or license material.
func TrackOperation(ctx context.Context, license string, eventType TelemetryEvent, mode TelemetryEventMode, diags *diag.Diagnostics, started time.Time) {
	svc := Get(ctx)
	if !svc.EnableTelemetry {
		return
	}

	item := NewTelemetryItem(ctx, license, eventType, mode, nil, nil)

	outcome := "success"
	errorCount := 0
	if diags != nil {
		for _, d := range *diags {
			if d.Severity() == diag.SeverityError {
				errorCount++
			}
		}
		if errorCount > 0 {
			outcome = "error"
		}
	}
	item.Properties["outcome"] = outcome
	item.Properties["error_count"] = errorCount
	item.Properties["duration_ms"] = time.Since(started).Milliseconds()

	svc.TrackEvent(ctx, item)
}

// sessionOnce keeps the session event to exactly one per provider process.
var sessionOnce = &sync.Once{}

// TrackSession emits one provider::configure event per provider process.
// It is the cleanest "a machine ran this provider" signal (one event per
// terraform run) and carries the Terraform CLI version and CI-ness.
func TrackSession(ctx context.Context, license string, terraformVersion string) {
	sessionOnce.Do(func() {
		svc := Get(ctx)
		if !svc.EnableTelemetry {
			return
		}

		item := NewTelemetryItem(ctx, license, EventProvider, ModeConfigure, nil, nil)
		item.Properties["terraform_version"] = terraformVersion
		item.Properties["is_ci"] = isCIEnvironment()

		svc.TrackEvent(ctx, item)
	})
}

func isCIEnvironment() bool {
	return os.Getenv("CI") != "" || os.Getenv("TF_IN_AUTOMATION") != ""
}
