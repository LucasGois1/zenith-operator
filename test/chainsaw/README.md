# Chainsaw Tests for Zenith Operator

This directory contains Chainsaw end-to-end tests for the Zenith Operator. These tests validate the operator's functionality by creating Function custom resources and verifying that the operator correctly orchestrates Tekton, Knative, and other components.

## Prerequisites

Before running these tests, ensure you have:

1. **Kubernetes cluster** with the following components installed:
   - Tekton Pipelines
   - Knative Serving
   - Tekton Catalog tasks: `git-clone` and `buildpacks-phases`

2. **Chainsaw CLI** installed:
   ```bash
   curl -L https://github.com/kyverno/chainsaw/releases/latest/download/chainsaw_linux_amd64.tar.gz -o /tmp/chainsaw.tar.gz
   tar -xzf /tmp/chainsaw.tar.gz -C /tmp
   sudo mv /tmp/chainsaw /usr/local/bin/
   ```

3. **Operator deployed** to the cluster:
   ```bash
   make deploy IMG=<your-operator-image>
   ```

## Test Scenarios

### 1. Basic Function Test (`basic-function/`)

Tests the complete lifecycle of a Function resource:
- Function CR creation
- PipelineRun creation for building the container image
- Status progression: Building → BuildSucceeded
- Knative Service creation after successful build
- Image digest population in status

**What it validates:**
- Git repository cloning works correctly
- Buildpacks build process completes successfully
- Status conditions are updated correctly
- Knative Service is created with the built image

### 2. ServiceAccount Secret Binding Test (`serviceaccount-secret/`)

Tests that the operator correctly manages registry credentials:
- Creates a registry secret
- Verifies the operator adds the secret to the default ServiceAccount's imagePullSecrets
- Ensures the ServiceAccount is updated before the PipelineRun starts

**What it validates:**
- Registry secret binding to ServiceAccount
- ServiceAccount update logic
- Proper credential propagation for image push/pull

### 3. Git Clone Validation Test (`git-clone-validation/`)

Tests that the operator correctly configures the git-clone task:
- Verifies the PipelineRun includes the correct git repository URL
- Verifies the PipelineRun includes the correct git revision
- Ensures the git-clone task parameters are properly set

**What it validates:**
- Git repository URL is correctly passed to the PipelineRun
- Git revision (branch/tag/commit) is correctly configured
- PipelineRun task configuration is correct

## Running the Tests

### Run all tests:
```bash
make test-chainsaw
```

### Run a specific test:
```bash
chainsaw test --test-dir test/chainsaw/basic-function
```

### Run tests with verbose output:
```bash
chainsaw test --test-dir test/chainsaw -v
```

### Run tests and keep resources for debugging:
```bash
chainsaw test --test-dir test/chainsaw --no-cleanup
```

## Test Configuration

The global test configuration is in `.chainsaw-test.yaml`:
- Default timeout: 5 minutes for assertions
- Namespace: `chainsaw-test` (automatically created and cleaned up)
- Parallel execution: 1 (sequential)

## Understanding Test Results

Each test follows this pattern:
1. **Apply**: Create resources (Function CR, Secrets, etc.)
2. **Assert**: Verify expected state (PipelineRun created, Status updated, etc.)
3. **Cleanup**: Delete resources

If a test fails, Chainsaw will:
- Show which assertion failed
- Display the actual vs expected state
- Provide logs from the resources

## Debugging Failed Tests

If a test fails:

1. **Check operator logs:**
   ```bash
   kubectl logs -n zenith-operator-system deployment/zenith-operator-controller-manager
   ```

2. **Check PipelineRun status:**
   ```bash
   kubectl get pipelineruns -n chainsaw-test
   kubectl describe pipelinerun <name> -n chainsaw-test
   ```

3. **Check Function status:**
   ```bash
   kubectl get functions -n chainsaw-test
   kubectl describe function <name> -n chainsaw-test
   ```

4. **Run with no cleanup to inspect resources:**
   ```bash
   chainsaw test --test-dir test/chainsaw --no-cleanup
   ```

## Adding New Tests

To add a new test:

1. Create a new directory under `test/chainsaw/`
2. Create a `chainsaw-test.yaml` file defining the test steps
3. Create YAML files for resources to apply
4. Create assertion YAML files to verify expected state
5. Update this README with the new test scenario

Example structure:
```
test/chainsaw/my-new-test/
├── chainsaw-test.yaml       # Test definition
├── function.yaml            # Resource to create
├── function-assert.yaml     # Expected state
└── cleanup.yaml             # Optional cleanup
```

## Test Function Repository

The tests use functions from: https://github.com/LucasGois1/zenith-test-functions

Currently available test functions:
- `go-hello/`: Simple Go HTTP server for basic testing

## Notes

- Tests use `ttl.sh` as a temporary container registry (images expire after 1 hour)
- Each test runs in an isolated namespace
- Tests are designed to be idempotent and can be run multiple times
- Build times may vary depending on network speed and cluster resources (typically 3-5 minutes)
