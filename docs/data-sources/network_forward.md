# incus_network_forward

Provides information about an Incus network forward.
See Incus network forward [configuration reference](https://linuxcontainers.org/incus/docs/main/howto/network_forwards/) for more details.

## Example Usage

```hcl
data "incus_network_forward" "this" {
  listen_address = "127.0.0.1"
  network = "parent"
}

output "network_forward_listen_address" {
  value = data.incus_network_forward.this.listen_address
}
```

## Argument Reference

* `listen_address` - **Required** - Listen Address of the network forward.

* `network` - **Required** - Name of the parent network.

* `project` - *Optional* - Name of the project where the network forward is be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.

## Attribute Reference

* `description` - Description of the network forward.

* `config` - Map of key/value pairs of config settings.

* `location` - Location of the network forward.

* `ports` - List of ports to forward.

The network forward ports supports:

* `description` - Description of the forward port.

* `protocol` - Protocol for port forward (either tcp or udp).

* `listen_port` - ListenPort(s) to forward (comma delimited ranges).

* `target_port` - Target port(s) to forward ListenPorts to (allows for many-to-one).

* `target_address` - Target address to forward ListenPorts to.

* `snat` - SNAT controls whether to apply a matching SNAT rule to new outgoing traffic from the target.
