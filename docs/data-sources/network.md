# incus_network

Provides information about an Incus network.
See Incus network [configuration reference](https://linuxcontainers.org/incus/docs/main/explanation/networks/) for more details.

## Example Usage

```hcl
data "incus_network" "this" {
  name = "default"
}

output "network_name" {
  value = data.incus_network.this.name
}
```

## Argument Reference

* `name` - **Required** - Name of the network.

* `project` - *Optional* - Name of the project where the network is be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.

## Attribute Reference

* `description` - Description of the network.

* `config` - Map of key/value pairs of config settings.
  [network config settings](https://linuxcontainers.org/incus/docs/main/howto/network_create/#network-types)

* `status` - Status of the network.

* `locations` - Locations of the network.

* `type` - Network type.
  [network type documentation](https://linuxcontainers.org/incus/docs/main/howto/network_create/#network-types)

* `managed` - Whether the network is managed by Incus.
