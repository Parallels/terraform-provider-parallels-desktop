package deploy

import (
	"bytes"
	"text/template"
)

var apiServiceTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>UserName</key>
  <string>root</string>
  <key>Label</key>
  <string>com.parallels.api-service</string>
  <key>ProgramArguments</key>
  <array>
    <string>{{ .Path }}/pd-api-service</string>
    <string>--port={{ .Port }}</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>ROOT_PASSWORD</key>
    <string>{{ .RootPassword }}</string>
    <key>SECURITY_PRIVATE_KEY</key>
    <string>{{ .EncryptionRsaKey }}</string>
    <key>HMAC_SECRET</key>
    <string>{{ .HmacSecret }}</string>
    <key>LOG_LEVEL</key>
    <string>{{ .LogLevel }}</string>
    <key>TLS_ENABLED</key>
    <string>{{ .EnableTLS }}</string>
    <key>TLS_PORT</key>
    <string>{{ .HostTLSPort }}</string>
    <key>TLS_CERTIFICATE</key>
    <string>{{ .TlsCertificate }}</string>
    <key>TLS_PRIVATE_KEY</key>
    <string>{{ .TlsPrivateKey }}</string>
  </dict>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardErrorPath</key>
  <string>/tmp/api-service.job.err</string>
  <key>StandardOutPath</key>
  <string>/tmp/api-service.job.out</string> 
</dict>
</plist>"`

type TemplateData struct {
	Path             string
	Port             string
	RootPassword     string
	EncryptionRsaKey string
	HmacSecret       string
	LogLevel         string
	EnableTLS        string
	HostTLSPort      string
	TlsCertificate   string
	TlsPrivateKey    string
}

func getApiServicePlist(path string, config ParallelsDesktopApiConfig) (string, error) {
	// Define the text template
	tmpl, err := template.New("parallels-api").Parse(apiServiceTemplate)
	if err != nil {
		return "", err
	}

	// Execute the template with a value
	var tpl bytes.Buffer
	templateData := TemplateData{
		Path:             path,
		Port:             config.Port.ValueString(),
		RootPassword:     config.RootPassword.ValueString(),
		EncryptionRsaKey: config.EncryptionRsaKey.ValueString(),
		HmacSecret:       config.HmacSecret.ValueString(),
		LogLevel:         config.LogLevel.ValueString(),
		HostTLSPort:      config.TLSPort.ValueString(),
		TlsCertificate:   config.TLSCertificate.ValueString(),
		TlsPrivateKey:    config.TLSPrivateKey.ValueString(),
	}
	if config.EnableTLS.ValueBool() {
		templateData.EnableTLS = "true"
	}

	err = tmpl.Execute(&tpl, templateData)
	if err != nil {
		return "", err
	}

	return tpl.String(), nil
}
