package pulp

import (
	"context"
	"os"
	"time"

	"github.com/go-logr/logr"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *PulpReconciler) pulpCacheController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// pulp-redis-data PVC
	// the PVC will be created only if a StorageClassName is provided
	if _, storageType := controllers.MultiStorageConfigured(pulp, "Cache"); storageType[0] == controllers.SCNameType {
		pvcFound := &corev1.PersistentVolumeClaim{}
		err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-redis-data", Namespace: pulp.Namespace}, pvcFound)
		pvc := redisDataPVC(pulp)
		if err != nil && errors.IsNotFound(err) {
			ctrl.SetControllerReference(pulp, pvc, r.Scheme)
			log.Info("Creating a new Pulp Redis Data PVC", "PVC.Namespace", pvc.Namespace, "PVC.Name", pvc.Name)
			err = r.Create(ctx, pvc)
			if err != nil {
				log.Error(err, "Failed to create new Pulp Redis Data PVC", "PVC.Namespace", pvc.Namespace, "PVC.Name", pvc.Name)
				r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new Redis Data PVC")
				return ctrl.Result{}, err
			}
			// PVC created successfully - return and requeue
			r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", "Redis Data PVC created")
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			log.Error(err, "Failed to get Pulp Redis Data PVC")
			return ctrl.Result{}, err
		}

		// Reconcile PVC
		if !equality.Semantic.DeepDerivative(pvc.Spec, pvcFound.Spec) {
			log.Info("The Redis PVC has been modified! Reconciling ...")
			r.recorder.Event(pulp, corev1.EventTypeNormal, "Updating", "Reconciling Redis PVC")
			ctrl.SetControllerReference(pulp, pvc, r.Scheme)
			err = r.Update(ctx, pvc)
			if err != nil {
				log.Error(err, "Error trying to update the Redis PVC object ... ")
				r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to reconcile Redis PVC")
				return ctrl.Result{}, err
			}
			r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", "Redis PVC reconciled")
			return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
		}
	}

	// redis-svc Service
	svcFound := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-redis-svc", Namespace: pulp.Namespace}, svcFound)
	svc := redisSvc(pulp)
	if err != nil && errors.IsNotFound(err) {
		ctrl.SetControllerReference(pulp, svc, r.Scheme)
		log.Info("Creating a new Redis Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.Create(ctx, svc)
		if err != nil {
			log.Error(err, "Failed to create new Redis Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new Redis Service")
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", "Redis Service created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Redis Service")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if !equality.Semantic.DeepDerivative(svc.Spec, svcFound.Spec) {
		log.Info("The Redis Service has been modified! Reconciling ...")
		ctrl.SetControllerReference(pulp, svc, r.Scheme)
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updating", "Reconciling Redis Service")
		err = r.Update(ctx, svc)
		if err != nil {
			log.Error(err, "Error trying to update the Redis Service object ... ")
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to reconcile Redis Service")
			return ctrl.Result{}, err
		}
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", "Redis Service reconciled")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
	}

	// redis Deployment
	deploymentFound := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-redis", Namespace: pulp.Namespace}, deploymentFound)
	dep := redisDeployment(pulp)
	if err != nil && errors.IsNotFound(err) {
		ctrl.SetControllerReference(pulp, dep, r.Scheme)
		log.Info("Creating a new Pulp Redis Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Pulp Redis Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new Redis Deployment")
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", "Redis Deployment created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Redis Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Deployment
	if !equality.Semantic.DeepDerivative(dep.Spec, deploymentFound.Spec) {
		log.Info("The Redis Deployment has been modified! Reconciling ...")
		ctrl.SetControllerReference(pulp, dep, r.Scheme)
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updating", "Reconciling Redis Deployment")
		err = r.Update(ctx, dep)
		if err != nil {
			log.Error(err, "Error trying to update the Redis Deployment object ... ")
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to reconcile Redis Deployment")
			return ctrl.Result{}, err
		}
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", "Redis Deployment reconciled")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
	}

	r.recorder.Event(pulp, corev1.EventTypeNormal, "RedisReady", "All Redis tasks ran successfully")
	return ctrl.Result{}, nil

}

// pulp-redis-data PVC
func redisDataPVC(m *repomanagerv1alpha1.Pulp) *corev1.PersistentVolumeClaim {
	// Define the new PVC
	pvc := &corev1.PersistentVolumeClaim{
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
			StorageClassName: &m.Spec.Cache.RedisStorageClass,
		},
	}
	return pvc
}

// redis-svc Service
func redisSvc(m *repomanagerv1alpha1.Pulp) *corev1.Service {
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

// redisDeployment returns a Redis Deployment object
func redisDeployment(m *repomanagerv1alpha1.Pulp) *appsv1.Deployment {

	replicas := int32(1)

	affinity := &corev1.Affinity{}
	if m.Spec.Cache.Affinity.NodeAffinity != nil {
		affinity.NodeAffinity = m.Spec.Cache.Affinity.NodeAffinity
	}

	nodeSelector := map[string]string{}
	if m.Spec.Cache.NodeSelector != nil {
		nodeSelector = m.Spec.Cache.NodeSelector
	}

	toleration := []corev1.Toleration{}
	if m.Spec.Cache.Tolerations != nil {
		toleration = m.Spec.Cache.Tolerations
	}

	_, storageType := controllers.MultiStorageConfigured(m, "Cache")
	volumeSource := corev1.VolumeSource{}
	// if SC defined, we should use the PVC provisioned by the operator
	if storageType[0] == controllers.SCNameType {
		volumeSource = corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: m.Name + "-redis-data",
			},
		}

		// if .spec.Cache.PVC defined we should use the PVC provisioned by user
	} else if storageType[0] == controllers.PVCType {
		volumeSource = corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: m.Spec.Cache.PVC,
			},
		}

		// if there is no SC nor PVC object storage defined we will mount an emptyDir
	} else if storageType[0] == controllers.EmptyDirType {
		volumeSource = corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		}
	}

	volumes := []corev1.Volume{
		{
			Name:         m.Name + "-redis-data",
			VolumeSource: volumeSource,
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			ReadOnly:  false,
			MountPath: "/data",
			Name:      m.Name + "-redis-data",
		},
	}

	readinessProbe := m.Spec.Cache.ReadinessProbe
	if readinessProbe == nil {
		readinessProbe = &corev1.Probe{
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
	}

	livenessProbe := m.Spec.Cache.LivenessProbe
	if livenessProbe == nil {
		livenessProbe = &corev1.Probe{
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
	}

	RedisImage := os.Getenv("RELATED_IMAGE_PULP_REDIS")
	if len(m.Spec.Cache.RedisImage) > 0 {
		RedisImage = m.Spec.Cache.RedisImage
	} else if RedisImage == "" {
		RedisImage = "redis:latest"
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
					NodeSelector:       nodeSelector,
					Tolerations:        toleration,
					ServiceAccountName: m.Name,
					Containers: []corev1.Container{{
						Name:            "redis",
						Image:           RedisImage,
						ImagePullPolicy: corev1.PullPolicy("IfNotPresent"),
						VolumeMounts:    volumeMounts,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 6379,
							Protocol:      "TCP",
						}},
						LivenessProbe:  livenessProbe,
						ReadinessProbe: readinessProbe,
						Resources:      m.Spec.Cache.RedisResourceRequirements,
					}},
					Volumes: volumes,
				},
			},
		},
	}
}
