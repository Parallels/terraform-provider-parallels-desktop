package sharedfolder

import (
	"context"
	"encoding/json"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type SharedFolder struct {
	Name        types.String `tfsdk:"name"`
	Path        types.String `tfsdk:"path"`
	Readonly    types.Bool   `tfsdk:"readonly"`
	Description types.String `tfsdk:"description"`
	Disabled    types.Bool   `tfsdk:"disabled"`
}

func (s *SharedFolder) Elements(ctx context.Context) []attr.Value {
	attrs := []attr.Value{
		s.Name,
		s.Path,
		s.Readonly,
		s.Description,
		s.Disabled,
	}

	return attrs
}

func (s *SharedFolder) ElementType(ctx context.Context) attr.Type {
	return basetypes.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":        types.StringType,
			"path":        types.StringType,
			"readonly":    types.BoolType,
			"description": types.StringType,
			"disabled":    types.BoolType,
		},
	}
}

func (s *SharedFolder) MapObject(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	attributeTypes := make(map[string]attr.Type)
	attributeTypes["name"] = types.StringType
	attributeTypes["path"] = types.StringType
	attributeTypes["readonly"] = types.BoolType
	attributeTypes["description"] = types.StringType
	attributeTypes["disabled"] = types.BoolType

	attrs := map[string]attr.Value{}
	attrs["name"] = s.Name
	attrs["path"] = s.Path
	attrs["readonly"] = s.Readonly
	attrs["description"] = s.Description
	attrs["disabled"] = s.Disabled

	return types.ObjectValue(attributeTypes, attrs)
}

func (s *SharedFolder) Add(ctx context.Context, config apiclient.HostConfig, vm apimodels.VirtualMachine) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}

	configSet := apimodels.NewVmConfigRequest(vm.User)
	op := apimodels.NewVmConfigRequestOperation(configSet)
	op.WithGroup("shared-folder")
	op.WithOperation("add")
	op.WithValue(s.Name.ValueString())
	if s.Path.ValueString() != "" {
		op.WithOption("path", s.Path.ValueString())
	}
	if s.Readonly.ValueBool() {
		op.WithOption("mode", "ro")
	} else {
		op.WithOption("mode", "rw")
	}
	if s.Description.ValueString() != "" {
		op.WithOption("shf-description", s.Description.ValueString())
	}
	if s.Disabled.ValueBool() {
		op.WithFlag("disable")
	} else {
		op.WithFlag("enable")
	}
	op.Append()

	response, diag := apiclient.ConfigureMachine(ctx, config, vm.ID, configSet)
	if diag.HasError() {
		diagnostic.Append(diag...)
		return diagnostic
	}
	out, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		tflog.Error(ctx, "Error marshalling response: "+err.Error())
	}
	tflog.Info(ctx, "response: "+string(out))

	if response == nil {
		diagnostic.AddError("There was an error adding the shared folder", "response is nil")
		return diagnostic
	}
	for _, op := range response.Operations {
		tflog.Info(ctx, "op error: "+op.Error)
		if op.Error != "" {
			diagnostic.AddError("There was an error adding the shared folder", op.Error)
			return diagnostic
		}
	}

	return diagnostic
}

func (s *SharedFolder) Update(ctx context.Context, config apiclient.HostConfig, vm apimodels.VirtualMachine) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}

	configSet := apimodels.NewVmConfigRequest(vm.User)
	op := apimodels.NewVmConfigRequestOperation(configSet)
	op.WithGroup("shared-folder")
	op.WithOperation("set")
	op.WithValue(s.Name.ValueString())
	if s.Path.ValueString() != "" {
		op.WithOption("path", s.Path.ValueString())
	}
	if s.Readonly.ValueBool() {
		op.WithOption("mode", "ro")
	} else {
		op.WithOption("mode", "rw")
	}
	if s.Description.ValueString() != "" {
		op.WithOption("shf-description", s.Description.ValueString())
	}
	if s.Disabled.ValueBool() {
		op.WithFlag("disabled")
	} else {
		op.WithFlag("enabled")
	}
	op.Append()

	response, diag := apiclient.ConfigureMachine(ctx, config, vm.ID, configSet)
	if diag.HasError() {
		diagnostic.Append(diag...)
		return diagnostic
	}
	out, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		tflog.Error(ctx, "Error marshalling response: "+err.Error())
	}
	tflog.Info(ctx, "response: "+string(out))

	if response == nil {
		diagnostic.AddError("There was an error updating the shared folder", "response is nil")
		return diagnostic
	}
	for _, op := range response.Operations {
		if op.Error != "" {
			diagnostic.AddError("There was an error updating the shared folder", op.Error)
			return diagnostic
		}
	}

	return diagnostic
}

func (s *SharedFolder) Delete(ctx context.Context, config apiclient.HostConfig, vm apimodels.VirtualMachine) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}

	configSet := apimodels.NewVmConfigRequest(vm.User)
	op := apimodels.NewVmConfigRequestOperation(configSet)
	op.WithGroup("shared-folder")
	op.WithOperation("delete")
	op.WithValue(s.Name.ValueString())
	op.Append()

	response, diag := apiclient.ConfigureMachine(ctx, config, vm.ID, configSet)
	if diag.HasError() {
		diagnostic.Append(diag...)
		return diagnostic
	}
	out, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		tflog.Error(ctx, "Error marshalling response: "+err.Error())
	}
	tflog.Info(ctx, "response: "+string(out))
	if response == nil {
		diagnostic.AddError("There was an error deleting the shared folder", "response is nil")
		return diagnostic
	}
	for _, op := range response.Operations {
		if op.Error != "" {
			diagnostic.AddError("There was an error deleting the shared folder", op.Error)
			return diagnostic
		}
	}

	return diagnostic
}
