package apiclient

import (
	"context"
	"fmt"

	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func DeleteVm(ctx context.Context, config HostConfig, machineId string) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}
	urlHost := helpers.GetHostUrl(config.Host)
	if machineId == "" {
		diagnostic.AddError("There was an error deleting the vm", "machineId is empty")
		return diagnostic
	}

	url := fmt.Sprintf("%s/machines/%s", helpers.GetHostApiVersionedBaseUrl(urlHost), machineId)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostic.AddError("There was an error getting the authenticator", err.Error())
		return diagnostic
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if _, err := client.DeleteDataFromClient(url, nil, auth, nil); err != nil {
		diagnostic.AddError("There was an error deleting the vm", err.Error())
		return diagnostic
	}

	tflog.Info(ctx, "Deleted vm "+machineId)

	return diagnostic
}
