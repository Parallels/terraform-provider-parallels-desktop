package reverseproxy

import (
	"context"
	"strings"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/common"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var allAddress = "0.0.0.0"

func Read() diag.Diagnostics {
	diagnostic := diag.Diagnostics{}
	return diagnostic
}

func Create(ctx context.Context, config apiclient.HostConfig, request []ReverseProxyHost) ([]ReverseProxyHost, diag.Diagnostics) {
	diagnostic := diag.Diagnostics{}
	for _, host := range request {
		if h, diag := createHost(ctx, config, host); diag.HasError() {
			diagnostic = append(diagnostic, diag...)
		} else {
			host.ID = h.ID
		}
	}

	return request, diagnostic
}

func Revert(ctx context.Context, config apiclient.HostConfig, currentHosts []ReverseProxyHost, requestHosts []ReverseProxyHost) ([]ReverseProxyHost, diag.Diagnostics) {
	diagnostic := diag.Diagnostics{}

	// we will delete all the hosts that are in the request as we do not know what was changed
	for _, host := range requestHosts {
		if exists, _ := apiclient.GetReverseProxyHost(ctx, config, host.GetHost()); exists != nil {
			if diag := deleteHost(ctx, config, host); diag.HasError() {
				tflog.Error(ctx, "Error deleting host "+host.GetHost())
			}
		}
	}

	// we will create all the hosts that are in the currentHosts as we do not know what was changed
	for i, host := range currentHosts {
		if exists, _ := apiclient.GetReverseProxyHost(ctx, config, host.GetHost()); exists == nil {
			if r, diag := createHost(ctx, config, host); diag.HasError() {
				diagnostic = append(diagnostic, diag...)
			} else {
				requestHosts[i].ID = r.ID
			}
		}
	}

	return requestHosts, diagnostic
}

func Update(ctx context.Context, config apiclient.HostConfig, currentHosts []ReverseProxyHost, requestHosts []ReverseProxyHost) ([]ReverseProxyHost, diag.Diagnostics) {
	diagnostic := diag.Diagnostics{}
	hasChanges := diff(currentHosts, requestHosts)
	if !hasChanges {
		return nil, diagnostic
	}

	// Getting a list of reverse proxy hosts to delete as they exist in the currentHosts and not in the requestHosts
	toDelete := getHostsToDelete(currentHosts, requestHosts)
	toCreate := getHostsToCreate(currentHosts, requestHosts)
	toUpdate := getHostsToUpdate(currentHosts, requestHosts)

	for _, host := range toDelete {
		if diag := deleteHost(ctx, config, host); diag.HasError() {
			diagnostic = append(diagnostic, diag...)
		}
	}

	for _, host := range toCreate {
		h, createDiag := createHost(ctx, config, host)
		if createDiag.HasError() {
			diagnostic = append(diagnostic, createDiag...)
		}
		requestHosts[host.Index].ID = h.ID
	}

	for _, host := range toUpdate {
		currentHost := currentHosts[host.Index]
		h, updateDiag := updateHost(ctx, config, currentHost, host)
		if updateDiag.HasError() {
			diagnostic = append(diagnostic, updateDiag...)
		}

		requestHosts[host.Index].ID = h.ID
	}

	return requestHosts, diagnostic
}

func Delete(ctx context.Context, config apiclient.HostConfig, request []ReverseProxyHost) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}
	for _, host := range request {
		if diag := deleteHost(ctx, config, host); diag.HasError() {
			diagnostic = append(diagnostic, diag...)
		}
	}
	return diagnostic
}

func diff(a, b []ReverseProxyHost) bool {
	if len(a) != len(b) {
		return true
	}
	for i, v := range a {
		if v.Diff(&b[i]) {
			return true
		}
	}

	return false
}

func getHostsToDelete(currentHosts, requestHosts []ReverseProxyHost) []ReverseProxyHost {
	toDelete := make([]ReverseProxyHost, 0)
	if len(currentHosts) > 0 && len(requestHosts) == 0 {
		for i, v := range currentHosts {
			v.Index = i
			toDelete = append(toDelete, v)
		}
		return toDelete
	}

	for i, v := range currentHosts {
		found := false
		vHost := v.GetHost()
		for _, r := range requestHosts {
			rHost := r.GetHost()
			if strings.EqualFold(vHost, rHost) {
				found = true
				break
			}
		}

		if !found {
			toDelete = append(toDelete, currentHosts[i])
		}
	}

	return toDelete
}

func getHostsToUpdate(currentHosts, requestHosts []ReverseProxyHost) []ReverseProxyHost {
	toUpdate := make([]ReverseProxyHost, 0)
	if len(requestHosts) == 0 || len(currentHosts) == 0 {
		return toUpdate
	}

	for i, v := range requestHosts {
		vHost := v.GetHost()
		for _, r := range currentHosts {
			rHost := r.GetHost()
			if strings.EqualFold(vHost, rHost) {
				if currentHosts[i].Diff(&v) {
					requestHosts[i].Index = i
					toUpdate = append(toUpdate, requestHosts[i])
				}
				break
			}
		}
	}

	return toUpdate
}

func getHostsToCreate(currentHosts, requestHosts []ReverseProxyHost) []ReverseProxyHost {
	toCreate := make([]ReverseProxyHost, 0)
	if len(requestHosts) == 0 {
		return toCreate
	}

	if len(currentHosts) == 0 {
		for i, v := range requestHosts {
			v.Index = i
			toCreate = append(toCreate, v)
		}
		return toCreate
	}

	for i, v := range requestHosts {
		found := false
		vHost := v.GetHost()
		for _, r := range currentHosts {
			rHost := r.GetHost()
			if strings.EqualFold(vHost, rHost) {
				found = true
				break
			}
		}

		if !found {
			requestHosts[i].Index = i
			toCreate = append(toCreate, requestHosts[i])
		}
	}

	return toCreate
}

func mapReverseProxyToApiModel(v ReverseProxyHost) apimodels.ReverseProxyHost {
	requestHost := apimodels.ReverseProxyHost{
		Port: v.Port,
	}

	if common.GetString(v.ID) != "" {
		requestHost.ID = common.GetString(v.ID)
	}

	if common.GetString(v.Host) == "" {
		requestHost.Host = allAddress
	} else {
		requestHost.Host = common.GetString(v.Host)
	}

	if v.Tls != nil {
		requestHost.Tls = &apimodels.ReverseProxyHostTls{
			Cert:    v.Tls.Certificate,
			Key:     v.Tls.PrivateKey,
			Enabled: v.Tls.Enabled,
		}
	}
	if v.Cors != nil {
		requestHost.Cors = &apimodels.ReverseProxyHostCors{
			Enabled:        v.Cors.Enabled,
			AllowedOrigins: v.Cors.AllowedOrigins,
			AllowedMethods: v.Cors.AllowedMethods,
			AllowedHeaders: v.Cors.AllowedHeaders,
		}
	}
	if v.TcpRoute != nil {
		requestHost.TcpRoute = &apimodels.ReverseProxyHostTcpRoute{}
		if common.GetString(v.TcpRoute.TargetHost) != "" {
			requestHost.TcpRoute.TargetHost = common.GetString(v.TcpRoute.TargetHost)
		}
		if common.GetString(v.TcpRoute.TargetPort) != "" {
			requestHost.TcpRoute.TargetPort = common.GetString(v.TcpRoute.TargetPort)
		}
		if common.GetString(v.TcpRoute.TargetVmId) != "" {
			requestHost.TcpRoute.TargetVmId = common.GetString(v.TcpRoute.TargetVmId)
		}
		if common.GetString(v.TcpRoute.TargetHost) == "" && common.GetString(v.TcpRoute.TargetVmId) == "" {
			requestHost.TcpRoute.TargetHost = allAddress
		}
	}

	if len(v.HttpRoute) > 0 {
		requestHost.HttpRoutes = make([]*apimodels.ReverseProxyHostHttpRoute, 0)
		for _, route := range v.HttpRoute {
			httpRoute := apimodels.ReverseProxyHostHttpRoute{
				TargetHost:      common.GetString(route.TargetHost),
				TargetPort:      common.GetString(route.TargetPort),
				TargetVmId:      common.GetString(route.TargetVmId),
				Path:            route.Path,
				Schema:          route.Schema,
				Pattern:         route.Pattern,
				RequestHeaders:  route.RequestHeaders,
				ResponseHeaders: route.ResponseHeaders,
			}
			if common.GetString(route.TargetHost) == "" && common.GetString(route.TargetVmId) == "" {
				httpRoute.TargetHost = allAddress
			}

			requestHost.HttpRoutes = append(requestHost.HttpRoutes, &httpRoute)
		}
	}

	return requestHost
}

func deleteHost(ctx context.Context, config apiclient.HostConfig, host ReverseProxyHost) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}

	if common.GetString(host.Host) == "" {
		host.Host = types.StringValue(allAddress)
	}

	if exists, _ := apiclient.GetReverseProxyHost(ctx, config, host.GetHost()); exists != nil {
		if diag := apiclient.DeleteReverseProxyHost(ctx, config, host.GetHost()); diag.HasError() {
			diagnostic = append(diagnostic, diag...)
		}
	}

	return diagnostic
}

func createHost(ctx context.Context, config apiclient.HostConfig, host ReverseProxyHost) (ReverseProxyHost, diag.Diagnostics) {
	diagnostic := diag.Diagnostics{}

	// if the host id is not set, we will check if the host exists and delete it
	if exists, _ := apiclient.GetReverseProxyHost(ctx, config, host.GetHost()); exists != nil {
		if diag := apiclient.DeleteReverseProxyHost(ctx, config, host.GetHost()); diag.HasError() {
			diagnostic = append(diagnostic, diag...)
		}
	}

	r, createDiag := apiclient.CreateReverseProxyHost(ctx, config, mapReverseProxyToApiModel(host))
	if createDiag.HasError() || r == nil {
		diagnostic = append(diagnostic, createDiag...)
		return host, diagnostic
	}

	host.ID = types.StringValue(r.ID)
	return host, diagnostic
}

func updateHost(ctx context.Context, config apiclient.HostConfig, currentHost ReverseProxyHost, requestHost ReverseProxyHost) (ReverseProxyHost, diag.Diagnostics) {
	diagnostic := diag.Diagnostics{}

	if exists, _ := apiclient.GetReverseProxyHost(ctx, config, currentHost.GetHost()); exists != nil {
		if diag := apiclient.DeleteReverseProxyHost(ctx, config, currentHost.GetHost()); diag.HasError() {
			diagnostic = append(diagnostic, diag...)
		}
	}

	if exists, diag := apiclient.GetReverseProxyHost(ctx, config, requestHost.GetHost()); exists != nil {
		if diag.HasError() {
			diagnostic = append(diagnostic, diag...)
		}

		requestHost.ID = types.StringValue(exists.ID)
		return requestHost, diagnostic
	}

	r, createDiag := apiclient.CreateReverseProxyHost(ctx, config, mapReverseProxyToApiModel(requestHost))
	if createDiag.HasError() {
		diagnostic = append(diagnostic, createDiag...)
	}

	requestHost.ID = types.StringValue(r.ID)
	return requestHost, diagnostic
}
