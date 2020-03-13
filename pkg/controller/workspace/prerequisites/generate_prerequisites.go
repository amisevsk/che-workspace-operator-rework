package prerequisites

import (
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/config"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func generatePrerequisites(workspace *v1alpha1.Workspace) ([]runtime.Object, error) {
	pvcStorageQuantity, err := resource.ParseQuantity(config.PVCStorageSize)
	if err != nil {
		return nil, err
	}

	k8sObjects := []runtime.Object{
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.ControllerCfg.GetWorkspacePVCName(),
				Namespace: workspace.Namespace,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						"storage": pvcStorageQuantity,
					},
				},
				StorageClassName: config.ControllerCfg.GetPVCStorageClassName(),
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "exec",
				Namespace: workspace.Namespace,
			},
			Rules: []rbacv1.PolicyRule{
				{
					Resources: []string{"pods/exec"},
					APIGroups: []string{""},
					Verbs:     []string{"create"},
				},
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "view-workspaces",
				Namespace: workspace.Namespace,
			},
			Rules: []rbacv1.PolicyRule{
				{
					Resources: []string{"workspaces"},
					APIGroups: []string{"workspace.che.eclipse.org"},
					Verbs:     []string{"get", "list"},
				},
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.ServiceAccount + "-view",
				Namespace: workspace.Namespace,
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "ClusterRole",
				Name: "view",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind: "Group",
					Name: "system:serviceaccounts:" + workspace.Namespace,
				},
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.ServiceAccount + "-exec",
				Namespace: workspace.Namespace,
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "Role",
				Name: "exec",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind: "Group",
					Name: "system:serviceaccounts:" + workspace.Namespace,
				},
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.ServiceAccount + "-view-workspaces",
				Namespace: workspace.Namespace,
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "Role",
				Name: "view-workspaces",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind: "Group",
					Name: "system:serviceaccounts:" + workspace.Namespace,
				},
			},
		},
	}
	return k8sObjects, nil
}
