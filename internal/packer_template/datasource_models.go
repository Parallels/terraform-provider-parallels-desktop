package packertemplate

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/filter"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// virtualMachinesDataSourceModel represents the data source schema for the virtual_machines data source.
type packerTemplateDataSourceModel struct {
	Authenticator *authenticator.Authentication `tfsdk:"authenticator"`
	Host          types.String                  `tfsdk:"host"`
	Filter        *filter.Filter                `tfsdk:"filter"`
	Templates     []packerTemplateModel         `tfsdk:"templates"`
}

// packerTemplateModel represents a virtual machine model with its properties.
type packerTemplateModel struct {
	ID          types.String            `tfsdk:"id"`
	Name        types.String            `tfsdk:"name"`
	Description types.String            `tfsdk:"description"`
	Specs       map[string]types.String `tfsdk:"specs"`
	Addons      []types.String          `tfsdk:"addons"`
	Defaults    map[string]types.String `tfsdk:"defaults"`
	Internal    types.Bool              `tfsdk:"internal"`
	Variables   map[string]types.String `tfsdk:"variables"`
}
