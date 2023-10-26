package packertemplate

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"terraform-provider-parallels/internal/clientmodels"
	"terraform-provider-parallels/internal/constants"
	"terraform-provider-parallels/internal/helpers"
	"terraform-provider-parallels/internal/models"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PackerTemplateVirtualMachineResource{}
var _ resource.ResourceWithImportState = &PackerTemplateVirtualMachineResource{}

func NewPackerTemplateVirtualMachineResource() resource.Resource {
	return &PackerTemplateVirtualMachineResource{}
}

// PackerTemplateVirtualMachineResource defines the resource implementation.
type PackerTemplateVirtualMachineResource struct {
	provider *models.ParallelsProviderModel
}

func (r *PackerTemplateVirtualMachineResource) getVms(ctx context.Context, host string, auth helpers.HttpCallerAuth, filterField, filterValue string) ([]clientmodels.VirtualMachine, error) {
	result := make([]clientmodels.VirtualMachine, 0)
	client := helpers.NewHttpCaller(ctx)

	filter := map[string]string{
		filterField: filterValue,
	}
	tflog.Info(ctx, "Getting filtered machines using filter "+filterField+" with value "+filterValue+" from "+host+"/api/v1/machines")
	url := fmt.Sprintf("%s/api/v1/machines", host)
	_, err := client.GetDataFromClient(url, &filter, auth, &result)

	if err != nil {
		return nil, err
	}
	tflog.Info(ctx, "Got "+strconv.Itoa(len(result))+" machines")

	return result, nil
}

func (r *PackerTemplateVirtualMachineResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_packer_template"
}

func (r *PackerTemplateVirtualMachineResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = getSchema(ctx)
}

func (r *PackerTemplateVirtualMachineResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*models.ParallelsProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *models.ParallelsProviderModel, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.provider = data
	return
}

func (r *PackerTemplateVirtualMachineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PackerVirtualMachineResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Setting the default timeout
	createTimeout, diags := data.Timeouts.Create(ctx, 60*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	auth, err := r.GetAuthenticator(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("error getting authenticator", err.Error())
		return
	}

	vm, err := r.getVms(ctx, data.Host.ValueString(), *auth, "Name", data.Name.String())
	if len(vm) > 0 {
		resp.Diagnostics.AddError("vm already exists", "vm already exists")
		return
	}

	client := helpers.NewHttpCaller(ctx)
	machineRequest := clientmodels.NewPackerTemplateVmRequest{
		Name:     data.Name.ValueString(),
		Template: data.Template.ValueString(),
	}
	if data.Owner.ValueString() != "" {
		machineRequest.Owner = data.Owner.ValueString()
	}
	if data.Specs != nil {
		if data.Specs.CpuCount.ValueString() != "" {
			machineRequest.Specs["cpu"] = data.Specs.CpuCount.ValueString()
		}
		if data.Specs.MemorySize.ValueString() != "" {
			machineRequest.Specs["memory"] = data.Specs.MemorySize.ValueString()
		}
		if data.Specs.DiskSize.ValueString() != "" {
			machineRequest.Specs["disk"] = data.Specs.DiskSize.ValueString()
		}
	}

	var machineResponse clientmodels.NewPackerTemplateVmResponse
	if clientResponse, err := client.PostDataToClient(fmt.Sprintf("%s/api/v1/machines", data.Host.ValueString()), nil, machineRequest, *auth, &machineResponse); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error creating vm: %v, api message: %s", err, clientResponse.ApiError.Message))
		}
		resp.Diagnostics.AddError("error creating vm", err.Error())

		return
	}

	data.ID = types.StringValue(machineResponse.ID)
	tflog.Info(ctx, "Created vm with id "+data.ID.ValueString())

	createdVM, err := r.getVms(ctx, data.Host.ValueString(), *auth, "ID", machineResponse.ID)
	if err != nil {
		resp.Diagnostics.AddError("error getting vm", err.Error())
		return
	}

	if len(createdVM) == 0 {
		resp.Diagnostics.AddError("vm was not found", "vm was not found")
		return
	}

	data.OsType = types.StringValue(createdVM[0].OS)
	var startResponse map[string]string
	if data.RunAfterCreate.ValueBool() {
		if _, err := client.GetDataFromClient(fmt.Sprintf("%s/api/v1/machines/%s/start", data.Host.ValueString(), data.ID.ValueString()), nil, *auth, &startResponse); err != nil {
			resp.Diagnostics.AddError("error starting vm", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PackerTemplateVirtualMachineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PackerVirtualMachineResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	// Setting the default timeout
	createTimeout, diags := data.Timeouts.Create(ctx, 60*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	auth, err := r.GetAuthenticator(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("error getting authenticator", err.Error())
		return
	}

	vms, err := r.getVms(ctx, data.Host.ValueString(), *auth, "ID", data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("error getting vm", err.Error())
		return
	}
	if len(vms) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(vms[0].Name)

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PackerTemplateVirtualMachineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PackerVirtualMachineResourceModel

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Setting the default timeout
	createTimeout, diags := data.Timeouts.Create(ctx, 60*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	auth, err := r.GetAuthenticator(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("error getting authenticator", err.Error())
		return
	}

	vms, err := r.getVms(ctx, data.Host.ValueString(), *auth, "ID", data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("error getting vm", err.Error())
		return
	}
	if len(vms) == 0 {
		resp.Diagnostics.AddError("vm was not found", "vm was not found")
		return
	}

	// Checking if the name is the same
	if data.Name.ValueString() != vms[0].Name {
		tflog.Info(ctx, "Updating vm name from "+vms[0].Name+" to "+data.Name.ValueString())
	}
	// Checking if the specs are the same
	if data.Specs != nil {
		client := helpers.NewHttpCaller(ctx)
		cpuCount := data.Specs.CpuCount.ValueString()
		memorySize := data.Specs.MemorySize.ValueString()

		hasChanges := false
		changes := clientmodels.VirtualMachineSetRequest{
			Owner:      vms[0].User,
			Operations: make([]*clientmodels.VirtualMachineSetOperation, 0),
		}

		vmCpuCount := strconv.Itoa(int(vms[0].Hardware.CPU.Cpus))
		if cpuCount != "" && vmCpuCount != cpuCount {
			tflog.Info(ctx, "Updating vm cpu count from "+vmCpuCount+" to "+cpuCount)
			hasChanges = true
			changes.Operations = append(changes.Operations, &clientmodels.VirtualMachineSetOperation{
				Group:     "cpu",
				Operation: "set",
				Value:     cpuCount,
			})
		}
		vmMemorySize := strings.ReplaceAll(vms[0].Hardware.Memory.Size, "Mb", "")
		if memorySize != "" && vmMemorySize != memorySize {
			tflog.Info(ctx, "Updating vm memory size from "+vmMemorySize+" to "+memorySize)
			hasChanges = true
			changes.Operations = append(changes.Operations, &clientmodels.VirtualMachineSetOperation{
				Group:     "memory",
				Operation: "set",
				Value:     memorySize,
			})
		}

		if hasChanges {
			var clientResponse clientmodels.VirtualMachineSetResponse
			if _, err := client.PostDataToClient(fmt.Sprintf("%s/api/v1/machines/%s/set", data.Host.ValueString(), data.ID.ValueString()), nil, changes, *auth, &clientResponse); err != nil {
				resp.Diagnostics.AddError("error updating vm", err.Error())
				return
			}
		}
	}

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PackerTemplateVirtualMachineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PackerVirtualMachineResourceModel
	//Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Setting the default timeout
	createTimeout, diags := data.Timeouts.Create(ctx, 60*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	auth, err := r.GetAuthenticator(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("error getting authenticator", err.Error())
		return
	}
	client := helpers.NewHttpCaller(ctx)

	vms, err := r.getVms(ctx, data.Host.ValueString(), *auth, "ID", data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("error getting vm", err.Error())
		return
	}
	if len(vms) == 0 {
		resp.Diagnostics.AddError("vm was not found", "vm was not found")
		return
	}

	if vms[0].State != "stopped" {
		if _, err := client.GetDataFromClient(fmt.Sprintf("%s/api/v1/machines/%s/stop", data.Host.ValueString(), data.ID.ValueString()), nil, *auth, nil); err != nil {
			resp.Diagnostics.AddError("error stopping vm", err.Error())
			return
		}
	}

	if _, err := client.DeleteDataFromClient(fmt.Sprintf("%s/api/v1/machines/%s", data.Host.ValueString(), data.ID.ValueString()), nil, *auth, nil); err != nil {
		resp.Diagnostics.AddError("error deleting vm", err.Error())
		return
	}

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PackerTemplateVirtualMachineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *PackerTemplateVirtualMachineResource) GetAuthenticator(ctx context.Context, state PackerVirtualMachineResourceModel) (*helpers.HttpCallerAuth, error) {
	client := helpers.NewHttpCaller(ctx)
	var auth helpers.HttpCallerAuth
	if state.Authenticator == nil {
		tflog.Info(ctx, "Authenticator is nil, using root access")
		password := r.provider.License.ValueString()
		token, err := client.GetJwtToken(state.Host.ValueString(), constants.RootUser, password)
		if err != nil {
			return nil, err
		}

		auth = helpers.HttpCallerAuth{
			BearerToken: token,
		}
		return &auth, nil
	} else {
		if state.Authenticator.Username.ValueString() != "" {
			password := state.Authenticator.Password.ValueString()
			token, err := client.GetJwtToken(state.Host.ValueString(), state.Authenticator.Username.ValueString(), password)
			if err != nil {
				return nil, err
			}
			auth = helpers.HttpCallerAuth{
				BearerToken: token,
			}
		} else {
			auth = helpers.HttpCallerAuth{
				ApiKey: state.Authenticator.ApiKey.ValueString(),
			}
		}
		return &auth, nil

	}
}
