# incus_network_address_set

Network address sets are a list of either IPv4, IPv6 addresses with or without CIDR suffix. They can be used in source or destination fields of [ACLs](https://linuxcontainers.org/incus/docs/main/howto/network_acls/#network-acls-rules-properties).

## Basic Example

```hcl
resource "incus_network_address_set" "this" {
    name        = "Network Address Set"
    description = "Network Address Set description"
    addresses   = ["10.0.0.2", "10.0.0.3"]
}
```

## ACL Example

```hcl
resource "incus_network_address_set" "this" {
    name        = "network_address_set"
    description = "Network Address Set description"
    addresses   = ["10.0.0.2", "10.0.0.3"]
}

resource "incus_network_acl" "this" {
  name = "network_acl"

  ingress = [
    {
      action           = "allow"
      source           = "$${incus_network_address_set.this.name}"
      destination_port = "22"
      protocol         = "tcp"
      description      = "Incoming SSH connections from ${incus_network_address_set.this.name}"
      state            = "logged"
    }
  ]
}
```

## Argument Reference

* `name` - **Required** - Name of the network address set.

* `addresses` - **Required** - IP addresses of the address set.

* `description` *Optional* - Description of the network address set.

* `project` - *Optional* - Name of the project where the network address set will be created.

* `remote` - *Optional* - The remote in which the resource will be created. If
 not provided, the provider's default remote will be used.

* `config` - *Optional* - Map of key/value pairs of [network address set config settings](https://linuxcontainers.org/incus/docs/main/howto/network_address_sets/#address-set-configuration-options)
