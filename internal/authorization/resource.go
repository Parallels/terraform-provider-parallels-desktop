package authorization

import (
	"context"
	"fmt"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &AuthorizationResource{}
	_ resource.ResourceWithImportState = &AuthorizationResource{}
)

func NewAuthorizationResource() resource.Resource {
	return &AuthorizationResource{}
}

// AuthorizationResource defines the resource implementation.
type AuthorizationResource struct {
	provider *models.ParallelsProviderModel
}

func (r *AuthorizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth"
}

func (r *AuthorizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = authorizationResourceSchema
}

func (r *AuthorizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
}

func (r *AuthorizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Creating Authorization Resource")
	var data AuthorizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		Host:                 data.Host.ValueString(),
		License:              r.provider.License.ValueString(),
		Authorization:        data.Authenticator,
		DisableTlsValidation: r.provider.DisableTlsValidation.ValueBool(),
	}

	usersNotCreated := make([]string, 0)
	apiKeysNotCreated := make([]string, 0)
	claimsNotCreated := make([]string, 0)
	rolesNotCreated := make([]string, 0)

	if len(data.Claims) > 0 {
		for i, claim := range data.Claims {
			existingClaim, diag := apiclient.GetClaim(ctx, hostConfig, claim.Name.ValueString())
			if diag.HasError() {
				claimsNotCreated = append(claimsNotCreated, claim.Name.ValueString())
				continue
			}
			if existingClaim != nil {
				tflog.Info(ctx, fmt.Sprintf("Claim %s found during create", claim.Name.ValueString()))
				claimsNotCreated = append(claimsNotCreated, claim.Name.ValueString())
				continue
			}

			response, diag := apiclient.CreateClaim(ctx, hostConfig, claim.Name.ValueString())
			if diag.HasError() {
				claimsNotCreated = append(claimsNotCreated, claim.Name.ValueString())
				continue
			} else {
				tflog.Info(ctx, fmt.Sprintf("Claim %s created", claim.Name.ValueString()))
				data.Claims[i].Id = types.StringValue(response.ID)
			}
		}
	}

	if len(data.Roles) > 0 {
		for i, role := range data.Roles {
			existingRole, diag := apiclient.GetRole(ctx, hostConfig, role.Name.ValueString())
			if diag.HasError() {
				rolesNotCreated = append(rolesNotCreated, role.Name.ValueString())
				continue
			}
			if existingRole != nil {
				tflog.Info(ctx, fmt.Sprintf("Role %s found during create", role.Name.ValueString()))
				rolesNotCreated = append(rolesNotCreated, role.Name.ValueString())
				continue
			}

			response, diag := apiclient.CreateRole(ctx, hostConfig, role.Name.ValueString())
			if diag.HasError() {
				rolesNotCreated = append(rolesNotCreated, role.Name.ValueString())
				continue
			} else {
				tflog.Info(ctx, fmt.Sprintf("Role %s created", role.Name.ValueString()))
				data.Roles[i].Id = types.StringValue(response.ID)
			}
		}
	}

	if len(data.ApiKeys) > 0 {
		for i, apiKey := range data.ApiKeys {
			existingApiKey, diag := apiclient.GetApiKey(ctx, hostConfig, apiKey.Key.ValueString())
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
				return
			}

			if existingApiKey != nil {
				tflog.Info(ctx, fmt.Sprintf("API Key %s found during create", apiKey.Name.ValueString()))
				resp.Diagnostics.AddError("api key already exists", fmt.Sprintf("api key %s already exists", apiKey.Name.ValueString()))
				return
			}

			request := apimodels.ApiKeyRequest{
				Name:   apiKey.Name.ValueString(),
				Key:    apiKey.Key.ValueString(),
				Secret: apiKey.Secret.ValueString(),
			}

			response, diag := apiclient.CreateApiKey(ctx, hostConfig, request)
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
				return
			} else {
				data.ApiKeys[i].Id = types.StringValue(response.ID)
				data.ApiKeys[i].ApiKey = types.StringValue(helpers.Base64Encode(fmt.Sprintf("%s:%s", apiKey.Key.ValueString(), apiKey.Secret.ValueString())))
			}
		}
	}

	if len(data.Users) > 0 {
		// Checking for duplicates
		for i, user := range data.Users {
			existingUser, diag := apiclient.GetUser(ctx, hostConfig, user.Username.ValueString())
			if diag.HasError() {
				usersNotCreated = append(usersNotCreated, user.Username.ValueString())
				continue
			}
			if existingUser != nil {
				tflog.Info(ctx, fmt.Sprintf("User %s found during create", user.Username.ValueString()))
				resp.Diagnostics.AddError("user already exists", fmt.Sprintf("user %s already exists", user.Username.ValueString()))
			}

			userModel := apimodels.UserRequest{
				Name:     user.Name.ValueString(),
				Email:    user.Email.ValueString(),
				Username: user.Username.ValueString(),
				Password: user.Password.ValueString(),
			}

			response, diag := apiclient.CreateUser(ctx, hostConfig, userModel)
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
				return
			} else {
				data.Users[i].Id = types.StringValue(response.ID)
			}

			for e, role := range user.Roles {
				roleResponse, diag := apiclient.AddRoleToUser(ctx, hostConfig, data.Users[i].Id.ValueString(), role.Name.ValueString())
				if diag.HasError() {
					resp.Diagnostics.AddError("role not found", fmt.Sprintf("role %s not found", role.Name))
					return
				}
				if roleResponse == nil {
					resp.Diagnostics.AddError("role not found", fmt.Sprintf("role %s not found", role.Name))
					return
				}

				data.Users[i].Roles[e].Id = types.StringValue(roleResponse.ID)
			}

			for e, claim := range user.Claims {
				claimResponse, diag := apiclient.AddClaimToUser(ctx, hostConfig, data.Users[i].Id.ValueString(), claim.Name.ValueString())
				if diag.HasError() {
					resp.Diagnostics.AddError("claim not found", fmt.Sprintf("role %s not found", claim.Name))
					return
				}
				if claimResponse == nil {
					resp.Diagnostics.AddError("claim not found", fmt.Sprintf("role %s not found", claim.Name))
					return
				}
				data.Users[i].Claims[e].Id = types.StringValue(claimResponse.ID)
			}
		}
	}

	// Save data into Terraform state
	if resp.Diagnostics.HasError() {
		if len(usersNotCreated) > 0 {
			for _, user := range usersNotCreated {
				for i, userBlock := range data.Users {
					if userBlock.Name.ValueString() == user {
						data.Users = append(data.Users[:i], data.Users[i+1:]...)
						break
					}
				}
				resp.Diagnostics.AddError("user not created", fmt.Sprintf("user %s not created", user))
			}
		}
		if len(apiKeysNotCreated) > 0 {
			for _, apiKey := range apiKeysNotCreated {
				for i, apiKeyBlock := range data.ApiKeys {
					if apiKeyBlock.Name.ValueString() == apiKey {
						data.ApiKeys = append(data.ApiKeys[:i], data.ApiKeys[i+1:]...)
						break
					}
				}
				resp.Diagnostics.AddError("api key not created", fmt.Sprintf("api key %s not created", apiKey))
			}
		}
		if len(claimsNotCreated) > 0 {
			for _, claim := range claimsNotCreated {
				for i, claimBlock := range data.Claims {
					if claimBlock.Name.ValueString() == claim {
						data.Claims = append(data.Claims[:i], data.Claims[i+1:]...)
						break
					}
				}
				resp.Diagnostics.AddError("claim not created", fmt.Sprintf("claim %s not created", claim))
			}
		}

		if len(rolesNotCreated) > 0 {
			for _, role := range rolesNotCreated {
				for i, roleBlock := range data.Claims {
					if roleBlock.Name.ValueString() == role {
						data.Roles = append(data.Roles[:i], data.Roles[i+1:]...)
						break
					}
				}
				resp.Diagnostics.AddError("role not created", fmt.Sprintf("role %s not created", role))
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *AuthorizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AuthorizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	// hostConfig := apiclient.HostConfig{
	// 	Host:          data.Host.ValueString(),
	// 	License:       r.provider.License.ValueString(),
	// 	Authorization: data.Authenticator,
	// }

	// for i, apiKey := range data.ApiKeys {
	// 	existingApiKey, diag := apiclient.GetApiKey(ctx, hostConfig, apiKey.Key.ValueString())
	// 	if diag.HasError() {
	// 		resp.Diagnostics.Append(diag...)
	// 		return
	// 	}
	// 	if existingApiKey == nil {
	// 		resp.Diagnostics.AddError("api key not found", fmt.Sprintf("api key %s not found", apiKey.Name.ValueString()))
	// 		return
	// 	}
	// 	data.ApiKeys[i].ApiKey = types.StringValue(helpers.Base64Encode(fmt.Sprintf("%s:%s", apiKey.Key.ValueString(), apiKey.Secret.ValueString())))
	// }

	// for _, user := range data.Users {
	// 	currentUser, diag := apiclient.GetUser(ctx, hostConfig, user.Username.ValueString())
	// 	if diag.HasError() {
	// 		resp.Diagnostics.Append(diag...)
	// 		return
	// 	}
	// 	if currentUser == nil {
	// 		resp.Diagnostics.AddError("user not found", fmt.Sprintf("user %s not found", user.Name.ValueString()))
	// 		return
	// 	}
	// }

	// for _, role := range data.Roles {
	// 	currentRole, diag := apiclient.GetRole(ctx, hostConfig, role.Name.ValueString())
	// 	if diag.HasError() {
	// 		resp.Diagnostics.Append(diag...)
	// 		return
	// 	}
	// 	if currentRole == nil {
	// 		resp.Diagnostics.AddError("role not found", fmt.Sprintf("role %s not found", role.Name.ValueString()))
	// 		return
	// 	}

	// }

	// if len(data.Claims) > 0 {
	// 	for _, claim := range data.Claims {
	// 		currentClaim, diag := apiclient.GetClaim(ctx, hostConfig, claim.Name.ValueString())
	// 		if diag.HasError() {
	// 			resp.Diagnostics.Append(diag...)
	// 			return
	// 		}
	// 		if currentClaim == nil {
	// 			resp.Diagnostics.AddError("claim not found", fmt.Sprintf("claim %s not found", claim.Name.ValueString()))
	// 			return
	// 		}
	// 	}
	// }

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *AuthorizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating Authorization Resource")
	var data AuthorizationResourceModel
	var currentData AuthorizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		Host:                 data.Host.ValueString(),
		License:              r.provider.License.ValueString(),
		Authorization:        data.Authenticator,
		DisableTlsValidation: r.provider.DisableTlsValidation.ValueBool(),
	}

	diag := updateClaims(ctx, hostConfig, &data, &currentData)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
	}

	diag = updateRoles(ctx, hostConfig, &data, &currentData)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
	}

	diag = updateApiKeys(ctx, hostConfig, &data, &currentData)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
	}

	diag = updateUsers(ctx, hostConfig, &data, &currentData)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *AuthorizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting Authorization Resource")
	var data AuthorizationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		Host:                 data.Host.ValueString(),
		License:              r.provider.License.ValueString(),
		Authorization:        data.Authenticator,
		DisableTlsValidation: r.provider.DisableTlsValidation.ValueBool(),
	}

	for _, apiKey := range data.ApiKeys {
		diag := apiclient.DeleteApiKey(ctx, hostConfig, apiKey.Id.ValueString())
		if diag.HasError() {
			tflog.Info(ctx, fmt.Sprintf("Error deleting api key %s", apiKey.Key.ValueString()))
			diag := apiclient.DeleteApiKey(ctx, hostConfig, apiKey.Key.ValueString())
			if diag.HasError() {
				tflog.Info(ctx, fmt.Sprintf("Error1 deleting api key %s", apiKey.Key.ValueString()))
				resp.Diagnostics.Append(diag...)
				return
			}
			tflog.Info(ctx, fmt.Sprintf("Api Key %s deleted", apiKey.Key.ValueString()))
		}
	}

	for _, user := range data.Users {
		diag := apiclient.DeleteUser(ctx, hostConfig, user.Id.ValueString())
		if diag.HasError() {
			diag := apiclient.DeleteUser(ctx, hostConfig, user.Username.ValueString())
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
			}
		}
	}

	for _, role := range data.Roles {
		diag := apiclient.DeleteRole(ctx, hostConfig, role.Id.ValueString())
		if diag.HasError() {
			diag := apiclient.DeleteRole(ctx, hostConfig, role.Name.ValueString())
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
			}
		}
	}

	for _, claim := range data.Claims {
		diag := apiclient.DeleteClaim(ctx, hostConfig, claim.Id.ValueString())
		if diag.HasError() {
			diag := apiclient.DeleteClaim(ctx, hostConfig, claim.Name.ValueString())
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
			}
		}
	}

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		tflog.Info(ctx, "Error deleting Authorization Resource")
		return
	}
}

func (r *AuthorizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func updateClaims(ctx context.Context, hostConfig apiclient.HostConfig, data, currentData *AuthorizationResourceModel) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	// If there is not any claim we delete all of them
	if len(data.Claims) == 0 {
		for _, claim := range currentData.Claims {
			diag := apiclient.DeleteClaim(ctx, hostConfig, claim.Id.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				return diagnostics
			}
		}
	}

	// Let's see if we need to delete any claim
	for _, claim := range currentData.Claims {
		found := false
		for _, newClaim := range data.Claims {
			if claim.Name.ValueString() == newClaim.Name.ValueString() {
				found = true
				break
			}
		}

		if !found {
			diag := apiclient.DeleteClaim(ctx, hostConfig, claim.Id.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				return diagnostics
			}
		}
	}

	// If there is not any claim we delete all of them
	for i, claim := range data.Claims {
		found := false
		for _, currentClaim := range currentData.Claims {
			if claim.Name.ValueString() == currentClaim.Name.ValueString() {
				found = true
				break
			}
		}
		if !found {
			tflog.Info(ctx, fmt.Sprintf("Creating claim %s", claim.Name.ValueString()))
			response, diag := apiclient.CreateClaim(ctx, hostConfig, claim.Name.ValueString())
			if diag.HasError() {
				tflog.Info(ctx, fmt.Sprintf("Error creating claim %s", claim.Name.ValueString()))

				diagnostics.Append(diag...)
			} else {
				tflog.Info(ctx, fmt.Sprintf("Claim %s created", claim.Name.ValueString()))
				data.Claims[i].Id = types.StringValue(response.ID)
			}
		} else {
			tflog.Info(ctx, fmt.Sprintf("Claim %s already exists", claim.Name.ValueString()))
			claimExists, diag := apiclient.GetClaim(ctx, hostConfig, claim.Name.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				break
			}
			if claimExists == nil {
				diagnostics.AddError("claim not found", fmt.Sprintf("claim %s not found", claim.Name))
				break
			}

			tflog.Info(ctx, fmt.Sprintf("Claim %s found, updating id %v", claimExists.Name, claimExists.ID))
			data.Claims[i].Id = types.StringValue(claimExists.ID)
		}
	}

	return diagnostics
}

func updateRoles(ctx context.Context, hostConfig apiclient.HostConfig, data, currentData *AuthorizationResourceModel) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}
	// If there is not any claim we delete all of them
	if len(data.Roles) == 0 {
		for _, role := range currentData.Roles {
			diag := apiclient.DeleteRole(ctx, hostConfig, role.Id.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				return diagnostics
			}
		}
	}

	// Let's see if we need to delete any role
	for _, role := range currentData.Roles {
		found := false
		for _, newRole := range data.Roles {
			if role.Name.ValueString() == newRole.Name.ValueString() {
				found = true
				break
			}
		}

		if !found {
			diag := apiclient.DeleteRole(ctx, hostConfig, role.Id.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				return diagnostics
			}
		}
	}

	// If there is not any claim we delete all of them
	for i, role := range data.Roles {
		found := false
		for _, currentRole := range currentData.Roles {
			if role.Name.ValueString() == currentRole.Name.ValueString() {
				found = true
				break
			}
		}
		if !found {
			tflog.Info(ctx, fmt.Sprintf("Creating Role %s", role.Name.ValueString()))
			response, diag := apiclient.CreateRole(ctx, hostConfig, role.Name.ValueString())
			if diag.HasError() {
				tflog.Info(ctx, fmt.Sprintf("Error creating role %s", role.Name.ValueString()))
				diagnostics.Append(diag...)
				data.Roles[i].Id = types.StringValue("-")
				continue
			} else {
				tflog.Info(ctx, fmt.Sprintf("Role %s created", role.Name.ValueString()))
				data.Roles[i].Id = types.StringValue(response.ID)
			}
		} else {
			tflog.Info(ctx, fmt.Sprintf("Role %s already exists", role.Name.ValueString()))
			roleExists, diag := apiclient.GetRole(ctx, hostConfig, role.Name.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				data.Roles[i].Id = types.StringValue("-")
				continue
			}
			if roleExists == nil {
				diagnostics.AddError("role not found", fmt.Sprintf("role %s not found", role.Name))
				data.Roles[i].Id = types.StringValue("-")
				continue
			}
			tflog.Info(ctx, fmt.Sprintf("Role %s found, updating id %v", roleExists.Name, roleExists.ID))
			data.Roles[i].Id = types.StringValue(roleExists.ID)
		}
	}

	return diagnostics
}

func updateApiKeys(ctx context.Context, hostConfig apiclient.HostConfig, data, currentData *AuthorizationResourceModel) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	// If there is not any api key we delete all of them
	if len(data.ApiKeys) == 0 {
		for _, apiKey := range currentData.ApiKeys {
			diag := apiclient.DeleteApiKey(ctx, hostConfig, apiKey.Id.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				return diagnostics
			}
		}
	}

	// Let's see if we need to delete any api key
	for _, apiKey := range currentData.ApiKeys {
		found := false
		for _, newApiKey := range data.ApiKeys {
			if apiKey.Name.ValueString() == newApiKey.Name.ValueString() {
				found = true
				break
			}
		}

		if !found {
			diag := apiclient.DeleteApiKey(ctx, hostConfig, apiKey.Id.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				return diagnostics
			}
		}
	}

	// If there is not any api key we delete all of them
	for i, apiKey := range data.ApiKeys {
		found := false
		for _, currentApiKey := range currentData.ApiKeys {
			if apiKey.Name.ValueString() == currentApiKey.Name.ValueString() {
				found = true
				break
			}
		}
		if !found {
			tflog.Info(ctx, fmt.Sprintf("Creating api key %s", apiKey.Name.ValueString()))
			request := apimodels.ApiKeyRequest{
				Name:   apiKey.Name.ValueString(),
				Key:    apiKey.Key.ValueString(),
				Secret: apiKey.Secret.ValueString(),
			}

			response, diag := apiclient.CreateApiKey(ctx, hostConfig, request)
			if diag.HasError() {
				tflog.Info(ctx, fmt.Sprintf("Error creating api key %s", apiKey.Name.ValueString()))
				diagnostics.Append(diag...)
				continue
			} else {
				tflog.Info(ctx, fmt.Sprintf("Api Key %s created", apiKey.Name.ValueString()))
				data.ApiKeys[i].Id = types.StringValue(response.ID)
				data.ApiKeys[i].ApiKey = types.StringValue(helpers.Base64Encode(fmt.Sprintf("%s:%s", apiKey.Key.ValueString(), apiKey.Secret.ValueString())))

			}
		} else {
			tflog.Info(ctx, fmt.Sprintf("Api Key %s already exists", apiKey.Name.ValueString()))
			apiKeyExists, diag := apiclient.GetApiKey(ctx, hostConfig, apiKey.Name.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				break
			}
			if apiKeyExists == nil {
				diagnostics.AddError("api key not found", fmt.Sprintf("api key %s not found", apiKey.Name))
				break
			}
			tflog.Info(ctx, fmt.Sprintf("ApiKey %s found, updating id %v", apiKeyExists.Name, apiKeyExists.ID))
			data.ApiKeys[i].Id = types.StringValue(apiKeyExists.ID)
			data.ApiKeys[i].ApiKey = types.StringValue(helpers.Base64Encode(fmt.Sprintf("%s:%s", data.ApiKeys[i].Key.ValueString(), data.ApiKeys[i].Secret.ValueString())))
		}
	}

	return diagnostics
}

func updateUsers(ctx context.Context, hostConfig apiclient.HostConfig, data, currentData *AuthorizationResourceModel) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	// If there is not any users, we delete all of them
	if len(data.Users) == 0 {
		for _, user := range currentData.Users {
			diag := apiclient.DeleteUser(ctx, hostConfig, user.Id.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				return diagnostics
			}
		}
	}

	// Let's see if we need to delete any user, the rule is simple if we have it in the current
	// state and not in the new state we delete it
	for _, user := range currentData.Users {
		found := false
		for _, newUser := range data.Users {
			if user.Name.ValueString() == newUser.Name.ValueString() {
				found = true
				break
			}
		}

		if !found {
			diag := apiclient.DeleteUser(ctx, hostConfig, user.Id.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				return diagnostics
			}
		}
	}

	// We will now check if there is any user like that in the current state,
	// if not we will create it otherwise we will just update the id
	for i, user := range data.Users {
		found := false
		for _, currentUser := range currentData.Users {
			if user.Username.ValueString() == currentUser.Username.ValueString() {
				found = true
				break
			}
		}
		if !found {
			tflog.Info(ctx, fmt.Sprintf("Creating User %s", user.Username.ValueString()))
			request := apimodels.UserRequest{
				Name:     user.Name.ValueString(),
				Email:    user.Email.ValueString(),
				Username: user.Username.ValueString(),
				Password: user.Password.ValueString(),
			}

			response, diag := apiclient.CreateUser(ctx, hostConfig, request)
			if diag.HasError() {
				tflog.Info(ctx, fmt.Sprintf("Error creating user %s", user.Username.ValueString()))
				diagnostics.Append(diag...)
				continue
			} else {
				tflog.Info(ctx, fmt.Sprintf("User %s created", user.Username.ValueString()))
				if diag := updateUserClaims(ctx, hostConfig, data, currentData, i); diag.HasError() {
					diagnostics.Append(diag...)
					continue
				}
				if diag := updateUserRoles(ctx, hostConfig, data, currentData, i); diag.HasError() {
					diagnostics.Append(diag...)
					continue
				}
				data.Users[i].Id = types.StringValue(response.ID)
			}
		} else {
			tflog.Info(ctx, fmt.Sprintf("User %s already exists", user.Username.ValueString()))
			userExists, diag := apiclient.GetUser(ctx, hostConfig, user.Username.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				continue
			}
			if userExists == nil {
				diagnostics.AddError("user not found", fmt.Sprintf("user %s not found", user.Username))
				continue
			}
			tflog.Info(ctx, fmt.Sprintf("User %s found, updating id %v", userExists.Name, userExists.ID))
			if diag := updateUserClaims(ctx, hostConfig, data, currentData, i); diag.HasError() {
				diagnostics.Append(diag...)
				continue
			}
			if diag := updateUserRoles(ctx, hostConfig, data, currentData, i); diag.HasError() {
				diagnostics.Append(diag...)
				continue
			}
			data.Users[i].Id = types.StringValue(userExists.ID)
		}
	}

	return diagnostics
}

func updateUserRoles(ctx context.Context, hostConfig apiclient.HostConfig, data, currentData *AuthorizationResourceModel, userIndex int) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	// We will now check if there is any user like that in the current state,
	// if not we will create it otherwise we will just update the id
	for i, role := range data.Users[userIndex].Roles {
		found := false
		for _, currentUserRole := range currentData.Users[userIndex].Roles {
			if role == currentUserRole {
				found = true
				break
			}
		}

		if !found {
			roleResponse, diag := apiclient.AddRoleToUser(ctx, hostConfig, data.Users[userIndex].Id.ValueString(), role.Name.ValueString())
			if diag.HasError() {
				diag := apiclient.DeleteUser(ctx, hostConfig, data.Users[i].Id.ValueString())
				if diag.HasError() {
					diagnostics.Append(diag...)
					continue
				}
			} else {
				data.Users[userIndex].Roles[i].Id = types.StringValue(roleResponse.ID)
			}
		} else {
			roleExists, diag := apiclient.GetRole(ctx, hostConfig, role.Name.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				continue
			}
			if roleExists == nil {
				diagnostics.AddError("role not found", fmt.Sprintf("role %s not found", role.Name))
				continue
			}
			data.Users[userIndex].Roles[i].Id = types.StringValue(roleExists.ID)
		}
	}

	return diagnostics
}

func updateUserClaims(ctx context.Context, hostConfig apiclient.HostConfig, data, currentData *AuthorizationResourceModel, userIndex int) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	// We will now check if there is any user like that in the current state,
	// if not we will create it otherwise we will just update the id
	for i, claim := range data.Users[userIndex].Claims {
		found := false
		for _, currentUserClaim := range currentData.Users[userIndex].Claims {
			if claim == currentUserClaim {
				found = true
				break
			}
		}

		if !found {
			claimResponse, diag := apiclient.AddClaimToUser(ctx, hostConfig, data.Users[userIndex].Id.ValueString(), claim.Name.ValueString())
			if diag.HasError() {
				diag := apiclient.DeleteUser(ctx, hostConfig, data.Users[i].Id.ValueString())
				if diag.HasError() {
					diagnostics.Append(diag...)
					continue
				}
			} else {
				data.Users[userIndex].Claims[i].Id = types.StringValue(claimResponse.ID)
			}
		} else {
			claimExists, diag := apiclient.GetClaim(ctx, hostConfig, claim.Name.ValueString())
			if diag.HasError() {
				diagnostics.Append(diag...)
				continue
			}
			if claimExists == nil {
				diagnostics.AddError("claim not found", fmt.Sprintf("claim %s not found", claim.Name))
				continue
			}
			data.Users[userIndex].Claims[i].Id = types.StringValue(claimExists.ID)
		}
	}

	return diagnostics
}
