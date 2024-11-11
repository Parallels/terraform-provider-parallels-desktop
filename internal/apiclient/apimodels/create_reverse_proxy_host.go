package apimodels

type ReverseProxyHostCreateRequest struct {
	Host       string                       `json:"host"`
	Port       string                       `json:"port"`
	Tls        *ReverseProxyHostTls         `json:"tls,omitempty"`
	Cors       *ReverseProxyHostCors        `json:"cors,omitempty"`
	HttpRoutes []*ReverseProxyHostHttpRoute `json:"http_routes,omitempty"`
	TcpRoute   *ReverseProxyHostTcpRoute    `json:"tcp_route,omitempty"`
}

type ReverseProxyHostUpdateRequest struct {
	Host string                `json:"host"`
	Port string                `json:"port"`
	Tls  *ReverseProxyHostTls  `json:"tls,omitempty"`
	Cors *ReverseProxyHostCors `json:"cors,omitempty"`
}
