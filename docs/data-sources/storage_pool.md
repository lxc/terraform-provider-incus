# incus_storage_pool

Provides information about an Incus storage pool.

## Example Usage

```hcl
data "incus_storage_pool" "default" {
  name = "default"
}

resource "incus_storage_volume" "small" {
  name         = "small"
  pool         = data.incus_storage_pool.default.name
  content_type = "block"
  config = {
    "size" = "1GiB",
  }
}
```

## Argument Reference

* `name` - **Required** - Name of the storage pool.

* `project` - *Optional* - Name of the project where the storage pool is be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.

## Attribute Reference

* `description` - Description of the storage pool.

* `driver` - Storage Pool driver.

* `config` - *Optional* - Map of key/value pairs of
  [storage pool config settings](https://linuxcontainers.org/incus/docs/main/reference/storage_drivers/).
