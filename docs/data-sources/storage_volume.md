# incus_storage_volume

Provides information about an Incus storage volume.
See Incus storage volume [configuration reference](https://linuxcontainers.org/incus/docs/main/howto/storage_volumes/) for more details.

## Example Usage

```hcl
data "incus_storage_volume" "this" {
  name = "default"
  type = "custom"
  storage_pool = "parent"
}

output "storage_volume_name" {
  value = data.incus_storage_volume.this.name
}
```

## Argument Reference

* `name` - **Required** - Name of the storage volume.

* `type` - **Required** - Storage Volume type.

* `storage_pool` - **Required** - Name of the parent storage pool.

* `project` - *Optional* - Name of the project where the storage volume is be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.

## Attribute Reference

* `description` - Description of the storage volume.

* `config` - Map of key/value pairs of config settings.
  [storage volume config settings](https://linuxcontainers.org/incus/docs/main/reference/storage_drivers/)

* `location` - Location of the storage volume.

* `content_type` - Storage Volume content type.
