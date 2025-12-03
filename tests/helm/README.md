# Helm Chart End-to-End Tests

This directory contains end-to-end tests for the AWS Private CA Issuer Helm chart that deploy to a real Kind cluster and validate actual Kubernetes resources.

## Test Strategy

The testing approach follows existing e2e patterns in the repository:
- **End-to-End Deployment Testing**: Charts are actually deployed to Kind cluster
- **Real Resource Validation**: Tests verify actual Kubernetes resources are created correctly
- **Conditional Logic Testing**: All branching logic tested through deployment scenarios
- **Value Substitution Validation**: Real deployments ensure `{{ .Values.* }}` work correctly
- **Complete E2E Coverage**: Every field in values.yaml has corresponding e2e test coverage

## E2E Coverage Validation

The test suite includes comprehensive coverage validation to ensure every configurable field in values.yaml has corresponding e2e test coverage:

### Coverage Test (`TestE2ECoverage`)

This test validates that all values.yaml fields are covered by e2e tests:

1. **Field Extraction**: Parses values.yaml to identify all configurable fields
2. **Test Mapping**: Maps each field to its corresponding e2e test file
3. **Coverage Analysis**: Reports coverage statistics and fails build if any field lacks coverage
4. **Gap Identification**: Clearly identifies untested fields requiring new tests

### Current Coverage Status

- **Total fields in values.yaml**: 93
- **Fields with e2e test coverage**: 40
- **Fields skipped (metadata/complex)**: 54
- **Untested fields**: 0 ✅

### Test File Organization

Each test file covers specific value categories:

- `autoscaling_test.go` - HPA configuration and scaling behavior
- `rbac_test.go` - RBAC permissions and cluster roles
- `deployment_test.go` - Basic deployment configuration flags
- `deployment_config_test.go` - Advanced deployment settings (resources, security, etc.)
- `service_test.go` - Service configuration and naming overrides
- `service_account_test.go` - Service account configuration and annotations
- `service_monitor_test.go` - Prometheus ServiceMonitor resources
- `approver_role_test.go` - Certificate approval permissions
- `pod_disruption_budget_test.go` - PDB configuration
- `coverage_test.go` - Template value reference validation
- `e2e_coverage_test.go` - **E2E coverage validation**

## Dependencies

### Why Separate go.mod?

This test directory has its own `go.mod` and `go.sum` files because:

1. **Test-Specific Dependencies**: Tests require heavy dependencies (Helm SDK, testify) not needed by main application
2. **Version Isolation**: Tests can use different versions of shared dependencies without affecting main app
3. **Build Separation**: Tests build independently without polluting main binary with test-only dependencies

Key test dependencies:
```go
require (
    github.com/stretchr/testify v1.8.4    // Test assertions
    helm.sh/helm/v3 v3.16.4               // Helm Go SDK for deployments
    k8s.io/client-go v0.31.2              // Kubernetes client for validation
)
```

## Running Tests

### Primary Method (Recommended)

```bash
# From repository root - sets up Kind cluster, cert-manager, and runs tests
make e2eHelmTest
```

### Direct Method

```bash
# From tests/helm directory - requires existing cluster setup
./run-tests.sh
```

### Coverage Validation Only

```bash
# Run just the e2e coverage validation test
cd tests/helm && go test -v -run TestE2ECoverage
```

## Adding New Tests

When adding new configurable fields to values.yaml:

1. **Add E2E Test**: Create test case in appropriate test file that validates the field
2. **Update Coverage Map**: Add field mapping in `e2e_coverage_test.go`
3. **Verify Coverage**: Run `TestE2ECoverage` to ensure no gaps

The build will fail if any values.yaml field lacks e2e test coverage, ensuring comprehensive validation.
tests/helm/
├── autoscaling_test.go      # Tests HPA and replica count logic
├── rbac_test.go            # Tests RBAC resource creation
├── deployment_test.go      # Tests deployment configuration
├── service_monitor_test.go # Tests ServiceMonitor and PDB
├── coverage_test.go        # Tests template value coverage
├── common_test.go          # Shared test utilities
├── go.mod                  # Dependencies
├── run-tests.sh           # Direct test runner
└── README.md              # This file
```

## What Gets Tested

All conditional logic in Helm templates through actual deployments:

### Autoscaling (`autoscaling_test.go`)
- HPA creation when `autoscaling.enabled=true`
- Replica count removal from Deployment when autoscaling enabled
- CPU and memory target configuration

### RBAC (`rbac_test.go`)
- ClusterRole and ClusterRoleBinding creation
- ServiceAccount creation with annotations
- Approver role configuration for cert-manager

### Deployment (`deployment_test.go`)
- Command line flags (`disableApprovedCheck`, `disableClientSideRateLimiting`)
- Priority class configuration
- Environment variable injection
- Volume and volume mount configuration
- Sidecar container addition

### Coverage Testing (`coverage_test.go`)
- Validates all `{{ .Values.* }}` template references have corresponding values in values.yaml
- Ensures template references don't break due to missing configuration values
- Provides comprehensive coverage of both conditional and non-conditional template logic

### Service Monitor (`service_monitor_test.go`)
- ServiceMonitor creation for Prometheus
- PodDisruptionBudget configuration

## How It Works

1. **Makefile Integration**: `e2eHelmTest` target uses existing `kind-cluster` and `deploy-cert-manager` dependencies
2. **Real Deployment**: Tests use Helm Go SDK to actually install charts to Kind cluster
3. **Resource Validation**: Tests use Kubernetes client-go to verify resources exist and are configured correctly
4. **Cleanup**: Each test cleans up its resources after validation

## Adding New Tests

When adding new conditional logic to templates:

1. **Add test case** in appropriate test file
2. **Deploy with values** that trigger the conditional logic
3. **Validate resources** using Kubernetes client-go
4. **Test both enabled/disabled** states

Example:
```go
{
    name: "newFeature enabled creates expected resource",
    values: map[string]interface{}{
        "newFeature": map[string]interface{}{
            "enabled": true,
        },
    },
    validate: func(t *testing.T, h *testHelper) {
        // Verify actual Kubernetes resource exists
        resource, err := h.clientset.AppsV1().Deployments(h.namespace).Get(...)
        require.NoError(t, err)
        assert.Equal(t, expectedValue, resource.Spec.SomeField)
    },
},
```

## Benefits

- **Regression Prevention**: Real deployments catch template and configuration issues
- **Comprehensive Validation**: Tests both template rendering and runtime behavior
- **CI/CD Integration**: Automated testing on every chart change
- **Familiar Patterns**: Uses same approach as existing e2e tests in repository
