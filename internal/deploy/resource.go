package deploy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"terraform-provider-parallels-desktop/internal/interfaces"
	"terraform-provider-parallels-desktop/internal/localclient"
	"terraform-provider-parallels-desktop/internal/models"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/orchestrator"
	"terraform-provider-parallels-desktop/internal/ssh"

	"terraform-provider-parallels-desktop/internal/telemetry"

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
	provider *models.ParallelsProviderModel
}

func (r *DeployResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deploy"
}

func (r *DeployResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = deployResourceSchemaV1
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
	tflog.Info(ctx, r.provider.License.ValueString())
}

func (r *DeployResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeployResourceModelV1
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventDeploy, telemetry.ModeCreate,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(telemetryEvent)

	if resp.Diagnostics.HasError() {
		return
	}

	var runClient interfaces.CommandClient
	var runClientError error

	if data.InstallLocal.ValueBool() {
		runClient = localclient.NewLocalClient()
	} else {
		runClient, runClientError = r.getSshClient(data)
		if runClientError != nil {
			resp.Diagnostics.AddError("Error creating SSH client", runClientError.Error())
			return
		}
	}

	parallelsClient := NewDevOpsServiceClient(ctx, runClient)

	dependencies, diag := r.installParallelsDesktop(parallelsClient)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	_, diag = r.installDevOpsService(&data, dependencies, parallelsClient)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	// getting parallels version
	if version, err := parallelsClient.GetVersion(); err != nil {
		resp.Diagnostics.AddError("Error getting parallels version", err.Error())
		return
	} else {
		data.CurrentVersion = types.StringValue(version)
	}

	// getting git version
	if version, err := parallelsClient.GetGitVersion(); err != nil {
		data.CurrentGitVersion = types.StringValue("-")
	} else {
		data.CurrentGitVersion = types.StringValue(version)
	}

	// getting packer version
	if version, err := parallelsClient.GetPackerVersion(); err != nil {
		data.CurrentPackerVersion = types.StringValue("-")
	} else {
		data.CurrentPackerVersion = types.StringValue(version)
	}

	// getting Vagrant version
	if version, err := parallelsClient.GetVagrantVersion(); err != nil {
		data.CurrentVagrantVersion = types.StringValue("-")
	} else {
		data.CurrentVagrantVersion = types.StringValue(version)
	}

	// getting parallels license
	if license, err := parallelsClient.GetLicense(); err != nil {
		if uninstallErrors := parallelsClient.UninstallDependencies(dependencies); len(uninstallErrors) > 0 {
			for _, uninstallError := range uninstallErrors {
				diag.AddError("Error uninstalling dependencies", uninstallError.Error())
			}
		}
		if err := parallelsClient.UninstallParallelsDesktop(); err != nil {
			resp.Diagnostics.AddError("Error uninstalling dependencies", err.Error())
		}
		if err := parallelsClient.UninstallDevOpsService(); err != nil {
			resp.Diagnostics.AddError("Error uninstalling parallels DevOps service", err.Error())
		}
		resp.Diagnostics.AddError("Error getting parallels license", err.Error())
		return
	} else {
		data.License = license.MapObject()
	}

	// Register with orchestrator if needed
	if data.Orchestrator != nil {
		apiData := data.Api
		host := strings.ReplaceAll(apiData.Attributes()["host"].String(), "\"", "")
		protocol := strings.ReplaceAll(apiData.Attributes()["protocol"].String(), "\"", "")
		port := strings.ReplaceAll(apiData.Attributes()["port"].String(), "\"", "")
		user := strings.ReplaceAll(apiData.Attributes()["user"].String(), "\"", "")
		password := strings.ReplaceAll(apiData.Attributes()["password"].String(), "\"", "")

		orchestratorConfig := orchestrator.OrchestratorRegistration{
			HostId:      data.Orchestrator.HostId,
			Schema:      types.StringValue(protocol),
			Host:        types.StringValue(host),
			Port:        types.StringValue(port),
			Description: data.Orchestrator.Description,
			Tags:        data.Orchestrator.Tags,
			HostCredentials: &authenticator.Authentication{
				Username: types.StringValue(user),
				Password: types.StringValue(password),
			},
			Orchestrator: data.Orchestrator.Orchestrator,
		}

		id, diag := orchestrator.RegisterWithHost(ctx, orchestratorConfig, r.provider.DisableTlsValidation.ValueBool())
		if diag.HasError() {
			if uninstallErrors := parallelsClient.UninstallDependencies(dependencies); len(uninstallErrors) > 0 {
				for _, uninstallError := range uninstallErrors {
					diag.AddError("Error uninstalling dependencies", uninstallError.Error())
				}
			}
			if err := parallelsClient.UninstallParallelsDesktop(); err != nil {
				resp.Diagnostics.AddError("Error uninstalling dependencies", err.Error())
			}
			if err := parallelsClient.UninstallDevOpsService(); err != nil {
				resp.Diagnostics.AddError("Error uninstalling parallels DevOps service", err.Error())
			}
			isRegistered, diags := orchestrator.IsAlreadyRegistered(ctx, orchestratorConfig, r.provider.DisableTlsValidation.ValueBool())
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			if isRegistered {
				if diag := orchestrator.UnregisterWithHost(ctx, orchestratorConfig, r.provider.DisableTlsValidation.ValueBool()); diag.HasError() {
					resp.Diagnostics.Append(diag...)
				}
			}

			resp.Diagnostics.Append(diag...)
			return
		}

		data.Orchestrator.HostId = types.StringValue(id)
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

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeployResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeployResourceModelV1
	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventDeploy, telemetry.ModeRead,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(telemetryEvent)

	tflog.Info(ctx, "Read request to see logs")
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var runClient interfaces.CommandClient
	var runClientError error

	if data.InstallLocal.ValueBool() {
		runClient = localclient.NewLocalClient()
	} else {
		runClient, runClientError = r.getSshClient(data)
		if runClientError != nil {
			resp.Diagnostics.AddError("Error creating SSH client", runClientError.Error())
			return
		}
	}
	parallelsClient := NewDevOpsServiceClient(ctx, runClient)

	// getting parallels version
	if version, err := parallelsClient.GetVersion(); err != nil {
		resp.Diagnostics.AddWarning("Error getting parallels version", err.Error())
		return
	} else {
		data.CurrentVersion = types.StringValue(version)
	}

	// getting parallels license
	if license, err := parallelsClient.GetLicense(); err != nil {
		resp.Diagnostics.AddWarning("Error getting parallels license", err.Error())
		return
	} else {
		data.License = license.MapObject()
	}

	// Getting parallels latest api version
	if version, err := parallelsClient.GetDevOpsVersion(); err != nil {
		planVersion := ParallelsDesktopDevOps{}
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
		planVersion := ParallelsDesktopDevOps{}
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
	var data DeployResourceModelV1
	var currentData DeployResourceModelV1

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventDeploy, telemetry.ModeUpdate,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(telemetryEvent)

	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var runClient interfaces.CommandClient
	var runClientError error

	if data.InstallLocal.ValueBool() {
		runClient = localclient.NewLocalClient()
	} else {
		runClient, runClientError = r.getSshClient(data)
		if runClientError != nil {
			resp.Diagnostics.AddError("Error creating SSH client", runClientError.Error())
			return
		}
	}

	var dependencies []string
	var restartDiag diag.Diagnostics

	parallelsClient := NewDevOpsServiceClient(ctx, runClient)

	// checking if we still have parallels desktop installed
	if _, err := parallelsClient.GetVersion(); err != nil {
		dependencies, restartDiag = r.installParallelsDesktop(parallelsClient)
		if restartDiag.HasError() {
			resp.Diagnostics.AddError("Error reinstalling Parallels desktop", err.Error())
			return
		}
	}

	// checking if we still have the devops service running
	_, devOpsErr := parallelsClient.GetDevOpsVersion()
	if devOpsErr != nil {
		r.installDevOpsService(&data, dependencies, parallelsClient)
	}

	// Check if the API config has changed
	if ApiConfigHasChanges(ctx, data.ApiConfig, currentData.ApiConfig) {
		if err := parallelsClient.UninstallDevOpsService(); err != nil {
			resp.Diagnostics.AddError("Error uninstalling parallels DevOps service", err.Error())
			return
		}
		if _, diag := r.installDevOpsService(&data, dependencies, parallelsClient); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	// restart parallels service
	if err := parallelsClient.RestartServer(); err != nil {
		dependencies, restartDiag = r.installParallelsDesktop(parallelsClient)
		if restartDiag.HasError() {
			resp.Diagnostics.AddError("Error restarting parallels service", err.Error())
			return
		}
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

	// getting git version
	if version, err := parallelsClient.GetGitVersion(); err != nil {
		resp.Diagnostics.AddError("Error getting git version", err.Error())
		return
	} else {
		data.CurrentGitVersion = types.StringValue(version)
	}

	// getting packer version
	if version, err := parallelsClient.GetPackerVersion(); err != nil {
		resp.Diagnostics.AddError("Error getting packer version", err.Error())
		return
	} else {
		data.CurrentPackerVersion = types.StringValue(version)
	}

	// getting Vagrant version
	if version, err := parallelsClient.GetVagrantVersion(); err != nil {
		resp.Diagnostics.AddError("Error getting vagrant version", err.Error())
		return
	} else {
		data.CurrentVagrantVersion = types.StringValue(version)
	}

	// getting parallels license
	if license, err := parallelsClient.GetLicense(); err != nil {
		resp.Diagnostics.AddError("Error getting parallels license", err.Error())
		return
	} else {
		data.License = license.MapObject()
	}

	// setting the same installed dependencies we had
	if data.InstalledDependencies.IsNull() || data.InstalledDependencies.IsUnknown() {
		data.InstalledDependencies = currentData.InstalledDependencies
	}

	hasChangesInDevOpsService := false
	if data.ApiConfig != nil {
		if currentData.ApiConfig == nil {
			hasChangesInDevOpsService = true
		} else {
			if data.ApiConfig.Port.ValueString() != currentData.ApiConfig.Port.ValueString() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.TLSPort.ValueString() != currentData.ApiConfig.TLSPort.ValueString() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.RootPassword.ValueString() != currentData.ApiConfig.RootPassword.ValueString() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.EncryptionRsaKey.ValueString() != currentData.ApiConfig.EncryptionRsaKey.ValueString() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.HmacSecret.ValueString() != currentData.ApiConfig.HmacSecret.ValueString() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.LogLevel.ValueString() != currentData.ApiConfig.LogLevel.ValueString() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.EnableTLS.ValueBool() != currentData.ApiConfig.EnableTLS.ValueBool() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.TLSPort.ValueString() != currentData.ApiConfig.TLSPort.ValueString() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.TLSCertificate.ValueString() != currentData.ApiConfig.TLSCertificate.ValueString() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.TLSPrivateKey.ValueString() != currentData.ApiConfig.TLSPrivateKey.ValueString() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.DisableCatalogCaching.ValueBool() != currentData.ApiConfig.DisableCatalogCaching.ValueBool() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.TokenDurationMinutes.ValueString() != currentData.ApiConfig.TokenDurationMinutes.ValueString() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.Mode.ValueString() != currentData.ApiConfig.Mode.ValueString() {
				hasChangesInDevOpsService = true
			}
			if data.ApiConfig.UseOrchestratorResources.ValueBool() != currentData.ApiConfig.UseOrchestratorResources.ValueBool() {
				hasChangesInDevOpsService = true
			}
		}
	}

	if hasChangesInDevOpsService {
		err := parallelsClient.UninstallDevOpsService()
		if err != nil {
			resp.Diagnostics.AddError("Error uninstalling parallels DevOps service", err.Error())
			return
		}
		if _, diag := r.installDevOpsService(&data, dependencies, parallelsClient); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
		tflog.Info(ctx, "Changes in DevOps service, restarting parallels service")
	} else {
		tflog.Info(ctx, "No changes in DevOps service")
	}

	installedVersion, getVersionError := parallelsClient.GetDevOpsVersion()
	if getVersionError != nil {
		if getVersionError.Error() == "Parallels Desktop DevOps Service not found" {
			_, apiDiag := r.installDevOpsService(&data, dependencies, parallelsClient)
			if apiDiag.HasError() {
				resp.Diagnostics.Append(apiDiag...)
				return
			}
		} else {
			resp.Diagnostics.AddError("Error getting parallels DevOps version", getVersionError.Error())
			return
		}
	}

	desiredApiData, err := data.Api.ToObjectValue(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error converting data.Api to object value", "")
		return
	}

	if !desiredApiData.IsUnknown() && !desiredApiData.IsNull() {
		desiredVersion := strings.ReplaceAll(desiredApiData.Attributes()["version"].String(), "\"", "")
		if installedVersion != desiredVersion {
			devOpsData, apiDiag := r.installDevOpsService(&data, dependencies, parallelsClient)
			if apiDiag.HasError() {
				resp.Diagnostics.Append(apiDiag...)
				return
			}
			if devOpsData != nil {
				tflog.Info(ctx, "DevOps is installed")
			}
		}
	} else {
		data.Api = currentData.Api
	}

	if data.Orchestrator != nil {
		if orchestrator.HasChanges(ctx, data.Orchestrator, currentData.Orchestrator) {
			apiData := data.Api
			host := strings.ReplaceAll(apiData.Attributes()["host"].String(), "\"", "")
			protocol := strings.ReplaceAll(apiData.Attributes()["protocol"].String(), "\"", "")
			port := strings.ReplaceAll(apiData.Attributes()["port"].String(), "\"", "")
			user := strings.ReplaceAll(apiData.Attributes()["user"].String(), "\"", "")
			password := strings.ReplaceAll(apiData.Attributes()["password"].String(), "\"", "")

			// checking if we already registered with orchestrator
			if currentData.Orchestrator != nil && currentData.Orchestrator.HostId.ValueString() != "" {
				if diag := orchestrator.UnregisterWithHost(ctx, *currentData.Orchestrator, r.provider.DisableTlsValidation.ValueBool()); diag.HasError() {
					resp.Diagnostics.Append(diag...)
					return
				}
			}

			orchestratorConfig := orchestrator.OrchestratorRegistration{
				HostId:      data.Orchestrator.HostId,
				Schema:      types.StringValue(protocol),
				Host:        types.StringValue(host),
				Port:        types.StringValue(port),
				Description: data.Orchestrator.Description,
				Tags:        data.Orchestrator.Tags,
				HostCredentials: &authenticator.Authentication{
					Username: types.StringValue(user),
					Password: types.StringValue(password),
				},
				Orchestrator: data.Orchestrator.Orchestrator,
			}

			isRegistered, diags := orchestrator.IsAlreadyRegistered(ctx, orchestratorConfig, r.provider.DisableTlsValidation.ValueBool())
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			if !isRegistered {
				id, diag := orchestrator.RegisterWithHost(ctx, orchestratorConfig, r.provider.DisableTlsValidation.ValueBool())
				if diag.HasError() {
					resp.Diagnostics.Append(diag...)
					return
				}

				data.Orchestrator.HostId = types.StringValue(id)
			} else {
				tflog.Info(ctx, "Already registered with orchestrator, skipping registration")
			}
		} else {
			data.Orchestrator.HostId = currentData.Orchestrator.HostId
		}
	} else if currentData.Orchestrator != nil {
		if currentData.Orchestrator.HostId.ValueString() != "" {
			if diag := orchestrator.UnregisterWithHost(ctx, *currentData.Orchestrator, r.provider.DisableTlsValidation.ValueBool()); diag.HasError() {
				resp.Diagnostics.Append(diag...)
				return
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DeployResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeployResourceModelV1

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventDeploy, telemetry.ModeDestroy,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(telemetryEvent)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var runClient interfaces.CommandClient
	var runClientError error

	if data.InstallLocal.ValueBool() {
		runClient = localclient.NewLocalClient()
	} else {
		runClient, runClientError = r.getSshClient(data)
		if runClientError != nil {
			resp.Diagnostics.AddError("Error creating SSH client", runClientError.Error())
			return
		}
	}

	parallelsService := NewDevOpsServiceClient(ctx, runClient)

	// deactivating parallels license
	if err := parallelsService.DeactivateLicense(); err != nil {
		resp.Diagnostics.AddWarning("Error deactivating parallels license", err.Error())
	}

	// uninstalling parallels desktop
	if err := parallelsService.UninstallParallelsDesktop(); err != nil {
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
	if uninstallErrors := parallelsService.UninstallDependencies(installedDependencies); len(uninstallErrors) > 0 {
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

	if err := parallelsService.UninstallDevOpsService(); err != nil {
		resp.Diagnostics.AddError("Error uninstalling parallels DevOps service", err.Error())
	}

	data.Api = types.ObjectUnknown(map[string]attr.Type{
		"version":  types.StringType,
		"host":     types.StringType,
		"user":     types.StringType,
		"password": types.StringType,
	})

	if data.Orchestrator != nil {
		if diag := orchestrator.UnregisterWithHost(ctx, *data.Orchestrator, r.provider.DisableTlsValidation.ValueBool()); diag.HasError() {
			resp.Diagnostics.Append(diag...)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DeployResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DeployResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &deployResourceSchemaV0,
			StateUpgrader: UpgradeState,
		},
	}
}

func UpgradeState(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var priorStateData DeployResourceModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	upgradedStateData := DeployResourceModelV1{
		SshConnection:         priorStateData.SshConnection,
		CurrentVersion:        priorStateData.CurrentVersion,
		CurrentPackerVersion:  priorStateData.CurrentPackerVersion,
		CurrentVagrantVersion: priorStateData.CurrentVagrantVersion,
		CurrentGitVersion:     priorStateData.CurrentGitVersion,
		License:               priorStateData.License,
		Orchestrator:          priorStateData.Orchestrator,
		ApiConfig: &ParallelsDesktopDevopsConfigV1{
			Port:                     priorStateData.ApiConfig.Port,
			Prefix:                   priorStateData.ApiConfig.Prefix,
			DevOpsVersion:            priorStateData.ApiConfig.DevOpsVersion,
			RootPassword:             priorStateData.ApiConfig.RootPassword,
			HmacSecret:               priorStateData.ApiConfig.HmacSecret,
			EncryptionRsaKey:         priorStateData.ApiConfig.EncryptionRsaKey,
			LogLevel:                 priorStateData.ApiConfig.LogLevel,
			EnableTLS:                priorStateData.ApiConfig.EnableTLS,
			TLSPort:                  priorStateData.ApiConfig.TLSPort,
			TLSCertificate:           priorStateData.ApiConfig.TLSCertificate,
			TLSPrivateKey:            priorStateData.ApiConfig.TLSPrivateKey,
			DisableCatalogCaching:    priorStateData.ApiConfig.DisableCatalogCaching,
			TokenDurationMinutes:     priorStateData.ApiConfig.TokenDurationMinutes,
			Mode:                     priorStateData.ApiConfig.Mode,
			UseOrchestratorResources: priorStateData.ApiConfig.UseOrchestratorResources,
			SystemReservedMemory:     priorStateData.ApiConfig.SystemReservedMemory,
			SystemReservedCpu:        priorStateData.ApiConfig.SystemReservedCpu,
			SystemReservedDisk:       priorStateData.ApiConfig.SystemReservedDisk,
			EnableLogging:            priorStateData.ApiConfig.EnableLogging,
			EnvironmentVariables:     make(map[string]basetypes.StringValue),
		},
		Api:                   priorStateData.Api,
		InstalledDependencies: priorStateData.InstalledDependencies,
		InstallLocal:          priorStateData.InstallLocal,
	}

	println(fmt.Sprintf("Upgrading state from version %v", upgradedStateData))

	resp.Diagnostics.Append(resp.State.Set(ctx, &upgradedStateData)...)
}

func (r *DeployResource) getSshClient(data DeployResourceModelV1) (*ssh.SshClient, error) {
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
	if err := sshClient.Connect(); err != nil {
		return nil, err
	}

	return sshClient, nil
}

func (r *DeployResource) installParallelsDesktop(parallelsClient *DevOpsServiceClient) ([]string, diag.Diagnostics) {
	diag := diag.Diagnostics{}
	var installDependenciesError error
	var installed_dependencies []string

	// installing dependencies
	installed_dependencies, installDependenciesError = parallelsClient.InstallDependencies()
	if installDependenciesError != nil {
		if uninstallErrors := parallelsClient.UninstallDependencies(installed_dependencies); len(uninstallErrors) > 0 {
			for _, uninstallError := range uninstallErrors {
				diag.AddError("Error uninstalling dependencies", uninstallError.Error())
			}
		}
		diag.AddError("Error installing dependencies", installDependenciesError.Error())
		return installed_dependencies, diag
	}

	// installing parallels desktop
	if err := parallelsClient.InstallParallelsDesktop(); err != nil {
		if uninstallErrors := parallelsClient.UninstallDependencies(installed_dependencies); len(uninstallErrors) > 0 {
			for _, uninstallError := range uninstallErrors {
				diag.AddError("Error uninstalling dependencies", uninstallError.Error())
			}
		}
		if err := parallelsClient.UninstallParallelsDesktop(); err != nil {
			diag.AddError("Error uninstalling dependencies", err.Error())
		}
		diag.AddError("Error installing parallels desktop", err.Error())
		return installed_dependencies, diag
	}

	// restarting parallels service
	if err := parallelsClient.RestartServer(); err != nil {
		if uninstallErrors := parallelsClient.UninstallDependencies(installed_dependencies); len(uninstallErrors) > 0 {
			for _, uninstallError := range uninstallErrors {
				diag.AddError("Error uninstalling dependencies", uninstallError.Error())
			}
		}
		if err := parallelsClient.UninstallParallelsDesktop(); err != nil {
			diag.AddError("Error uninstalling dependencies", err.Error())
		}
		diag.AddError("Error restarting parallels service", err.Error())
		return installed_dependencies, diag
	}

	key := r.provider.License.ValueString()
	username := r.provider.MyAccountUser.ValueString()
	password := r.provider.MyAccountPassword.ValueString()

	// installing parallels license
	if err := parallelsClient.InstallLicense(key, username, password); err != nil {
		if uninstallErrors := parallelsClient.UninstallDependencies(installed_dependencies); len(uninstallErrors) > 0 {
			for _, uninstallError := range uninstallErrors {
				diag.AddError("Error uninstalling dependencies", uninstallError.Error())
			}
		}
		if err := parallelsClient.UninstallParallelsDesktop(); err != nil {
			diag.AddError("Error uninstalling dependencies", err.Error())
		}
		diag.AddError("Error installing parallels license", err.Error())
		return installed_dependencies, diag
	}

	if installed_dependencies == nil {
		installed_dependencies = []string{}
	}

	return installed_dependencies, diag
}

func (r *DeployResource) installDevOpsService(data *DeployResourceModelV1, dependencies []string, parallelsClient *DevOpsServiceClient) (*ParallelsDesktopDevOps, diag.Diagnostics) {
	diag := diag.Diagnostics{}
	targetPort := "8080"
	targetTlsPort := "8443"
	apiVersion := "latest"

	// Installing parallels DevOps service
	var config ParallelsDesktopDevopsConfigV1
	if data.ApiConfig == nil {
		config = ParallelsDesktopDevopsConfigV1{
			DevOpsVersion: types.StringValue(apiVersion),
			Port:          types.StringValue(targetPort),
			TLSPort:       types.StringValue(targetTlsPort),
		}
	} else {
		config = *data.ApiConfig
	}

	if config.RootPassword.ValueString() == "" {
		config.RootPassword = r.provider.License
	}

	_, err := parallelsClient.InstallDevOpsService(r.provider.License.ValueString(), config)
	if err != nil {
		if uninstallErrors := parallelsClient.UninstallDependencies(dependencies); len(uninstallErrors) > 0 {
			for _, uninstallError := range uninstallErrors {
				diag.AddError("Error uninstalling dependencies", uninstallError.Error())
			}
		}
		if err := parallelsClient.UninstallParallelsDesktop(); err != nil {
			diag.AddError("Error uninstalling dependencies", err.Error())
		}
		if err := parallelsClient.UninstallDevOpsService(); err != nil {
			diag.AddError("Error uninstalling parallels DevOps service", err.Error())
		}

		diag.AddError("Error installing parallels DevOps service", err.Error())
		return nil, diag
	}

	currentVersion, err := parallelsClient.GetDevOpsVersion()
	if err != nil {
		if uninstallErrors := parallelsClient.UninstallDependencies(dependencies); len(uninstallErrors) > 0 {
			for _, uninstallError := range uninstallErrors {
				diag.AddError("Error uninstalling dependencies", uninstallError.Error())
			}
		}
		if err := parallelsClient.UninstallParallelsDesktop(); err != nil {
			diag.AddError("Error uninstalling dependencies", err.Error())
		}
		if err := parallelsClient.UninstallDevOpsService(); err != nil {
			diag.AddError("Error uninstalling parallels DevOps service", err.Error())
		}
		diag.AddError("Error getting parallels api version", err.Error())
		return nil, diag
	}

	apiData := ParallelsDesktopDevOps{
		Version:  types.StringValue(currentVersion),
		Host:     types.StringValue(data.SshConnection.Host.ValueString()),
		Port:     types.StringValue(targetPort),
		Protocol: types.StringValue("http"),
		User:     types.StringValue("root@localhost"),
	}

	if config.EnableTLS.ValueBool() {
		apiData.Protocol = types.StringValue("https")
		apiData.Port = types.StringValue(targetTlsPort)
	} else {
		apiData.Protocol = types.StringValue("http")
		apiData.Port = types.StringValue(targetPort)
	}

	apiData.Password = config.RootPassword

	data.Api = apiData.MapObject()

	return &apiData, diag
}
