package common

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lxc/incus/v7/shared/api"
)

// InterfaceModel represents Incus instance network interface.
type InterfaceModel struct {
	RealName    types.String `tfsdk:"name"`
	State       types.String `tfsdk:"state"`
	Type        types.String `tfsdk:"type"`
	IPAddresses types.List   `tfsdk:"ip_addresses"`
}

// IPModel is a wrapper of the interface IP address.
type IPModel struct {
	Address types.String `tfsdk:"address"`
	Family  types.String `tfsdk:"family"`
	Scope   types.String `tfsdk:"scope"`
}

// ToInterfaceMapType converts provided intance networks into types.Map. The function
// also accepts instance config which is used to determine configuration interface name.
func ToInterfaceMapType(ctx context.Context, instanceNetworks map[string]api.InstanceStateNetwork, instanceConfig map[string]string) (types.Map, diag.Diagnostics) {
	ipType := map[string]attr.Type{
		"address": types.StringType,
		"family":  types.StringType,
		"scope":   types.StringType,
	}

	interfaceType := map[string]attr.Type{
		"name":         types.StringType,
		"state":        types.StringType,
		"type":         types.StringType,
		"ip_addresses": types.ListType{ElemType: types.ObjectType{AttrTypes: ipType}},
	}

	nullMap := types.MapNull(types.ObjectType{AttrTypes: interfaceType})
	if len(instanceNetworks) == 0 {
		return nullMap, nil
	}

	interfaces := make(map[string]InterfaceModel, len(instanceNetworks))
	for name, net := range instanceNetworks {
		// Find volatile entry that contains mac address of the network
		// interface. If addresses match, extract the config name of the
		// interface from the config key (volatile.<if_name>.hwaddr).
		cfgInfName := ""
		for k, v := range instanceConfig {
			if v == net.Hwaddr {
				cfgInfName = strings.SplitN(k, ".", 3)[1]
				break
			}
		}

		if cfgInfName == "" {
			// We did not find a matching config interface, therefore
			// do not export it.
			continue
		}

		inf := InterfaceModel{
			RealName: types.StringValue(name),
			State:    types.StringValue(net.State),
			Type:     types.StringValue(net.Type),
		}

		netAddresses := make([]IPModel, 0, len(net.Addresses))
		for _, addr := range net.Addresses {
			addrType := IPModel{
				Address: types.StringValue(addr.Address),
				Family:  types.StringValue(addr.Family),
				Scope:   types.StringValue(addr.Scope),
			}

			netAddresses = append(netAddresses, addrType)
		}

		addresses, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: ipType}, netAddresses)
		if diags.HasError() {
			return nullMap, diags
		}

		inf.IPAddresses = addresses
		interfaces[cfgInfName] = inf
	}

	return types.MapValueFrom(ctx, types.ObjectType{AttrTypes: interfaceType}, interfaces)
}
