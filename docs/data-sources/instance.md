# incus_instance

Provides information about an Incus instance.
See Incus instance [configuration reference](https://linuxcontainers.org/incus/docs/main/explanation/instance_config/) for more details.

## Example Usage

```hcl
data "incus_instance" "this" {
  name = "default"
}

output "instance_name" {
  value = data.incus_instance.this.name
}
```

## Argument Reference

* `name` - **Required** - Name of the instance.

* `project` - *Optional* - Name of the project where the instance is be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

* `description` - Description of the instance.

* `config` - Map of key/value pairs of config settings.
  [instance config settings](https://linuxcontainers.org/incus/docs/main/reference/instance_options/)

* `status` - Status of the instance.

* `location` - Location of the instance.

* `device` - Device definitions. See reference below.

* `type` - Instance type.

* `architecture` - Architecture name.

* `ephemeral` - Whether the instance is ephemeral (deleted on shutdown).

* `profiles` - List of profiles applied to the instance.

* `stateful` - Whether the instance is stateful.

The `device` blocks support:

* `name` - Name of the device.

* `type` - Type of the device Must be one of none, disk, nic,
  unix-char, unix-block, usb, gpu, infiniband, proxy, unix-hotplug, tpm, pci.

* `properties` - Map of key/value pairs of
  [device properties](https://linuxcontainers.org/incus/docs/main/reference/devices/).
