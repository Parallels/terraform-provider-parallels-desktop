package common

import (
	"context"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/schemas/sharedfolder"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func SharedFoldersBlockOnCreate(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, planSharedFolder []*sharedfolder.SharedFolder) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	for _, sharedFolder := range planSharedFolder {
		tflog.Info(ctx, "Shared folder "+sharedFolder.Name.ValueString()+" does not exist, creating it")
		diag := sharedFolder.Add(ctx, hostConfig, *vm)
		if diag.HasError() {
			diagnostics.Append(diag...)
			return diagnostics
		}
	}

	return diagnostics
}

func SharedFoldersBlockOnUpdate(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, planSharedFolder, stateSharedFolder []*sharedfolder.SharedFolder) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	// First lets remove the current shared folders in the current state that do not exist in the new state
	for _, current := range stateSharedFolder {
		// checking if this is a new one or if it existed before
		if planSharedFolder != nil {
			var newSharedFolder *sharedfolder.SharedFolder
			for _, sharedFolder := range planSharedFolder {
				if current.Name.ValueString() == sharedFolder.Name.ValueString() {
					newSharedFolder = sharedFolder
					break
				}
			}

			if newSharedFolder == nil {
				// It no longer exists, we need to delete it
				tflog.Info(ctx, "Current shared folder "+current.Name.ValueString()+" needs deleting as is does not exist in the current state")
				diag := current.Delete(ctx, hostConfig, *vm)
				if diag.HasError() {
					diagnostics.Append(diag...)
					return diagnostics
				}
			}
		} else {
			// There is no new shared folders so we need to remove it
			tflog.Info(ctx, "Current shared folder "+current.Name.ValueString()+" needs deleting as there is no new shared folders")
			diag := current.Delete(ctx, hostConfig, *vm)
			if diag.HasError() {
				diagnostics.Append(diag...)
				return diagnostics
			}
		}
	}

	for _, sharedFolder := range planSharedFolder {
		// checking if this is a new one or if it existed before
		var currentSharedFolder *sharedfolder.SharedFolder

		for _, current := range stateSharedFolder {
			if current.Name.ValueString() == sharedFolder.Name.ValueString() {
				currentSharedFolder = current
				break
			}
		}

		if currentSharedFolder == nil {
			// It exists, we need to update it
			tflog.Info(ctx, "Shared folder "+sharedFolder.Name.ValueString()+" does not exist, creating it")
			diag := sharedFolder.Add(ctx, hostConfig, *vm)
			if diag.HasError() {
				diagnostics.Append(diag...)
				return diagnostics
			}
		} else {
			hasChanges := false
			if currentSharedFolder.Description.ValueString() != sharedFolder.Description.ValueString() {
				hasChanges = true
			}
			if currentSharedFolder.Disabled.ValueBool() != sharedFolder.Disabled.ValueBool() {
				hasChanges = true
			}
			if currentSharedFolder.Path.ValueString() != sharedFolder.Path.ValueString() {
				hasChanges = true
			}
			if currentSharedFolder.Readonly.ValueBool() != sharedFolder.Readonly.ValueBool() {
				hasChanges = true
			}

			if currentSharedFolder.Name.ValueString() != sharedFolder.Name.ValueString() {
				tflog.Info(ctx, "Shared folder "+sharedFolder.Name.ValueString()+" changed the name, deleting the current one and creating a new one")
				// this is a special case that needs to delete the current one and create a new one
				currentDiag := currentSharedFolder.Delete(ctx, hostConfig, *vm)
				if currentDiag.HasError() {
					diagnostics.Append(currentDiag...)
					return diagnostics
				}
				newDiag := sharedFolder.Add(ctx, hostConfig, *vm)
				if newDiag.HasError() {
					diagnostics.Append(newDiag...)
					return diagnostics
				}
			}

			if hasChanges {
				tflog.Info(ctx, "Shared folder "+sharedFolder.Name.ValueString()+" exist, updating it")
				// It does not exist, we need to create it
				diag := sharedFolder.Update(ctx, hostConfig, *vm)
				if diag.HasError() {
					diagnostics.Append(diag...)
					return diagnostics
				}
			} else {
				tflog.Info(ctx, "Shared folder "+sharedFolder.Name.ValueString()+" exist but no changes found")
			}
		}
	}

	return diagnostics
}
