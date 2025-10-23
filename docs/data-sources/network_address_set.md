# incus_network_address_set

Provides information about an Incus network address set.
See Incus network address set [configuration reference](https://linuxcontainers.org/incus/docs/main/howto/network_address_sets/) for more details.

## Example Usage

```hcl
data "incus_network_address_set" "this" {
  name = "default"
}

output "network_address_set_name" {
  value = data.incus_network_address_set.this.name
}
```

## Argument Reference

* `name` - **Required** - Name of the network address set.

* `project` - *Optional* - Name of the project where the network address set is be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

* `description` - Description of the network address set.

* `config` - Map of key/value pairs of config settings.

* `addresses` - List of network addresses.
