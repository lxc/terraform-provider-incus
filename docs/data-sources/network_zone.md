# incus_network_zone

Provides information about an Incus network zone.
See Incus network zone [configuration reference](https://linuxcontainers.org/incus/docs/main/howto/network_zones/) for more details.

## Example Usage

```hcl
data "incus_network_zone" "this" {
  name = "default"
}

output "network_zone_name" {
  value = data.incus_network_zone.this.name
}
```

## Argument Reference

* `name` - **Required** - Name of the network zone.

* `project` - *Optional* - Name of the project where the network zone is be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.

## Attribute Reference

* `description` - Description of the network zone.

* `config` - Map of key/value pairs of config settings.
