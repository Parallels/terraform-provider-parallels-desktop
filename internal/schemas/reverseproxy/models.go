package reverseproxy

import (
	"terraform-provider-parallels-desktop/internal/common"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type ReverseProxyHost struct {
	Index     int                       `tfsdk:"-"`
	ID        basetypes.StringValue     `tfsdk:"id"`
	Host      basetypes.StringValue     `tfsdk:"host"`
	Port      string                    `tfsdk:"port"`
	Cors      *ReverseProxyCors         `tfsdk:"cors"`
	Tls       *ReverseProxyTls          `tfsdk:"tls"`
	HttpRoute []*ReverseProxyHttpRoute  `tfsdk:"http_routes"`
	TcpRoute  *ReverseProxyHostTcpRoute `tfsdk:"tcp_route"`
}

func (o *ReverseProxyHost) Copy() ReverseProxyHost {
	result := ReverseProxyHost{}
	if o == nil {
		return result
	}

	result.ID = o.ID
	result.Host = o.Host
	result.Port = o.Port
	if o.Cors != nil {
		cors := o.Cors.Copy()
		result.Cors = &cors
	}
	if o.Tls != nil {
		tls := o.Tls.Copy()
		result.Tls = &tls
	}
	if len(o.HttpRoute) > 0 {
		result.HttpRoute = make([]*ReverseProxyHttpRoute, len(o.HttpRoute))
		for i, v := range o.HttpRoute {
			route := v.Copy()
			result.HttpRoute[i] = &route
		}
	}
	if o.TcpRoute != nil {
		tcpRoute := o.TcpRoute.Copy()
		result.TcpRoute = &tcpRoute
	}

	return result
}

func (o *ReverseProxyHost) GetHost() string {
	if o == nil {
		return ""
	}

	host := common.GetString(o.Host)
	if host == "" {
		host = allAddress
	}

	if o.Port != "" {
		return host + ":" + o.Port
	}

	return host
}

func (o *ReverseProxyHost) Diff(other *ReverseProxyHost) bool {
	if o == nil && other == nil {
		return false
	}
	if o == nil && other != nil {
		return true
	}
	if o != nil && other == nil {
		return true
	}
	if common.GetString(o.ID) != common.GetString(other.ID) {
		return true
	}
	if common.GetString(o.Host) != common.GetString(other.Host) {
		return true
	}
	if o.Port != other.Port {
		return true
	}
	if o.Cors.Diff(other.Cors) {
		return true
	}
	if o.Tls.Diff(other.Tls) {
		return true
	}
	if len(o.HttpRoute) != len(other.HttpRoute) {
		return true
	}
	for i, v := range o.HttpRoute {
		if v.Diff(other.HttpRoute[i]) {
			return true
		}
	}

	return o.TcpRoute.Diff(other.TcpRoute)
}

type ReverseProxyHostTcpRoute struct {
	TargetPort basetypes.StringValue `tfsdk:"target_port"`
	TargetHost basetypes.StringValue `tfsdk:"target_host"`
	TargetVmId basetypes.StringValue `tfsdk:"target_vm_id"`
}

func (o *ReverseProxyHostTcpRoute) GetHost() string {
	host := common.GetString(o.TargetHost)
	port := common.GetString(o.TargetPort)
	vmId := common.GetString(o.TargetVmId)

	if host == "" && vmId == "" {
		host = allAddress
	}
	if host == "" && vmId != "" {
		host = vmId
	}

	if port != "" {
		host += ":" + port
	}

	return host
}

func (o *ReverseProxyHostTcpRoute) Copy() ReverseProxyHostTcpRoute {
	result := ReverseProxyHostTcpRoute{}
	if o == nil {
		return result
	}

	result.TargetPort = o.TargetPort
	result.TargetHost = o.TargetHost
	result.TargetVmId = o.TargetVmId

	return result
}

func (o *ReverseProxyHostTcpRoute) Diff(other *ReverseProxyHostTcpRoute) bool {
	if o == nil && other == nil {
		return false
	}
	if o == nil && other != nil {
		return true
	}
	if o != nil && other == nil {
		return true
	}
	if common.GetString(o.TargetPort) != common.GetString(other.TargetPort) {
		return true
	}
	if common.GetString(o.TargetHost) != common.GetString(other.TargetHost) {
		return true
	}
	if common.GetString(o.TargetVmId) != common.GetString(other.TargetVmId) {
		return true
	}
	return false
}

type ReverseProxyCors struct {
	Enabled        bool     `tfsdk:"enabled"`
	AllowedOrigins []string `tfsdk:"allowed_origins"`
	AllowedMethods []string `tfsdk:"allowed_methods"`
	AllowedHeaders []string `tfsdk:"allowed_headers"`
}

func (o *ReverseProxyCors) Copy() ReverseProxyCors {
	if o == nil {
		return ReverseProxyCors{}
	}
	c := *o
	return c
}

func (o *ReverseProxyCors) Diff(other *ReverseProxyCors) bool {
	if o == nil && other == nil {
		return false
	}
	if o == nil && other != nil {
		return true
	}
	if o != nil && other == nil {
		return true
	}
	if o.Enabled != other.Enabled {
		return true
	}
	if len(o.AllowedOrigins) != len(other.AllowedOrigins) {
		return true
	}
	for i, v := range o.AllowedOrigins {
		if other.AllowedOrigins[i] != v {
			return true
		}
	}
	if len(o.AllowedMethods) != len(other.AllowedMethods) {
		return true
	}
	for i, v := range o.AllowedMethods {
		if other.AllowedMethods[i] != v {
			return true
		}
	}
	if len(o.AllowedHeaders) != len(other.AllowedHeaders) {
		return true
	}
	for i, v := range o.AllowedHeaders {
		if other.AllowedHeaders[i] != v {
			return true
		}
	}
	return false
}

type ReverseProxyTls struct {
	Enabled     bool   `tfsdk:"enabled"`
	Certificate string `tfsdk:"certificate"`
	PrivateKey  string `tfsdk:"private_key"`
}

func (o *ReverseProxyTls) Copy() ReverseProxyTls {
	if o == nil {
		return ReverseProxyTls{}
	}
	c := *o
	return c
}

func (o *ReverseProxyTls) Diff(other *ReverseProxyTls) bool {
	if o == nil && other == nil {
		return false
	}
	if o == nil && other != nil {
		return true
	}
	if o != nil && other == nil {
		return true
	}
	if o.Enabled != other.Enabled {
		return true
	}
	if o.Certificate != other.Certificate {
		return true
	}
	if o.PrivateKey != other.PrivateKey {
		return true
	}
	return false
}

type ReverseProxyHttpRoute struct {
	TargetPort      basetypes.StringValue `tfsdk:"target_port"`
	TargetHost      basetypes.StringValue `tfsdk:"target_host"`
	TargetVmId      basetypes.StringValue `tfsdk:"target_vm_id"`
	Path            string                `tfsdk:"path"`
	Pattern         string                `tfsdk:"pattern"`
	Schema          string                `tfsdk:"schema"`
	RequestHeaders  map[string]string     `tfsdk:"request_headers"`
	ResponseHeaders map[string]string     `tfsdk:"response_headers"`
}

func (o *ReverseProxyHttpRoute) GetHost() string {
	host := common.GetString(o.TargetHost)
	port := common.GetString(o.TargetPort)
	vmId := common.GetString(o.TargetVmId)

	if host == "" && vmId == "" {
		host = allAddress
	}
	if host == "" && vmId != "" {
		host = vmId
	}

	if port != "" {
		host += ":" + port
	}

	if o.Schema != "" {
		host = o.Schema + "://" + host
	}
	return host
}

func (o *ReverseProxyHttpRoute) Copy() ReverseProxyHttpRoute {
	result := ReverseProxyHttpRoute{}
	result.Path = o.Path
	result.Pattern = o.Pattern
	result.TargetHost = o.TargetHost
	result.TargetPort = o.TargetPort
	result.TargetVmId = o.TargetVmId
	result.Schema = o.Schema
	result.RequestHeaders = o.RequestHeaders
	result.ResponseHeaders = o.ResponseHeaders

	return result
}

func (o *ReverseProxyHttpRoute) Diff(other *ReverseProxyHttpRoute) bool {
	if o == nil && other == nil {
		return false
	}
	if o == nil && other != nil {
		return true
	}
	if o != nil && other == nil {
		return true
	}
	if o.Path != other.Path {
		return true
	}
	if o.Pattern != other.Pattern {
		return true
	}
	if common.GetString(o.TargetHost) != common.GetString(other.TargetHost) {
		return true
	}
	if common.GetString(o.TargetPort) != common.GetString(other.TargetPort) {
		return true
	}
	if common.GetString(o.TargetVmId) != common.GetString(other.TargetVmId) {
		return true
	}
	if o.Schema != other.Schema {
		return true
	}
	if len(o.RequestHeaders) != len(other.RequestHeaders) {
		return true
	}
	for k, v := range o.RequestHeaders {
		if other.RequestHeaders[k] != v {
			return true
		}
	}
	if len(o.ResponseHeaders) != len(other.ResponseHeaders) {
		return true
	}
	for k, v := range o.ResponseHeaders {
		if other.ResponseHeaders[k] != v {
			return true
		}
	}
	return false
}

func ReverseProxyHostsDiff(a, b []*ReverseProxyHost) bool {
	if len(a) != len(b) {
		return true
	}
	for i, v := range a {
		if v.Diff(b[i]) {
			return true
		}
	}
	return false
}

func CopyReverseProxyHosts(a []*ReverseProxyHost) []ReverseProxyHost {
	if a == nil {
		return nil
	}
	b := make([]ReverseProxyHost, len(a))
	for i, v := range a {
		b[i] = v.Copy()
	}
	return b
}
