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
    <string>--port</string>
    <string>{{ .Port }}</string>
  </array>
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
	Path string
	Port string
}

func getApiServicePlist(path, port string) (string, error) {
	// Define the text template
	tmpl, err := template.New("parallels-api").Parse(apiServiceTemplate)
	if err != nil {
		return "", err
	}

	// Execute the template with a value
	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, TemplateData{
		Path: path,
		Port: port,
	})
	if err != nil {
		return "", err
	}

	return tpl.String(), nil
}
