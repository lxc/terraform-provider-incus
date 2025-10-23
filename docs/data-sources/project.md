# incus_project

Provides information about an Incus project.

## Example Usage

```hcl
data "incus_project" "this" {
  name = "default"
}

output "project_name" {
  value = data.incus_project.this.name
}
```

## Argument Reference

* `name` - **Required** - Name of the project.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

* `description` - Description of the project.

* `config` - Map of key/value pairs of config settings.
  [instance config settings](https://linuxcontainers.org/incus/docs/main/reference/instance_options/)
