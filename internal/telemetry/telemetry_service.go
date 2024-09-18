package telemetry

import (
	"context"
	"fmt"

	"github.com/amplitude/analytics-go/amplitude"
	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type TelemetryService struct {
	ctx             context.Context
	client          amplitude.Client
	EnableTelemetry bool
	CallBackChan    chan types.ExecuteResult
}

func (t *TelemetryService) TrackEvent(item TelemetryItem) {
	if !t.EnableTelemetry {
		tflog.Debug(t.ctx, "[Telemetry] Telemetry is disabled, ignoring event track")
		return
	}

	tflog.Debug(t.ctx, fmt.Sprintf("[Telemetry] Sending Amplitude Tracking event %s", item.Type))

	// Create a new event
	if len(item.UserID) < 5 {
		if item.DeviceId != "" {
			item.UserID = fmt.Sprintf("%s@%s", item.UserID, item.DeviceId)
		} else {
			item.UserID = fmt.Sprintf("%s@service", item.UserID)
		}
	}
	if len(item.DeviceId) < 5 {
		item.DeviceId = "service"
	}

	ev := amplitude.Event{
		UserID:          item.UserID,
		DeviceID:        item.DeviceId,
		EventType:       item.Type,
		EventProperties: item.Properties,
	}

	// Log the event
	t.client.Track(ev)
}

func (t *TelemetryService) Callback(result types.ExecuteResult) {
	if result.Code < 200 || result.Code >= 300 {
		tflog.Debug(t.ctx, fmt.Sprintf("[Telemetry] Failed to send event to Amplitude: %v", result.Message))
		if result.Code == 401 || result.Code == 403 || result.Message == "Invalid API key" {
			tflog.Error(t.ctx, "[Telemetry] Disabling telemetry as received invalid key")
			t.EnableTelemetry = false
		}
	} else {
		tflog.Debug(t.ctx, "[Telemetry] Event sent to Amplitude")
	}

	t.CallBackChan <- result
}

func (t *TelemetryService) Flush() {
	t.client.Flush()
}

func (t *TelemetryService) Close() {
	t.client.Shutdown()
}
