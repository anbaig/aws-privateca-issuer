package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *TestHelper) InstallChart(values map[string]interface{}) *release.Release {
	mode := GetTestMode()
	modeNames := map[TestMode]string{
		LocalMode: "local",
		BetaMode:  "beta",
		ProdMode:  "production",
	}
	h.T.Logf("Starting chart installation in %s mode with values: %+v", modeNames[mode], values)

	settings := cli.New()
	settings.KubeConfig = "/tmp/pca_kubeconfig"
	actionConfig := new(action.Configuration)

	err := actionConfig.Init(settings.RESTClientGetter(), h.Namespace, "secret", func(format string, v ...interface{}) {
		h.T.Logf("Helm: "+format, v...)
	})
	if !assert.NoError(h.T, err, "Failed to initialize Helm action config") {
		return nil
	}
	h.T.Logf("Helm action configuration initialized successfully")

	// Generate unique release name
	releaseName := fmt.Sprintf("%s-%d", ReleasePrefix, time.Now().UnixNano())
	h.T.Logf("Generated release name: %s", releaseName)

	install := action.NewInstall(actionConfig)
	install.ReleaseName = releaseName
	install.Namespace = h.Namespace
	install.CreateNamespace = true
	install.Wait = false
	install.Timeout = 2 * time.Minute

	var chart *chart.Chart

	if mode == ProdMode {
		// Production mode: Use chart from Helm registry
		h.T.Logf("Production mode: Installing from Helm registry")

		// Use Helm's built-in repository functionality
		install.ChartPathOptions.RepoURL = ProdChartRepo
		chartPath, err := install.ChartPathOptions.LocateChart("aws-privateca-issuer", settings)
		if err != nil {
			h.T.Logf("Failed to locate chart from repository: %v", err)
			return nil
		}

		chart, err = loader.Load(chartPath)
		if !assert.NoError(h.T, err, "Failed to load chart from repository") {
			return nil
		}
		h.T.Logf("Production chart loaded successfully: %s-%s", chart.Name(), chart.Metadata.Version)
	} else {
		// Local and Beta modes: Use local chart
		h.T.Logf("%s mode: Loading chart from local path: %s",
			map[TestMode]string{LocalMode: "Local", BetaMode: "Beta"}[mode], LocalChartPath)
		chart, err = loader.Load(LocalChartPath)
		if !assert.NoError(h.T, err, "Failed to load local chart") {
			return nil
		}
		h.T.Logf("Local chart loaded successfully: %s-%s", chart.Name(), chart.Metadata.Version)
	}

	// Set default values for testing if not provided
	if values == nil {
		values = make(map[string]interface{})
	}

	// Configure image based on mode
	if mode == LocalMode {
		// Local mode: Override with local ECR image if not specified
		if _, exists := values["image"]; !exists {
			values["image"] = map[string]interface{}{
				"repository": "public.ecr.aws/k1n1h4h4/cert-manager-aws-privateca-issuer",
				"tag":        "v1.2.7",
				"pullPolicy": "IfNotPresent",
			}
		}
	} else if mode == BetaMode {
		// Beta mode: Override with testECR image if not specified
		if _, exists := values["image"]; !exists {
			values["image"] = map[string]interface{}{
				"repository": "public.ecr.aws/cert-manager-aws-privateca-issuer/cert-manager-aws-privateca-issuer-test",
				"tag":        "latest",
				"pullPolicy": "Always",
			}
		}
	}
	// Production mode: Use chart's default image values (no overrides)

	// Set common test defaults if not already specified
	if _, exists := values["livenessProbe"]; !exists {
		values["livenessProbe"] = map[string]interface{}{
			"enabled": false,
		}
	}
	if _, exists := values["readinessProbe"]; !exists {
		values["readinessProbe"] = map[string]interface{}{
			"enabled": false,
		}
	}
	if _, exists := values["approverRole"]; !exists {
		values["approverRole"] = map[string]interface{}{
			"enabled": false,
		}
	}

	h.T.Logf("Final values for installation: %+v", values)

	h.T.Logf("Installing chart...")
	release, err := install.Run(chart, values)
	if !assert.NoError(h.T, err, "Failed to install chart") {
		return nil
	}

	h.T.Logf("Helm release %s installed successfully", release.Name)
	h.T.Logf("Release manifest length: %d", len(release.Manifest))

	// Validate chart version for production mode
	if err := h.ValidateChartVersion(release); err != nil {
		h.T.Errorf("Chart version validation failed: %v", err)
		// Cleanup the release since validation failed
		h.UninstallChart(release.Name)
		return nil
	}

	time.Sleep(2 * time.Second) // Give time for resources to be created

	// List all resources to debug what was actually created
	pods, _ := h.Clientset.CoreV1().Pods(h.Namespace).List(context.TODO(), metav1.ListOptions{})
	h.T.Logf("Pods created: %d", len(pods.Items))

	deployments, _ := h.Clientset.AppsV1().Deployments(h.Namespace).List(context.TODO(), metav1.ListOptions{})
	h.T.Logf("Deployments created: %d", len(deployments.Items))
	for _, dep := range deployments.Items {
		h.T.Logf("  - Deployment: %s", dep.Name)
	}

	services, _ := h.Clientset.CoreV1().Services(h.Namespace).List(context.TODO(), metav1.ListOptions{})
	h.T.Logf("Services created: %d", len(services.Items))
	for _, svc := range services.Items {
		h.T.Logf("  - Service: %s", svc.Name)
	}

	return release
}

func (h *TestHelper) UninstallChart(releaseName string) {
	settings := cli.New()
	settings.KubeConfig = "/tmp/pca_kubeconfig"
	actionConfig := new(action.Configuration)

	err := actionConfig.Init(settings.RESTClientGetter(), h.Namespace, "secret", func(format string, v ...interface{}) {
		h.T.Logf(format, v...)
	})
	if err != nil {
		return
	}

	uninstall := action.NewUninstall(actionConfig)
	uninstall.Run(releaseName)
}
