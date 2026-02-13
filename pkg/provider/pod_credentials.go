package provider

import (
	"context"

	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// extractPodCredentials extracts the service account token and config maps required for the Pod.
func (p *MacOSExecProvider) extractPodCredentials(ctx context.Context, pod *corev1.Pod) (map[string]*corev1.ConfigMap, string, error) {
	var serviceAccountToken string
	configMaps := map[string]*corev1.ConfigMap{}

	if pod.Spec.AutomountServiceAccountToken == nil || *pod.Spec.AutomountServiceAccountToken {
		svcProj, cmProj := findProjections(pod)
		if err := p.populateConfigMaps(ctx, pod.Namespace, cmProj, configMaps); err != nil {
			return nil, "", err
		}

		if svcProj != nil {
			token, err := p.createServiceAccountToken(ctx, pod.Namespace, pod.Spec.ServiceAccountName, svcProj)
			if err != nil {
				return nil, "", err
			}
			serviceAccountToken = token
		}
	}

	return configMaps, serviceAccountToken, nil
}

// populateConfigMaps fetches and populates the config maps based on the ConfigMapProjection.
func (p *MacOSExecProvider) populateConfigMaps(ctx context.Context, namespace string, cmProj *corev1.ConfigMapProjection, configMaps map[string]*corev1.ConfigMap) error {
	if cmProj != nil {
		// use core client directly instead of lister due to better nature of caching
		configMap, err := p.k8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, cmProj.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		configMaps[cmProj.Name] = configMap
	}
	return nil
}

// createServiceAccountToken creates a token for the service account.
func (p *MacOSExecProvider) createServiceAccountToken(ctx context.Context, namespace, saName string, svcProj *corev1.ServiceAccountTokenProjection) (string, error) {
	var audiences []string
	if svcProj.Audience != "" {
		audiences = []string{svcProj.Audience}
	}
	tokenReq := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: svcProj.ExpirationSeconds,
		},
	}
	req, err := p.k8sClient.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, saName, tokenReq, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	// TODO: ideally implement token rotation
	return req.Status.Token, nil
}

// findProjections finds the service account and config map projections from the Pod's volumes.
func findProjections(pod *corev1.Pod) (*corev1.ServiceAccountTokenProjection, *corev1.ConfigMapProjection) {
	var svcProj *corev1.ServiceAccountTokenProjection
	var cmProj *corev1.ConfigMapProjection
	for _, vol := range pod.Spec.Volumes {
		if vol.Projected != nil {
			for _, source := range vol.Projected.Sources {
				if source.ServiceAccountToken != nil {
					svcProj = source.ServiceAccountToken
				}
				if source.ConfigMap != nil {
					cmProj = source.ConfigMap
				}
				if svcProj != nil && cmProj != nil {
					break
				}
			}
		}
	}
	return svcProj, cmProj
}
