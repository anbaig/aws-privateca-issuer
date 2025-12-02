package helm

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConditionalValuesCoverage(t *testing.T) {
	// Define values that trigger conditional logic in templates
	conditionalKeys := []string{
		"autoscaling.enabled",
		"rbac.create",
		"serviceAccount.create",
		"serviceMonitor.create",
		"disableApprovedCheck",
		"disableClientSideRateLimiting",
		"approverRole.enabled",
		"podDisruptionBudget",
		"priorityClassName",
		"env",
		"volumes",
		"volumeMounts",
		"extraContainers",
		"podLabels",
	}

	// Load values.yaml
	valuesFile, err := os.ReadFile("../../charts/aws-pca-issuer/values.yaml")
	require.NoError(t, err)

	var values map[string]interface{}
	err = yaml.Unmarshal(valuesFile, &values)
	require.NoError(t, err)

	var missing []string
	for _, key := range conditionalKeys {
		if !hasNestedKey(values, key) {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		t.Errorf("Conditional values missing from values.yaml: %v", missing)
	}
}

func TestTemplateValueReferences(t *testing.T) {
	// Parse all template files for .Values references
	templateDir := "../../charts/aws-pca-issuer/templates"
	valueRefs := extractValueReferences(t, templateDir)

	// Load values.yaml
	valuesFile, err := os.ReadFile("../../charts/aws-pca-issuer/values.yaml")
	require.NoError(t, err)

	var values map[string]interface{}
	err = yaml.Unmarshal(valuesFile, &values)
	require.NoError(t, err)

	// Skip validation for optional values that may be commented out
	skipValidation := map[string]bool{
		"autoscaling.targetMemoryUtilizationPercentage": true,
	}

	// Check each template reference exists in values
	var missing []string
	for _, ref := range valueRefs {
		if skipValidation[ref] {
			continue
		}
		if !hasNestedKey(values, ref) {
			missing = append(missing, ref)
		}
	}

	if len(missing) > 0 {
		t.Errorf("Template references missing from values.yaml: %v", missing)
	}

	t.Logf("Validated %d template value references", len(valueRefs))
}

func extractValueReferences(t *testing.T, templateDir string) []string {
	var refs []string
	valueRegex := regexp.MustCompile(`\{\{\s*\.Values\.([a-zA-Z0-9_.]+)\s*\}\}`)

	err := filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		matches := valueRegex.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) > 1 {
				refs = append(refs, match[1])
			}
		}

		return nil
	})

	require.NoError(t, err)

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, ref := range refs {
		if !seen[ref] {
			seen[ref] = true
			unique = append(unique, ref)
		}
	}

	return unique
}

func hasNestedKey(data map[string]interface{}, key string) bool {
	parts := strings.Split(key, ".")
	current := data

	for i, part := range parts {
		if val, exists := current[part]; exists {
			if i == len(parts)-1 {
				return true // Found the final key
			}
			if nested, ok := val.(map[string]interface{}); ok {
				current = nested
			} else {
				return false // Path exists but not as nested map
			}
		} else {
			return false // Key doesn't exist
		}
	}

	return false
}
