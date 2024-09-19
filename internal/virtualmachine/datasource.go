package virtualmachine

import (
	"context"
	"fmt"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/models"

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
	resp.Schema = virtualMachineDataSourceSchema
}

func (d *VirtualMachinesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data virtualMachinesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		Host:                 data.Host.ValueString(),
		License:              d.provider.License.ValueString(),
		Authorization:        data.Authenticator,
		DisableTlsValidation: d.provider.DisableTlsValidation.ValueBool(),
	}

	vms, diag := apiclient.GetVms(ctx, hostConfig, data.Filter.FieldName.ValueString(), data.Filter.Value.ValueString())
	if diag.HasError() {
		diag.Append(diag...)
		return
	}

	for _, machine := range vms {
		stateMachine := virtualMachineModel{
			HostIP:      types.StringValue("-"),
			ID:          types.StringValue(machine.ID),
			Name:        types.StringValue(machine.Name),
			Description: types.StringValue(machine.Description),
			OSType:      types.StringValue(machine.OS),
			State:       types.StringValue(machine.State),
			Home:        types.StringValue(machine.Home),
		}

		data.Machines = append(data.Machines, stateMachine)
	}

	if data.Machines == nil {
		data.Machines = make([]virtualMachineModel, 0)
	}

	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
}
