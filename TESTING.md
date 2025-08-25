# Testing

## Setting up VSCode to run tests locally

To set up VSCode to run the tests locally, follow these steps:

1. Create a `.vscode` directory in the root of your project if it doesn't already exist.
2. Inside the `.vscode` directory, create a `settings.json` file.
3. Add the following configuration to the `settings.json` file:

   ```json
   {
     "go.testTimeout": "300s",
     "go.testEnvVars": {
       "TF_ACC": "1"
     }
   }
   ```

This configuration will enable verbose test output, set a test timeout of 30 seconds, run tests on save, and set the necessary environment variables for acceptance tests.

### Using a different Incus Remote

Add `INCUS_REMOTE` to `go.testEnvVars` to use a different Incus Remote instead of `local`.

```json
{
  "go.testTimeout": "300s",
  "go.testEnvVars": {
    "TF_ACC": "1",
    "INCUS_REMOTE": "incus-dev"
  }
}
```

## Setting up an Incus cluster for the clustered acceptance tests

The easiest way to get an Incus cluster for the clustered acceptance tests is
to use [incus-deploy](https://github.com/lxc/incus-deploy).

Follow the instructions in the `incus-deploy` repository to set up your cluster
using terraform (or tofu) and ansible. As a precondition, you have to install
Incus on your machine, e.g. by following the instructions on
[zabbly/incus](https://github.com/zabbly/incus).

With the cluster up and running, the acceptance tests can be executed using
the following command:

```bash
TF_ACC=1 go test ./... -v
```

If the cluster is not the default remote (`local`), you can specify the remote
by setting the `INCUS_REMOTE` environment variable:

```bash
INCUS_REMOTE=your-remote TF_ACC=1 go test ./... -v
```
