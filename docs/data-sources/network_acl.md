# incus_network_acl

Provides information about an Incus network ACL.
See Incus network ACL [configuration reference](https://linuxcontainers.org/incus/docs/main/howto/network_acls/) for more details.

## Example Usage

```hcl
data "incus_network_acl" "this" {
  name = "default"
}

output "network_acl_name" {
  value = data.incus_network_acl.this.name
}
```

## Argument Reference

* `name` - **Required** - Name of the network ACL.

* `project` - *Optional* - Name of the project where the network ACL is be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

* `description` - Description of the network ACL.

* `config` - Map of key/value pairs of config settings.

* `egress` - List of egress rules.

* `ingress` - List of ingress rules.

The network ACL egress supports:

* `action` - Action to perform on rule match.

* `source` - Source address.

* `destination` - Destination address.

* `protocol` - Protocol (e.g., tcp, udp).

* `source_port` - Source port.

* `destination_port` - Destination port.

* `icmp_type` - Type of ICMP message.

* `icmp_code` - ICMP message code (for ICMP protocol).

* `description` - Description of the rule.

* `state` - State of the rule.

The network ACL ingress supports:

* `action` - Action to perform on rule match.

* `source` - Source address.

* `destination` - Destination address.

* `protocol` - Protocol (e.g., tcp, udp).

* `source_port` - Source port.

* `destination_port` - Destination port.

* `icmp_type` - Type of ICMP message.

* `icmp_code` - ICMP message code (for ICMP protocol).

* `description` - Description of the rule.

* `state` - State of the rule.
