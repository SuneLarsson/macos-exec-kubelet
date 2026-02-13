package volumes_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/internal/volumes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateContainerMounts(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name                string
		container           corev1.Container
		pod                 *corev1.Pod
		serviceAccountToken string
		configMaps          map[string]*corev1.ConfigMap
		expectedMounts      []volumes.Mount
		expectError         bool
	}{
		{
			name: "HostPath volume",
			container: corev1.Container{
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "test-volume",
						MountPath: "/mnt/test",
					},
				},
			},
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "test-volume",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/tmp/hostpath",
								},
							},
						},
					},
				},
			},
			expectedMounts: []volumes.Mount{
				{
					Name:          "test-volume",
					HostPath:      "/tmp/hostpath",
					ContainerPath: "/mnt/test",
					ReadOnly:      false,
				},
			},
		},
		{
			name: "EmptyDir volume",
			container: corev1.Container{
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "emptydir-volume",
						MountPath: "/mnt/emptydir",
					},
				},
			},
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "emptydir-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
			expectedMounts: []volumes.Mount{
				{
					Name:          "emptydir-volume",
					HostPath:      filepath.Join(tempDir, "emptydir-volume"),
					ContainerPath: "/mnt/emptydir",
					ReadOnly:      false,
				},
			},
		},
		{
			name: "Projected volume with ServiceAccountToken",
			container: corev1.Container{
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "projected-volume",
						MountPath: "/mnt/projected",
					},
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "projected-volume",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									Sources: []corev1.VolumeProjection{
										{
											ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
												Path: "token",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			serviceAccountToken: "test-token",
			expectedMounts: []volumes.Mount{
				{
					Name:          "projected-volume",
					HostPath:      filepath.Join(tempDir, "projected-volume"),
					ContainerPath: "/mnt/projected",
					ReadOnly:      false,
				},
			},
		},
		{
			name: "Projected volume with ConfigMap",
			container: corev1.Container{
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "configmap-volume",
						MountPath: "/mnt/configmap",
					},
				},
			},
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "configmap-volume",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									Sources: []corev1.VolumeProjection{
										{
											ConfigMap: &corev1.ConfigMapProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-configmap",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "config-key",
														Path: "config-path",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			configMaps: map[string]*corev1.ConfigMap{
				"test-configmap": {
					Data: map[string]string{
						"config-key": "config-value",
					},
				},
			},
			expectedMounts: []volumes.Mount{
				{
					Name:          "configmap-volume",
					HostPath:      filepath.Join(tempDir, "configmap-volume"),
					ContainerPath: "/mnt/configmap",
					ReadOnly:      false,
				},
			},
		},
		{
			name: "Projected volume with DownwardAPI (namespace)",
			container: corev1.Container{
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "downwardapi-volume",
						MountPath: "/mnt/downwardapi",
					},
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "downwardapi-volume",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									Sources: []corev1.VolumeProjection{
										{
											DownwardAPI: &corev1.DownwardAPIProjection{
												Items: []corev1.DownwardAPIVolumeFile{
													{
														Path: "namespace",
														FieldRef: &corev1.ObjectFieldSelector{
															FieldPath: "metadata.namespace",
														},
														Mode: func(i int32) *int32 {
															return &i
														}(0644),
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedMounts: []volumes.Mount{
				{
					Name:          "downwardapi-volume",
					HostPath:      filepath.Join(tempDir, "downwardapi-volume"),
					ContainerPath: "/mnt/downwardapi",
					ReadOnly:      false,
				},
			},
		},
		{
			name: "Volume not found in Pod spec",
			container: corev1.Container{
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "non-existent-volume",
						MountPath: "/mnt/non-existent",
					},
				},
			},
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{},
				},
			},
			expectedMounts: []volumes.Mount{},
		},
		{
			name: "Unsupported volume type",
			container: corev1.Container{
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "unsupported-volume",
						MountPath: "/mnt/unsupported",
					},
				},
			},
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "unsupported-volume",
							VolumeSource: corev1.VolumeSource{
								// This is an unsupported volume type for this function
								Secret: &corev1.SecretVolumeSource{
									SecretName: "my-secret",
								},
							},
						},
					},
				},
			},
			expectedMounts: []volumes.Mount{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mounts, err := volumes.CreateContainerMounts(context.Background(), tempDir, tt.container, tt.pod, tt.serviceAccountToken, tt.configMaps)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedMounts, mounts)
			}
		})
	}
}
