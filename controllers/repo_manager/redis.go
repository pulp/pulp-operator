package repo_manager

import (
	"context"
	"os"
	"time"

	"github.com/go-logr/logr"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

func (r *RepoManagerReconciler) pulpCacheController(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, log logr.Logger) (ctrl.Result, error) {

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-API-Ready"
	funcResources := controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}

	// pulp-redis-data PVC
	// the PVC will be created only if a StorageClassName is provided
	if _, storageType := controllers.MultiStorageConfigured(pulp, "Cache"); storageType[0] == controllers.SCNameType {
		pvcName := settings.DefaultCachePVC(pulp.Name)
		pvcFound := &corev1.PersistentVolumeClaim{}
		err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: pulp.Namespace}, pvcFound)
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
	svcName := settings.CacheService(pulp.Name)
	svcFound := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: pulp.Namespace}, svcFound)
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
	deploymentName := settings.CACHE.DeploymentName(pulp.Name)
	deploymentFound := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: pulp.Namespace}, deploymentFound)
	dep := redisDeployment(pulp, funcResources)
	if err != nil && errors.IsNotFound(err) {
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

	// Ensure the deployment spec is as expected
	if requeue, err := controllers.ReconcileObject(funcResources, dep, deploymentFound, conditionType, controllers.PulpDeployment{}); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	// Update managedCache status
	pulp.Status.ManagedCacheEnabled = pulp.Spec.Cache.Enabled
	r.Status().Update(ctx, pulp)

	r.recorder.Event(pulp, corev1.EventTypeNormal, "RedisReady", "All Redis tasks ran successfully")
	return ctrl.Result{}, nil

}

// pulp-redis-data PVC
func redisDataPVC(m *repomanagerpulpprojectorgv1beta2.Pulp) *corev1.PersistentVolumeClaim {

	storageClass := &m.Spec.Cache.RedisStorageClass

	storageSize := "1Gi"
	if storageResourceQuantity := m.Spec.Cache.RedisResourceRequirements.Requests.Storage(); !storageResourceQuantity.IsZero() {
		storageSize = storageResourceQuantity.String()
	}

	// Define the new PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.DefaultCachePVC(m.Name),
			Namespace: m.Namespace,
			Labels:    labelsForCache(m),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(storageSize),
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.PersistentVolumeAccessMode("ReadWriteOnce"),
			},
			StorageClassName: storageClass,
		},
	}
	return pvc
}

// redis-svc Service
func redisSvc(m *repomanagerpulpprojectorgv1beta2.Pulp) *corev1.Service {
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: 6379}
	port := m.Spec.Cache.RedisPort
	if port == 0 {
		port = 6379
	}

	labels := labelsForCache(m)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.CacheService(m.Name),
			Namespace: m.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port:       int32(port),
				Protocol:   servicePortProto,
				TargetPort: targetPort,
				Name:       "redis-6379",
			}},
		},
	}
}

// redisDeployment returns a Redis Deployment object
func redisDeployment(m *repomanagerpulpprojectorgv1beta2.Pulp, funcResources controllers.FunctionResources) *appsv1.Deployment {

	replicas := int32(1)
	ls := labelsForCache(m)
	affinity := &corev1.Affinity{}
	if m.Spec.Cache.Affinity != nil {
		affinity = m.Spec.Cache.Affinity
	}

	// if no strategy is defined in pulp CR we are setting `strategy.Type` with the
	// default value ("RollingUpdate"), this will be helpful during the reconciliation
	// when a strategy was previously defined and eventually the field is removed
	strategy := m.Spec.Cache.Strategy
	if strategy.Type == "" {
		strategy.Type = "RollingUpdate"
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
				ClaimName: settings.DefaultCachePVC(m.Name),
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

	redisImage := os.Getenv("RELATED_IMAGE_PULP_REDIS")
	if len(m.Spec.Cache.RedisImage) > 0 {
		redisImage = m.Spec.Cache.RedisImage
	} else if redisImage == "" {
		redisImage = "docker.io/library/redis:latest"
	}

	resources := m.Spec.Cache.RedisResourceRequirements

	removeStorageDefinition(&resources)

	deploymentAnnotations := map[string]string{}
	if m.Spec.Cache.DeploymentAnnotations != nil {
		deploymentAnnotations = m.Spec.Cache.DeploymentAnnotations
	}
	// set standard annotations that cannot be overridden by users
	deploymentAnnotations["email"] = "pulp-dev@redhat.com"
	deploymentAnnotations["ignore-check.kube-linter.io/unset-cpu-requirements"] = "Temporarily disabled"
	deploymentAnnotations["ignore-check.kube-linter.io/unset-memory-requirements"] = "Temporarily disabled"
	deploymentAnnotations["ignore-check.kube-linter.io/no-node-affinity"] = "Do not check node affinity"

	runAsUser := int64(700)
	fsGroup := int64(700)

	// deployment definition
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        settings.CACHE.DeploymentName(m.Name),
			Namespace:   m.Namespace,
			Annotations: deploymentAnnotations,
			Labels:      ls,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: strategy,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Affinity:           affinity,
					NodeSelector:       nodeSelector,
					Tolerations:        toleration,
					ServiceAccountName: settings.PulpServiceAccount(m.Name),
					SecurityContext:    &corev1.PodSecurityContext{RunAsUser: &runAsUser, FSGroup: &fsGroup},
					Containers: []corev1.Container{{
						Name:            "redis",
						Image:           redisImage,
						ImagePullPolicy: corev1.PullPolicy("IfNotPresent"),
						VolumeMounts:    volumeMounts,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 6379,
							Protocol:      "TCP",
						}},
						LivenessProbe:   livenessProbe,
						ReadinessProbe:  readinessProbe,
						Resources:       resources,
						SecurityContext: controllers.SetDefaultSecurityContext(),
					}},
					Volumes: volumes,
				},
			},
		},
	}

	controllers.AddHashLabel(funcResources, dep)
	ctrl.SetControllerReference(m, dep, funcResources.Scheme)
	return dep
}

// removeStorageDefinition ensures that no storage definition is present in resourceRequirements
// we need to get rid of it because cache.redis_resource_requirements is a corev1.ResourceRequirements (which can contain storage definition)
// but storage is not a valid value for container resources
func removeStorageDefinition(resources *corev1.ResourceRequirements) {
	if resources.Requests.Storage() != nil {
		delete(resources.Requests, "storage")
	}
	if resources.Limits.Storage() != nil {
		delete(resources.Limits, "storage")
	}
}

// deprovisionCache removes Redis resources in case cache is not enabled anymore
// or in case of a new definition with an external Redis instance
func (r *RepoManagerReconciler) deprovisionCache(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, log logr.Logger) (ctrl.Result, error) {
	// redis-svc Service
	svcName := settings.CacheService(pulp.Name)
	svcFound := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: pulp.Namespace}, svcFound)
	if !errors.IsNotFound(err) {
		log.Info("Removing Redis service", "Service.Namespace", pulp.Namespace, "Service.Name", svcName)
		r.Delete(ctx, svcFound)
	}

	// redis Deployment
	deploymentName := settings.CACHE.DeploymentName(pulp.Name)
	deploymentFound := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: pulp.Namespace}, deploymentFound)
	if !errors.IsNotFound(err) {
		log.Info("Removing Redis deployment", "Deployment.Namespace", pulp.Namespace, "Deployment.Name", deploymentName)
		r.Delete(ctx, deploymentFound)
	}

	// Update managedCache status
	pulp.Status.ManagedCacheEnabled = pulp.Spec.Cache.Enabled
	r.Status().Update(ctx, pulp)

	return ctrl.Result{}, nil
}

// labelsForCache returns the labels for selecting the resources
// belonging to the given pulp CR name.
func labelsForCache(m *repomanagerpulpprojectorgv1beta2.Pulp) map[string]string {
	return settings.PulpcoreLabels(*m, "cache")
}

// managedCacheDisabled returns true if
// * there is no definition for external cache
// * the managed cache (deployed by pulp-operator) has a different definition than the status
func managedCacheDisabled(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return len(pulp.Spec.Cache.ExternalCacheSecret) == 0 && pulp.Spec.Cache.Enabled != pulp.Status.ManagedCacheEnabled
}
