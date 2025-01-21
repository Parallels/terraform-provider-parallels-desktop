package authenticator

import (
	"context"

	"terraform-provider-parallels-desktop/internal/constants"
	"terraform-provider-parallels-desktop/internal/helpers"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func GetAuthenticator(ctx context.Context, host string, license string, authenticator *Authentication, disableTLSVerification bool) (*helpers.HttpCallerAuth, error) {
	client := helpers.NewHttpCaller(ctx, disableTLSVerification)
	var auth helpers.HttpCallerAuth
	if authenticator == nil {
		tflog.Info(ctx, "Authenticator is nil, using root access")
		password := license
		token, err := client.GetJwtToken(ctx, host, constants.RootUser, password)
		if err != nil {
			return nil, err
		}

		auth = helpers.HttpCallerAuth{
			BearerToken: token,
		}
		return &auth, nil
	} else {
		if authenticator.Username.ValueString() != "" {
			password := authenticator.Password.ValueString()
			token, err := client.GetJwtToken(ctx, host, authenticator.Username.ValueString(), password)
			if err != nil {
				return nil, err
			}
			auth = helpers.HttpCallerAuth{
				BearerToken: token,
			}
		} else {
			auth = helpers.HttpCallerAuth{
				ApiKey: authenticator.ApiKey.ValueString(),
			}
		}
		return &auth, nil
	}
}
