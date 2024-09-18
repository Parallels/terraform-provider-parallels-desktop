package telemetry

type TelemetryEvent string

const (
	EventDeploy      TelemetryEvent = "PD-TERRAFORM-PROVIDER::DEPLOY"
	EventVagrant     TelemetryEvent = "PD-TERRAFORM-PROVIDER::VAGRANT"
	EventRemoteImage TelemetryEvent = "PD-TERRAFORM-PROVIDER::REMOTE_IMAGE"
	EventCloneVm     TelemetryEvent = "PD-TERRAFORM-PROVIDER::CLONE_VM"
)

type TelemetryEventMode string

const (
	ModeCreate  TelemetryEventMode = "CREATE"
	ModeUpdate  TelemetryEventMode = "UPDATE"
	ModeDestroy TelemetryEventMode = "DESTROY"
	ModeRead    TelemetryEventMode = "READ"
)
