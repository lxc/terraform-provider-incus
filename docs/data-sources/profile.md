# incus_profile

Provides information about an Incus profile.

## Example Usage

```hcl
data "incus_profile" "this" {
  name = "default"
}

output "profile_name" {
  value = data.incus_profile.this.name
}
```

## Argument Reference

* `name` - **Required** - Name of the profile.

* `project` - *Optional* - Name of the project where the profile is be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.

## Attribute Reference

* `description` - Description of the profile.

* `config` - Map of key/value pairs of config settings.
  [instance config settings](https://linuxcontainers.org/incus/docs/main/reference/instance_options/)

* `device` - Device definition. See reference below.

The `device` block supports:

* `name` - Name of the device.

* `type` - Type of the device Must be one of none, disk, nic,
  unix-char, unix-block, usb, gpu, infiniband, proxy, unix-hotplug, tpm, pci.

* `properties` - Map of key/value pairs of
  [device properties](https://linuxcontainers.org/incus/docs/main/reference/devices/).
