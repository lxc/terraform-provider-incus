# incus_network_load_balancer

Provides information about an Incus network load balancer.
See Incus network load balancer [configuration reference](https://linuxcontainers.org/incus/docs/main/howto/network_load_balancers/) for more details.

## Example Usage

```hcl
data "incus_network_load_balancer" "this" {
  listen_address = "127.0.0.1"
  network = "parent"
}

output "network_load_balancer_listen_address" {
  value = data.incus_network_load_balancer.this.listen_address
}
```

## Argument Reference

* `listen_address` - **Required** - Listen Address of the network load balancer.

* `network` - **Required** - Name of the parent network.

* `project` - *Optional* - Name of the project where the network load balancer is be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.

## Attribute Reference

* `description` - Description of the network load balancer.

* `config` - Map of key/value pairs of config settings.

* `location` - Location of the network load balancer.

* `backends` - List of load balancer backends.

* `ports` - List of load balancer ports.

The network load balancer backends supports:

* `name` - Name of the load balancer backend.

* `description` - Description of the load balancer backend.

* `target_port` - TargetPort(s) for the forward ListenPorts to (allows for many-to one).

* `target_address` - TargetAddress to forward ListenPorts to.

The network load balancer ports supports:

* `description` - Description of the load balancer port.

* `protocol` - Protocol for load balancer (either tcp or udp).

* `listen_port` - ListenPort(s) for the load balancer (comma delimited ranges).

* `target_backend` - TargetBackend backend names to load balance ListenPorts to.
