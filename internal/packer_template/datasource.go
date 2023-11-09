package packertemplate

import (
	"context"
	"fmt"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &PackerTemplateDataSource{}
	_ datasource.DataSourceWithConfigure = &PackerTemplateDataSource{}
)

func NewPackerTemplateDataSource() datasource.DataSource {
	return &PackerTemplateDataSource{}
}

type PackerTemplateDataSource struct {
	provider *models.ParallelsProviderModel
}

func (d *PackerTemplateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PackerTemplateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_packer_template"
}

func (d *PackerTemplateDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = packerTemplateDataSourceSchema
}

func (d *PackerTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data packerTemplateDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		Host:          data.Host.ValueString(),
		License:       d.provider.License.ValueString(),
		Authorization: data.Authenticator,
	}

	templates, diag := apiclient.GetPackerTemplates(ctx, hostConfig, data.Filter.FieldName.ValueString(), data.Filter.Value.ValueString())
	if diag.HasError() {
		diag.Append(diag...)
		return
	}

	for _, template := range templates {
		templateModel := packerTemplateModel{
			ID:          types.StringValue(template.ID),
			Name:        types.StringValue(template.Name),
			Description: types.StringValue(template.Description),
			Specs:       make(map[string]types.String),
			Addons:      make([]types.String, 0),
			Defaults:    make(map[string]types.String),
			Internal:    types.BoolValue(template.Internal),
			Variables:   make(map[string]types.String),
		}
		for key, value := range template.Specs {
			templateModel.Specs[key] = types.StringValue(value)
		}
		for _, value := range template.Addons {
			templateModel.Addons = append(templateModel.Addons, types.StringValue(value))
		}
		for key, value := range template.Defaults {
			templateModel.Defaults[key] = types.StringValue(value)
		}
		for key, value := range template.Variables {
			templateModel.Variables[key] = types.StringValue(value)
		}

		data.Templates = append(data.Templates, templateModel)
	}

	if data.Templates == nil {
		data.Templates = make([]packerTemplateModel, 0)
	}

	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
}
