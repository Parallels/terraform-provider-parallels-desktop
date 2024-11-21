package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/amplitude/analytics-go/amplitude"
	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	globalTelemetryService *TelemetryService
	lock                          = &sync.Mutex{}
	AMPLITUDE_API_KEY      string = ""
	VERSION                       = ""
)

func New(context context.Context) *TelemetryService {
	svc := &TelemetryService{
		EnableTelemetry: true,
		ctx:             context,
		CallBackChan:    make(chan types.ExecuteResult),
	}

	// Getting the code inbuilt api key
	key := AMPLITUDE_API_KEY

	if key == "" {
		tflog.Warn(context, "Telemetry disabled as no API key found")
		svc.EnableTelemetry = false
		return svc
	}

	config := amplitude.NewConfig(key)
	config.FlushQueueSize = 100
	config.FlushInterval = time.Second * 3
	// adding a callback to read what is the status
	config.ExecuteCallback = func(result types.ExecuteResult) {
		svc.Callback(result)
	}

	svc.client = amplitude.NewClient(config)

	globalTelemetryService = svc
	return svc
}

func Get(context context.Context) *TelemetryService {
	if globalTelemetryService == nil {
		lock.Lock()

		globalTelemetryService = New(context)
		lock.Unlock()
		return globalTelemetryService
	}

	return globalTelemetryService
}

func TrackEvent(context context.Context, item TelemetryItem) {
	svc := Get(context)
	if !svc.EnableTelemetry {
		return
	}

	svc.TrackEvent(item)
}
