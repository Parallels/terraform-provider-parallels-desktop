package prlctl

import (
	"context"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type PrlCtlCmd struct {
	Operation types.String      `tfsdk:"operation"`
	Flags     []types.String    `tfsdk:"flags"`
	Options   []PrlCtlCmdOption `tfsdk:"options"`
}

type PrlCtlCmdOption struct {
	Flag  string `tfsdk:"flag"`
	Value string `tfsdk:"value"`
}

func (s *PrlCtlCmd) Apply(ctx context.Context, config apiclient.HostConfig, vm apimodels.VirtualMachine) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}
	diagnostics := diag.Diagnostics{}
	refreshVm, diag := apiclient.GetVm(ctx, config, vm.ID)
	if diag.HasError() {
		diagnostics.Append(diag...)
		return diagnostics
	}

	if refreshVm.State != "stopped" {
		result, stateDiag := apiclient.SetMachineState(ctx, config, refreshVm.ID, apiclient.MachineStateOpStop)
		if stateDiag.HasError() {
			diagnostic.Append(stateDiag...)
		}
		if !result {
			diagnostic.AddError("error stopping vm", "error stopping vm")
		}
		tflog.Info(ctx, "Waiting for vm "+refreshVm.Name+" to stop")
	}

	vmConfigRequest := apimodels.NewVmConfigRequest(refreshVm.User)
	op := apimodels.NewVmConfigRequestOperation(vmConfigRequest)
	op.WithGroup("cmd")
	op.WithOperation(s.Operation.ValueString())

	for _, flag := range s.Flags {
		op.WithFlag(flag.ValueString())
	}
	for _, option := range s.Options {
		op.WithOption(option.Flag, option.Value)
	}

	op.Append()

	_, resultDiagnostic := apiclient.ConfigureMachine(ctx, config, refreshVm.ID, vmConfigRequest)
	if resultDiagnostic.HasError() {
		diagnostic.Append(resultDiagnostic...)
		return diagnostic
	}

	return diagnostic
}
