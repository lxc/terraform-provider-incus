package common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DeviceModel struct {
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	Properties types.Map    `tfsdk:"properties"`
}

// ToDeviceMap converts devices from types.Set into map[string]map[string]string.
func ToDeviceMap(ctx context.Context, deviceSet types.Set) (map[string]map[string]string, diag.Diagnostics) {
	if deviceSet.IsNull() || deviceSet.IsUnknown() {
		return make(map[string]map[string]string), nil
	}

	deviceList := make([]DeviceModel, 0, len(deviceSet.Elements()))
	diags := deviceSet.ElementsAs(ctx, &deviceList, false)
	if diags.HasError() {
		return nil, diags
	}

	devices := make(map[string]map[string]string, len(deviceList))
	for _, d := range deviceList {
		deviceName := d.Name.ValueString()
		deviceType := d.Type.ValueString()

		device, diags := ToConfigMap(ctx, d.Properties)
		if diags.HasError() {
			return nil, diags
		}

		device["type"] = deviceType
		devices[deviceName] = device
	}

	return devices, nil
}

// ToDeviceSetType converts devices from map[string]map[string]string
// into types.Set.
func ToDeviceSetType(ctx context.Context, devices map[string]map[string]string) (types.Set, diag.Diagnostics) {
	return toDeviceSetType(ctx, devices, types.SetNull(types.ObjectType{AttrTypes: deviceType()}), false)
}

// ToDeviceSetTypePreservingNulls converts devices from map[string]map[string]string
// into types.Set, preserving null property values from the model when the API
// omits those properties.
func ToDeviceSetTypePreservingNulls(ctx context.Context, devices map[string]map[string]string, modelDevices types.Set) (types.Set, diag.Diagnostics) {
	return toDeviceSetType(ctx, devices, modelDevices, true)
}

func deviceType() map[string]attr.Type {
	return map[string]attr.Type{
		"name":       types.StringType,
		"type":       types.StringType,
		"properties": types.MapType{ElemType: types.StringType},
	}
}

func toDeviceSetType(ctx context.Context, devices map[string]map[string]string, modelDevices types.Set, preserveModelNulls bool) (types.Set, diag.Diagnostics) {
	deviceType := deviceType()
	nilSet := types.SetNull(types.ObjectType{AttrTypes: deviceType})

	if len(devices) == 0 {
		return nilSet, nil
	}

	modelDeviceProperties := map[string]map[string]*string{}
	if preserveModelNulls {
		var diags diag.Diagnostics
		modelDeviceProperties, diags = deviceModelPropertiesByName(ctx, modelDevices)
		if diags.HasError() {
			return nilSet, diags
		}
	}

	deviceList := make([]DeviceModel, 0, len(devices))
	for key := range devices {
		properties := devices[key]

		deviceName := types.StringValue(key)
		deviceType := types.StringValue(properties["type"])

		deviceProperties := make(map[string]*string, len(properties))
		for k, v := range properties {
			// Remove type from properties, as we manage it
			// outside properties.
			if k == "type" {
				continue
			}

			v := v
			deviceProperties[k] = &v
		}

		if preserveModelNulls {
			for k, v := range modelDeviceProperties[key] {
				if v != nil {
					continue
				}

				if _, ok := deviceProperties[k]; !ok {
					deviceProperties[k] = nil
				}
			}
		}

		deviceModelProperties, diags := types.MapValueFrom(ctx, types.StringType, deviceProperties)
		if diags.HasError() {
			return nilSet, diags
		}

		dev := DeviceModel{
			Name:       deviceName,
			Type:       deviceType,
			Properties: deviceModelProperties,
		}

		deviceList = append(deviceList, dev)
	}

	return types.SetValueFrom(ctx, types.ObjectType{AttrTypes: deviceType}, deviceList)
}

func deviceModelPropertiesByName(ctx context.Context, modelDevices types.Set) (map[string]map[string]*string, diag.Diagnostics) {
	modelDeviceProperties := map[string]map[string]*string{}
	if modelDevices.IsNull() || modelDevices.IsUnknown() {
		return modelDeviceProperties, nil
	}

	deviceList := make([]DeviceModel, 0, len(modelDevices.Elements()))
	diags := modelDevices.ElementsAs(ctx, &deviceList, false)
	if diags.HasError() {
		return nil, diags
	}

	for _, d := range deviceList {
		if d.Name.IsNull() || d.Name.IsUnknown() || d.Properties.IsNull() || d.Properties.IsUnknown() {
			continue
		}

		properties := make(map[string]*string, len(d.Properties.Elements()))
		diags := d.Properties.ElementsAs(ctx, &properties, false)
		if diags.HasError() {
			return nil, diags
		}

		modelDeviceProperties[d.Name.ValueString()] = properties
	}

	return modelDeviceProperties, nil
}
