# incus_network_integration

Manage integrations between the local Incus deployment and remote networks hosted on Incus or other platforms. Currently available only for [OVN networks](https://linuxcontainers.org/incus/docs/main/reference/network_ovn/#network-ovn).

## Basic Example

```hcl
resource "incus_network_integration" "this" {
    name = "ovn-region"
    type = "ovn"

    config = {
        "ovn.northbound_connection" = "tcp:[192.0.2.12]:6645,tcp:[192.0.3.13]:6645,tcp:[192.0.3.14]:6645"
        "ovn.southbound_connection" = "tcp:[192.0.2.12]:6646,tcp:[192.0.3.13]:6646,tcp:[192.0.3.14]:6646"
    }
}
```

## Peer Example

```hcl
resource "incus_network" "default" {
  name = "default"
  type = "ovn"

  config = {
    "ipv4.address" = "192.168.2.0/24"
    "ipv4.nat"     = "true"
  }
}

resource "incus_network_integration" "this" {
    name = "ovn-region"
    type = "ovn"

    config = {
        "ovn.northbound_connection" = "tcp:[192.0.2.12]:6645,tcp:[192.0.3.13]:6645,tcp:[192.0.3.14]:6645"
        "ovn.southbound_connection" = "tcp:[192.0.2.12]:6646,tcp:[192.0.3.13]:6646,tcp:[192.0.3.14]:6646"
    }
}

resource "incus_network_peer" "this" {
    name               = "ovn-peer"
    network            = incus_network.default.name
    target_integration = incus_network_integration.this.name
    type               = "remote"
}
```

## Argument Reference

* `name` - **Required** - Name of the network integration.

* `type` **Required** - The type of the network integration. Currently, only supports `ovn` type.

* `description` *Optional* - Description of the network integration.

* `project` - *Optional* - Name of the project where the network will be created.

* `remote` - *Optional* - The remote in which the resource will be created. If
 not provided, the provider's default remote will be used.

* `config` - *Optional* - Map of key/value pairs of [network integration config settings](https://linuxcontainers.org/incus/docs/main/howto/network_integrations/)
