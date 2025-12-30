package repo_manager_test

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	routev1 "github.com/openshift/api/route/v1"
	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OperatorType  = "pulp"
	PulpName      = OperatorType + "-operator"
	PulpNamespace = "default"
	StsName       = PulpName + "-database"
	ApiName       = PulpName + "-api"
	ContentName   = PulpName + "-content"
	WorkerName    = PulpName + "-worker"
	DBVolumeName  = PulpName + "-postgres"

	timeout  = time.Minute
	interval = time.Second
)

var _ = Describe("Pulp controller", Ordered, func() {
	ctx := context.Background()

	format.MaxLength = 0

	labelsSts := map[string]string{
		"app.kubernetes.io/name":       OperatorType + "-database",
		"app.kubernetes.io/instance":   OperatorType + "-database-" + PulpName,
		"app.kubernetes.io/component":  "database",
		"app.kubernetes.io/part-of":    OperatorType,
		"app.kubernetes.io/managed-by": OperatorType + "-operator",
		"app":                          "pulp-database",
		"pulp_cr":                      PulpName,
	}

	labelsStsPods := map[string]string{"custom": "database"}
	maps.Copy(labelsStsPods, labelsSts)

	labelsApi := map[string]string{
		"app.kubernetes.io/name":       OperatorType + "-api",
		"app.kubernetes.io/instance":   OperatorType + "-api-" + PulpName,
		"app.kubernetes.io/component":  "api",
		"app.kubernetes.io/part-of":    OperatorType,
		"app.kubernetes.io/managed-by": PulpName,
		"app":                          "pulp-api",
		"pulp_cr":                      PulpName,
	}

	labelsApiPods := map[string]string{"custom": "api"}
	maps.Copy(labelsApiPods, labelsApi)

	labelsContent := map[string]string{
		"app.kubernetes.io/name":       OperatorType + "-content",
		"app.kubernetes.io/instance":   OperatorType + "-content-" + PulpName,
		"app.kubernetes.io/component":  "content",
		"app.kubernetes.io/part-of":    OperatorType,
		"app.kubernetes.io/managed-by": PulpName,
		"app":                          "pulp-content",
		"pulp_cr":                      PulpName,
	}

	labelsContentPods := map[string]string{"custom": "content"}
	maps.Copy(labelsContentPods, labelsContent)

	labelsWorker := map[string]string{
		"app.kubernetes.io/name":       OperatorType + "-worker",
		"app.kubernetes.io/instance":   OperatorType + "-worker-" + PulpName,
		"app.kubernetes.io/component":  "worker",
		"app.kubernetes.io/part-of":    OperatorType,
		"app.kubernetes.io/managed-by": PulpName,
		"app":                          "pulp-worker",
		"pulp_cr":                      PulpName,
	}

	labelsWorkerPods := map[string]string{"custom": "worker"}
	maps.Copy(labelsWorkerPods, labelsWorker)

	replicasSts := int32(1)
	replicasApi := int32(1)
	replicasContent := int32(1)
	replicasWorker := int32(1)

	envVarsSts := []corev1.EnvVar{
		{
			Name: "POSTGRESQL_DATABASE",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: PulpName + "-postgres-configuration",
					},
					Key: "database",
				},
			},
		},
		{
			Name: "POSTGRESQL_USER",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: PulpName + "-postgres-configuration",
					},
					Key: "username",
				},
			},
		},
		{
			Name: "POSTGRESQL_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: PulpName + "-postgres-configuration",
					},
					Key: "password",
				},
			},
		},
		{
			Name: "POSTGRES_DB",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: PulpName + "-postgres-configuration",
					},
					Key: "database",
				},
			},
		},
		{
			Name: "POSTGRES_USER",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: PulpName + "-postgres-configuration",
					},
					Key: "username",
				},
			},
		},
		{
			Name: "POSTGRES_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: PulpName + "-postgres-configuration",
					},
					Key: "password",
				},
			},
		},
		{Name: "PGDATA", Value: "/var/lib/postgresql/data/pgdata"},
		{Name: "POSTGRES_INITDB_ARGS", Value: "--auth-host=scram-sha-256"},
		{Name: "POSTGRES_HOST_AUTH_METHOD", Value: "scram-sha-256"},
	}

	customEnvVar := corev1.EnvVar{Name: "TEST", Value: "test ok"}
	envVarsInitContainer := []corev1.EnvVar{
		{Name: "POSTGRES_SERVICE_HOST", Value: PulpName + "-database-svc"},
		{Name: "POSTGRES_SERVICE_PORT", Value: "5432"},
		customEnvVar,
	}

	envVarsApi := []corev1.EnvVar{
		customEnvVar,
		{Name: "PULP_GUNICORN_TIMEOUT", Value: strconv.Itoa(90)},
		{Name: "PULP_API_WORKERS", Value: strconv.Itoa(2)},
		{Name: "PULP_GUNICORN_ACCESS_LOGFORMAT", Value: `pulp [%({correlation-id}o)s]: %(h)s %(l)s %(u)s %(t)s "%(r)s" %(s)s %(b)s "%(f)s" "%(a)s"`},
		{Name: "POSTGRES_SERVICE_HOST", Value: PulpName + "-database-svc"},
		{Name: "POSTGRES_SERVICE_PORT", Value: "5432"},
		{Name: "REDIS_SERVICE_HOST", Value: PulpName + "-redis-svc." + PulpNamespace},
		{Name: "REDIS_SERVICE_PORT", Value: strconv.Itoa(6379)},
	}

	envVarsContent := []corev1.EnvVar{
		customEnvVar,
		{Name: "PULP_GUNICORN_TIMEOUT", Value: strconv.Itoa(90)},
		{Name: "PULP_CONTENT_WORKERS", Value: strconv.Itoa(2)},
		{Name: "PULP_GUNICORN_ACCESS_LOGFORMAT", Value: `%a %t "%r" %s %b "%{Referer}i" "%{User-Agent}i"`},
		{Name: "POSTGRES_SERVICE_HOST", Value: PulpName + "-database-svc"},
		{Name: "POSTGRES_SERVICE_PORT", Value: "5432"},
		{Name: "REDIS_SERVICE_HOST", Value: PulpName + "-redis-svc." + PulpNamespace},
		{Name: "REDIS_SERVICE_PORT", Value: strconv.Itoa(6379)},
	}

	envVarsWorker := []corev1.EnvVar{
		customEnvVar,
		{Name: "POSTGRES_SERVICE_HOST", Value: PulpName + "-database-svc"},
		{Name: "POSTGRES_SERVICE_PORT", Value: "5432"},
		{Name: "REDIS_SERVICE_HOST", Value: PulpName + "-redis-svc." + PulpNamespace},
		{Name: "REDIS_SERVICE_PORT", Value: strconv.Itoa(6379)},
	}

	volumeMountsSts := []corev1.VolumeMount{
		{
			Name:      DBVolumeName,
			MountPath: "/var/lib/postgresql/data",
			SubPath:   "data",
		},
	}
	volumeMountsInitContainer := []corev1.VolumeMount{
		{
			Name:      PulpName + "-server",
			MountPath: "/etc/pulp/settings.py",
			SubPath:   "settings.py",
			ReadOnly:  true,
		},
		{
			Name:      PulpName + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
		{
			Name:      "file-storage",
			MountPath: "/var/lib/pulp",
		},
	}

	volumeMountsApi := []corev1.VolumeMount{
		{
			Name:      PulpName + "-server",
			MountPath: "/etc/pulp/settings.py",
			SubPath:   "settings.py",
			ReadOnly:  true,
		},
		{
			Name:      PulpName + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
		{
			Name:      PulpName + "-admin-password",
			MountPath: "/etc/pulp/pulp-admin-password",
			SubPath:   "admin-password",
			ReadOnly:  true,
		},
		{
			Name:      "file-storage",
			MountPath: "/var/lib/pulp",
		},
		{
			Name:      PulpName + "-container-auth-certs",
			MountPath: "/etc/pulp/keys/container_auth_private_key.pem",
			SubPath:   "container_auth_private_key.pem",
			ReadOnly:  true,
		},
		{
			Name:      PulpName + "-container-auth-certs",
			MountPath: "/etc/pulp/keys/container_auth_public_key.pem",
			SubPath:   "container_auth_public_key.pem",
			ReadOnly:  true,
		},
	}

	volumeMountsContent := []corev1.VolumeMount{
		{
			Name:      PulpName + "-server",
			MountPath: "/etc/pulp/settings.py",
			SubPath:   "settings.py",
			ReadOnly:  true,
		},
		{
			Name:      PulpName + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
		{
			Name:      "file-storage",
			MountPath: "/var/lib/pulp",
		},
	}

	volumeMountsWorker := []corev1.VolumeMount{
		{
			Name:      PulpName + "-server",
			MountPath: "/etc/pulp/settings.py",
			SubPath:   "settings.py",
			ReadOnly:  true,
		},
		{
			Name:      PulpName + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
		{
			Name:      PulpName + "-ansible-tmp",
			MountPath: "/.ansible/tmp",
		},
		{
			Name:      "file-storage",
			MountPath: "/var/lib/pulp",
		},
	}

	volumesApi := []corev1.Volume{
		{
			Name: PulpName + "-server",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: PulpName + "-server",
					Items: []corev1.KeyToPath{{
						Key:  "settings.py",
						Path: "settings.py",
					}},
				},
			},
		},
		{
			Name: PulpName + "-db-fields-encryption",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: PulpName + "-db-fields-encryption",
					Items: []corev1.KeyToPath{{
						Key:  "database_fields.symmetric.key",
						Path: "database_fields.symmetric.key",
					}},
				},
			},
		},
		{
			Name: PulpName + "-admin-password",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: PulpName + "-admin-password",
					Items: []corev1.KeyToPath{{
						Path: "admin-password",
						Key:  "password",
					}},
				},
			},
		},
		{
			Name: "file-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PulpName + "-file-storage",
				},
			},
		},
		{
			Name: PulpName + "-container-auth-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: PulpName + "-container-auth",
					Items: []corev1.KeyToPath{
						{
							Key:  "container_auth_public_key.pem",
							Path: "container_auth_public_key.pem",
						},
						{
							Key:  "container_auth_private_key.pem",
							Path: "container_auth_private_key.pem",
						},
					},
				},
			},
		},
	}

	volumesContent := []corev1.Volume{
		{
			Name: PulpName + "-server",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: PulpName + "-server",
					Items: []corev1.KeyToPath{{
						Key:  "settings.py",
						Path: "settings.py",
					}},
				},
			},
		},
		{
			Name: PulpName + "-db-fields-encryption",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: PulpName + "-db-fields-encryption",
					Items: []corev1.KeyToPath{{
						Key:  "database_fields.symmetric.key",
						Path: "database_fields.symmetric.key",
					}},
				},
			},
		},
		{
			Name: "file-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PulpName + "-file-storage",
				},
			},
		},
	}

	volumesWorker := []corev1.Volume{
		{
			Name: PulpName + "-server",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: PulpName + "-server",
					Items: []corev1.KeyToPath{{
						Key:  "settings.py",
						Path: "settings.py",
					}},
				},
			},
		},
		{
			Name: PulpName + "-db-fields-encryption",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: PulpName + "-db-fields-encryption",
					Items: []corev1.KeyToPath{{
						Key:  "database_fields.symmetric.key",
						Path: "database_fields.symmetric.key",
					}},
				},
			},
		},
		{
			Name: PulpName + "-ansible-tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "file-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PulpName + "-file-storage",
				},
			},
		},
	}

	livenessProbeSts := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/bin/sh",
					"-i",
					"-c",
					"pg_isready -U " + OperatorType + " -h 127.0.0.1 -p 5432",
				},
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		FailureThreshold:    6,
		SuccessThreshold:    1,
	}

	readinessProbeSts := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/bin/sh",
					"-i",
					"-c",
					"pg_isready -U " + OperatorType + " -h 127.0.0.1 -p 5432",
				},
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		FailureThreshold:    6,
		SuccessThreshold:    1,
	}

	readinessProbeApi := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/usr/bin/readyz.py",
					"/pulp/api/v3/status/",
				},
			},
		},
		FailureThreshold:    1,
		InitialDelaySeconds: 3,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		TimeoutSeconds:      10,
	}

	livenessProbeApi := &corev1.Probe{
		FailureThreshold: 10,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/pulp/api/v3/status/",
				Port: intstr.IntOrString{
					IntVal: 24817,
				},
				Scheme: corev1.URIScheme("HTTP"),
			},
		},
		InitialDelaySeconds: 3,
		PeriodSeconds:       20,
		SuccessThreshold:    1,
		TimeoutSeconds:      10,
	}

	volumeClaimTemplate := []corev1.PersistentVolumeClaim{{
		ObjectMeta: metav1.ObjectMeta{
			Name: DBVolumeName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("5Gi"),
				},
			},
		},
	}}

	// this is the expected database statefulset that should be
	// provisioned by pulp controller
	expectedSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      StsName,
			Namespace: PulpNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       OperatorType + "-database",
				"app.kubernetes.io/instance":   OperatorType + "-database-" + PulpName,
				"app.kubernetes.io/component":  "database",
				"app.kubernetes.io/part-of":    OperatorType,
				"app.kubernetes.io/managed-by": OperatorType + "-operator",
				"app":                          "pulp-database",
				"pulp_cr":                      PulpName,
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicasSts,
			Selector: &metav1.LabelSelector{
				MatchLabels: labelsSts,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labelsStsPods,
				},
				Spec: corev1.PodSpec{
					Affinity:           &corev1.Affinity{},
					ServiceAccountName: PulpName,
					Containers: []corev1.Container{{
						Image: "docker.io/library/postgres:13",
						Name:  "postgres",
						Env:   envVarsSts,
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(5432),
							Name:          "postgres",
							Protocol:      corev1.ProtocolTCP,
						}},
						LivenessProbe:  livenessProbeSts,
						ReadinessProbe: readinessProbeSts,
						VolumeMounts:   volumeMountsSts,
						Resources:      corev1.ResourceRequirements{},
					}},
					RestartPolicy: corev1.RestartPolicyAlways,
				},
			},
			VolumeClaimTemplates: volumeClaimTemplate,
		},
	}

	apiInitContainers := []corev1.Container{
		{
			Name:    "init-container",
			Image:   "quay.io/pulp/pulp-minimal:latest",
			Env:     envVarsInitContainer,
			Command: []string{"/bin/sh"},
			Args: []string{
				"-c",
				`mkdir -p /var/lib/pulp/{media,assets,tmp}
/usr/bin/wait_on_postgres.py
/usr/bin/wait_on_database_migrations.sh`,
			},
			VolumeMounts: volumeMountsInitContainer,
		},
	}

	commonInitContainers := []corev1.Container{
		{
			Name:    "init-container",
			Image:   "quay.io/pulp/pulp-minimal:latest",
			Env:     envVarsInitContainer,
			Command: []string{"/bin/sh"},
			Args: []string{
				"-c",
				`/usr/bin/wait_on_postgres.py
/usr/bin/wait_on_database_migrations.sh`,
			},
			VolumeMounts: volumeMountsInitContainer,
		},
	}

	apiContainers := []corev1.Container{{
		Name:    "api",
		Image:   "quay.io/pulp/pulp-minimal:latest",
		Command: []string{"/bin/sh"},
		Args: []string{
			"-c",
			`if which pulpcore-api
then
  PULP_API_ENTRYPOINT=("pulpcore-api")
else
  PULP_API_ENTRYPOINT=("gunicorn" "pulpcore.app.wsgi:application" "--name" "pulp-api")
fi
exec "${PULP_API_ENTRYPOINT[@]}" \
--bind "[::]:24817" \
--timeout "${PULP_GUNICORN_TIMEOUT}" \
--workers "${PULP_API_WORKERS}" \
--access-logformat "${PULP_GUNICORN_ACCESS_LOGFORMAT}" \
--access-logfile -`,
		},
		Env: envVarsApi,
		Ports: []corev1.ContainerPort{{
			ContainerPort: 24817,
			Protocol:      "TCP",
		}},
		LivenessProbe:  livenessProbeApi,
		ReadinessProbe: readinessProbeApi,
		VolumeMounts:   volumeMountsApi,
	}}

	// this is the expected api deployment
	expectedApiDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApiName,
			Namespace: PulpNamespace,
			Annotations: map[string]string{
				"email": "pulp-dev@redhat.com",
				"ignore-check.kube-linter.io/no-node-affinity": "Do not check node affinity",
			},
			Labels: map[string]string{
				"app.kubernetes.io/name":       OperatorType + "-api",
				"app.kubernetes.io/instance":   OperatorType + "-api-" + PulpName,
				"app.kubernetes.io/component":  "api",
				"app.kubernetes.io/part-of":    OperatorType,
				"app.kubernetes.io/managed-by": PulpName,
				"app":                          "pulp-api",
				"pulp_cr":                      PulpName,
				"owner":                        "pulp-dev",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicasApi,
			Selector: &metav1.LabelSelector{
				MatchLabels: labelsApi,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labelsApiPods,
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": "api",
					},
				},
				Spec: corev1.PodSpec{
					Affinity:           &corev1.Affinity{},
					ServiceAccountName: PulpName,
					Volumes:            volumesApi,
					InitContainers:     apiInitContainers,
					Containers:         apiContainers,
				},
			},
		},
	}

	// this is the expected content deployment
	expectedContentDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ContentName,
			Namespace: PulpNamespace,
			Annotations: map[string]string{
				"email": "pulp-dev@redhat.com",
				"ignore-check.kube-linter.io/no-node-affinity": "Do not check node affinity",
			},
			Labels: map[string]string{
				"app.kubernetes.io/name":       OperatorType + "-content",
				"app.kubernetes.io/instance":   OperatorType + "-content-" + PulpName,
				"app.kubernetes.io/component":  "content",
				"app.kubernetes.io/part-of":    OperatorType,
				"app.kubernetes.io/managed-by": PulpName,
				"app":                          "pulp-content",
				"pulp_cr":                      PulpName,
				"owner":                        "pulp-dev",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicasContent,
			Selector: &metav1.LabelSelector{
				MatchLabels: labelsContent,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labelsContentPods,
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": "content",
					},
				},
				Spec: corev1.PodSpec{
					Affinity:                  &corev1.Affinity{},
					SecurityContext:           &corev1.PodSecurityContext{},
					NodeSelector:              map[string]string{},
					Tolerations:               []corev1.Toleration{},
					Volumes:                   volumesContent,
					ServiceAccountName:        PulpName,
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
					InitContainers:            commonInitContainers,
					Containers: []corev1.Container{{
						Name:            "content",
						Image:           "quay.io/pulp/pulp-minimal:latest",
						ImagePullPolicy: corev1.PullPolicy("Always"),
						Command:         []string{"/bin/sh"},
						Args: []string{
							"-c",
							`if which pulpcore-content
then
  PULP_CONTENT_ENTRYPOINT=("pulpcore-content")
else
  PULP_CONTENT_ENTRYPOINT=("gunicorn" "pulpcore.content:server" "--worker-class" "aiohttp.GunicornWebWorker" "--name" "pulp-content")
fi
exec "${PULP_CONTENT_ENTRYPOINT[@]}" \
--bind "[::]:24816" \
--timeout "${PULP_GUNICORN_TIMEOUT}" \
--workers "${PULP_CONTENT_WORKERS}" \
--access-logformat "${PULP_GUNICORN_ACCESS_LOGFORMAT}" \
--access-logfile -
`,
						},
						Resources: corev1.ResourceRequirements{},
						Env:       envVarsContent,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 24816,
							Protocol:      "TCP",
						}},
						// LivenessProbe:  livenessProbe,
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"/usr/bin/readyz.py",
										"/pulp/content/",
									},
								},
							},
							FailureThreshold:    1,
							InitialDelaySeconds: 3,
							PeriodSeconds:       10,
							SuccessThreshold:    1,
							TimeoutSeconds:      10,
						},
						VolumeMounts: volumeMountsContent,
					}},
				},
			},
		},
	}

	// this is the expected worker deployment
	expectedWorkerDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      WorkerName,
			Namespace: PulpNamespace,
			Annotations: map[string]string{
				"email": "pulp-dev@redhat.com",
				"ignore-check.kube-linter.io/no-node-affinity": "Do not check node affinity",
			},
			Labels: map[string]string{
				"app.kubernetes.io/name":       OperatorType + "-worker",
				"app.kubernetes.io/instance":   OperatorType + "-worker-" + PulpName,
				"app.kubernetes.io/component":  "worker",
				"app.kubernetes.io/part-of":    OperatorType,
				"app.kubernetes.io/managed-by": PulpName,
				"owner":                        "pulp-dev",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicasWorker,
			Selector: &metav1.LabelSelector{
				MatchLabels: labelsWorker,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labelsWorkerPods,
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": "worker",
					},
				},
				Spec: corev1.PodSpec{
					Affinity:                  &corev1.Affinity{},
					SecurityContext:           &corev1.PodSecurityContext{},
					NodeSelector:              map[string]string{},
					Tolerations:               []corev1.Toleration{},
					Volumes:                   volumesWorker,
					ServiceAccountName:        PulpName,
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
					InitContainers:            commonInitContainers,
					Containers: []corev1.Container{{
						Name:            "worker",
						Image:           "quay.io/pulp/pulp-minimal:latest",
						ImagePullPolicy: corev1.PullPolicy("Always"),
						Command:         []string{"/usr/bin/pulp-worker"},
						Env:             envVarsWorker,
						// LivenessProbe:  livenessProbe,
						// ReadinessProbe: readinessProbe,
						VolumeMounts: volumeMountsWorker,
						Resources:    corev1.ResourceRequirements{},
					}},
				},
			},
		},
	}

	createdPulp := &pulpv1.Pulp{}
	createdSts := &appsv1.StatefulSet{}
	createdApiDeployment := &appsv1.Deployment{}
	createdContentDeployment := &appsv1.Deployment{}
	createdWorkerDeployment := &appsv1.Deployment{}

	// instantiate a pulp CR
	BeforeAll(func() {

		postgresStorageClass := "standard"

		// [TODO] Instead of using this hardcoded pulp CR we should
		// use the samples from config/samples/ folder during each
		// pipeline workflow execution
		// this is the example pulp CR
		pulp := &pulpv1.Pulp{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PulpName,
				Namespace: PulpNamespace,
			},
			Spec: pulpv1.PulpSpec{
				Cache: pulpv1.Cache{
					Enabled:           true,
					RedisStorageClass: "standard",
					PodLabels: map[string]string{
						"custom": "cache",
					},
				},
				ImageVersion:    "latest",
				ImageWebVersion: "latest",
				Api: pulpv1.Api{
					Replicas: 1,
					EnvVars:  []corev1.EnvVar{customEnvVar},
					PodLabels: map[string]string{
						"custom": "api",
					},
				},
				Content: pulpv1.Content{
					Replicas: 1,
					EnvVars:  []corev1.EnvVar{customEnvVar},
					PodLabels: map[string]string{
						"custom": "content",
					},
				},
				Worker: pulpv1.Worker{
					Replicas: 1,
					EnvVars:  []corev1.EnvVar{customEnvVar},
					PodLabels: map[string]string{
						"custom": "worker",
					},
				},
				Web: pulpv1.Web{
					Replicas: 1,
					PodLabels: map[string]string{
						"custom": "web",
					},
				},
				Database: pulpv1.Database{
					PostgresStorageClass:        &postgresStorageClass,
					PostgresStorageRequirements: "5Gi",
					PodLabels: map[string]string{
						"custom": "database",
					},
				},
				FileStorageAccessMode: "ReadWriteOnce",
				FileStorageSize:       "2Gi",
				FileStorageClass:      "standard",
				IngressType:           "nodeport",
			},
		}

		// test should fail if Pulp CR is not created
		By("Checking Pulp CR instance creation")
		Expect(k8sClient.Create(ctx, pulp)).Should(Succeed())

		// test should fail if Pulp CR is not found
		By("Checking Pulp CR being present")
		objectGet(ctx, createdPulp, PulpName)

		// make sure that there is no tasks running before proceeding
		//waitPulpOperatorFinish(ctx, createdPulp)
	})

	Context("When creating a Database statefulset", func() {
		It("Should follow the spec from pulp CR", func() {

			// test should fail if sts is not found
			By("Checking sts being found")
			objectGet(ctx, createdSts, StsName)

			// using DeepDerivative to ignore comparison of unset fields from "expectedSts"
			// that are present on "predicate"
			var isEqual = func(predicate interface{}) bool {
				return equality.Semantic.DeepDerivative(expectedSts.Spec.Template, predicate)
			}

			//waitPulpOperatorFinish(ctx, createdPulp)

			// test should fail if sts is not with the desired spec
			By("Checking sts expected Name")
			Expect(createdSts.Name).Should(Equal(expectedSts.Name))
			By("Checking sts expected Labels")
			Expect(createdSts.Labels).Should(Equal(expectedSts.Labels))
			By("Checking sts expected Replicas")
			Expect(createdSts.Spec.Replicas).Should(Equal(expectedSts.Spec.Replicas))
			By("Checking sts expected Selector")
			Expect(createdSts.Spec.Selector).Should(Equal(expectedSts.Spec.Selector))
			By("Checking sts expected Template")
			Expect(createdSts.Spec.Template).Should(Satisfy(isEqual))
		})
	})

	Context("When updating Database statefulset", func() {
		It("Should be reconciled with what is defined in pulp CR", func() {
			By("Modifying the number of replicas")

			// make sure that there is no tasks running before proceeding
			//waitPulpOperatorFinish(ctx, createdPulp)
			replicas := int32(3)
			createdSts.Spec.Replicas = &replicas
			objectUpdate(ctx, createdSts)

			// we expect that pulp controller rollback createdSts.spec.replicas to 1
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: StsName, Namespace: PulpNamespace}, createdSts)
				return *createdSts.Spec.Replicas == *expectedSts.Spec.Replicas
			}, timeout, interval).Should(BeTrue())

			By("Modifying the container image name")
			newName := "mysql:latest"
			createdSts.Spec.Template.Spec.Containers[0].Image = newName
			objectUpdate(ctx, createdSts)

			// we expect that pulp controller rollback the container image
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: StsName, Namespace: PulpNamespace}, createdSts)
				return createdSts.Spec.Template.Spec.Containers[0].Image == expectedSts.Spec.Template.Spec.Containers[0].Image
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When updating a database definition in pulp CR", func() {
		It("Should reconcile the database sts", func() {
			By("Modifying database image")

			// make sure that there is no tasks running before proceeding
			objectGet(ctx, createdSts, StsName)
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: PulpName, Namespace: PulpNamespace}, createdPulp)
				createdPulp.Spec.Database.PostgresImage = "postgres:12"
				if err := k8sClient.Update(ctx, createdPulp); err != nil {
					fmt.Println("Error trying to update object: ", err)
					return false
				}
				return createdPulp.Spec.Database.PostgresImage == "postgres:12"
			}, timeout, interval).Should(BeTrue())

			// we expect that pulp controller update sts with the new image defined in pulp CR
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: StsName, Namespace: PulpNamespace}, createdSts)
				return createdSts.Spec.Template.Spec.Containers[0].Image == "postgres:12"
			}, timeout, interval).Should(BeTrue())

		})
	})

	Context("When creating API deployment", func() {
		It("Should follow the spec from pulp CR", func() {
			By("Checking api deployment being found")
			objectGet(ctx, createdApiDeployment, ApiName)

			var isEqual = func(predicate interface{}) bool {
				return equality.Semantic.DeepDerivative(expectedApiDeployment.Spec.Template, predicate)
			}

			By("Checking api deployment expected Template")
			Expect(createdApiDeployment.Spec.Template).Should(Satisfy(isEqual))
		})
	})

	Context("When updating an API definition in pulp CR", func() {
		It("Should reconcile the api deployment", func() {
			By("Modifying the base image")

			// make sure that there is no tasks running before proceeding
			//waitPulpOperatorFinish(ctx, createdPulp)
			objectGet(ctx, createdPulp, PulpName)
			createdPulp.Spec.Image = "quay.io/pulp/pulp2"
			createdPulp.Spec.ImageVersion = "stable"
			createdPulp.Spec.ImageWebVersion = "stable"
			objectUpdate(ctx, createdPulp)

			// we expect that pulp controller update sts with the new image defined in pulp CR
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: ApiName, Namespace: PulpNamespace}, createdApiDeployment)
				return createdApiDeployment.Spec.Template.Spec.Containers[0].Image == "quay.io/pulp/pulp2:stable"
			}, timeout, interval).Should(BeTrue())

			// rollback the config to not impact other tests
			objectGet(ctx, createdPulp, PulpName)
			createdPulp.Spec.Image = "quay.io/pulp/pulp-minimal"
			createdPulp.Spec.ImageVersion = "latest"
			createdPulp.Spec.ImageWebVersion = "latest"
			objectUpdate(ctx, createdPulp)

		})
	})

	Context("When modifying the deployment api", func() {
		It("Should reconcile according to pulp CR spec", func() {
			By("Restoring the config")

			// not sure why, but for this specific scenario i saw this error once:
			//   "the object has been modified; please apply your changes to the latest version and try again"
			// adding this retry to get the newer version of api deployment just in case
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: ApiName, Namespace: PulpNamespace}, createdApiDeployment)
				replicasApi = int32(5)
				createdApiDeployment.Spec.Replicas = &replicasApi
				if err := k8sClient.Update(ctx, createdApiDeployment); err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: ApiName, Namespace: PulpNamespace}, createdApiDeployment)
				return *createdApiDeployment.Spec.Replicas == int32(1)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When creating Content deployment", func() {
		It("Should follow the spec from pulp CR", func() {
			By("Checking content deployment being found")
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: ContentName, Namespace: PulpNamespace}, createdContentDeployment)
				return equality.Semantic.DeepDerivative(expectedContentDeployment.Spec.Template, createdContentDeployment.Spec.Template)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When updating a Content definition in pulp CR", func() {
		It("Should reconcile the content deployment", func() {
			By("Modifying the base image")

			// make sure that there is no tasks running before proceeding
			//waitPulpOperatorFinish(ctx, createdPulp)
			objectGet(ctx, createdPulp, PulpName)
			createdPulp.Spec.Image = "quay.io/pulp/pulp2"
			createdPulp.Spec.ImageVersion = "stable"
			createdPulp.Spec.ImageWebVersion = "stable"
			// before trying to update an object we are doing another get to try to workaround
			// the issue: "the object has been modified; please apply your changes to the latest version and try again"
			objectUpdate(ctx, createdPulp)

			// we expect that pulp controller update sts with the new image defined in pulp CR
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: ContentName, Namespace: PulpNamespace}, createdContentDeployment)
				return createdContentDeployment.Spec.Template.Spec.Containers[0].Image == "quay.io/pulp/pulp2:stable"
			}, timeout, interval).Should(BeTrue())

			// rollback the config to not impact other tests
			//waitPulpOperatorFinish(ctx, createdPulp)
			objectGet(ctx, createdPulp, PulpName)
			createdPulp.Spec.Image = "quay.io/pulp/pulp-minimal"
			createdPulp.Spec.ImageVersion = "latest"
			createdPulp.Spec.ImageWebVersion = "latest"
			objectUpdate(ctx, createdPulp)
		})
	})

	Context("When modifying the deployment content", func() {
		It("Should reconcile according to pulp CR spec", func() {
			By("Restoring the config")
			// make sure that there is no tasks running before proceeding
			//waitPulpOperatorFinish(ctx, createdPulp)

			// get the current deployment content spec
			objectGet(ctx, createdContentDeployment, ContentName)
			replicasContent = int32(5)
			createdContentDeployment.Spec.Replicas = &replicasContent
			objectUpdate(ctx, createdContentDeployment)

			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: ContentName, Namespace: PulpNamespace}, createdContentDeployment)
				return *createdContentDeployment.Spec.Replicas == int32(1)
			}, timeout, interval).Should(BeTrue())

		})
	})

	Context("When creating Worker deployment", func() {
		It("Should follow the spec from pulp CR", func() {
			By("Checking worker deployment being found")
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: WorkerName, Namespace: PulpNamespace}, createdWorkerDeployment)
				return equality.Semantic.DeepDerivative(expectedWorkerDeployment.Spec.Template, createdWorkerDeployment.Spec.Template)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When updating a Worker definition in pulp CR", func() {
		It("Should reconcile the worker deployment", func() {
			By("Modifying the base image")
			// make sure that there is no tasks running before proceeding
			//waitPulpOperatorFinish(ctx, createdPulp)
			objectGet(ctx, createdPulp, PulpName)
			createdPulp.Spec.Image = "quay.io/pulp/pulp2"
			createdPulp.Spec.ImageVersion = "stable"
			createdPulp.Spec.ImageWebVersion = "stable"
			objectUpdate(ctx, createdPulp)

			// we expect that pulp controller update sts with the new image defined in pulp CR
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: WorkerName, Namespace: PulpNamespace}, createdWorkerDeployment)
				return createdWorkerDeployment.Spec.Template.Spec.Containers[0].Image == "quay.io/pulp/pulp2:stable"
			}, timeout, interval).Should(BeTrue())

			// rollback the config to not impact other tests
			//waitPulpOperatorFinish(ctx, createdPulp)
			objectGet(ctx, createdPulp, PulpName)
			createdPulp.Spec.Image = "quay.io/pulp/pulp-minimal"
			createdPulp.Spec.ImageVersion = "latest"
			createdPulp.Spec.ImageWebVersion = "latest"
			objectUpdate(ctx, createdPulp)
		})
	})

	Context("When modifying the deployment worker", func() {
		It("Should reconcile according to pulp CR spec", func() {
			By("Restoring the config")
			// make sure that there is no tasks running before proceeding
			//waitPulpOperatorFinish(ctx, createdPulp)

			// get the current deployment worker spec
			objectGet(ctx, createdWorkerDeployment, WorkerName)
			replicasWorker = int32(5)
			createdWorkerDeployment.Spec.Replicas = &replicasWorker
			objectUpdate(ctx, createdWorkerDeployment)

			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{Name: WorkerName, Namespace: PulpNamespace}, createdWorkerDeployment)
				return *createdWorkerDeployment.Spec.Replicas == int32(1)
			}, timeout, interval).Should(BeTrue())
		})
	})

	// [WIP] This spec should NOT be used. It is not working because we could
	// not find a good solutions for issues:
	// - route crd not bootstrapped (https://github.com/kubernetes-sigs/controller-runtime/issues/1191)
	// - could not understand why the podList from route.go is always empty during the tests
	// - what happens with the exec command to run route_paths.py during the tests?
	Context("When ingress_type is defined as route", func() {
		It("Should not deploy pulp-web resources and still expose services", func() {

			if strings.ToLower(createdPulp.Spec.IngressType) != "route" {
				Skip("IngressType != route")
			}

			// make sure that there is no tasks running before proceeding
			//waitPulpOperatorFinish(ctx, createdPulp)

			By("Creating the default root route path")

			routeName := createdPulp.Name
			expectedRoutes := make(map[string]interface{})
			expectedRoutes[routeName] = struct {
				Path, TargetPort, ServiceName string
			}{"/", "api-24817", createdPulp.Name + "-api-svc"}

			route := &routev1.Route{}
			k8sClient.Get(ctx, types.NamespacedName{Name: routeName, Namespace: PulpNamespace}, route)
			Expect(route.Spec.Host).Should(Equal(createdPulp.Spec.RouteHost))
			Expect(route.Spec.Path).Should(Equal(expectedRoutes[routeName].(struct{ Path, TargetPort, ServiceName string }).Path))
			Expect(route.Spec.Port.TargetPort).Should(Equal(expectedRoutes[routeName].(struct{ Path, TargetPort, ServiceName string }).TargetPort))
			Expect(route.Spec.To.Name).Should(Equal(expectedRoutes[routeName].(struct{ Path, TargetPort, ServiceName string }).ServiceName))

			By("Creating the default content route path")
			routeName = createdPulp.Name + "-content"
			expectedRoutes[routeName] = struct {
				Path, TargetPort, ServiceName string
			}{"/pulp/content/", "api-24816", createdPulp.Name + "-content-svc"}

			k8sClient.Get(ctx, types.NamespacedName{Name: routeName, Namespace: PulpNamespace}, route)
			Expect(route.Spec.Host).Should(Equal(createdPulp.Spec.RouteHost))
			Expect(route.Spec.Path).Should(Equal(expectedRoutes[routeName].(struct{ Path, TargetPort, ServiceName string }).Path))
			Expect(route.Spec.Port.TargetPort).Should(Equal(expectedRoutes[routeName].(struct{ Path, TargetPort, ServiceName string }).TargetPort))
			Expect(route.Spec.To.Name).Should(Equal(expectedRoutes[routeName].(struct{ Path, TargetPort, ServiceName string }).ServiceName))

			By("Creating the default api-v3 route path")
			routeName = createdPulp.Name + "-api-v3"
			expectedRoutes[routeName] = struct {
				Path, TargetPort, ServiceName string
			}{"/pulp/api/v3", "api-24817", createdPulp.Name + "-api-svc"}

			k8sClient.Get(ctx, types.NamespacedName{Name: routeName, Namespace: PulpNamespace}, route)
			Expect(route.Spec.Host).Should(Equal(createdPulp.Spec.RouteHost))
			Expect(route.Spec.Path).Should(Equal(expectedRoutes[routeName].(struct{ Path, TargetPort, ServiceName string }).Path))
			Expect(route.Spec.Port.TargetPort).Should(Equal(expectedRoutes[routeName].(struct{ Path, TargetPort, ServiceName string }).TargetPort))
			Expect(route.Spec.To.Name).Should(Equal(expectedRoutes[routeName].(struct{ Path, TargetPort, ServiceName string }).ServiceName))

			By("Creating the default auth route path")
			routeName = createdPulp.Name + "-auth"
			expectedRoutes[routeName] = struct {
				Path, TargetPort, ServiceName string
			}{"/auth/login", "api-24817", createdPulp.Name + "-api-svc"}

			k8sClient.Get(ctx, types.NamespacedName{Name: routeName, Namespace: PulpNamespace}, route)
			Expect(route.Spec.Host).Should(Equal(createdPulp.Spec.RouteHost))
			Expect(route.Spec.Path).Should(Equal(expectedRoutes[routeName].(struct{ Path, TargetPort, ServiceName string }).Path))
			Expect(route.Spec.Port.TargetPort).Should(Equal(expectedRoutes[routeName].(struct{ Path, TargetPort, ServiceName string }).TargetPort))
			Expect(route.Spec.To.Name).Should(Equal(expectedRoutes[routeName].(struct{ Path, TargetPort, ServiceName string }).ServiceName))

			By("Making sure no deployment/pulp-web is provisioned")
			webDeployment := &appsv1.Deployment{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: PulpName + "-web", Namespace: PulpNamespace}, webDeployment)
			Expect(err).ShouldNot(BeEmpty())
			Expect(errors.IsNotFound(err)).Should(BeTrue())

			By("Making sure no svc/pulp-web is provisioned")
			webSvc := &corev1.Service{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: PulpName + "-web-svc", Namespace: PulpNamespace}, webSvc)
			Expect(err).ShouldNot(BeEmpty())
			Expect(errors.IsNotFound(err)).Should(BeTrue())

		})
	})

})

// objectUpdate waits and retries until an update request returns without error.
// a common cause that it is needed is because sometimes the object has been modified
// during the update request and we try to modify an old version of it
func objectUpdate[T client.Object](ctx context.Context, object T) {
	Eventually(func() bool {
		if err := k8sClient.Update(ctx, object); err != nil {
			fmt.Println("Error trying to update object: ", err)
			return false
		}
		return true
	}, timeout, interval).Should(BeTrue())
}

// objectGet waits and retries until a get request returns without error.
func objectGet[T client.Object](ctx context.Context, object T, objectName string) {
	Eventually(func() bool {
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: objectName, Namespace: PulpNamespace}, object); err != nil {
			fmt.Println("Error trying to get object: ", err)
			return false
		}
		return true
	}, timeout, interval).Should(BeTrue())
}
