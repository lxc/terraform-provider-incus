# incus_network_integration

Provides information about an Incus network integration.
See Incus network integration [configuration reference](https://linuxcontainers.org/incus/docs/main/howto/network_integrations/) for more details.

## Example Usage

```hcl
data "incus_network_integration" "this" {
  name = "default"
}

output "network_integration_name" {
  value = data.incus_network_integration.this.name
}
```

## Argument Reference

* `name` - **Required** - Name of the network integration.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

* `description` - Description of the network integration.

* `config` - Map of key/value pairs of config settings.

* `type` - Integration type.
