package authorization

import (
	"context"
	"fmt"
	"terraform-provider-parallels/internal/clientmodels"
	"terraform-provider-parallels/internal/constants"
	"terraform-provider-parallels/internal/helpers"
	"terraform-provider-parallels/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AuthorizationResource{}
var _ resource.ResourceWithImportState = &AuthorizationResource{}

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
	return
}

func (r *AuthorizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AuthorizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	client := helpers.NewHttpCaller(ctx)
	password := r.provider.License.ValueString()
	token, err := client.GetJwtToken(data.Host.ValueString(), constants.RootUser, password)
	if err != nil {
		resp.Diagnostics.AddError("error getting token", err.Error())
		return
	}
	auth := helpers.HttpCallerAuth{
		BearerToken: token,
	}

	usersNotCreated := make([]string, 0)
	apiKeysNotCreated := make([]string, 0)

	if len(data.ApiKeys) > 0 {
		// Checking for duplicates
		for i, apiKey := range data.ApiKeys {
			if response, err := r.createApiKey(ctx, data.Host.ValueString(), auth, apiKey.Name.ValueString(), apiKey.Key.ValueString(), apiKey.Secret.ValueString()); err != nil {
				resp.Diagnostics.AddError("error creating api key", err.Error())
				apiKeysNotCreated = append(apiKeysNotCreated, apiKey.Name.ValueString())
			} else {
				data.ApiKeys[i].Id = types.StringValue(response.ID)
				data.ApiKeys[i].ApiKey = types.StringValue(helpers.Base64Encode(fmt.Sprintf("%s:%s", apiKey.Key.ValueString(), apiKey.Secret.ValueString())))
			}
		}
	}

	if len(data.Users) > 0 {
		// Checking for duplicates
		for i, user := range data.Users {
			userModel := clientmodels.CreateUserRequest{
				Name:     user.Name.ValueString(),
				Email:    user.Email.ValueString(),
				Username: user.Username.ValueString(),
				Password: user.Password.ValueString(),
			}
			if response, err := r.createUser(ctx, data.Host.ValueString(), auth, userModel); err != nil {
				resp.Diagnostics.AddError("error creating user", err.Error())
				usersNotCreated = append(usersNotCreated, user.Name.ValueString())
			} else {
				data.Users[i].Id = types.StringValue(response.ID)
			}
			for e, role := range user.Roles {
				if roleResponse, err := r.addRoleToUser(ctx, data.Host.ValueString(), auth, data.Users[i].Id.ValueString(), role.Name.ValueString()); err != nil {
					resp.Diagnostics.AddError("error adding roles to user", err.Error())
					usersNotCreated = append(usersNotCreated, user.Name.ValueString())
					client.DeleteDataFromClient(data.Host.ValueString()+"/api/v1/auth/users/"+data.Users[i].Id.ValueString(), nil, auth, nil)
					break
				} else {
					data.Users[i].Roles[e].Id = types.StringValue(roleResponse.ID)
				}
			}
			for e, claim := range user.Claims {
				if claimResponse, err := r.addClaimToUser(ctx, data.Host.ValueString(), auth, data.Users[i].Id.ValueString(), claim.Name.ValueString()); err != nil {
					resp.Diagnostics.AddError("error adding claims to user", err.Error())
					usersNotCreated = append(usersNotCreated, user.Name.ValueString())
					client.DeleteDataFromClient(data.Host.ValueString()+"/api/v1/auth/users/"+data.Users[i].Id.ValueString(), nil, auth, nil)
					break
				} else {
					data.Users[i].Claims[e].Id = types.StringValue(claimResponse.ID)
				}
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

	client := helpers.NewHttpCaller(ctx)
	password := r.provider.License.ValueString()
	token, err := client.GetJwtToken(data.Host.ValueString(), constants.RootUser, password)
	if err != nil {
		resp.Diagnostics.AddError("error getting token", err.Error())
		return
	}
	auth := helpers.HttpCallerAuth{
		BearerToken: token,
	}

	if len(data.ApiKeys) > 0 {
		// Checking for duplicates
		for i, apiKey := range data.ApiKeys {
			var currentApiKey clientmodels.APIKeyResponse
			client.GetDataFromClient(data.Host.ValueString()+"/api/v1/auth/api_keys/"+apiKey.Key.ValueString(), nil, auth, &currentApiKey)
			if currentApiKey.ID != "" {
				tflog.Info(ctx, fmt.Sprintf("API Key %s found during read", apiKey.Id.ValueString()))
				data.ApiKeys[i].Id = types.StringValue(currentApiKey.ID)
				data.ApiKeys[i].ApiKey = types.StringValue(helpers.Base64Encode(fmt.Sprintf("%s:%s", apiKey.Key.ValueString(), apiKey.Secret.ValueString())))
			} else {
				tflog.Info(ctx, fmt.Sprintf("API Key %s not found during read", apiKey.Id.ValueString()))
				resp.Diagnostics.AddError("API Key not found", fmt.Sprintf("API Key %s not found during read", apiKey.Id.ValueString()))
				return
			}
		}
	}

	if len(data.ApiKeys) > 0 {
		// Checking for duplicates
		for i, user := range data.Users {
			var currentUser clientmodels.APIKeyResponse
			client.GetDataFromClient(data.Host.ValueString()+"/api/v1/auth/users/"+user.Id.ValueString(), nil, auth, &currentUser)
			if currentUser.ID != "" {
				tflog.Info(ctx, fmt.Sprintf("User %s found during read", user.Id.ValueString()))
				data.Users[i].Id = types.StringValue(currentUser.ID)
			} else {
				tflog.Info(ctx, fmt.Sprintf("user %s not found during read", user.Id.ValueString()))
				resp.Diagnostics.AddError("user not found", fmt.Sprintf("user %s not found during read", user.Id.ValueString()))
				return
			}
		}
	}

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *AuthorizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AuthorizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := helpers.NewHttpCaller(ctx)
	password := r.provider.License.ValueString()
	token, err := client.GetJwtToken(data.Host.ValueString(), constants.RootUser, password)
	if err != nil {
		resp.Diagnostics.AddError("error getting token", err.Error())
		return
	}
	auth := helpers.HttpCallerAuth{
		BearerToken: token,
	}

	if len(data.ApiKeys) > 0 {
		// Checking for duplicates
		for i, apiKey := range data.ApiKeys {
			if apiKey.Id.IsNull() || apiKey.Id.IsUnknown() || apiKey.Id.ValueString() == "" {
				tflog.Info(ctx, fmt.Sprintf("New API Key %s found, creating", apiKey.Id.ValueString()))
				if response, err := r.createApiKey(ctx, data.Host.ValueString(), auth, apiKey.Name.ValueString(), apiKey.Key.ValueString(), apiKey.Secret.ValueString()); err != nil {
					resp.Diagnostics.AddError("error creating api key", err.Error())
				} else {
					data.ApiKeys[i].Id = types.StringValue(response.ID)
					data.ApiKeys[i].ApiKey = types.StringValue(helpers.Base64Encode(fmt.Sprintf("%s:%s", apiKey.Key.ValueString(), apiKey.Secret.ValueString())))
				}
			} else {
				existingApiKey := clientmodels.APIKeyResponse{}
				client.GetDataFromClient(data.Host.ValueString()+"/api/v1/auth/api_keys/"+apiKey.Id.ValueString(), nil, auth, &existingApiKey)
				if existingApiKey.ID != "" {
					tflog.Info(ctx, fmt.Sprintf("API Key %s found, deleting", apiKey.Id.ValueString()))
					if _, err := client.DeleteDataFromClient(data.Host.ValueString()+"/api/v1/auth/api_keys/"+apiKey.Id.ValueString(), nil, auth, nil); err != nil {
						resp.Diagnostics.AddError("error deleting api key", err.Error())
					}
				} else {
					tflog.Info(ctx, fmt.Sprintf("API Key %s not found, creating", apiKey.Id.ValueString()))
				}

				if response, err := r.createApiKey(ctx, data.Host.ValueString(), auth, apiKey.Name.ValueString(), apiKey.Key.ValueString(), apiKey.Secret.ValueString()); err != nil {
					resp.Diagnostics.AddError("error creating api key", err.Error())
				} else {
					data.ApiKeys[i].Id = types.StringValue(response.ID)
					data.ApiKeys[i].ApiKey = types.StringValue(helpers.Base64Encode(fmt.Sprintf("%s:%s", apiKey.Key.ValueString(), apiKey.Secret.ValueString())))
				}
			}
		}
	}

	if len(data.Users) > 0 {
		// Checking for duplicates
		for i, user := range data.Users {
			userModel := clientmodels.CreateUserRequest{
				Name:     user.Name.ValueString(),
				Email:    user.Email.ValueString(),
				Username: user.Username.ValueString(),
				Password: user.Password.ValueString(),
			}
			if user.Id.IsNull() || user.Id.IsUnknown() || user.Id.ValueString() == "" {
				tflog.Info(ctx, fmt.Sprintf("New user %s found, creating", user.Id.ValueString()))

				if response, err := r.createUser(ctx, data.Host.ValueString(), auth, userModel); err != nil {
					resp.Diagnostics.AddError("error creating user", err.Error())
				} else {
					data.Users[i].Id = types.StringValue(response.ID)
				}
			} else {
				existingUser := clientmodels.CreateUserResponse{}
				client.GetDataFromClient(data.Host.ValueString()+"/api/v1/auth/users/"+user.Id.ValueString(), nil, auth, &existingUser)
				if existingUser.ID != "" {
					tflog.Info(ctx, fmt.Sprintf("User %s found, deleting", user.Id.ValueString()))
					if _, err := client.DeleteDataFromClient(data.Host.ValueString()+"/api/v1/auth/users/"+user.Id.ValueString(), nil, auth, nil); err != nil {
						resp.Diagnostics.AddError("error deleting user", err.Error())
					}
				} else {
					tflog.Info(ctx, fmt.Sprintf("user %s not found, creating", user.Id.ValueString()))
				}

				if response, err := r.createUser(ctx, data.Host.ValueString(), auth, userModel); err != nil {
					resp.Diagnostics.AddError("error creating user", err.Error())
				} else {
					data.Users[i].Id = types.StringValue(response.ID)
				}
			}
		}
	}

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	return
}

func (r *AuthorizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AuthorizationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := helpers.NewHttpCaller(ctx)
	password := r.provider.License.ValueString()
	token, err := client.GetJwtToken(data.Host.ValueString(), constants.RootUser, password)
	if err != nil {
		resp.Diagnostics.AddError("error getting token", err.Error())
		return
	}
	auth := helpers.HttpCallerAuth{
		BearerToken: token,
	}

	if len(data.ApiKeys) > 0 {
		// Checking for duplicates
		deletedKeys := make([]string, 0)
		for _, apiKey := range data.ApiKeys {
			if apiKey.Id.ValueString() != "" {
				if _, err := client.DeleteDataFromClient(data.Host.ValueString()+"/api/v1/auth/api_keys/"+apiKey.Id.ValueString(), nil, auth, nil); err != nil {
					resp.Diagnostics.AddError("error deleting api key", err.Error())

				}
				deletedKeys = append(deletedKeys, apiKey.Id.ValueString())
			}
		}
		for len(deletedKeys) > 0 {
			for i, apiKey := range data.ApiKeys {
				if apiKey.Id.ValueString() == deletedKeys[0] {
					data.ApiKeys = append(data.ApiKeys[:i], data.ApiKeys[i+1:]...)
					deletedKeys = deletedKeys[1:]
					break
				}
			}
		}
	}

	if len(data.Users) > 0 {
		// Checking for duplicates
		deletedUsers := make([]string, 0)
		for _, apiKey := range data.Users {
			if apiKey.Id.ValueString() != "" {
				if _, err := client.DeleteDataFromClient(data.Host.ValueString()+"/api/v1/auth/users/"+apiKey.Id.ValueString(), nil, auth, nil); err != nil {
					resp.Diagnostics.AddError("error deleting user", err.Error())
				}
				deletedUsers = append(deletedUsers, apiKey.Id.ValueString())
			}
		}
		for len(deletedUsers) > 0 {
			for i, apiKey := range data.Users {
				if apiKey.Id.ValueString() == deletedUsers[0] {
					data.Users = append(data.Users[:i], data.Users[i+1:]...)
					deletedUsers = deletedUsers[1:]
					break
				}
			}
		}
	}

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *AuthorizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AuthorizationResource) createApiKey(ctx context.Context, host string, auth helpers.HttpCallerAuth, name, key, secret string) (*clientmodels.APIKeyResponse, error) {
	client := helpers.NewHttpCaller(ctx)
	tflog.Info(ctx, "Creating API Key "+name)
	request := clientmodels.APIKeyRequest{
		Name:   name,
		Key:    key,
		Secret: secret,
	}
	var response clientmodels.APIKeyResponse

	if resp, err := client.PostDataToClient(host+"/api/v1/auth/api_keys", nil, request, auth, &response); err != nil {
		if resp != nil && resp.ApiError != nil {
			return nil, fmt.Errorf("error creating api key %s, %v", name, resp.ApiError.Message)
		} else {
			return nil, err
		}
	}

	return &response, nil
}

func (r *AuthorizationResource) createUser(ctx context.Context, host string, auth helpers.HttpCallerAuth, user clientmodels.CreateUserRequest) (*clientmodels.APIKeyResponse, error) {
	client := helpers.NewHttpCaller(ctx)
	tflog.Info(ctx, "Creating User "+user.Name)
	var response clientmodels.APIKeyResponse

	if resp, err := client.PostDataToClient(host+"/api/v1/auth/users", nil, user, auth, &response); err != nil {
		if resp != nil && resp.ApiError != nil {
			return nil, fmt.Errorf("error creating user %s, %v", user.Name, resp.ApiError.Message)
		} else {
			return nil, err
		}
	}

	return &response, nil
}

func (r *AuthorizationResource) addRoleToUser(ctx context.Context, host string, auth helpers.HttpCallerAuth, userId string, role string) (*clientmodels.ClaimRole, error) {
	if role == "" {
		return nil, nil
	}

	client := helpers.NewHttpCaller(ctx)
	tflog.Info(ctx, "Adding Role "+role+" to User "+userId)
	requestBody := clientmodels.UserClaimRoleCreate{
		Name: role,
	}
	var respClaim clientmodels.ClaimRole
	if resp, err := client.PostDataToClient(fmt.Sprintf("%s/api/v1/auth/users/%s/claims", host, userId), nil, &requestBody, auth, &respClaim); err != nil {
		if resp != nil && resp.ApiError != nil {
			return nil, fmt.Errorf("error creating user %s role %s, %v", userId, role, resp.ApiError.Message)
		} else {
			return nil, err
		}
	}

	return &respClaim, nil
}

func (r *AuthorizationResource) addClaimToUser(ctx context.Context, host string, auth helpers.HttpCallerAuth, userId string, claim string) (*clientmodels.ClaimRole, error) {
	if claim == "" {
		return nil, nil
	}

	client := helpers.NewHttpCaller(ctx)
	tflog.Info(ctx, "Adding Claim "+claim+" to User "+userId)
	requestBody := clientmodels.UserClaimRoleCreate{
		Name: claim,
	}
	var respClaim clientmodels.ClaimRole
	if resp, err := client.PostDataToClient(fmt.Sprintf("%s/api/v1/auth/users/%s/claims", host, userId), nil, &requestBody, auth, &respClaim); err != nil {
		if resp != nil && resp.ApiError != nil {
			return nil, fmt.Errorf("error creating user %s claim %s, %v", userId, claim, resp.ApiError.Message)
		} else {
			return nil, err
		}
	}

	return &respClaim, nil
}
