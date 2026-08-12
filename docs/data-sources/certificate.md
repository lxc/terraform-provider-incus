# incus_certificate

Provides information about an Incus certificate.

## Example Usage

```hcl
data "incus_certificate" "this" {
  fingerprint = "default"
}

output "certificate_fingerprint" {
  value = data.incus_certificate.this.fingerprint
}
```

## Argument Reference

* `fingerprint` - **Required** - Fingerprint of the certificate.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

* `description` - Description of the certificate.

* `name` - Certificate name.

* `certificate` - PEM-encoded X.509 certificate.

* `type` - Certificate usage type.

* `restricted` - Whether the certificate is restricted to listed projects.

* `projects` - Projects permitted to use the certificate.
