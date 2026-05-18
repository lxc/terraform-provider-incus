package storage_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/lxc/terraform-provider-incus/internal/storage"
)

func TestStoragePoolPreserveUserConfig_Source(t *testing.T) {
	userSource := "/dev/disk/by-id/test-disk"
	apiSource := "pool1"

	model := storage.StoragePoolModel{
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

func TestStoragePoolPreserveUserConfig_NonZFSDriver(t *testing.T) {
	userSource := "/dev/disk/by-id/test-disk"
	apiSource := "pool1"

	model := storage.StoragePoolModel{
		Config: types.MapValueMust(types.StringType, map[string]attr.Value{
			"source": types.StringValue(userSource),
		}),
	}

	stateConfig := map[string]*string{
		"source": &apiSource,
	}

	stateConfig = model.PreserveUserConfig("lvm", stateConfig)

	if stateConfig["source"] == nil {
		t.Fatal("expected source to remain set")
	}

	if got := *stateConfig["source"]; got != apiSource {
		t.Fatalf("expected source %q, got %q", apiSource, got)
	}
}

func TestStoragePoolConfigValueChanged(t *testing.T) {
	ctx := context.Background()

	stateConfig := types.MapValueMust(types.StringType, map[string]attr.Value{
		"source": types.StringValue("/dev/disk/by-id/disk-a"),
	})

	planConfig := types.MapValueMust(types.StringType, map[string]attr.Value{
		"source": types.StringValue("/dev/disk/by-id/disk-b"),
	})

	if !storage.ConfigValueChanged(ctx, stateConfig, planConfig, "source") {
		t.Fatal("expected source change to be detected")
	}
}

func TestStoragePoolConfigValueChanged_Unchanged(t *testing.T) {
	ctx := context.Background()

	stateConfig := types.MapValueMust(types.StringType, map[string]attr.Value{
		"source": types.StringValue("/dev/disk/by-id/disk-a"),
	})

	planConfig := types.MapValueMust(types.StringType, map[string]attr.Value{
		"source": types.StringValue("/dev/disk/by-id/disk-a"),
	})

	if storage.ConfigValueChanged(ctx, stateConfig, planConfig, "source") {
		t.Fatal("expected unchanged source not to be detected as changed")
	}
}
