# generate-datasources

`generate-datasources` is a tool to generate data sources for the the Terraform
provider. It is controlled by the config file `generate-datasources.yaml`.

## Usage

```shell
go run ./cmd/generate-datasources
```

For development or debugging of `generate-datasources`, the following flags might be helpful:

```none
Usage of generate-datasources:
      --config string          filename of the configfile (default "generate-datasources.yaml")
  -d, --debug                  Show all debug messages
      --only-entity string     Limit code generation to this entity
      --only-template string   Limit code generation to this template
  -v, --verbose                Show all information messages
```

With `--only-*`, the generator can be instructed to only generate some defined parts:

* `entity` is the name of the entity as defined in the config file, e.g. `network`.
* `template` is the name of the template file in `./cmd/generate-datasources/tmpl`, e.g. `datasource.go.gotmpl`.

## Config `generate-datasources.yaml`

The config settings available per entity, for which the data source should be
generated, are defined and documented in [config.go](./config.go).

## Templates

The templates used by `generate-datasources` are [Go templates](https://pkg.go.dev/text/template).

Templating is used for two purposes:

* target file name and path
* content of the generated file

There are two kinds of template files:

* regular: these templates are executed for each entity (resource).
* global: these templates are only executed once for all entities.

### Arguments

For the *regular* templates, the arguments defined in `entityArgs` for the
currently generated entity are available.

For *global* templates, a map with all entities is passed, where the key is
the name of the entity (as provided in the config file) and the value is the
respective `entityArgs` instance.

The type `entityArgs` is defined and documented in [main.go](./main.go).

### Functions

Additionally to the functions and operators provided by  [Go templates](https://pkg.go.dev/text/template) also the complete set of functions provided by the [sprig](https://masterminds.github.io/sprig/) library is available, with some functions being replaced with equivalents
(see below).

The following specialized functions are provided:

* `pascalcase`: Convert a string in snake case to pascal case (`some_name` -> `SomeName`)
* `camelcase`: Convert a string in snake case to camel case (`some_name` -> `someName`)
* `kebabcase`: Convert a string in snake case to kebab case (`some_name` -> `some-name`)
* `titlecase`: Convert a string in snake case to title case (`some_name` -> `Some Name`)
* `words`: Split a string in snake case into words (`some_name` -> `some name`)

These functions do value acronyms and will convert them to the appropriate case as well.
For the list of supported acronyms, please refer to [funcs.go](./funcs.go).
