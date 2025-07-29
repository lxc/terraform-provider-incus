package common

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestStripConfig_PreservesEmptyStrings(t *testing.T) {
	// Test that StripConfig preserves empty string values from user config
	ctx := context.Background()

	// Create user config with empty string values
	userConfigMap := map[string]*string{
		"cloud-init.network-config": strPtr(""),
		"cloud-init.vendor-data":    strPtr(""),
		"user.user-data":           strPtr(""),
		"user.test":                strPtr("value"),
		"user.empty":               strPtr(""),
		"user.null":                nil,
	}

	// Convert to types.Map
	modelConfig, _ := types.MapValueFrom(ctx, types.StringType, userConfigMap)

	// Simulate resource config from API (Incus returns empty strings)
	resConfig := map[string]string{
		"cloud-init.network-config": "",
		"cloud-init.vendor-data":    "",
		"user.user-data":           "",
		"user.test":                "value",
		"user.empty":               "",
		"volatile.uuid":            "some-uuid", // computed key
		"image.description":        "Ubuntu",    // computed key
	}

	// Call StripConfig
	result := StripConfig(resConfig, modelConfig, []string{"volatile.", "image."})

	// Verify empty strings are preserved
	assert.NotNil(t, result["cloud-init.network-config"])
	assert.Equal(t, "", *result["cloud-init.network-config"])

	assert.NotNil(t, result["cloud-init.vendor-data"])
	assert.Equal(t, "", *result["cloud-init.vendor-data"])

	assert.NotNil(t, result["user.user-data"])
	assert.Equal(t, "", *result["user.user-data"])

	assert.NotNil(t, result["user.empty"])
	assert.Equal(t, "", *result["user.empty"])

	// Non-empty value preserved
	assert.NotNil(t, result["user.test"])
	assert.Equal(t, "value", *result["user.test"])

	// Null value preserved
	assert.Nil(t, result["user.null"])

	// Computed keys not included (unless in user config)
	_, hasVolatile := result["volatile.uuid"]
	assert.False(t, hasVolatile)

	_, hasImage := result["image.description"]
	assert.False(t, hasImage)
}

func strPtr(s string) *string {
	return &s
}