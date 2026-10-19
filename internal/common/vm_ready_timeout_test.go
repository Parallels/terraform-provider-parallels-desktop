package common

import (
	"context"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseVmReadyTimeout(t *testing.T) {
	tests := []struct {
		name    string
		value   types.String
		want    time.Duration
		wantErr bool
	}{
		{"null uses default", types.StringNull(), DefaultVmReadyTimeout, false},
		{"empty uses default", types.StringValue(""), DefaultVmReadyTimeout, false},
		{"valid duration", types.StringValue("30m"), 30 * time.Minute, false},
		{"invalid duration", types.StringValue("soon"), 0, true},
		{"non-positive duration", types.StringValue("-5m"), 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := ParseVmReadyTimeout(tt.value)
			if diags.HasError() != tt.wantErr || got != tt.want {
				t.Fatalf("got %v (err=%v), want %v (err=%v)", got, diags.HasError(), tt.want, tt.wantErr)
			}
		})
	}
}

func TestEnsureMachineStoppedNilVm(t *testing.T) {
	vm, diags := EnsureMachineStopped(context.Background(), apiclient.HostConfig{}, nil)
	if vm != nil || !diags.HasError() {
		t.Fatalf("expected nil vm and an error diagnostic, got %v %v", vm, diags)
	}
}
