package repo_manager_restore

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type PodReplicas struct {
	Api, Content, Worker, Web int32
}

// restorePulpCR recreates the pulp CR with the content from backup
func (r *RepoManagerRestoreReconciler) restorePulpCR(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore, backupDir string, pod *corev1.Pod) (PodReplicas, error) {
	pulp := &repomanagerpulpprojectorgv1beta2.Pulp{}
	podReplicas := PodReplicas{}

	// we'll recreate pulp instance only if it was not found
	// in situations like during a pulpRestore reconcile loop (because of an error, for example) pulp instance could have been previously created
	// this will avoid an infinite reconciliation loop trying to recreate a resource that already exists
	if err := r.Get(ctx, types.NamespacedName{Name: pulpRestore.Spec.DeploymentName, Namespace: pulpRestore.Namespace}, pulp); err != nil && errors.IsNotFound(err) {
		log := r.RawLogger
		log.Info("Restoring " + pulpRestore.Spec.DeploymentName + " CR ...")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Restoring "+pulpRestore.Spec.DeploymentName+" CR", "Restoring"+pulpRestore.Spec.DeploymentName+"CR")
		execCmd := []string{
			"cat", backupDir + "/cr_object",
		}
		cmdOutput, err := controllers.ContainerExec(r, pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)
		if err != nil {
			log.Error(err, "Failed to get cr_object backup file!")
			r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to get cr_object backup file!", "FailedGet"+pulpRestore.Spec.DeploymentName+"CR")
			return PodReplicas{}, err
		}

		pulp := repomanagerpulpprojectorgv1beta2.Pulp{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pulpRestore.Spec.DeploymentName,
				Namespace: pulpRestore.Namespace,
			},
		}

		// [DEPRECATED] Temporarily adding to keep compatibility with ansible version
		ansible, _ := r.isAnsibleBackup(ctx, pulpRestore, backupDir, pod)
		if !ansible {
			json.Unmarshal([]byte(cmdOutput), &pulp.Spec)
		} else {
			json.Unmarshal([]byte(parseUnquotedJson(cmdOutput)), &pulp.Spec)
			log.V(1).Info("Restoring Pulp CR from ansible ...", "CR", pulp.Spec)

			// in ansible version the pvc used by database pods was created by the statefulset volumeClaimTemplates
			// in the current version, if users do not provide a SC or PVC the sts will use emptyDir (no PVC will be created through volumeClaimTemplates)
			// for ansible restoration we are manually creating the PVC to "replicate" the volumeClaimTemplates behavior
			postgresPVCName, err := r.deployPostgresPVC(ctx, pulpRestore, &pulp)
			if err != nil {
				return PodReplicas{}, err
			}

			// Configure Pulp CR to use this new PVC
			pulp.Spec.Database.PVC = postgresPVCName

			// Ansible version deploys Redis by default
			// cache_enabled is defined in playbook, so we cannot recover this from the copied CR
			pulp.Spec.Cache.Enabled = true

			// in ansible version if no object storage is defined, the operator will deploy a pvc
			// in the current version, if users do not provide a SC or PVC or Object Storage credentials
			// the operator will deploy pulp pods with emptyDir
			// for ansible restoration we are manually creating the PVC to "replicate" ansible behavior
			pulpPVCName, err := r.deployPulpPVC(ctx, pulpRestore, &pulp)
			if err != nil {
				return PodReplicas{}, err
			}
			pulp.Spec.PVC = pulpPVCName
		}

		// store the number of replicas so we can rescale with the same amount later
		podReplicas = PodReplicas{
			Api:     pulp.Spec.Api.Replicas,
			Content: pulp.Spec.Content.Replicas,
			Worker:  pulp.Spec.Worker.Replicas,
			Web:     pulp.Spec.Web.Replicas,
		}

		pulp.Spec.Api.Replicas = 0
		pulp.Spec.Content.Replicas = 0
		pulp.Spec.Worker.Replicas = 0
		pulp.Spec.Web.Replicas = 0

		if err = r.Create(ctx, &pulp); err != nil {
			log.Error(err, "Error trying to restore "+pulpRestore.Spec.DeploymentName+" CR!")
			r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to restore cr_object!", "FailedRestore"+pulpRestore.Spec.DeploymentName+"CR")
			return PodReplicas{}, err
		}

		log.Info(pulpRestore.Spec.DeploymentName + " CR restored!")
	}

	return podReplicas, nil
}

// scaleDeployments will rescale the deployments with:
// - if KeepBackupReplicasCount = true  - it will keep the same amount of replicas from backup
// - if KeepBackupReplicasCount = false - it will deploy 1 replica for each component
func (r *RepoManagerRestoreReconciler) scaleDeployments(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore, podReplicas PodReplicas) error {
	log := r.RawLogger
	pulp := &repomanagerpulpprojectorgv1beta2.Pulp{}

	if err := r.Get(ctx, types.NamespacedName{Name: pulpRestore.Spec.DeploymentName, Namespace: pulpRestore.Namespace}, pulp); err != nil && errors.IsNotFound(err) {
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Pulp CR")
		return err
	}

	if pulpRestore.Spec.KeepBackupReplicasCount {
		pulp.Spec.Api.Replicas = podReplicas.Api
		pulp.Spec.Content.Replicas = podReplicas.Content
		pulp.Spec.Worker.Replicas = podReplicas.Worker
		pulp.Spec.Web.Replicas = podReplicas.Web
	} else {
		pulp.Spec.Api.Replicas = 1
		pulp.Spec.Content.Replicas = 1
		pulp.Spec.Worker.Replicas = 1
		isNginxIngress := strings.ToLower(pulp.Spec.IngressType) == "ingress" && !controllers.IsNginxIngressSupported(r, pulp.Spec.IngressClassName)
		if strings.ToLower(pulp.Spec.IngressType) != "route" && !isNginxIngress {
			pulp.Spec.Web.Replicas = 1
		}
	}

	if err := r.Update(ctx, pulp); err != nil {
		log.Error(err, "Failed to scale up deployment replicas!")
		return err
	}

	log.Info("Waiting operator tasks ...")
	// wait operator finish update before proceeding
	for timeout := 0; timeout < 18; timeout++ {
		time.Sleep(time.Second * 10)

		// [TODO] we should use the operator status to make sure that it finished its execution, but the
		// .status.condition is not reflecting the real state.
		// pulp-api and pulp-web were not READY and Pulp-Operator-Finished-Execution was set to true
		/* r.Get(ctx, types.NamespacedName{Name: pulpRestore.Spec.DeploymentName, Namespace: pulpRestore.Namespace}, pulp)
		if v1.IsStatusConditionPresentAndEqual(pulp.Status.Conditions, cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType)+"-Operator-Finished-Execution", metav1.ConditionTrue) {
			break
		} */

		apiDeployment := &appsv1.Deployment{}
		r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-api", Namespace: pulp.Namespace}, apiDeployment)
		if apiDeployment.Status.ReadyReplicas == apiDeployment.Status.Replicas {
			break
		}
	}

	// There is a small interval in which pulp-web stays in a READY state and crash after a few seconds because pulp-api
	// didn't finish it boot process. This sleep is a workaround to try to mitigate this.
	// [TODO] add readiness probe to pulp-web pods
	time.Sleep(time.Second * 60)
	for timeout := 0; timeout < 18; timeout++ {
		webDeployment := &appsv1.Deployment{}
		r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-web", Namespace: pulp.Namespace}, webDeployment)
		if webDeployment.Status.ReadyReplicas == webDeployment.Status.Replicas {
			break
		}
		time.Sleep(time.Second * 10)
	}

	return nil
}

// [DEPRECATED] Temporarily adding to keep compatibility with ansible version
// ansible backup of Pulp CR produces a "non-formatted" file (it is a "json" without quotes in key and values).
// since it is not formatted we cannot just unmarshal it to Pulp type
func parseUnquotedJson(pulpCR string) string {
	re := regexp.MustCompile(`([a-zA-Z0-9-_]+):\s*(http(s)?:\/\/[a-z0-9.-]+(:[0-9]+)?|[a-zA-Z0-9-_./]+)`)

	simpleFields := re.ReplaceAllString(string(pulpCR), `"$1": "$2"`)
	re = regexp.MustCompile(`([a-zA-Z0-9-_]+):\s*(\{)`)

	objectFields := re.ReplaceAllString(string(simpleFields), `"$1": $2`)
	re = regexp.MustCompile(`(["a-zA-Z0-9-_]+):\s*\[(.*?)\]`)

	return re.ReplaceAllString(string(objectFields), `"$1": ["$2"]`)
}

// getDeploymentType returns the deployment_type (if not provided default is "pulp")
func getDeploymentType(pulp *repomanagerpulpprojectorgv1beta2.Pulp) string {
	deploymentType := pulp.Spec.DeploymentType
	if len(pulp.Spec.DeploymentType) == 0 {
		deploymentType = "pulp"
	}
	return deploymentType
}

// [DEPRECATED] Temporarily adding to keep compatibility with ansible version
func (r *RepoManagerRestoreReconciler) deployPostgresPVC(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore, pulp *repomanagerpulpprojectorgv1beta2.Pulp) (string, error) {

	log := r.RawLogger
	postgresPVCName := "postgres-" + pulp.Name
	deploymentType := getDeploymentType(pulp)
	postgresPVC := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, types.NamespacedName{Name: postgresPVCName, Namespace: pulp.Namespace}, postgresPVC); err != nil {
		postgresPVC = &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      postgresPVCName,
				Namespace: pulpRestore.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/name":       "postgres",
					"app.kubernetes.io/instance":   "postgres-" + pulp.Name,
					"app.kubernetes.io/component":  "database",
					"app.kubernetes.io/part-of":    deploymentType,
					"app.kubernetes.io/managed-by": deploymentType + "-operator",
					"owner":                        "pulp-dev",
					"app":                          "postgresql",
					"pulp_cr":                      pulp.Name,
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				StorageClassName: pulp.Spec.PostgresStorageClass,
			},
		}

		if pulp.Spec.PostgresResourceRequirements != nil {
			postgresPVC.Spec.Resources = *pulp.Spec.PostgresResourceRequirements
		} else {
			postgresPVC.Spec.Resources.Requests = corev1.ResourceList{
				corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("8Gi"),
			}
		}

		if err = r.Create(ctx, postgresPVC); err != nil {
			log.Error(err, "Error trying to create the database PVC!")
			r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to create the database PVC!", "FailedCreateDBPVC")
			return "", err
		}
	}
	return postgresPVCName, nil
}

// [DEPRECATED] Temporarily adding to keep compatibility with ansible version
func (r *RepoManagerRestoreReconciler) deployPulpPVC(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore, pulp *repomanagerpulpprojectorgv1beta2.Pulp) (string, error) {

	log := r.RawLogger
	deploymentType := getDeploymentType(pulp)

	// in ansible, if no azure nor s3 secret are provided it means it should deploy a PVC
	if len(pulp.Spec.ObjectStorageAzureSecret) == 0 && len(pulp.Spec.ObjectStorageS3Secret) == 0 {
		pulpPVCName := pulpRestore.Spec.DeploymentName + "-file-storage"
		pulpPVC := &corev1.PersistentVolumeClaim{}
		if err := r.Get(ctx, types.NamespacedName{Name: pulpPVCName, Namespace: pulp.Namespace}, pulpPVC); err != nil {
			pulpPVC = &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pulpPVCName,
					Namespace: pulpRestore.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":       deploymentType + "-storage",
						"app.kubernetes.io/instance":   deploymentType + "-storage-" + pulpRestore.Spec.DeploymentName,
						"app.kubernetes.io/component":  "storage",
						"app.kubernetes.io/part-of":    deploymentType,
						"app.kubernetes.io/managed-by": deploymentType + "-operator",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					StorageClassName: &pulp.Spec.FileStorageClass,
				},
			}

			pulpPVC.Spec.Resources.Requests = corev1.ResourceList{
				corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(pulp.Spec.FileStorageSize),
			}

			if err = r.Create(ctx, pulpPVC); err != nil {
				log.Error(err, "Error trying to create Pulp PVC!")
				r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to create Pulp PVC!", "FailedCreatePUlpPVC")
				return "", err
			}
		}

		return pulpPVCName, nil
	}

	// if object storage secret provided, we should not return a PVC
	return "", nil
}
