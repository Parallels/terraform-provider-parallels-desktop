package virtualmachine

import (
	"context"
	"fmt"
	"time"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/models"
	data_models "terraform-provider-parallels-desktop/internal/virtualmachine/models"
	"terraform-provider-parallels-desktop/internal/virtualmachine/schemas"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &VirtualMachinesDataSource{}
	_ datasource.DataSourceWithConfigure = &VirtualMachinesDataSource{}
)

func NewVirtualMachinesDataSource() datasource.DataSource {
	return &VirtualMachinesDataSource{}
}

type VirtualMachinesDataSource struct {
	provider *models.ParallelsProviderModel
}

func (d *VirtualMachinesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.provider = data
}

func (d *VirtualMachinesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (d *VirtualMachinesDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schemas.VirtualMachineDataSourceSchemaV2
}

func (d *VirtualMachinesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data data_models.VirtualMachinesDataSourceModelV2

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// selecting if this is a standalone host or an orchestrator
	isOrchestrator := false
	var host string
	if data.Orchestrator.ValueString() != "" {
		isOrchestrator = true
		host = data.Orchestrator.ValueString()
	} else {
		host = data.Host.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		Host:                 host,
		IsOrchestrator:       isOrchestrator,
		License:              d.provider.License.ValueString(),
		Authorization:        data.Authenticator,
		DisableTlsValidation: d.provider.DisableTlsValidation.ValueBool(),
	}

	retryAttempts := 10
	for {
		needsRefresh := false
		data.Machines = make([]data_models.VirtualMachineModelV2, 0)
		vms, diag := apiclient.GetVms(ctx, hostConfig, data.Filter.FieldName.ValueString(), data.Filter.Value.ValueString())
		if diag.HasError() {
			diag.Append(diag...)
			return
		}

		for _, machine := range vms {
			stateMachine := data_models.VirtualMachineModelV2{
				HostIP:             types.StringValue("-"),
				ID:                 types.StringValue(machine.ID),
				Name:               types.StringValue(machine.Name),
				Description:        types.StringValue(machine.Description),
				OSType:             types.StringValue(machine.OS),
				State:              types.StringValue(machine.State),
				Home:               types.StringValue(machine.Home),
				ExternalIp:         types.StringValue(machine.HostExternalIpAddress),
				InternalIp:         types.StringValue(machine.InternalIpAddress),
				OrchestratorHostId: types.StringValue(machine.HostId),
			}
			if stateMachine.State.ValueString() == "running" && stateMachine.InternalIp.ValueString() == "" && data.WaitForNetworkUp.ValueBool() {
				needsRefresh = true
				time.Sleep(5 * time.Second) // wait for 5 seconds to give the network time to come up
				break
			}

			if stateMachine.InternalIp.ValueString() == "" {
				stateMachine.InternalIp = types.StringValue("-")
			}

			data.Machines = append(data.Machines, stateMachine)
		}

		if !needsRefresh {
			break
		}

		if retryAttempts == 0 {
			resp.Diagnostics.AddError("timeout waiting for network to be up", "timeout waiting for network to be up")
			return
		}

		retryAttempts--
	}

	if data.Machines == nil {
		data.Machines = make([]data_models.VirtualMachineModelV2, 0)
	}

	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
}
