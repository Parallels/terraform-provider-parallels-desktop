package deploy

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/url"
	"slices"
	"strings"
	"time"

	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/deploy/models"
	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func retryableServiceError(err error) bool {
	var status *helpers.HTTPStatusError
	if errors.As(err, &status) {
		return status.StatusCode >= 500
	}
	var cert x509.UnknownAuthorityError
	var invalid x509.CertificateInvalidError
	var hostname x509.HostnameError
	if errors.As(err, &cert) || errors.As(err, &invalid) || errors.As(err, &hostname) {
		return false
	}
	var network net.Error
	return errors.As(err, &network) || errors.Is(err, context.DeadlineExceeded)
}

func pollService(ctx context.Context, timeout time.Duration, check func(context.Context) (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	delay := 200 * time.Millisecond
	var last error
	for {
		attempt, cancelAttempt := context.WithTimeout(ctx, 5*time.Second)
		done, err := check(attempt)
		cancelAttempt()
		if err == nil && done {
			return nil
		}
		if err != nil && !retryableServiceError(err) {
			return err
		}
		last = err
		timer := time.NewTimer(delay + time.Duration(rand.Int64N(int64(delay/4)+1))) // #nosec G404 -- jitter only spreads readiness retries.
		select {
		case <-ctx.Done():
			timer.Stop()
			if last != nil {
				return fmt.Errorf("service readiness timed out: %w", last)
			}
			return ctx.Err()
		case <-timer.C:
		}
		delay = min(delay*2, 5*time.Second)
	}
}

func waitForHost(ctx context.Context, cfg models.EffectiveServiceConfig, insecure bool) error {
	caller := helpers.NewHttpCaller(ctx, insecure)
	token := ""
	return pollService(ctx, 2*time.Minute, func(attempt context.Context) (bool, error) {
		if token == "" {
			var err error
			token, err = caller.GetJwtToken(attempt, cfg.Endpoint, "root@localhost", cfg.Environment["ROOT_PASSWORD"])
			if err != nil {
				return false, err
			}
		}
		if !slices.Contains(strings.Split(cfg.Environment["ENABLED_MODULES"], ","), "host") {
			return true, nil
		}
		_, err := caller.GetDataFromClient(attempt, helpers.GetHostApiVersionedBaseUrl(cfg.Endpoint)+"/machines", nil, &helpers.HttpCallerAuth{BearerToken: token}, nil)
		return err == nil, err
	})
}

func deploymentConfig(ctx context.Context, data *models.DeployResourceModelV3, license string) (models.EffectiveServiceConfig, error) {
	host := "localhost"
	if data.SshConnection != nil {
		host = data.SshConnection.Host.ValueString()
	}
	cfg, err := models.ResolveServiceConfig(ctx, data.ApiConfig, license, host, data.Orchestrator != nil)
	if err != nil {
		return cfg, err
	}
	if cfg.Version != "" && cfg.Version != "latest" {
		version, err := normalizeRelease(cfg.Version)
		if err != nil {
			return cfg, err
		}
		if err := validateReleaseSupport(version); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

func validateOrchestratorAuth(auth *authenticator.Authentication) error {
	if auth == nil {
		return errors.New("orchestrator authentication is required")
	}
	if auth.ApiKey.IsUnknown() || auth.Username.IsUnknown() || auth.Password.IsUnknown() {
		return errors.New("orchestrator authentication must be known before deployment")
	}
	key, user, pass := auth.ApiKey.ValueString(), auth.Username.ValueString(), auth.Password.ValueString()
	if key != "" && (user != "" || pass != "") {
		return errors.New("use an orchestrator API key or username and password, not both")
	}
	if key == "" && (user == "" || pass == "") {
		return errors.New("orchestrator requires an API key or both username and password")
	}
	return nil
}

func (r *DeployResource) registerWithOrchestrator(ctx context.Context, data, current *models.DeployResourceModelV3) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if data.Orchestrator == nil {
		setUnregisteredState(data)
		return diagnostics
	}
	fail := func(err error) diag.Diagnostics {
		diagnostics.AddError("Error registering with orchestrator", err.Error()+". The installed host and database were preserved; verify connectivity from the orchestrator to the Host API before retrying.")
		return diagnostics
	}
	cfg, err := deploymentConfig(ctx, data, r.provider.License.ValueString())
	if err != nil {
		return fail(err)
	}
	details := data.Orchestrator.Orchestrator
	if details == nil {
		return fail(errors.New("orchestrator connection details are required"))
	}
	if err := validateOrchestratorDetails(details); err != nil {
		return fail(err)
	}
	caller := helpers.NewHttpCaller(ctx, r.provider.DisableTlsValidation.ValueBool())
	auth := &helpers.HttpCallerAuth{ApiKey: details.UseAuthentication.ApiKey.ValueString()}
	if auth.ApiKey == "" {
		token, err := caller.GetJwtToken(ctx, details.GetHost(), details.UseAuthentication.Username.ValueString(), details.UseAuthentication.Password.ValueString())
		if err != nil {
			return fail(err)
		}
		auth.BearerToken = token
	}
	base := helpers.GetHostApiVersionedBaseUrl(details.GetHost()) + "/orchestrator/hosts"
	id := registrationID(data)
	if current != nil && current.Orchestrator != nil && current.Orchestrator.Orchestrator != nil {
		if current.Orchestrator.Orchestrator.GetHost() != details.GetHost() && registrationID(current) != "" {
			return fail(errors.New("changing orchestrators requires removing the existing registration first"))
		}
		if id == "" {
			id = registrationID(current)
		}
	}
	lookup := func(attempt context.Context) (*apimodels.OrchestratorHost, error) {
		var hosts []apimodels.OrchestratorHost
		if _, err := caller.GetDataFromClient(attempt, base, nil, auth, &hosts); err != nil {
			return nil, err
		}
		if id != "" {
			for i := range hosts {
				if hosts[i].ID == id {
					return &hosts[i], nil
				}
			}
			return nil, nil
		}
		var match *apimodels.OrchestratorHost
		for i := range hosts {
			h := &hosts[i]
			if helpers.GetHostApiBaseUrl(h.Host) == cfg.Endpoint {
				if match != nil && match.ID != h.ID {
					return nil, errors.New("multiple orchestrator hosts match the registration identity")
				}
				match = h
			}
		}
		return match, nil
	}
	record, err := lookup(ctx)
	if err != nil {
		return fail(err)
	}
	request := apimodels.OrchestratorHostRequest{Host: cfg.Endpoint, Description: data.Orchestrator.Description.ValueString(), Tags: data.Orchestrator.Tags, Authentication: &apimodels.OrchestratorAuthentication{Username: "root@localhost", Password: cfg.Environment["ROOT_PASSWORD"]}}
	save := func(record *apimodels.OrchestratorHost) error {
		if err := setRegisteredState(data, record); err != nil {
			return err
		}
		id = record.ID
		return nil
	}
	if record != nil {
		if err := save(record); err != nil {
			return fail(err)
		}
		// Released service update endpoints cannot change tags or clear a description.
		tagsEqual := func(a, b []string) bool {
			a = slices.Clone(a)
			b = slices.Clone(b)
			slices.Sort(a)
			slices.Sort(b)
			return slices.Equal(a, b)
		}
		if !tagsEqual(record.Tags, request.Tags) || (record.Description != "" && request.Description == "") {
			return fail(errors.New("this service API cannot update tags or clear a description in place; remove the registration explicitly before changing these fields"))
		}
		if _, err := caller.PutDataToClient(ctx, base+"/"+url.PathEscape(id), nil, request, auth, nil); err != nil {
			return fail(err)
		}
	} else {
		id = "" // The previous ID was confirmed absent; reconcile a new POST by endpoint.
		var created apimodels.OrchestratorHostResponse
		_, postErr := caller.PostDataToClient(ctx, base, nil, request, auth, &created)
		if postErr != nil {
			// Never repeat a POST after an uncertain outcome. Reconcile reads instead.
			if !retryableServiceError(postErr) {
				return fail(postErr)
			}
			err = pollService(ctx, 30*time.Second, func(attempt context.Context) (bool, error) {
				var e error
				record, e = lookup(attempt)
				return record != nil, e
			})
			if err != nil || record == nil {
				return fail(fmt.Errorf("registration outcome is uncertain; inspect the orchestrator before retrying: %w", postErr))
			}
			if err := save(record); err != nil {
				return fail(err)
			}
		} else {
			if created.ID == "" {
				return fail(errors.New("orchestrator returned an empty registration ID"))
			}
			if err := save(&apimodels.OrchestratorHost{ID: created.ID, Host: cfg.Endpoint}); err != nil {
				return fail(err)
			}
		}
	}
	err = pollService(ctx, 2*time.Minute, func(attempt context.Context) (bool, error) {
		var confirmed apimodels.OrchestratorHost
		_, err := caller.GetDataFromClient(attempt, base+"/"+url.PathEscape(id), nil, auth, &confirmed)
		if err != nil {
			return false, err
		}
		if confirmed.ID != id || helpers.GetHostApiBaseUrl(confirmed.Host) != cfg.Endpoint {
			return false, errors.New("registration readback did not match the expected host identity and endpoint")
		}
		if err := save(&confirmed); err != nil {
			return false, err
		}
		return confirmed.Enabled && strings.EqualFold(confirmed.State, "healthy"), nil
	})
	if err != nil {
		return fail(err)
	}
	return diagnostics
}

func preparePartialDeploymentState(ctx context.Context, data *models.DeployResourceModelV3, dependencies []string) {
	for _, value := range []*types.String{&data.CurrentVersion, &data.CurrentGitVersion, &data.CurrentPackerVersion, &data.CurrentVagrantVersion, &data.ExternalIp, &data.OrchestratorHost, &data.OrchestratorHostId} {
		if value.IsUnknown() {
			*value = types.StringNull()
		}
	}
	if data.IsRegisteredInOrchestrator.IsUnknown() {
		data.IsRegisteredInOrchestrator = types.BoolValue(false)
	}
	if data.Orchestrator != nil && data.Orchestrator.HostId.IsUnknown() {
		data.Orchestrator.HostId = types.StringNull()
	}
	for _, host := range data.ReverseProxyHosts {
		if host.ID.IsUnknown() {
			host.ID = types.StringNull()
		}
	}
	if data.License.IsUnknown() {
		data.License = types.ObjectNull(data.License.AttributeTypes(ctx))
	}
	if data.InstalledDependencies.IsNull() || data.InstalledDependencies.IsUnknown() {
		list, _ := types.ListValueFrom(ctx, types.StringType, dependencies)
		data.InstalledDependencies = list
	}
}
