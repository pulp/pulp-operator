package pulp_restore

import (
	"context"
	"encoding/json"
	"time"

	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type PodReplicas struct {
	Api, Content, Worker, Web int32
}

// restorePulpCR recreates the pulp CR with the content from backup
func (r *PulpRestoreReconciler) restorePulpCR(ctx context.Context, pulpRestore *repomanagerv1alpha1.PulpRestore, backupDir string, pod *corev1.Pod) (PodReplicas, error) {
	pulp := &repomanagerv1alpha1.Pulp{}
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

		pulp := repomanagerv1alpha1.Pulp{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pulpRestore.Spec.DeploymentName,
				Namespace: pulpRestore.Namespace,
			},
		}

		json.Unmarshal([]byte(cmdOutput), &pulp.Spec)

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

// scaleDeployments will update pulp CR with 1 replica for each core component
func (r *PulpRestoreReconciler) scaleDeployments(ctx context.Context, pulpRestore *repomanagerv1alpha1.PulpRestore, podReplicas PodReplicas) error {
	log := r.RawLogger
	pulp := &repomanagerv1alpha1.Pulp{}

	if err := r.Get(ctx, types.NamespacedName{Name: pulpRestore.Spec.DeploymentName, Namespace: pulpRestore.Namespace}, pulp); err != nil && errors.IsNotFound(err) {
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Pulp CR")
		return err
	}

	/* Not sure if this is a good idea to promptly restore with the same number of replicas from backup.
	   After a restore procedure it is common to manually double-check the integrity of the data and the health
	   of the components. With fewer replicas it is easier to troubleshoot and identify possible reprovision issues.
	   Another common scenario of the restore procedure is a DR. Some DR environments have less resources
	   (because of cost, for example) than prod and starting with a single replica and letting users scale them
	   manually also seems to be a good idea. */
	pulp.Spec.Api.Replicas = podReplicas.Api
	pulp.Spec.Content.Replicas = podReplicas.Content
	pulp.Spec.Worker.Replicas = podReplicas.Worker
	pulp.Spec.Web.Replicas = podReplicas.Web

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
