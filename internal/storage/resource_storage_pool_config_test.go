package storage

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStoragePoolPreserveUserConfig_Source(t *testing.T) {
	userSource := "/dev/disk/by-id/test-disk"
	apiSource := "pool1"

	model := StoragePoolModel{
		Config: types.MapValueMust(types.StringType, map[string]attr.Value{
			"source": types.StringValue(userSource),
		}),
	}

	stateConfig := map[string]*string{
		"source": &apiSource,
	}

	stateConfig = model.PreserveUserConfig("zfs", stateConfig)

	if stateConfig["source"] == nil {
		t.Fatal("expected source to be preserved")
	}

	if got := *stateConfig["source"]; got != userSource {
		t.Fatalf("expected source %q, got %q", userSource, got)
	}
}

func TestStoragePoolPreserveUserConfig_UnsupportedDriver(t *testing.T) {
	userSource := "/dev/disk/by-id/test-disk"
	apiSource := "pool1"

	model := StoragePoolModel{
		Config: types.MapValueMust(types.StringType, map[string]attr.Value{
			"source": types.StringValue(userSource),
		}),
	}

	stateConfig := map[string]*string{
		"source": &apiSource,
	}

	stateConfig = model.PreserveUserConfig("dir", stateConfig)

	if stateConfig["source"] == nil {
		t.Fatal("expected source to remain set")
	}

	if got := *stateConfig["source"]; got != apiSource {
		t.Fatalf("expected source %q, got %q", apiSource, got)
	}
}
