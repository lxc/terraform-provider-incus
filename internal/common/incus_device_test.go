package common

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToDeviceMap_skipsNullProperties(t *testing.T) {
	ctx := context.Background()
	devices := mustDeviceSet(ctx, []DeviceModel{
		{
			Name: types.StringValue("eth0"),
			Type: types.StringValue("nic"),
			Properties: types.MapValueMust(types.StringType, map[string]attr.Value{
				"nictype": types.StringValue("bridged"),
				"parent":  types.StringValue("br0"),
				"hwaddr":  types.StringNull(),
			}),
		},
	})

	actual, diags := ToDeviceMap(ctx, devices)
	require.False(t, diags.HasError(), diags.Errors())

	assert.Equal(t, map[string]map[string]string{
		"eth0": {
			"type":    "nic",
			"nictype": "bridged",
			"parent":  "br0",
		},
	}, actual)
}

func TestToDeviceSetTypePreservingNulls_preservesNullProperties(t *testing.T) {
	ctx := context.Background()
	modelDevices := mustDeviceSet(ctx, []DeviceModel{
		{
			Name: types.StringValue("eth0"),
			Type: types.StringValue("nic"),
			Properties: types.MapValueMust(types.StringType, map[string]attr.Value{
				"nictype": types.StringValue("bridged"),
				"parent":  types.StringValue("br0"),
				"hwaddr":  types.StringNull(),
			}),
		},
	})

	actual, diags := ToDeviceSetTypePreservingNulls(ctx, map[string]map[string]string{
		"eth0": {
			"type":    "nic",
			"nictype": "bridged",
			"parent":  "br0",
		},
	}, modelDevices)
	require.False(t, diags.HasError(), diags.Errors())

	device := singleDevice(t, ctx, actual)
	props := map[string]*string{}
	diags = device.Properties.ElementsAs(ctx, &props, false)
	require.False(t, diags.HasError(), diags.Errors())

	assert.Equal(t, "eth0", device.Name.ValueString())
	assert.Equal(t, "nic", device.Type.ValueString())
	assert.Equal(t, "bridged", *props["nictype"])
	assert.Equal(t, "br0", *props["parent"])
	assert.Contains(t, props, "hwaddr")
	assert.Nil(t, props["hwaddr"])
	assert.NotContains(t, props, "type")
}

func TestToDeviceSetType_doesNotPreserveNullProperties(t *testing.T) {
	ctx := context.Background()

	actual, diags := ToDeviceSetType(ctx, map[string]map[string]string{
		"eth0": {
			"type":    "nic",
			"nictype": "bridged",
			"parent":  "br0",
		},
	})
	require.False(t, diags.HasError(), diags.Errors())

	device := singleDevice(t, ctx, actual)
	props := map[string]*string{}
	diags = device.Properties.ElementsAs(ctx, &props, false)
	require.False(t, diags.HasError(), diags.Errors())

	assert.NotContains(t, props, "hwaddr")
	assert.NotContains(t, props, "type")
}

func mustDeviceSet(ctx context.Context, devices []DeviceModel) types.Set {
	deviceSet, diags := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: deviceType()}, devices)
	if diags.HasError() {
		panic(diags.Errors())
	}

	return deviceSet
}

func singleDevice(t *testing.T, ctx context.Context, devices types.Set) DeviceModel {
	t.Helper()

	deviceList := make([]DeviceModel, 0, 1)
	diags := devices.ElementsAs(ctx, &deviceList, false)

	require.False(t, diags.HasError(), diags.Errors())
	require.Len(t, deviceList, 1)

	return deviceList[0]
}
