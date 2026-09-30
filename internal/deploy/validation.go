package deploy

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"terraform-provider-parallels-desktop/internal/schemas/orchestrator"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"terraform-provider-parallels-desktop/internal/deploy/models"
)

var _ resource.ResourceWithValidateConfig = &DeployResource{}
var _ resource.ResourceWithUpgradeState = &DeployResource{}

func (r *DeployResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	// Configuration may depend on other resources. Apply performs the same validation
	// with resolved values before opening SSH or installing dependencies.
	if !req.Config.Raw.IsFullyKnown() {
		return
	}
	var data models.DeployResourceModelV3
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Provider configuration is not necessarily available during validation. The
	// compatibility password fallback is checked against its real value at apply.
	if _, err := deploymentConfig(ctx, &data, "provider-license-fallback"); err != nil {
		resp.Diagnostics.AddError("Invalid DevOps configuration", err.Error())
	}
	if data.Orchestrator != nil {
		if data.Orchestrator.Orchestrator == nil {
			resp.Diagnostics.AddError("Invalid orchestrator configuration", "orchestrator connection details are required")
			return
		}
		if err := validateOrchestratorDetails(data.Orchestrator.Orchestrator); err != nil {
			resp.Diagnostics.AddError("Invalid orchestrator authentication", err.Error())
		}
	}
}

func validateReleaseSupport(version string) error {
	var major, minor, patch int
	if _, err := fmt.Sscanf(strings.Split(version, "-")[0], "%d.%d.%d", &major, &minor, &patch); err != nil || major != 1 || (minor == 0 && patch < 4) {
		return fmt.Errorf("this deployment adapter requires DevOps Service >=1.0.4 and <2.0.0")
	}
	return nil
}

func validateOrchestratorDetails(details *orchestrator.OrchestratorDetails) error {
	if details == nil {
		return fmt.Errorf("orchestrator connection details are required")
	}
	if details.Host.IsUnknown() || details.Port.IsUnknown() || details.Schema.IsUnknown() {
		return fmt.Errorf("orchestrator endpoint must be known before deployment")
	}
	endpoint, err := url.Parse(details.GetHost())
	if err != nil || endpoint.Hostname() == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return fmt.Errorf("orchestrator host must be an HTTP(S) endpoint without userinfo, query or fragment")
	}
	if port := endpoint.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return fmt.Errorf("orchestrator port must be between 1 and 65535")
		}
	}
	return validateOrchestratorAuth(details.UseAuthentication)
}
