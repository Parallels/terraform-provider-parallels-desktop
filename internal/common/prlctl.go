package common

import (
	"context"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/schemas/prlctl"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func PrlCtlBlockOnCreate(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, planPrlCtl []*prlctl.PrlCtlCmd) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	for _, cmd := range planPrlCtl {
		tflog.Info(ctx, "PrlCtl running command "+cmd.Operation.ValueString())
		diag := cmd.Apply(ctx, hostConfig, *vm)
		if diag.HasError() {
			diagnostics.Append(diag...)
			return diagnostics
		}
	}

	return diagnostics
}

func PrlCtlBlockOnUpdate(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, planPrlCtl []*prlctl.PrlCtlCmd) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	// the prlctl command acts always the same and it needs to be executed regardless of the saved state
	// so we just run it and return
	for _, cmd := range planPrlCtl {
		tflog.Info(ctx, "PrlCtl running command "+cmd.Operation.ValueString())
		diag := cmd.Apply(ctx, hostConfig, *vm)
		if diag.HasError() {
			diagnostics.Append(diag...)
		}
	}

	return diagnostics
}

func PrlCtlBlockHasChanges(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, planPrlCtl, statePrlCtl []*prlctl.PrlCtlCmd) bool {
	if planPrlCtl == nil && statePrlCtl == nil {
		return false
	}

	if planPrlCtl == nil && statePrlCtl != nil {
		return false
	}

	if planPrlCtl != nil && statePrlCtl == nil {
		return true
	}

	if planPrlCtl != nil && statePrlCtl != nil {
		if len(planPrlCtl) != len(statePrlCtl) {
			return true
		}

		for i, cmd := range planPrlCtl {
			if cmd.Operation.ValueString() != statePrlCtl[i].Operation.ValueString() {
				return true
			}
		}
	}

	return false
}
