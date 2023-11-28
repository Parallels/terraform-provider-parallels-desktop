package vmspecs

import (
	"context"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type VmSpecs struct {
	Force      types.Bool   `tfsdk:"force"`
	CpuCount   types.String `tfsdk:"cpu_count"`
	MemorySize types.String `tfsdk:"memory_size"`
	DiskSize   types.String `tfsdk:"disk_size"`
}

func (s *VmSpecs) Schema() map[string]schema.Attribute {
	return SchemaBlock.Attributes
}

func (s *VmSpecs) Elements() []attr.Value {
	attrs := []attr.Value{
		s.CpuCount,
		s.MemorySize,
		s.DiskSize,
	}

	return attrs
}

func (s *VmSpecs) ElementType() attr.Type {
	return basetypes.ObjectType{
		AttrTypes: map[string]attr.Type{
			"force":       types.BoolType,
			"cpu_count":   types.StringType,
			"memory_size": types.StringType,
			"disk_size":   types.StringType,
		},
	}
}

func (s *VmSpecs) MapObject() (basetypes.ObjectValue, diag.Diagnostics) {
	attributeTypes := make(map[string]attr.Type)
	attributeTypes["force"] = types.BoolType
	attributeTypes["cpu_count"] = types.StringType
	attributeTypes["memory_size"] = types.StringType
	attributeTypes["disk_size"] = types.StringType

	attrs := map[string]attr.Value{}
	attrs["force"] = s.Force
	attrs["cpu_count"] = s.CpuCount
	attrs["memory_size"] = s.MemorySize
	attrs["disk_size"] = s.DiskSize

	return types.ObjectValue(attributeTypes, attrs)
}

func (s *VmSpecs) Apply(ctx context.Context, config apiclient.HostConfig, vm apimodels.VirtualMachine) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}

	if vm.State != "stopped" {
		diagnostic.AddError("vm must be stopped", "vm must be stopped")
		return diagnostic
	}

	vmConfigRequest := apimodels.NewVmConfigRequest(vm.User)
	if s.CpuCount.ValueString() != "" {
		op := apimodels.NewVmConfigRequestOperation(vmConfigRequest)
		op.WithGroup("cpu")
		op.WithOperation("set")
		op.WithValue(s.CpuCount.ValueString())
		op.Append()
	}

	if s.MemorySize.ValueString() != "" {
		op := apimodels.NewVmConfigRequestOperation(vmConfigRequest)
		op.WithGroup("memory")
		op.WithOperation("set")
		op.WithValue(s.MemorySize.ValueString())
		op.Append()
	}

	_, resultDiagnostic := apiclient.ConfigureMachine(ctx, config, vm.ID, vmConfigRequest)
	if resultDiagnostic.HasError() {
		diagnostic.Append(resultDiagnostic...)
		return diagnostic
	}

	return diagnostic
}
