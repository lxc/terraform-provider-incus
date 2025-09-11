# incus_storage_bucket

Provides information about an Incus storage bucket.
See Incus storage bucket [configuration reference](https://linuxcontainers.org/incus/docs/main/howto/storage_buckets/) for more details.

## Example Usage

```hcl
data "incus_storage_bucket" "this" {
  name = "default"
  storage_pool = "parent"
}

output "storage_bucket_name" {
  value = data.incus_storage_bucket.this.name
}
```

## Argument Reference

* `name` - **Required** - Name of the storage bucket.

* `storage_pool` - **Required** - Name of the parent storage pool.

* `project` - *Optional* - Name of the project where the storage bucket is be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.

## Attribute Reference

* `description` - Description of the storage bucket.

* `config` - Map of key/value pairs of config settings.
  [storage bucket config settings](https://linuxcontainers.org/incus/docs/main/reference/storage_drivers/)

* `location` - Location of the storage bucket.

* `s3_url` - Storage Bucket S3 URL.
