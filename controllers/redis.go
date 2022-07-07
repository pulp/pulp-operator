package controllers

import (
	"context"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *PulpReconciler) pulpCacheController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// pulp-redis-data PVC
	pvcFound := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-redis-data", Namespace: pulp.Namespace}, pvcFound)
	if err != nil && errors.IsNotFound(err) {
		pvc := redisDataPVC(pulp)
		ctrl.SetControllerReference(pulp, pvc, r.Scheme)
		log.Info("Creating a new Pulp Redis Data PVC", "PVC.Namespace", pvc.Namespace, "PVC.Name", pvc.Name)
		err = r.Create(ctx, pvc)
		if err != nil {
			log.Error(err, "Failed to create new Pulp Redis Data PVC", "PVC.Namespace", pvc.Namespace, "PVC.Name", pvc.Name)
			return ctrl.Result{}, err
		}
		// PVC created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Redis Data PVC")
		return ctrl.Result{}, err
	}

	// redis-svc Service
	svcFound := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-redis-svc", Namespace: pulp.Namespace}, svcFound)
	if err != nil && errors.IsNotFound(err) {
		svc := redisSvc(pulp)
		ctrl.SetControllerReference(pulp, svc, r.Scheme)
		log.Info("Creating a new Redis Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.Create(ctx, svc)
		if err != nil {
			log.Error(err, "Failed to create new Redis Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Redis Service")
		return ctrl.Result{}, err
	}

	// redis Deployment
	deploymentFound := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-redis", Namespace: pulp.Namespace}, deploymentFound)
	if err != nil && errors.IsNotFound(err) {
		dep := redisDeployment(pulp)
		ctrl.SetControllerReference(pulp, dep, r.Scheme)
		log.Info("Creating a new Pulp Redis Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Pulp Redis Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Redis Deployment")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil

}

// pulp-redis-data PVC
func redisDataPVC(m *repomanagerv1alpha1.Pulp) *corev1.PersistentVolumeClaim {
	// Define the new PVC
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-redis-data",
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "redis",
				"app.kubernetes.io/instance":   "redis-" + m.Name,
				"app.kubernetes.io/component":  "cache",
				"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
				"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("1Gi"),
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.PersistentVolumeAccessMode("ReadWriteOnce"),
			},
			StorageClassName: &m.Spec.RedisStorageClass,
		},
	}
}

// redis-svc Service
func redisSvc(m *repomanagerv1alpha1.Pulp) *corev1.Service {
	/*
		serviceInternalTrafficPolicyCluster := corev1.ServiceInternalTrafficPolicyType("Cluster")
		ipFamilyPolicyType := corev1.IPFamilyPolicyType("SingleStack")
		serviceAffinity := corev1.ServiceAffinity("None")
		serviceType := corev1.ServiceType("ClusterIP")
	*/
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: 6379}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-redis-svc",
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "redis",
				"app.kubernetes.io/instance":   "redis-" + m.Name,
				"app.kubernetes.io/component":  "cache",
				"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
				"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/name":       "redis",
				"app.kubernetes.io/instance":   "redis-" + m.Name,
				"app.kubernetes.io/component":  "cache",
				"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
				"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
			},
			Ports: []corev1.ServicePort{{
				Port:       6379,
				Protocol:   servicePortProto,
				TargetPort: targetPort,
				Name:       "redis-6379",
			}},
		},
	}
}

// redis Deployment

// deploymentForPulpApi returns a pulp-api Deployment object
func redisDeployment(m *repomanagerv1alpha1.Pulp) *appsv1.Deployment {

	replicas := int32(1)

	affinity := &corev1.Affinity{}
	if m.Spec.Api.Affinity.NodeAffinity != nil {
		affinity.NodeAffinity = m.Spec.Api.Affinity.NodeAffinity
	}

	volumes := []corev1.Volume{
		{
			Name: m.Name + "-redis-data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: m.Name + "-redis-data",
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			ReadOnly:  false,
			MountPath: "/data",
			Name:      m.Name + "-redis-data",
		},
	}

	readinessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/bin/sh",
					"-i",
					"-c",
					"redis-cli -h 127.0.0.1 -p 6379",
				},
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		TimeoutSeconds:      5,
		FailureThreshold:    5,
		SuccessThreshold:    1,
	}

	livenessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/bin/sh",
					"-i",
					"-c",
					"redis-cli -h 127.0.0.1 -p 6379",
				},
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		SuccessThreshold:    1,
		FailureThreshold:    5,
		TimeoutSeconds:      5,
	}

	// deployment definition
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-redis",
			Namespace: m.Namespace,
			Annotations: map[string]string{
				"email": "pulp-dev@redhat.com",
				"ignore-check.kube-linter.io/unset-cpu-requirements":    "Temporarily disabled",
				"ignore-check.kube-linter.io/unset-memory-requirements": "Temporarily disabled",
				"ignore-check.kube-linter.io/no-node-affinity":          "Do not check node affinity",
			},
			Labels: map[string]string{
				"app.kubernetes.io/name":       "redis",
				"app.kubernetes.io/instance":   "redis-" + m.Name,
				"app.kubernetes.io/component":  "cache",
				"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
				"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":       "redis",
					"app.kubernetes.io/instance":   "redis-" + m.Name,
					"app.kubernetes.io/component":  "cache",
					"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
					"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":       "redis",
						"app.kubernetes.io/instance":   "redis-" + m.Name,
						"app.kubernetes.io/component":  "cache",
						"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
						"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
					},
				},
				Spec: corev1.PodSpec{
					Affinity:           affinity,
					ServiceAccountName: "pulp-operator-go-controller-manager",
					Containers: []corev1.Container{{
						Name:            "redis",
						Image:           m.Spec.RedisImage,
						ImagePullPolicy: corev1.PullPolicy("IfNotPresent"),
						VolumeMounts:    volumeMounts,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 6379,
							Protocol:      "TCP",
						}},
						LivenessProbe:  livenessProbe,
						ReadinessProbe: readinessProbe,
						Resources:      m.Spec.RedisResourceRequirements,
					}},
					Volumes: volumes,
				},
			},
		},
	}
}
