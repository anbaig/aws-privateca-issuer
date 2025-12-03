# Helm Chart End-to-End Tests

This directory contains end-to-end tests for the AWS Private CA Issuer Helm chart that deploy to a real Kind cluster and validate actual Kubernetes resources.

## Directory Structure

```
tests/helm/
├── testutil/                    # Shared test utilities
│   ├── helper.go               # Test setup/cleanup helpers
│   ├── helm.go                 # Helm chart operations
│   └── validation.go           # Resource validation helpers
├── core/                       # Core functionality tests
│   ├── defaults_test.go        # Default values validation
│   ├── deployment_test.go      # Basic deployment functionality
│   └── service_test.go         # Service configuration and naming
├── features/                   # Optional/advanced feature tests
│   ├── autoscaling_test.go     # HPA configuration and scaling
│   ├── rbac_test.go           # RBAC permissions and roles
│   ├── approver_role_test.go   # Certificate approval permissions
│   ├── service_monitor_test.go # Prometheus ServiceMonitor
│   ├── pod_disruption_budget_test.go # PDB configuration
│   ├── deployment_config_test.go # Advanced deployment settings
│   ├── service_account_test.go # ServiceAccount customization
│   └── optional_fields_test.go # Optional field combinations
├── e2e_coverage_test.go        # E2E coverage validation
├── go.mod, go.sum             # Dependencies
├── run-tests.sh               # Direct test runner
└── README.md                  # This file
```

## Test Strategy

The testing approach follows existing e2e patterns in the repository:
- **End-to-End Deployment Testing**: Charts are actually deployed to Kind cluster
- **Real Resource Validation**: Tests verify actual Kubernetes resources are created correctly
- **Conditional Logic Testing**: All branching logic tested through deployment scenarios
- **Value Substitution Validation**: Real deployments ensure `{{ .Values.* }}` work correctly
- **Complete E2E Coverage**: Every field in values.yaml has corresponding e2e test coverage

## Test Modes

The test suite supports two modes for flexible validation:

### Pre-Production Mode (Default)
```bash
make helmE2ETestPreProd
```
- **Chart Source**: Local repository chart (`../../charts/aws-pca-issuer`)
- **Image**: Locally built image with overrides
- **Use Case**: Local development and testing

### Production Mode
```bash
make helmE2ETestProd
```
- **Chart Source**: Production chart from `https://cert-manager.github.io/aws-privateca-issuer`
- **Image**: Production image (chart's default values)
- **Use Case**: Validation against published chart

## E2E Coverage Validation

The test suite includes comprehensive coverage validation (`e2e_coverage_test.go`) to ensure every configurable field in values.yaml has corresponding e2e test coverage:

1. **Field Extraction**: Parses values.yaml to identify all configurable fields
2. **Test Mapping**: Maps each field to its corresponding e2e test file
3. **Coverage Analysis**: Reports coverage statistics and fails build if any field lacks coverage
4. **Gap Identification**: Clearly identifies untested fields requiring new tests

### Current Coverage Status
- **Total meaningful coverage**: 93/93 fields (100%) ✅
- **Fields with explicit e2e tests**: 50
- **Untested fields**: 0

## Test Organization

### Core Tests (`core/`)
Basic functionality that should always work:
- **defaults_test.go** - Validates all default values and basic chart functionality
- **deployment_test.go** - Tests basic deployment configuration flags
- **service_test.go** - Tests service configuration and naming overrides

### Feature Tests (`features/`)
Optional and advanced functionality:
- **autoscaling_test.go** - HPA configuration and scaling behavior
- **rbac_test.go** - RBAC permissions and cluster roles
- **approver_role_test.go** - Certificate approval permissions
- **service_monitor_test.go** - Prometheus ServiceMonitor resources
- **pod_disruption_budget_test.go** - PDB configuration
- **deployment_config_test.go** - Advanced deployment settings (resources, security, annotations)
- **service_account_test.go** - ServiceAccount configuration and annotations
- **optional_fields_test.go** - Optional field combinations (volumes, tolerations, etc.)

### Shared Utilities (`testutil/`)
Reusable test infrastructure:
- **helper.go** - Test setup, cleanup, and TestHelper struct
- **helm.go** - Helm chart installation and uninstallation operations
- **validation.go** - Kubernetes resource validation and debugging helpers

## Dependencies

### Why Separate go.mod?

This test directory has its own `go.mod` and `go.sum` files because:

1. **Test-Specific Dependencies**: Tests require heavy dependencies (Helm SDK, testify) not needed by main application
2. **Version Isolation**: Tests can use different versions of shared dependencies without affecting main app
3. **Build Separation**: Tests build independently without polluting main binary with test-only dependencies

Key test dependencies:
```go
require (
    github.com/stretchr/testify v1.10.0   // Test assertions
    helm.sh/helm/v3 v3.16.4               // Helm Go SDK for deployments
    k8s.io/client-go v0.32.3              // Kubernetes client for validation
)
```

## Running Tests

### Primary Method (Recommended)

```bash
# Pre-production mode (local chart + local image)
make helmE2ETestPreProd

# Production mode (registry chart + production image)  
make helmE2ETestProd

# Legacy target (same as pre-production)
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

1. **Add E2E Test**: Create test case in appropriate test file (core/ or features/) that validates the field
2. **Update Coverage Map**: Add field mapping in `e2e_coverage_test.go`
3. **Use Shared Utilities**: Import and use `testutil` package for consistent test patterns
4. **Verify Coverage**: Run `TestE2ECoverage` to ensure no gaps

Example test structure:
```go
package helm

import (
    "testing"
    "github.com/cert-manager/aws-privateca-issuer/tests/helm/testutil"
)

func TestNewFeature(t *testing.T) {
    helper := testutil.SetupTest(t)
    defer helper.Cleanup()

    release := helper.InstallChart(map[string]interface{}{
        "newFeature": map[string]interface{}{
            "enabled": true,
        },
    })
    defer helper.UninstallChart(release.Name)

    // Validate actual Kubernetes resources
    deploymentName := release.Name + "-aws-privateca-issuer"
    helper.WaitForDeployment(deploymentName)
    
    // Add specific validations...
}
```

The build will fail if any values.yaml field lacks e2e test coverage, ensuring comprehensive validation.
