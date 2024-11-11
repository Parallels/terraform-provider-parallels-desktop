package apimodels

type ReverseProxyHost struct {
	ID         string                       `json:"id"`
	Host       string                       `json:"host"`
	Port       string                       `json:"port"`
	Tls        *ReverseProxyHostTls         `json:"tls,omitempty"`
	Cors       *ReverseProxyHostCors        `json:"cors,omitempty"`
	HttpRoutes []*ReverseProxyHostHttpRoute `json:"http_routes,omitempty"`
	TcpRoute   *ReverseProxyHostTcpRoute    `json:"tcp_route,omitempty"`
}

type ReverseProxyHostTls struct {
	Enabled bool   `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Cert    string `json:"cert,omitempty" yaml:"cert,omitempty"`
	Key     string `json:"key,omitempty" yaml:"key,omitempty"`
}

type ReverseProxyHostTcpRoute struct {
	ID         string `json:"id,omitempty" yaml:"id,omitempty"`
	TargetPort string `json:"target_port,omitempty" yaml:"target_port,omitempty"`
	TargetHost string `json:"target_host,omitempty" yaml:"target_host,omitempty"`
	TargetVmId string `json:"target_vm_id,omitempty" yaml:"target_vm_id,omitempty"`
}

type ReverseProxyHostCors struct {
	Enabled        bool     `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	AllowedOrigins []string `json:"allowed_origins,omitempty" yaml:"allowed_origins,omitempty"`
	AllowedMethods []string `json:"allowed_methods,omitempty" yaml:"allowed_methods,omitempty"`
	AllowedHeaders []string `json:"allowed_headers,omitempty" yaml:"allowed_headers,omitempty"`
}

type ReverseProxyHostHttpRoute struct {
	ID              string            `json:"id,omitempty" yaml:"id,omitempty"`
	Path            string            `json:"path,omitempty" yaml:"path,omitempty"`
	TargetVmId      string            `json:"target_vm_id,omitempty" yaml:"target_vm_id,omitempty"`
	TargetHost      string            `json:"target_host,omitempty" yaml:"target_host,omitempty"`
	TargetPort      string            `json:"target_port,omitempty" yaml:"target_port,omitempty"`
	Schema          string            `json:"schema,omitempty" yaml:"scheme,omitempty"`
	Pattern         string            `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	RequestHeaders  map[string]string `json:"request_headers,omitempty" yaml:"request_headers,omitempty"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty" yaml:"response_headers,omitempty"`
}
