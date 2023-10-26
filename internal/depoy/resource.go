package deploy

import (
	"context"
	"errors"
	"fmt"
	"terraform-provider-parallels/internal/models"
	"terraform-provider-parallels/internal/ssh"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &VirtualMachineStateResource{}
var _ resource.ResourceWithImportState = &VirtualMachineStateResource{}

func NewVirtualMachineStateResource() resource.Resource {
	return &VirtualMachineStateResource{}
}

// VirtualMachineStateResource defines the resource implementation.
type VirtualMachineStateResource struct {
	provider *models.ParallelsProviderModel
}

func (r *VirtualMachineStateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deploy"
}

func (r *VirtualMachineStateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = deployResourceSchema
}

func (r *VirtualMachineStateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		tflog.Info(ctx, "No provider data")
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
	tflog.Info(ctx, r.provider.License.ValueString())
	return
}

func (r *VirtualMachineStateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeployResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	sshClient, err := r.getSshClient(data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SSH client", err.Error())
		return
	}

	parallelsClient := NewParallelsServerClient(ctx, sshClient)
	// installing dependencies
	if err := parallelsClient.InstallDependencies(); err != nil {
		resp.Diagnostics.AddError("Error installing dependencies", err.Error())
		return
	}
	// installing parallels desktop
	if err := parallelsClient.InstallParallelsDesktop(); err != nil {
		resp.Diagnostics.AddError("Error installing parallels desktop", err.Error())
		return
	}
	// restarting parallels service
	if err := parallelsClient.RestartServer(); err != nil {
		resp.Diagnostics.AddError("Error restarting parallels service", err.Error())
		return
	}

	key := r.provider.License.ValueString()
	username := r.provider.MyAccountUser.ValueString()
	password := r.provider.MyAccountPassword.ValueString()

	// installing parallels license
	if err := parallelsClient.InstallLicense(key, username, password); err != nil {
		resp.Diagnostics.AddError("Error installing parallels license", err.Error())
		return
	}

	targetPort := "8080"
	if !data.ApiPort.IsNull() && data.ApiPort.ValueString() != "" {
		targetPort = data.ApiPort.ValueString()
	}

	apiVersion := "latest"
	if !data.InstallApiVersion.IsNull() && data.InstallApiVersion.ValueString() != "" {
		apiVersion = data.InstallApiVersion.ValueString()
	}

	// Installing parallels api service
	_, err = parallelsClient.InstallApiService(r.provider.License.ValueString(), apiVersion, targetPort)
	if err != nil {
		resp.Diagnostics.AddError("Error installing parallels api service", err.Error())
		return
	}
	currentVersion, err := parallelsClient.GetApiVersion()
	if err != nil {
		resp.Diagnostics.AddError("Error getting parallels api version", err.Error())
		return
	}

	apiData := ParallelsApi{
		Version:  types.StringValue(currentVersion),
		Protocol: types.StringValue("http"),
		Host:     types.StringValue(data.SshConnection.Host.ValueString()),
		Port:     types.StringValue(targetPort),
		User:     types.StringValue("root@localhost"),
	}
	password, err = parallelsClient.GenerateDefaultRootPassword()
	apiData.Password = types.StringValue(password)
	data.Api = apiData.MapObject()

	// getting parallels version
	if version, err := parallelsClient.GetVersion(); err != nil {
		resp.Diagnostics.AddError("Error getting parallels version", err.Error())
		return
	} else {
		data.CurrentVersion = types.StringValue(version)
	}

	// getting parallels license
	if license, err := parallelsClient.GetLicense(); err != nil {
		resp.Diagnostics.AddError("Error getting parallels license", err.Error())
		return
	} else {
		data.License = license.MapObject()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtualMachineStateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeployResourceModel

	tflog.Info(ctx, "Read request to see logs")
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	sshClient, err := r.getSshClient(data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SSH client", err.Error())
		return
	}

	parallelsClient := NewParallelsServerClient(ctx, sshClient)

	// getting parallels version
	if version, err := parallelsClient.GetVersion(); err != nil {
		resp.Diagnostics.AddError("Error getting parallels version", err.Error())
		return
	} else {
		data.CurrentVersion = types.StringValue(version)
	}

	// getting parallels license
	if license, err := parallelsClient.GetLicense(); err != nil {
		resp.Diagnostics.AddError("Error getting parallels license", err.Error())
		return
	} else {
		data.License = license.MapObject()
	}

	// Getting parallels latest api version
	if version, err := parallelsClient.GetApiVersion(); err != nil {
		planVersion := ParallelsApi{}
		if !data.Api.IsNull() {
			if diags := data.Api.As(ctx, &planVersion, basetypes.ObjectAsOptions{}); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			planVersion.Version = types.StringValue("-")
			data.Api = planVersion.MapObject()
		}
	} else {
		planVersion := ParallelsApi{}
		if !data.Api.IsNull() {
			if diags := data.Api.As(ctx, &planVersion, basetypes.ObjectAsOptions{}); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			planVersion.Version = types.StringValue(version)
			data.Api = planVersion.MapObject()
		}
	}

	tflog.Info(ctx, "Finished Reading")
	// Set refreshed state
	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VirtualMachineStateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DeployResourceModel

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sshClient, err := r.getSshClient(data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SSH client", err.Error())
		return
	}

	parallelsClient := NewParallelsServerClient(ctx, sshClient)
	// restart parallels service
	if err := parallelsClient.RestartServer(); err != nil {
		resp.Diagnostics.AddError("Error restarting parallels service", err.Error())
		return
	}

	if r.provider.License.ValueString() != "" {
		// Licenses are the same, no changes
		equal, err := parallelsClient.CompareLicenses(r.provider.License.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error comparing parallels licenses", err.Error())
			return
		}

		if !equal {
			currentLicense, err := parallelsClient.GetLicense()
			if err != nil {
				resp.Diagnostics.AddError("Error getting parallels license", err.Error())
				return
			}
			if currentLicense.State.ValueString() == "valid" {
				// deactivating parallels license
				if err := parallelsClient.DeactivateLicense(); err != nil {
					resp.Diagnostics.AddError("Error deactivating parallels license", err.Error())
					return
				}
			}
		}

		if r.provider.License.ValueString() != "" {
			// install parallels license
			key := r.provider.License.ValueString()
			username := r.provider.MyAccountUser.ValueString()
			password := r.provider.MyAccountPassword.ValueString()

			// installing parallels license
			if err := parallelsClient.InstallLicense(key, username, password); err != nil {
				resp.Diagnostics.AddError("Error installing parallels license", err.Error())
				return
			}
		}
	} else {
		if !data.License.IsNull() || !data.License.IsUnknown() {
			// deactivating parallels license
			if err := parallelsClient.DeactivateLicense(); err != nil {
				resp.Diagnostics.AddError("Error deactivating parallels license", err.Error())
				return
			}
		}
	}

	// getting parallels version
	if version, err := parallelsClient.GetVersion(); err != nil {
		resp.Diagnostics.AddError("Error getting parallels version", err.Error())
		return
	} else {
		data.CurrentVersion = types.StringValue(version)
	}

	// getting parallels license
	if license, err := parallelsClient.GetLicense(); err != nil {
		resp.Diagnostics.AddError("Error getting parallels license", err.Error())
		return
	} else {
		data.License = license.MapObject()
	}

	targetPort := "8080"
	if !data.ApiPort.IsNull() && data.ApiPort.ValueString() != "" {
		targetPort = data.ApiPort.ValueString()
	}

	targetApiVersion := "latest"
	if !data.InstallApiVersion.IsNull() && data.InstallApiVersion.ValueString() != "" {
		targetApiVersion = data.InstallApiVersion.ValueString()
	}

	_, err = parallelsClient.InstallApiService(r.provider.License.ValueString(), targetApiVersion, targetPort)
	if err != nil {
		resp.Diagnostics.AddError("Error installing parallels api service", err.Error())
		return
	}
	currentVersion, err := parallelsClient.GetApiVersion()
	if err != nil {
		resp.Diagnostics.AddError("Error getting parallels api version", err.Error())
		return
	}

	apiData := ParallelsApi{
		Version:  types.StringValue(currentVersion),
		Protocol: types.StringValue("http"),
		Host:     types.StringValue(data.SshConnection.Host.ValueString()),
		Port:     types.StringValue(targetPort),
		User:     types.StringValue("root@localhost"),
	}
	password, err := parallelsClient.GenerateDefaultRootPassword()
	apiData.Password = types.StringValue(password)
	data.Api = apiData.MapObject()

	diags = resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	return
}

func (r *VirtualMachineStateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeployResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	sshClient, err := r.getSshClient(data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SSH client", err.Error())
		return
	}

	parallelsService := NewParallelsServerClient(ctx, sshClient)

	// deactivating parallels license
	if err := parallelsService.DeactivateLicense(); err != nil {
		resp.Diagnostics.AddError("Error deactivating parallels license", err.Error())
		return
	}

	// uninstalling parallels desktop
	if err := parallelsService.UninstallParallelsDesktop(); err != nil {
		resp.Diagnostics.AddError("Error uninstalling parallels desktop", err.Error())
		return
	}

	// uninstalling dependencies
	if err := parallelsService.UninstallDependencies(); err != nil {
		resp.Diagnostics.AddError("Error uninstalling dependencies", err.Error())
		return
	}

	// Save data into Terraform state
	data.CurrentVersion = types.StringValue("-")
	data.License = types.ObjectUnknown(map[string]attr.Type{
		"state":      types.StringType,
		"key":        types.StringType,
		"restricted": types.BoolType,
	})

	if err := parallelsService.UninstallApiService(); err != nil {
		resp.Diagnostics.AddError("Error uninstalling parallels api service", err.Error())
		return
	}

	data.Api = types.ObjectUnknown(map[string]attr.Type{
		"version":  types.StringType,
		"host":     types.StringType,
		"user":     types.StringType,
		"password": types.StringType,
	})

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VirtualMachineStateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *VirtualMachineStateResource) getSshClient(data DeployResourceModel) (*ssh.SshClient, error) {
	if data.SshConnection.Host.IsNull() {
		return nil, errors.New("Host is required")
	}
	if data.SshConnection.User.IsNull() {
		return nil, errors.New("User is required")
	}
	if data.SshConnection.Password.IsNull() && data.SshConnection.PrivateKey.IsNull() {
		return nil, errors.New("Password or PrivateKey is required")
	}

	// Create a new SSH client
	auth := ssh.SshAuthorization{
		User:       data.SshConnection.User.ValueString(),
		Password:   data.SshConnection.Password.ValueString(),
		PrivateKey: data.SshConnection.PrivateKey.ValueString(),
	}

	sshClient, err := ssh.NewSshClient(data.SshConnection.Host.ValueString(), data.SshConnection.HostPort.ValueString(), auth)
	if err != nil {
		return nil, err
	}
	if err := sshClient.Connect(); err != nil {
		return nil, err
	}

	return sshClient, nil
}
