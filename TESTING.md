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
