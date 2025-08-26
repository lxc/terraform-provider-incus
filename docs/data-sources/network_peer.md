# incus_network_peer

Provides information about an Incus network peer.
See Incus network peer [configuration reference](https://linuxcontainers.org/incus/docs/main/howto/network_peers/https://linuxcontainers.org/incus/docs/main/howto/network_ovn_peers/) for more details.

## Example Usage

```hcl
data "incus_network_peer" "this" {
  name = "default"
  network = "parent"
}

output "network_peer_name" {
  value = data.incus_network_peer.this.name
}
```

## Argument Reference

* `name` - **Required** - Name of the network peer.

* `network` - **Required** - Name of the parent network.

* `project` - *Optional* - Name of the project where the network peer is be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.

## Attribute Reference

* `description` - Description of the network peer.

* `config` - Map of key/value pairs of config settings.

* `status` - Status of the network peer.

* `target_project` - Target project for the network peer.

* `target_network` - Target network for the network peer.

* `target_integration` - Target integration for the network peer.

* `type` - Network peer type.
