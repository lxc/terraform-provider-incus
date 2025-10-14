# incus_network_peer

Incus allows creating peer routing relationships between two OVN networks. Using this method, traffic between the two
networks can go directly from one OVN network to the other and thus stays within the OVN subsystem, rather than transiting
through the uplink network.

-> The peer resource is exclusively compatible with OVN (Open Virtual Network).

For more information, please refer to [How to create peer routing relationships](https://linuxcontainers.org/incus/docs/main/howto/network_ovn_peers/)
in the official Incus documentation.

## Example Usage

```hcl
resource "incus_network" "lan0" {
  name = "lan0"
  type = "ovn"

  config = {
    # ...
  }
}

resource "incus_network" "lan1" {
  name = "lan1"
  type = "ovn"

  config = {
    # ...
  }
}

resource "incus_network_peer" "lan0_lan1"{
  name           = "lab0-lan1"
  description    = "A meaningful description"
  network        = incus_network.lan0.name
  project        = "default"
  target_network = incus_network.lan1.name
  target_project = "default"
}

resource "incus_network_peer" "lan1_lan0"{
  name           = "lab1-lan0"
  description    = "A meaningful description"
  network        = incus_network.lan1.name
  project        = "default"
  target_network = incus_network.lan0.name
  target_project = "default"
}
```

## Argument Reference

* `name` - **required** - Name of the network peering on the local network

* `network` - **Required** - Name of the local network.

* `target_network` - **required** - Which network to create a peering with (required at create time for local peers)

* `description`  - *Optional* - Description of the network peering

* `config` - *Optional* - Configuration options as key/value pairs (only user.* custom keys supported)

* `type` - *Optional* - Type of network peering

* `target_intergration` - *Optional* - Name of the integration (required at create time for remote peers)

* `target_project` - *Optional* - Which project the target network exists in (required at create time for local peers)

* `project` - *Optional* - Name of the project where the network is located.

* `remote` - *Optional* - The remote in which the resource will be created. If
    not provided, the provider's default remote will be used.

## Attribute Reference

No attributes are exported.
