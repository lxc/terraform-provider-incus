# incus_server

Provides information about an Incus server setup. This resource supports
clustered as well as standalone setups and will treat them accordingly.
Standalone setups will look like a cluster with a single member.

## Example Usage

```hcl
data "incus_server" "local" {
  remote = "local"
}
```

## Example prevent execution if any server is not online

```hcl
data "incus_server" "cluster" {
  remote = "cluster"

  lifecycle {
    postcondition {
      condition     = alltrue([for i, v in self.members : v.status == "Online"])
      error_message = "All servers must be online."
    }
  }
}
```

## Argument Reference

* `remote` - *Optional* - The remote for which the Incus server setup should be
  queried. If not provided, the provider's default remote will be used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments
above:

* `is_clustered` - Whether this is a clustered setup.

* `members` - List of cluster members of the setup. See reference below.

The `member` block contains:

* `addresses` - List of addresses of the cluster member.

* `server_name` - Server name of the cluster member.

* `status` - Status of the cluster member. Possible values are
  `Online`, `Evacuated`, `Offline`, `Blocked`.

## Notes

* For standalone setups, the `members` attribute will contain a single entry
  with the server's own `addresses` and `server_name`. The `status` will be
  always set to `Online`.
