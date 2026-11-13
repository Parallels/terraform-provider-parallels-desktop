package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/amplitude/analytics-go/amplitude"
	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// flushTimeout caps how long a tracked event may delay a terraform operation
// while it is flushed to Amplitude. Events not delivered in time fall back to
// the background flush interval and may be lost if the provider process exits
// first; telemetry must never meaningfully slow down an operation.
const flushTimeout = 3 * time.Second

type TelemetryService struct {
	client          amplitude.Client
	EnableTelemetry bool
	CallBackChan    chan types.ExecuteResult
}

func (t *TelemetryService) TrackEvent(ctx context.Context, item TelemetryItem) {
	if !t.EnableTelemetry {
		tflog.Debug(ctx, "[Telemetry] Telemetry is disabled, ignoring event track")
		return
	}

	tflog.Debug(ctx, "[Telemetry] Sending Amplitude Tracking event "+item.Type)

	ev := amplitude.Event{
		UserID:          item.UserID,
		DeviceID:        item.DeviceId,
		EventType:       item.Type,
		Groups:          item.Groups,
		EventProperties: item.Properties,
	}

	// Log the event
	t.client.Track(ev)

	// The terraform provider process is short-lived and is killed as soon as
	// the operation finishes, so anything still sitting in the SDK queue at
	// that point is lost. Flush synchronously (bounded) right after tracking
	// to guarantee delivery without delaying the operation noticeably.
	t.FlushWithTimeout(flushTimeout)
}

// FlushWithTimeout blocks up to timeout for queued events to be delivered.
func (t *TelemetryService) FlushWithTimeout(timeout time.Duration) {
	if !t.EnableTelemetry || t.client == nil {
		return
	}

	done := make(chan struct{})
	go func() {
		defer func() {
			// The Amplitude client may panic if shut down concurrently; never
			// let telemetry take down a terraform operation.
			_ = recover()
			close(done)
		}()
		t.client.Flush()
	}()

	select {
	case <-done:
	case <-time.After(timeout):
	}
}

func (t *TelemetryService) Callback(ctx context.Context, result types.ExecuteResult) {
	if result.Code < 200 || result.Code >= 300 {
		tflog.Debug(ctx, fmt.Sprintf("[Telemetry] Failed to send event to Amplitude: %v", result.Message))
		if result.Code == 401 || result.Code == 403 || result.Message == "Invalid API key" {
			tflog.Error(ctx, "[Telemetry] Disabling telemetry as received invalid key")
			t.EnableTelemetry = false
		}
	} else {
		tflog.Debug(ctx, "[Telemetry] Event sent to Amplitude")
	}

	// Never block the SDK callback goroutine: nothing in the provider reads
	// CallBackChan in production, and a blocked send leaks a goroutine per
	// flush (and would make Shutdown() hang forever).
	select {
	case t.CallBackChan <- result:
	default:
	}
}

func (t *TelemetryService) Flush() {
	t.client.Flush()
}

func (t *TelemetryService) Close() {
	t.client.Shutdown()
}
