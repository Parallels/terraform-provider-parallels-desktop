package apiclient

import (
	"context"
	"fmt"

	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func DeleteReverseProxyHost(ctx context.Context, config HostConfig, host string) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}
	urlHost := helpers.GetHostUrl(config.Host)
	if host == "" {
		diagnostic.AddError("There was an error deleting the reverse proxy host", "host is empty")
		return diagnostic
	}

	var url string
	if config.IsOrchestrator {
		url = fmt.Sprintf("%s/orchestrator/hosts/%s/reverse-proxy/hosts/%s", helpers.GetHostApiVersionedBaseUrl(urlHost), config.HostId, host)
	} else {
		url = fmt.Sprintf("%s/reverse-proxy/hosts/%s", helpers.GetHostApiVersionedBaseUrl(urlHost), host)
	}

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostic.AddError("There was an error getting the authenticator", err.Error())
		return diagnostic
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if _, err := client.DeleteDataFromClient(ctx, url, nil, auth, nil); err != nil {
		diagnostic.AddError("There was an error deleting the reverse proxy host", err.Error())
		return diagnostic
	}

	tflog.Info(ctx, "Deleted reverse proxy host "+host)

	return diagnostic
}
