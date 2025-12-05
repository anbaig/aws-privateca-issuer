package helm

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestE2ECoverage(t *testing.T) {
	// Load values.yaml
	valuesFile, err := os.ReadFile("../../charts/aws-pca-issuer/values.yaml")
	require.NoError(t, err)

	var values map[string]interface{}
	err = yaml.Unmarshal(valuesFile, &values)
	require.NoError(t, err)

	// Extract all field paths from values.yaml
	allFields := extractAllFieldPaths(values, "")

	// Define which fields are tested by which e2e tests
	testedFields := map[string]string{
		// Autoscaling fields - tested in autoscaling_test.go
		"autoscaling.enabled":                        "autoscaling_test.go",
		"autoscaling.minReplicas":                    "autoscaling_test.go",
		"autoscaling.maxReplicas":                    "autoscaling_test.go",
		"autoscaling.targetCPUUtilizationPercentage": "autoscaling_test.go",

		// RBAC fields - tested in rbac_test.go
		"rbac.create": "rbac_test.go",

		// ServiceAccount fields - tested in service_account_test.go
		"serviceAccount.create":      "rbac_test.go",
		"serviceAccount.name":        "service_account_test.go",
		"serviceAccount.annotations": "service_account_test.go",

		// Deployment fields - tested in deployment_test.go and deployment_config_test.go
		"disableApprovedCheck":          "deployment_test.go",
		"disableClientSideRateLimiting": "deployment_config_test.go",
		"podAnnotations":                "deployment_config_test.go",
		"podLabels":                     "deployment_config_test.go",

		// ServiceMonitor fields - tested in service_monitor_test.go
		"serviceMonitor.create": "service_monitor_test.go",

		// Defaults test - validates all default values
		"replicaCount":              "defaults_test.go",
		"revisionHistoryLimit":      "defaults_test.go",
		"image":                     "defaults_test.go",
		"image.repository":          "defaults_test.go",
		"image.pullPolicy":          "defaults_test.go",
		"image.tag":                 "defaults_test.go",
		"resources":                 "defaults_test.go",
		"resources.limits.cpu":      "defaults_test.go",
		"resources.limits.memory":   "defaults_test.go",
		"resources.requests.cpu":    "defaults_test.go",
		"resources.requests.memory": "defaults_test.go",
		"securityContext":           "defaults_test.go",
		"podSecurityContext":        "defaults_test.go",
		"service":                   "defaults_test.go",
		"service.type":              "defaults_test.go",
		"service.port":              "defaults_test.go",
		"podDisruptionBudget":       "defaults_test.go",
		"affinity":                  "defaults_test.go",
		"topologySpreadConstraints": "defaults_test.go",

		// Naming fields - tested in service_test.go
		"nameOverride":     "service_test.go",
		"fullnameOverride": "service_test.go",

		// ApproverRole fields - tested in approver_role_test.go
		"approverRole":                    "approver_role_test.go",
		"approverRole.enabled":            "approver_role_test.go",
		"approverRole.serviceAccountName": "approver_role_test.go",
		"approverRole.namespace":          "approver_role_test.go",

		// RBAC parent - tested in rbac_test.go
		"rbac": "rbac_test.go",

		// ServiceAccount parent - tested in service_account_test.go
		"serviceAccount": "service_account_test.go",

		// ServiceMonitor parent - tested in service_monitor_test.go
		"serviceMonitor": "service_monitor_test.go",

		// Autoscaling parent - tested in autoscaling_test.go
		"autoscaling": "autoscaling_test.go",

		// Optional fields - tested in optional_fields_test.go
		"imagePullSecrets":  "optional_fields_test.go",
		"tolerations":       "optional_fields_test.go",
		"nodeSelector":      "optional_fields_test.go",
		"volumes":           "optional_fields_test.go",
		"volumeMounts":      "optional_fields_test.go",
		"extraContainers":   "optional_fields_test.go",
		"env":               "optional_fields_test.go",
		"priorityClassName": "optional_fields_test.go",
	}

	// Fields that don't need e2e testing
	skipFields := map[string]bool{}

	// Find untested fields
	var untestedFields []string
	for _, field := range allFields {
		if skipFields[field] {
			continue
		}
		if _, tested := testedFields[field]; !tested {
			// Check if parent field is tested (covers child fields)
			if !isParentTested(field, testedFields) {
				untestedFields = append(untestedFields, field)
			}
		}
	}

	// Report results
	if len(untestedFields) > 0 {
		t.Errorf("The following values.yaml fields lack e2e test coverage:\n%s\n\nAdd tests for these fields or update the testedFields map if they are already tested.",
			strings.Join(untestedFields, "\n"))
	}

	// Calculate meaningful coverage (tested + covered by parents + skipped = total coverage)
	coveredByParents := len(allFields) - len(testedFields) - len(skipFields) - len(untestedFields)
	meaningfulCoverage := len(testedFields) + coveredByParents
	coveragePercent := float64(meaningfulCoverage) / float64(len(allFields)) * 100

	t.Logf("E2E Coverage Analysis:")
	t.Logf("- Fields with explicit e2e tests: %d", len(testedFields))
	t.Logf("- Untested fields: %d", len(untestedFields))
	t.Logf("- Total meaningful coverage: %d/%d (%.1f%%)", meaningfulCoverage, len(allFields), coveragePercent)
}

func extractAllFieldPaths(data interface{}, prefix string) []string {
	var paths []string

	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			fullPath := key
			if prefix != "" {
				fullPath = prefix + "." + key
			}
			paths = append(paths, fullPath)
			paths = append(paths, extractAllFieldPaths(value, fullPath)...)
		}
	case []interface{}:
		for i, item := range v {
			indexPath := prefix + "." + string(rune('0'+i))
			paths = append(paths, indexPath)
			paths = append(paths, extractAllFieldPaths(item, indexPath)...)
		}
	default:
		// Leaf node - already added by parent
	}

	return paths
}

func isParentTested(field string, testedFields map[string]string) bool {
	parts := strings.Split(field, ".")
	// Check each parent level (from most specific to least)
	for i := len(parts) - 1; i > 0; i-- {
		parent := strings.Join(parts[:i], ".")
		if _, tested := testedFields[parent]; tested {
			return true
		}
	}
	return false
}
