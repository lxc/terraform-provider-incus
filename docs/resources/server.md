# incus_server

Manages the configuration of an Incus server.

An Incus server can be a standalone server or a cluster of servers.
A full reference of server config options is available in the [documentation](https://linuxcontainers.org/incus/docs/main/server_config/).

## Example Usage

```hcl
resource "incus_server" "test" {
  config = {
    "logging.loki01.target.type"     = "loki"
    "logging.loki01.target.address"  = "https://loki01.int.example.net"
    "logging.loki01.target.username" = "foo"
    "logging.loki01.target.password" = "bar"
    "logging.loki01.types"           = "lifecycle,network-acl"
    "logging.loki01.lifecycle.types" = "instance"
  }
}
```

## Argument Reference

* `config` - *Optional* - Map of key/value pairs of
  [server config settings](https://linuxcontainers.org/incus/docs/main/server_config/).

* `remote` - *Optional* - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster where the config
  options should be applied. This is in particular important for config options
  with `local` scope.

## Notes

* The server may have configuration options set prior to the server being managed
  through Terraform. These configuration keys are recorded by the resource and
  preserved as long as they are not changed by the Terraform configuration.
  Upon destruction of the resource, these config options will remain on the
  server with their last value.
* If multiple `incus_server` resources are defined for the same `remote`, e.g.
  in order to configure local and global settings for the same feature, they
  likely need to be defined as dependent (e.g. use `depends_on` meta argument)
  in order to ensure proper application of the configuration.
  Race conditions between multiple instances of `incus_server` resources are
  prevented by the provider.
