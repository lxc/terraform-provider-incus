# terraform-provider-incus

Use Terraform to manage Incus resources.

## Description

This provider connects to the Incus daemon over local Unix socket or HTTPS.

It makes use of the [Incus client library](https://github.com/lxc/incus), which
currently looks in `~/.config/incus` for `client.crt`
and `client.key` files to authenticate against the Incus daemon.

To generate these files and store them in the Incus client config, follow these
[steps](https://linuxcontainers.org/incus/docs/main/howto/server_expose/#authenticate-with-the-incus-server).
Alternatively, the Incus Terraform provider can generate them on demand if
`generate_client_certificates` is set to true.

Minimum required Incus version is **`0.3.0`**.

## Basic Example

This is all that is needed if the Incus remotes have been defined out of band via
the `incus` client.

```hcl
provider "incus" {
}
```

## Specifying Multiple Remotes

If you're running `terraform` from a system where Incus is not installed then you
can define all the remotes in the Provider config:

```hcl
provider "incus" {
  generate_client_certificates = true
  accept_remote_certificate    = true

  remote {
    name    = "incus-server-1"
    address = "https://10.1.1.8"
    token   = "token"
    default = true
  }

  remote {
    name    = "incus-server-2"
    address = "https://10.1.2.8"
    token   = "token"
  }
}
```

## Configuration Reference

The following arguments are supported:

* `remote` - *Optional* - Specifies an Incus remote (Incus server) to connect
  to. See the `remote` reference below for details.

* `config_dir` - *Optional* - The directory to look for existing Incus
  configuration. Defaults to `$HOME/.config/incus`

* `generate_client_certificates` - *Optional* - Automatically generate the Incus
  client certificate if it does not exist. Valid values are `true` and `false`.
  This can also be set with the `INCUS_GENERATE_CLIENT_CERTS` Environment
  variable. Defaults to `false`.

* `accept_remote_certificate` - *Optional* - Automatically accept the Incus
  remote's certificate. Valid values are `true` and `false`. If this is not set
  to `true`, you must accept the certificate out of band of Terraform. This can
  also be set with the `INCUS_ACCEPT_SERVER_CERTIFICATE` environment variable.
  Defaults to `false`

* `default_remote` - *Optional* - The `address` of the default remote to use.
	The specified remote will then be used when one is not specified in a resource.
	If you choose to _not_ set a default_remote and do not specify
	a remote in a resource, this provider will attempt to connect to an Incus
	server running on the same host through the UNIX socket. See `Undefined Remote`
	for more information.
	The default can also be set with the `INCUS_REMOTE` Environment variable.

The `remote` block supports:

* `address` - *Optional* - The address of the Incus remote.

* `name` - *Optional* - The name of the Incus remote.

* `protocol` - *Optional* - The server protocol to use. Valid values are `incus`,`oci`, or `simplestreams`.

* `auth_type` - *Optional* - Server authentication type. Valid values are `tls` or `oidc`. ( Only for the `incus` protocol )

* `token` - *Optional* - The one-time trust [token](https://linuxcontainers.org/incus/docs/main/authentication/#adding-client-certificates-using-tokens) used for initial authentication with the Incus remote.

* `public` - *Optional* - Public image server. Valid values are `true` and `false`.

* `default_project` - *Optional* - Default project to configure. ( Only for the `incus` protocol ).

## Undefined Remote

If you choose to *not* define a `remote`, this provider will attempt
to connect to an Incus server running on the same host through the UNIX
socket.

## Environment Variable Remote

It is possible to define a single `remote` through environment variables.
The required variables are:

* `INCUS_REMOTE` - The name of the remote.
* `INCUS_ADDR` - The address of the Incus remote.
* `INCUS_PROTOCOL` - The server protocol to use.
* `INCUS_AUTHTYPE` - Server authentication type.
* `INCUS_TOKEN` - The trust token of the Incus remote.
* `INCUS_DEFAULTPROJECT` - Default project to configure.
* `INCUS_PUBLIC` - Public image server.

## PKI Support

Incus is capable of [authenticating via PKI](https://linuxcontainers.org/incus/docs/main/authentication/#using-a-pki-system). In order to do this, you must
generate appropriate certificates on *both* the remote/server side and client
side. Details on how to generate these certificates is out of scope of this
document.
