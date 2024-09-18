package provider

import (
	"context"

	"terraform-provider-parallels-desktop/internal/authorization"
	clonevm "terraform-provider-parallels-desktop/internal/clone_vm"
	deploy "terraform-provider-parallels-desktop/internal/deploy"
	"terraform-provider-parallels-desktop/internal/models"
	"terraform-provider-parallels-desktop/internal/remoteimage"
	"terraform-provider-parallels-desktop/internal/vagrantbox"
	"terraform-provider-parallels-desktop/internal/virtualmachine"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &ParallelsProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ParallelsProvider{
			version: version,
		}
	}
}

// ParallelsProvider is the provider implementation.
type ParallelsProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Metadata returns the provider type name.
func (p *ParallelsProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "parallels-desktop"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *ParallelsProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Parallels Desktop provider allows you to manage Parallels Desktop virtual machines in a remote environment.
You will need a business edition license to use this provider. To get your license, please visit [Parallels Desktop](https://www.parallels.com/products/desktop/).

You can also join our community on [Discord](https://discord.gg/aFsrjbkN) channel.`,
		Attributes: map[string]schema.Attribute{
			"license": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "Parallels Desktop Pro or Business license",
				Description:         "Parallels Desktop Pro or Business license",
			},
			"my_account_user": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Parallels Desktop My Account user",
				Description:         "Parallels Desktop My Account user",
			},
			"my_account_password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Parallels Desktop My Account password",
				Description:         "Parallels Desktop My Account password",
			},
		},
	}
}

func (p *ParallelsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config models.ParallelsProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.License.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("license"),
			"The license is required",
			"The provider needs a Parallels Desktop Pro or Business license to work",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	data := models.ParallelsProviderModel{
		License:           config.License,
		MyAccountUser:     config.MyAccountUser,
		MyAccountPassword: config.MyAccountPassword,
	}

	resp.DataSourceData = &data
	resp.ResourceData = &data
}

// DataSources defines the data sources implemented in the provider.
func (p *ParallelsProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		virtualmachine.NewVirtualMachinesDataSource,
		// packertemplate.NewPackerTemplateDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *ParallelsProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// virtualmachinestate.NewVirtualMachineStateResource,
		deploy.NewDeployResource,
		// packertemplate.NewPackerTemplateVirtualMachineResource,
		authorization.NewAuthorizationResource,
		vagrantbox.NewVagrantBoxResource,
		remoteimage.NewRemoteVmResource,
		clonevm.NewCloneVmResource,
	}
}
