package testutil

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *TestHelper) WaitForDeployment(name string) {
	// Just check that the deployment exists, don't wait for readiness
	// Add initial delay to allow Helm to create resources
	time.Sleep(2 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			// List all deployments to debug
			deployments, err := h.Clientset.AppsV1().Deployments(h.Namespace).List(context.TODO(), metav1.ListOptions{})
			if err == nil {
				h.T.Logf("Available deployments in namespace %s:", h.Namespace)
				for _, dep := range deployments.Items {
					h.T.Logf("  - %s (Ready: %d/%d)", dep.Name, dep.Status.ReadyReplicas, dep.Status.Replicas)

					// Log deployment conditions if not ready
					if dep.Status.ReadyReplicas != dep.Status.Replicas {
						h.T.Logf("    Deployment %s conditions:", dep.Name)
						for _, cond := range dep.Status.Conditions {
							h.T.Logf("      %s: %s - %s", cond.Type, cond.Status, cond.Message)
						}

						// Get pod details for this deployment
						h.logPodFailures(dep.Name)
					}
				}
			}
			h.T.Errorf("Timeout waiting for deployment %s to be created", name)
			return
		default:
			_, err := h.Clientset.AppsV1().Deployments(h.Namespace).Get(context.TODO(), name, metav1.GetOptions{})
			if err == nil {
				// Print resources for debugging
				h.PrintResourcesForDebugging(name)
				return // Deployment exists, that's enough for our tests
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func (h *TestHelper) logPodFailures(deploymentName string) {
	pods, err := h.Clientset.CoreV1().Pods(h.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/name=aws-privateca-issuer,app.kubernetes.io/instance=%s", deploymentName),
	})
	if err != nil {
		h.T.Logf("    Failed to get pods for deployment %s: %v", deploymentName, err)
		return
	}

	for _, pod := range pods.Items {
		h.T.Logf("    Pod %s: Phase=%s", pod.Name, pod.Status.Phase)

		// Log container statuses
		for _, containerStatus := range pod.Status.ContainerStatuses {
			h.T.Logf("      Container %s: Ready=%t, RestartCount=%d",
				containerStatus.Name, containerStatus.Ready, containerStatus.RestartCount)

			if containerStatus.State.Waiting != nil {
				h.T.Logf("        Waiting: %s - %s",
					containerStatus.State.Waiting.Reason, containerStatus.State.Waiting.Message)
			}
			if containerStatus.State.Terminated != nil {
				h.T.Logf("        Terminated: %s - %s (Exit Code: %d)",
					containerStatus.State.Terminated.Reason,
					containerStatus.State.Terminated.Message,
					containerStatus.State.Terminated.ExitCode)
			}
		}

		// Log pod conditions
		for _, cond := range pod.Status.Conditions {
			if cond.Status != "True" {
				h.T.Logf("      Condition %s: %s - %s", cond.Type, cond.Status, cond.Message)
			}
		}
	}
}

func (h *TestHelper) PrintResourcesForDebugging(deploymentName string) {
	h.T.Logf("=== KUBERNETES RESOURCES VALIDATION ===")

	// Print Deployment details
	if dep, err := h.Clientset.AppsV1().Deployments(h.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.T.Logf("✓ Deployment: %s", dep.Name)
		h.T.Logf("  Image: %s", dep.Spec.Template.Spec.Containers[0].Image)
		h.T.Logf("  Args: %v", dep.Spec.Template.Spec.Containers[0].Args)
		if dep.Spec.Replicas != nil {
			h.T.Logf("  Replicas: %d", *dep.Spec.Replicas)
		} else {
			h.T.Logf("  Replicas: <nil> (managed by HPA)")
		}
	}

	// Print HPA if exists
	if hpa, err := h.Clientset.AutoscalingV2().HorizontalPodAutoscalers(h.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.T.Logf("✓ HPA: %s", hpa.Name)
		h.T.Logf("  Min/Max Replicas: %d/%d", *hpa.Spec.MinReplicas, hpa.Spec.MaxReplicas)
	}

	// Print Service
	if svc, err := h.Clientset.CoreV1().Services(h.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.T.Logf("✓ Service: %s", svc.Name)
		h.T.Logf("  Type: %s, Port: %d", svc.Spec.Type, svc.Spec.Ports[0].Port)
	}

	// Print ServiceAccount
	if sa, err := h.Clientset.CoreV1().ServiceAccounts(h.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.T.Logf("✓ ServiceAccount: %s", sa.Name)
	}

	// Print PodDisruptionBudget
	if pdb, err := h.Clientset.PolicyV1().PodDisruptionBudgets(h.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.T.Logf("✓ PodDisruptionBudget: %s", pdb.Name)
		if pdb.Spec.MaxUnavailable != nil {
			h.T.Logf("  MaxUnavailable: %s", pdb.Spec.MaxUnavailable.String())
		}
	}

	// Print ClusterRole and ClusterRoleBinding
	if cr, err := h.Clientset.RbacV1().ClusterRoles().Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.T.Logf("✓ ClusterRole: %s", cr.Name)
	}
	if crb, err := h.Clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.T.Logf("✓ ClusterRoleBinding: %s", crb.Name)
	}

	h.T.Logf("=== END RESOURCES VALIDATION ===")
}
