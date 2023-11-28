package sshconnection

import "github.com/hashicorp/terraform-plugin-framework/types"

type SshConnection struct {
	Host       types.String `tfsdk:"host"`
	HostPort   types.String `tfsdk:"host_port"`
	User       types.String `tfsdk:"user"`
	Password   types.String `tfsdk:"password"`
	PrivateKey types.String `tfsdk:"private_key"`
}
