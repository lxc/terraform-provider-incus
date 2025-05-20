# incus_image_alias

Manages a locally-stored Incus image alias.

## Example Usage

```hcl
resource "incus_image" "alpine" {
  source_image = {
    remote = "images"
    name   = "alpine/edge"
  }
}

resource "incus_image_alias" "alpine" {
  alias       = "alpine"
  description = "Alpine Edge"
  fingerprint = incus_image.alpine.fingerprint
}
```

## Argument Reference

* `alias` - *Required* - An alias to assign to the image after pulling.

* `description` - *Optional* - Description of the image alias.

* `fingerprint` - *Required* - The unique hash fingperint of the image.

* `project` - *Optional* - Name of the project where the image will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.
