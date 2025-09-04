# incus_storage_pool

Provides information about an Incus storage pool.
See Incus storage pool [configuration reference](https://linuxcontainers.org/incus/docs/main/howto/storage_pools/) for more details.

## Example Usage

```hcl
data "incus_storage_pool" "this" {
  name = "default"
}

output "storage_pool_name" {
  value = data.incus_storage_pool.this.name
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

* `config` - Map of key/value pairs of config settings.
  [storage pool config settings](https://linuxcontainers.org/incus/docs/main/reference/storage_drivers/)

* `status` - Status of the storage pool.

* `driver` - Storage Pool driver.
