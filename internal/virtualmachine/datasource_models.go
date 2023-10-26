package virtualmachine

import "github.com/hashicorp/terraform-plugin-framework/types"

// virtualMachinesDataSourceModel represents the data source schema for the virtual_machines data source.
type virtualMachinesDataSourceModel struct {
	Host     types.String               `tfsdk:"host"`
	Filter   virtualMachinesFilterModel `tfsdk:"filter"`
	Machines []virtualMachineModel      `tfsdk:"machines"`
}

// virtualMachinesFilterModel represents a filter for virtual machines.
type virtualMachinesFilterModel struct {
	State types.String `tfsdk:"state"`
	Id    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
}

// virtualMachineModel represents a virtual machine model with its properties.
type virtualMachineModel struct {
	HostIP      types.String `tfsdk:"host_ip"`     // The IP address of the host machine.
	ID          types.String `tfsdk:"id"`          // The unique identifier of the virtual machine.
	Name        types.String `tfsdk:"name"`        // The name of the virtual machine.
	Description types.String `tfsdk:"description"` // The description of the virtual machine.
	OSType      types.String `tfsdk:"os_type"`     // The type of the operating system installed on the virtual machine.
	State       types.String `tfsdk:"state"`       // The state of the virtual machine.
	Home        types.String `tfsdk:"home"`        // The path to the virtual machine home directory.
}
