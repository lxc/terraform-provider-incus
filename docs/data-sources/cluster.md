# incus_cluster

Provides information about an Incus cluster.

## Example Usage

```hcl
data "incus_cluster" "this" {
  # No arguments are required.
}
```

## Example prevent execution if any cluster member is not online

```hcl
data "incus_cluster" "this" {
  remote = "cluster"

  lifecycle {
    postcondition {
      condition     = alltrue(self.is_clustered ? [for i, v in self.members : v.status == "Online"] : [])
      error_message = "All servers must be online."
    }
  }
}
```

## Example create resource for each cluster member

In this example, we define the server configuration [`core.bgp_address`](https://linuxcontainers.org/incus/docs/main/server_config/#core-configuration),
 which has scope `local`, on each cluster member.

```hcl
data "incus_cluster" "this" {}

resource "incus_server" "nodes" {
  for_each = data.incus_cluster.this.members

  target = data.incus_cluster.this.is_clustered ? each.key : null

  config = {
    "core.bgp_address" = ":179"
  }
}
```

## Argument Reference

* `remote` - *Optional* - The remote for which the Incus cluster information
  should be queried. If not provided, the provider's default remote will be used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments
above:

* `is_clustered` - Whether this is a clustered setup.

* `members`: A map of cluster members. The key is the member name and the value
  is a member object. See reference below.

The `member` block contains:

* `address` - Address of the cluster member, that is used for cluster communication.

* `architecture` - Architecture of the cluster member (e.g. x86_64, aarch64).
  See [Architectures](https://linuxcontainers.org/incus/docs/main/architectures/)
  for all possible values.

* `description` - Description of the cluster member.

* `failure_domain` - Failure domain of the cluster member.

* `groups` - A list of groups the cluster member belongs to.

* `roles` - A list of roles assigned to the cluster member.

* `status` - Status of the cluster member. Possible values are
  `Online`, `Evacuated`, `Offline`, `Blocked`.

## Notes

* For non-clustered setups, the `members` attribute will be `null`.
