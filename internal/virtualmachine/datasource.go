package virtualmachine

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"terraform-provider-parallels/internal/clientmodels"
	"terraform-provider-parallels/internal/constants"
	"terraform-provider-parallels/internal/helpers"
	"terraform-provider-parallels/internal/models"

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
	host   string
	client *http.Client
}

func (d *VirtualMachinesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*models.ParallelsProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.host = config.License.String()
	d.client = &http.Client{}
}

func (d *VirtualMachinesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (d *VirtualMachinesDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = virtualMachineDataSourceSchema
}

func (d *VirtualMachinesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state virtualMachinesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if state.Host.IsUnknown() {
		resp.Diagnostics.AddError("host is required", "host is required")
		return
	}

	baseUrl := state.Host.ValueString()
	if strings.HasSuffix(baseUrl, "/") {
		baseUrl = baseUrl[:len(baseUrl)-1]
	}
	baseUrl = fmt.Sprintf("%s%s%s", baseUrl, constants.API_PREFIX, "/machines")

	var jsonResponse []clientmodels.VirtualMachine

	caller := helpers.NewHttpCaller(ctx)
	_, err := caller.GetDataFromClient(baseUrl, nil, helpers.HttpCallerAuth{}, &jsonResponse)
	if err != nil {
		resp.Diagnostics.AddError("error getting machines", err.Error())
		return
	}

	for _, machine := range jsonResponse {
		stateMachine := virtualMachineModel{
			HostIP:      types.StringValue("-"),
			ID:          types.StringValue(machine.ID),
			Name:        types.StringValue(machine.Name),
			Description: types.StringValue(machine.Description),
			OSType:      types.StringValue(machine.OS),
			State:       types.StringValue(machine.State),
			Home:        types.StringValue(machine.Home),
		}
		if !state.Filter.Id.IsUnknown() || !state.Filter.Name.IsUnknown() || !state.Filter.State.IsUnknown() {
			if !state.Filter.Id.IsNull() {
				// Compile the regex pattern
				regex, err := regexp.Compile(state.Filter.Id.ValueString())
				if err != nil {
					resp.Diagnostics.AddError("error compiling ID regex", err.Error())
					return
				}
				if regex.MatchString(machine.ID) {
					state.Machines = append(state.Machines, stateMachine)
					continue
				}
			}
			if !state.Filter.Name.IsNull() {
				// Compile the regex pattern
				regex, err := regexp.Compile(state.Filter.Name.ValueString())
				if err != nil {
					resp.Diagnostics.AddError("error compiling Name regex", err.Error())
					return
				}
				if regex.MatchString(machine.Name) {
					state.Machines = append(state.Machines, stateMachine)
					continue
				}
			}
			if !state.Filter.State.IsNull() {
				// Compile the regex pattern
				regex, err := regexp.Compile(state.Filter.State.ValueString())
				if err != nil {
					resp.Diagnostics.AddError("error compiling State regex", err.Error())
					return
				}
				if regex.MatchString(machine.State) {
					state.Machines = append(state.Machines, stateMachine)
					continue
				}
			}
		} else {
			state.Machines = append(state.Machines, stateMachine)
		}
	}

	if state.Machines == nil {
		state.Machines = make([]virtualMachineModel, 0)
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
}
