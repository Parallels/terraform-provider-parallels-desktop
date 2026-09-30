package deploy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"terraform-provider-parallels-desktop/internal/deploy/schemas"
	"terraform-provider-parallels-desktop/internal/interfaces"
	"terraform-provider-parallels-desktop/internal/localclient"
	"terraform-provider-parallels-desktop/internal/models"
	"terraform-provider-parallels-desktop/internal/schemas/reverseproxy"
	"terraform-provider-parallels-desktop/internal/ssh"
	"terraform-provider-parallels-desktop/internal/telemetry"

	deploy_models "terraform-provider-parallels-desktop/internal/deploy/models"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &DeployResource{}
	_ resource.ResourceWithImportState = &DeployResource{}
)

func NewDeployResource() resource.Resource {
	return &DeployResource{}
}

// DeployResource defines the resource implementation.
type DeployResource struct {
	provider             *models.ParallelsProviderModel
	commandClientFactory func(context.Context, deploy_models.DeployResourceModelV3) (interfaces.ContextCommandClient, error)
}

func (r *DeployResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deploy"
}

func (r *DeployResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schemas.DeployResourceSchemaV3
}

func (r *DeployResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
}

func (r *DeployResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data deploy_models.DeployResourceModelV3
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventDeploy, telemetry.ModeCreate,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(ctx, telemetryEvent)

	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := deploymentConfig(ctx, &data, r.provider.License.ValueString()); err != nil {
		resp.Diagnostics.AddError("Invalid DevOps configuration", err.Error())
		return
	}
	if data.Orchestrator != nil {
		if data.Orchestrator.Orchestrator == nil {
			resp.Diagnostics.AddError("Invalid orchestrator configuration", "orchestrator connection details are required")
			return
		}
		if err := validateOrchestratorDetails(data.Orchestrator.Orchestrator); err != nil {
			resp.Diagnostics.AddError("Invalid orchestrator authentication", err.Error())
			return
		}
	}
	runClient, runClientError := r.commandClient(ctx, data)
	if runClientError != nil {
		resp.Diagnostics.AddError("Error creating command client", runClientError.Error())
		return
	}

	defer runClient.Close()

	parallelsClient := NewDevOpsServiceClient(ctx, runClient)

	dependencies, diag := r.installParallelsDesktop(ctx, parallelsClient, data.KeepAfterError.ValueBool())
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	defer func() {
		if resp.Diagnostics.HasError() && !data.Api.IsNull() && !data.Api.IsUnknown() {
			// Once configured, the service and database remain managed even if readiness or registration fails.
			preparePartialDeploymentState(ctx, &data, dependencies)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		}
	}()
	_, diag = r.installDevOpsService(ctx, &data, dependencies, parallelsClient)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	// getting parallels version
	if version, err := parallelsClient.GetVersion(ctx); err != nil {
		resp.Diagnostics.AddError("Error getting parallels version", err.Error())
		return
	} else {
		data.CurrentVersion = types.StringValue(version)
	}

	// getting git version
	if version, err := parallelsClient.GetGitVersion(ctx); err != nil {
		data.CurrentGitVersion = types.StringValue("-")
	} else {
		data.CurrentGitVersion = types.StringValue(version)
	}

	// getting packer version
	if version, err := parallelsClient.GetPackerVersion(ctx); err != nil {
		data.CurrentPackerVersion = types.StringValue("-")
	} else {
		data.CurrentPackerVersion = types.StringValue(version)
	}

	// getting Vagrant version
	if version, err := parallelsClient.GetVagrantVersion(ctx); err != nil {
		data.CurrentVagrantVersion = types.StringValue("-")
	} else {
		data.CurrentVagrantVersion = types.StringValue(version)
	}

	// getting parallels license
	if license, err := parallelsClient.GetLicense(ctx); err != nil {
		resp.Diagnostics.AddError("Error getting parallels license", err.Error())
		return
	} else {
		data.License = license.MapObject()
	}

	// Register with orchestrator if needed, otherwise set fields to known zero values
	if data.Orchestrator == nil {
		setUnregisteredState(&data)
	}
	if data.Orchestrator != nil {
		diag := r.registerWithOrchestrator(ctx, &data, nil)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	var installedDependencies []attr.Value
	if len(dependencies) > 0 {
		for _, dep := range dependencies {
			installedDependencies = append(installedDependencies, types.StringValue(dep))
		}
	} else {
		installedDependencies = []attr.Value{}
	}

	installDependenciesListValue, diags := types.ListValue(types.StringType, installedDependencies)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	data.InstalledDependencies = installDependenciesListValue

	hostConfig := data.GenerateApiHostConfig(ctx, r.provider)

	if len(data.ReverseProxyHosts) > 0 {
		rpHostsCopy := reverseproxy.CopyReverseProxyHosts(data.ReverseProxyHosts)
		result, createDiag := reverseproxy.Create(ctx, hostConfig, rpHostsCopy)
		if createDiag.HasError() {
			resp.Diagnostics.Append(createDiag...)
			if diag := reverseproxy.Delete(ctx, hostConfig, rpHostsCopy); diag.HasError() {
				tflog.Error(ctx, "Error deleting reverse proxy hosts")
			}
			return
		}

		for i := range result {
			data.ReverseProxyHosts[i].ID = result[i].ID
		}
	}

	if data.SshConnection != nil {
		data.ExternalIp = types.StringValue(strings.ReplaceAll(data.SshConnection.Host.String(), "\"", ""))
	} else {
		data.ExternalIp = types.StringValue("localhost")
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeployResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data deploy_models.DeployResourceModelV3
	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventDeploy, telemetry.ModeRead,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(ctx, telemetryEvent)

	tflog.Info(ctx, "Read request to see logs")
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.refreshRegistration(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runClient, runClientError := r.commandClient(ctx, data)
	if runClientError != nil {
		resp.Diagnostics.AddError("Error creating command client", runClientError.Error())
		return
	}
	defer runClient.Close()

	parallelsClient := NewDevOpsServiceClient(ctx, runClient)

	// getting parallels version
	if version, err := parallelsClient.GetVersion(ctx); err != nil {
		resp.Diagnostics.AddWarning("Error getting parallels version", err.Error())
		return
	} else {
		data.CurrentVersion = types.StringValue(version)
	}

	// getting parallels license
	if license, err := parallelsClient.GetLicense(ctx); err != nil {
		resp.Diagnostics.AddWarning("Error getting parallels license", err.Error())
		return
	} else {
		data.License = license.MapObject()
	}

	// Getting parallels latest api version
	if version, err := parallelsClient.GetDevOpsVersion(ctx); err != nil {
		planVersion := deploy_models.ParallelsDesktopDevOps{}
		if !data.Api.IsNull() {
			if diags := data.Api.As(ctx, &planVersion, basetypes.ObjectAsOptions{}); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			planVersion.Version = types.StringValue("-")
			data.Api = planVersion.MapObject()
		} else {
			planVersion.Version = types.StringValue("-")
			data.Api = planVersion.MapObject()
		}
	} else {
		planVersion := deploy_models.ParallelsDesktopDevOps{}
		if !data.Api.IsNull() {
			if diags := data.Api.As(ctx, &planVersion, basetypes.ObjectAsOptions{}); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			planVersion.Version = types.StringValue(version)
			data.Api = planVersion.MapObject()
		} else {
			planVersion.Version = types.StringValue("-")
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

func (r *DeployResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data deploy_models.DeployResourceModelV3
	var currentData deploy_models.DeployResourceModelV3

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventDeploy, telemetry.ModeUpdate,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(ctx, telemetryEvent)

	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := deploymentConfig(ctx, &data, r.provider.License.ValueString()); err != nil {
		resp.Diagnostics.AddError("Invalid DevOps configuration", err.Error())
		return
	}
	if data.Orchestrator != nil {
		if data.Orchestrator.Orchestrator == nil {
			resp.Diagnostics.AddError("Invalid orchestrator configuration", "orchestrator connection details are required")
			return
		}
		if err := validateOrchestratorDetails(data.Orchestrator.Orchestrator); err != nil {
			resp.Diagnostics.AddError("Invalid orchestrator authentication", err.Error())
			return
		}
	}
	data.IsRegisteredInOrchestrator = currentData.IsRegisteredInOrchestrator
	data.OrchestratorHostId = currentData.OrchestratorHostId
	data.OrchestratorHost = currentData.OrchestratorHost
	if data.Orchestrator != nil && currentData.Orchestrator != nil {
		data.Orchestrator.HostId = currentData.Orchestrator.HostId
	}
	runClient, runClientError := r.commandClient(ctx, data)
	if runClientError != nil {
		resp.Diagnostics.AddError("Error creating command client", runClientError.Error())
		return
	}

	var dependencies []string
	var restartDiag diag.Diagnostics

	defer runClient.Close()

	parallelsClient := NewDevOpsServiceClient(ctx, runClient)

	// checking if we still have parallels desktop installed
	if _, err := parallelsClient.GetVersion(ctx); err != nil {
		if !executableMissing(err, "prlsrvctl") {
			resp.Diagnostics.AddError("Error inspecting Parallels Desktop", err.Error())
			return
		}
		dependencies, restartDiag = r.installParallelsDesktop(ctx, parallelsClient, data.KeepAfterError.ValueBool())
		if restartDiag.HasError() {
			resp.Diagnostics.AddError("Error reinstalling Parallels desktop", err.Error())
			return
		}
	}

	serviceConfigured := false
	defer func() {
		if resp.Diagnostics.HasError() {
			// Preserve previously applied configuration when a transaction fails.
			// Publish changed API configuration only after the canonical file was installed.
			partial := currentData
			if serviceConfigured {
				partial.Api = data.Api
				partial.ApiConfig = data.ApiConfig
			}
			if data.OrchestratorHostId.ValueString() != "" {
				partial.IsRegisteredInOrchestrator = data.IsRegisteredInOrchestrator
				partial.OrchestratorHostId = data.OrchestratorHostId
				partial.OrchestratorHost = data.OrchestratorHost
				if partial.Orchestrator == nil {
					partial.Orchestrator = data.Orchestrator
				}
				if partial.Orchestrator != nil {
					registration := *partial.Orchestrator
					registration.HostId = data.OrchestratorHostId
					partial.Orchestrator = &registration
				}
			} else if !data.IsRegisteredInOrchestrator.IsUnknown() && !data.IsRegisteredInOrchestrator.ValueBool() && data.Orchestrator == nil {
				partial.Orchestrator = nil
				partial.IsRegisteredInOrchestrator = types.BoolValue(false)
				partial.OrchestratorHostId = types.StringNull()
				partial.OrchestratorHost = types.StringNull()
			}
			preparePartialDeploymentState(ctx, &partial, dependencies)
			resp.Diagnostics.Append(resp.State.Set(ctx, &partial)...)
		}
	}()
	apiChanged := deploy_models.ApiConfigHasChanges(ctx, data.ApiConfig, currentData.ApiConfig)
	effective, configErr := deploymentConfig(ctx, &data, r.provider.License.ValueString())
	if configErr != nil {
		resp.Diagnostics.AddError("Invalid DevOps configuration", configErr.Error())
		return
	}
	if !currentData.Api.IsNull() && !currentData.Api.IsUnknown() {
		priorAPI := currentData.Api.Attributes()
		for key, desired := range map[string]string{"host": effective.Host, "port": effective.Port, "protocol": effective.Protocol, "password": effective.Environment["ROOT_PASSWORD"]} {
			prior, ok := priorAPI[key].(types.String)
			if !ok || prior.ValueString() != desired {
				apiChanged = true
			}
		}
	}
	if data.Orchestrator != nil && currentData.Orchestrator == nil {
		apiChanged = true
	}

	_, devOpsErr := parallelsClient.GetDevOpsVersion(ctx)
	if devOpsErr != nil && !executableMissing(devOpsErr, "prldevops") {
		resp.Diagnostics.AddError("Error inspecting DevOps Service", devOpsErr.Error())
		return
	}
	data.Api = currentData.Api
	if apiChanged || devOpsErr != nil {
		api, diagnostics := r.installDevOpsService(ctx, &data, dependencies, parallelsClient)
		serviceConfigured = api != nil
		if diagnostics.HasError() {
			resp.Diagnostics.Append(diagnostics...)
			return
		}
	}

	// restart parallels service
	if err := parallelsClient.RestartServer(ctx); err != nil {
		resp.Diagnostics.AddError("Error restarting parallels service", err.Error())
		return
	}

	if r.provider.License.ValueString() != "" {
		// Licenses are the same, no changes
		equal, err := parallelsClient.CompareLicenses(ctx, r.provider.License.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error comparing parallels licenses", err.Error())
			return
		}

		if !equal {
			currentLicense, err := parallelsClient.GetLicense(ctx)
			if err != nil {
				resp.Diagnostics.AddError("Error getting parallels license", err.Error())
				return
			}
			if currentLicense.State.ValueString() == "valid" {
				// deactivating parallels license
				if err := parallelsClient.DeactivateLicense(ctx); err != nil {
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
			if err := parallelsClient.InstallLicense(ctx, key, username, password); err != nil {
				resp.Diagnostics.AddError("Error installing parallels license", err.Error())
				return
			}
		}
	} else if !data.License.IsNull() || !data.License.IsUnknown() {
		// deactivating parallels license
		if err := parallelsClient.DeactivateLicense(ctx); err != nil {
			resp.Diagnostics.AddError("Error deactivating parallels license", err.Error())
			return
		}
	}

	// getting parallels version
	if version, err := parallelsClient.GetVersion(ctx); err != nil {
		data.CurrentVagrantVersion = types.StringValue("-")
	} else {
		data.CurrentVersion = types.StringValue(version)
	}

	// getting git version
	if version, err := parallelsClient.GetGitVersion(ctx); err != nil {
		data.CurrentVagrantVersion = types.StringValue("-")
	} else {
		data.CurrentGitVersion = types.StringValue(version)
	}

	// getting packer version
	if version, err := parallelsClient.GetPackerVersion(ctx); err != nil {
		data.CurrentVagrantVersion = types.StringValue("-")
	} else {
		data.CurrentPackerVersion = types.StringValue(version)
	}

	// getting Vagrant version
	if version, err := parallelsClient.GetVagrantVersion(ctx); err != nil {
		data.CurrentVagrantVersion = types.StringValue("-")
	} else {
		data.CurrentVagrantVersion = types.StringValue(version)
	}

	// getting parallels license
	if license, err := parallelsClient.GetLicense(ctx); err != nil {
		resp.Diagnostics.AddError("Error getting parallels license", err.Error())
		return
	} else {
		data.License = license.MapObject()
	}

	// setting the same installed dependencies we had
	if data.InstalledDependencies.IsNull() || data.InstalledDependencies.IsUnknown() {
		data.InstalledDependencies = currentData.InstalledDependencies
	}

	switch {
	case data.Orchestrator != nil:
		if !serviceConfigured {
			if err := waitForHost(ctx, effective, r.provider.DisableTlsValidation.ValueBool()); err != nil {
				resp.Diagnostics.AddError("Host API is not ready", err.Error())
				return
			}
		}
		diagnostics := r.registerWithOrchestrator(ctx, &data, &currentData)
		if diagnostics.HasError() {
			resp.Diagnostics.Append(diagnostics...)
			return
		}
	case currentData.Orchestrator != nil:
		diagnostics := r.unregisterWithOrchestrator(ctx, &currentData)
		if diagnostics.HasError() {
			resp.Diagnostics.Append(diagnostics...)
			return
		}
		setUnregisteredState(&data)
	default:
		setUnregisteredState(&data)
	}

	hostConfig := data.GenerateApiHostConfig(ctx, r.provider)

	if reverseproxy.ReverseProxyHostsDiff(data.ReverseProxyHosts, currentData.ReverseProxyHosts) {
		copyCurrentRpHosts := reverseproxy.CopyReverseProxyHosts(currentData.ReverseProxyHosts)
		copyRpHosts := reverseproxy.CopyReverseProxyHosts(data.ReverseProxyHosts)

		results, updateDiag := reverseproxy.Update(ctx, hostConfig, copyCurrentRpHosts, copyRpHosts)
		if updateDiag.HasError() {
			resp.Diagnostics.Append(updateDiag...)
			revertResults, _ := reverseproxy.Revert(ctx, hostConfig, copyCurrentRpHosts, copyRpHosts)
			for i := range revertResults {
				data.ReverseProxyHosts[i].ID = revertResults[i].ID
			}
			return
		}

		for i := range results {
			data.ReverseProxyHosts[i].ID = results[i].ID
		}
	} else {
		for i := range currentData.ReverseProxyHosts {
			data.ReverseProxyHosts[i].ID = currentData.ReverseProxyHosts[i].ID
		}
	}

	if currentData.ExternalIp.ValueString() == "" ||
		strings.ReplaceAll(currentData.ExternalIp.ValueString(), "\"", "") != strings.ReplaceAll(data.ExternalIp.ValueString(), "\"", "") {
		if data.SshConnection != nil {
			data.ExternalIp = types.StringValue(strings.ReplaceAll(data.SshConnection.Host.String(), "\"", ""))
		} else {
			data.ExternalIp = types.StringValue("localhost")
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DeployResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data deploy_models.DeployResourceModelV3

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventDeploy, telemetry.ModeDestroy,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(ctx, telemetryEvent)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	runClient, runClientError := r.commandClient(ctx, data)
	if runClientError != nil {
		resp.Diagnostics.AddError("Error creating command client", runClientError.Error())
		return
	}

	defer runClient.Close()

	parallelsService := NewDevOpsServiceClient(ctx, runClient)

	// deactivating parallels license
	if data.Orchestrator != nil {
		if diag := r.unregisterWithOrchestrator(ctx, &data); diag.HasError() {
			resp.Diagnostics.Append(diag...)
		}
	}

	hostConfig := data.GenerateApiHostConfig(ctx, r.provider)

	if len(data.ReverseProxyHosts) > 0 {
		rpHostsCopy := reverseproxy.CopyReverseProxyHosts(data.ReverseProxyHosts)
		if diag := reverseproxy.Delete(ctx, hostConfig, rpHostsCopy); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if err := parallelsService.DeactivateLicense(ctx); err != nil {
		resp.Diagnostics.AddWarning("Error deactivating parallels license", err.Error())
	}

	// uninstalling parallels desktop
	if err := parallelsService.UninstallParallelsDesktop(ctx); err != nil {
		resp.Diagnostics.AddWarning("Error uninstalling parallels desktop", err.Error())
	}

	var installedDependencies []string
	if !data.InstalledDependencies.IsNull() {
		for _, dep := range data.InstalledDependencies.Elements() {
			if strVal, ok := dep.(types.String); ok {
				installedDependencies = append(installedDependencies, strVal.ValueString())
			}
		}
	}

	// uninstalling dependencies
	if uninstallErrors := parallelsService.UninstallDependencies(ctx, installedDependencies); len(uninstallErrors) > 0 {
		for _, err := range uninstallErrors {
			resp.Diagnostics.AddWarning("Error uninstalling dependencies", err.Error())
		}
	}

	// Save data into Terraform state
	data.CurrentVersion = types.StringValue("-")
	data.License = types.ObjectUnknown(map[string]attr.Type{
		"state":      types.StringType,
		"key":        types.StringType,
		"restricted": types.BoolType,
	})

	if err := parallelsService.UninstallDevOpsService(ctx); err != nil {
		if data.InstallLocal.ValueBool() {
			// For local installs, downgrade to warning so terraform destroy isn't blocked.
			// The error message includes manual cleanup steps if needed.
			resp.Diagnostics.AddWarning("Partial cleanup of DevOps service", err.Error())
		} else {
			resp.Diagnostics.AddError("Error uninstalling parallels DevOps service", err.Error())
		}
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

func (r *DeployResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Versions 0, 1 and the released schema version 2 are additive predecessors.
// Decode raw state against the current type so omitted attributes become null,
// preserving all existing values (including fields absent from the old V2 Go model).
func (r *DeployResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{0: {StateUpgrader: upgradeDeploymentState}, 1: {StateUpgrader: upgradeDeploymentState}, 2: {StateUpgrader: upgradeDeploymentState}}
}

func upgradeDeploymentState(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	if req.RawState == nil {
		resp.Diagnostics.AddError("Missing previous deployment state", "Cannot upgrade without raw state")
		return
	}
	value, err := req.RawState.Unmarshal(schemas.DeployResourceSchemaV3.Type().TerraformType(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Could not upgrade deployment state", err.Error())
		return
	}
	resp.State.Raw = value
}

func (r *DeployResource) commandClient(ctx context.Context, data deploy_models.DeployResourceModelV3) (interfaces.ContextCommandClient, error) {
	if r.commandClientFactory != nil {
		return r.commandClientFactory(ctx, data)
	}
	if data.InstallLocal.ValueBool() {
		return localclient.NewLocalClient(), nil
	}
	return r.getSshClient(ctx, data)
}

func (r *DeployResource) getSshClient(ctx context.Context, data deploy_models.DeployResourceModelV3) (*ssh.SshClient, error) {
	if data.SshConnection == nil {
		return nil, errors.New("ssh_connection is required for remote deployment; use install_local = true for local deployment")
	}
	if data.SshConnection.Host.IsNull() {
		return nil, errors.New("host is required")
	}
	if data.SshConnection.User.IsNull() {
		return nil, errors.New("user is required")
	}
	if data.SshConnection.Password.IsNull() && data.SshConnection.PrivateKey.IsNull() {
		return nil, errors.New("password or PrivateKey is required")
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
	if err := sshClient.ConnectContext(ctx); err != nil {
		sshClient.Close()
		return nil, err
	}

	return sshClient, nil
}

func (r *DeployResource) installParallelsDesktop(ctx context.Context, parallelsClient *DevOpsServiceClient, keepAfterError bool) ([]string, diag.Diagnostics) {
	diag := diag.Diagnostics{}
	var installDependenciesError error
	var installed_dependencies []string
	cleanupDependencies := func() []error {
		if keepAfterError {
			return nil
		}
		return parallelsClient.UninstallDependencies(ctx, installed_dependencies)
	}
	mandatoryDependencies := []string{
		"brew",
		"git",
		"vagrant",
	}

	// installing dependencies
	installed_dependencies, installDependenciesError = parallelsClient.InstallDependencies(ctx, mandatoryDependencies)
	if installDependenciesError != nil {
		if uninstallErrors := cleanupDependencies(); len(uninstallErrors) > 0 {
			for _, uninstallError := range uninstallErrors {
				diag.AddError("Error uninstalling dependencies", uninstallError.Error())
			}
		}
		diag.AddError("Error installing dependencies", installDependenciesError.Error())
		return installed_dependencies, diag
	}

	// installing parallels desktop
	if err := parallelsClient.InstallParallelsDesktop(ctx); err != nil {
		if uninstallErrors := cleanupDependencies(); len(uninstallErrors) > 0 {
			for _, uninstallError := range uninstallErrors {
				diag.AddError("Error uninstalling dependencies", uninstallError.Error())
			}
		}

		diag.AddError("Error installing parallels desktop", err.Error())
		return installed_dependencies, diag
	}

	// restarting parallels service
	if err := parallelsClient.RestartServer(ctx); err != nil {
		if uninstallErrors := cleanupDependencies(); len(uninstallErrors) > 0 {
			for _, uninstallError := range uninstallErrors {
				diag.AddError("Error uninstalling dependencies", uninstallError.Error())
			}
		}

		diag.AddError("Error restarting parallels service", err.Error())
		return installed_dependencies, diag
	}

	key := r.provider.License.ValueString()
	username := r.provider.MyAccountUser.ValueString()
	password := r.provider.MyAccountPassword.ValueString()

	// For local deployments, check if already licensed before trying to install
	skipLicense := false
	if license, err := parallelsClient.GetLicense(ctx); err == nil && license != nil {
		if license.State.ValueString() == "valid" || license.State.ValueString() == "active" {
			skipLicense = true
		}
	}

	// installing parallels license (skip if already licensed or key is empty for local installs)
	if !skipLicense && key != "" {
		if err := parallelsClient.InstallLicense(ctx, key, username, password); err != nil {
			if uninstallErrors := cleanupDependencies(); len(uninstallErrors) > 0 {
				for _, uninstallError := range uninstallErrors {
					diag.AddError("Error uninstalling dependencies", uninstallError.Error())
				}
			}

			diag.AddError("Error installing parallels license", err.Error())
			return installed_dependencies, diag
		}
	} else if !skipLicense && key == "" {
		diag.AddError("Error installing parallels license", "No license key provided and no valid license found. Set the license in the provider configuration.")
		return installed_dependencies, diag
	}

	if installed_dependencies == nil {
		installed_dependencies = []string{}
	}

	return installed_dependencies, diag
}

func (r *DeployResource) installDevOpsService(ctx context.Context, data *deploy_models.DeployResourceModelV3, dependencies []string, client *DevOpsServiceClient) (*deploy_models.ParallelsDesktopDevOps, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	cfg, err := deploymentConfig(ctx, data, r.provider.License.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid DevOps configuration", err.Error())
		return nil, diagnostics
	}
	version, err := client.installConfiguredService(ctx, cfg)
	if err != nil {
		diagnostics.AddError("Error configuring DevOps Service", err.Error())
		return nil, diagnostics
	}
	api := deploy_models.ParallelsDesktopDevOps{Version: types.StringValue(version), Host: types.StringValue(cfg.Host), Port: types.StringValue(cfg.Port), Protocol: types.StringValue(cfg.Protocol), User: types.StringValue("root@localhost"), Password: types.StringValue(cfg.Environment["ROOT_PASSWORD"])}
	data.Api = api.MapObject()
	if err := waitForHost(ctx, cfg, r.provider.DisableTlsValidation.ValueBool()); err != nil {
		diagnostics.AddError("Host API is not ready", "The Terraform runner could not authenticate and reach the configured Host API: "+err.Error()+". Check the canonical configuration, launchd process, TLS trust and network access. The installed service and database were preserved.")
	}
	return &api, diagnostics
}
