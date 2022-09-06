package pulp_restore

import (
	"context"
	"time"

	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// restoreDatabaseData scales down the pods and runs a pg_restore
func (r *PulpRestoreReconciler) restoreDatabaseData(ctx context.Context, pulpRestore *repomanagerv1alpha1.PulpRestore, backupDir string, pod *corev1.Pod) error {
	log := log.FromContext(ctx)
	backupFile := "pulp.db"

	//[TODO] fix this
	// WORKAROUND!!!! Giving some time to pulp CR be created
	// sometimes the scale down process was failing with kube-api returning an error
	// because the object has been modified and asking to try again
	// I think one of the reasons that it is happening is because pulp CR
	// was in the middle of its "creation process" and this kludge did a "relief"
	time.Sleep(5 * time.Second)

	// retrieve pg credentials and address
	pgConfig := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: pulpRestore.Status.PostgresSecret, Namespace: pulpRestore.Namespace}, pgConfig); err != nil {
		log.Error(err, "Failed to find postgres-configuration secret")
		return err
	}

	// wait until database pod is ready
	log.Info("Waiting db pod get into a READY state ...")
	r.waitDBReady(ctx, pulpRestore.Namespace, pulpRestore.Spec.DeploymentName+"-database")

	// run pg_restore
	execCmd := []string{
		"pg_restore", "-d",
		"postgresql://" + string(pgConfig.Data["username"]) + ":" + string(pgConfig.Data["password"]) + "@" + string(pgConfig.Data["host"]) + ":" + string(pgConfig.Data["port"]) + "/" + string(pgConfig.Data["database"]),
		backupDir + "/" + backupFile,
	}

	log.Info("Running db restore ...")
	if _, err := controllers.ContainerExec(r, pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace); err != nil {
		log.Error(err, "Failed to restore postgres data")
		return err
	}

	log.Info("Database restore finished!")

	return nil
}

// waitDBReady waits until db container gets into a "READY" state
func (r *PulpRestoreReconciler) waitDBReady(ctx context.Context, namespace, stsName string) error {
	var err error
	for timeout := 0; timeout < 120; timeout++ {
		sts := &appsv1.StatefulSet{}
		err = r.Get(ctx, types.NamespacedName{Name: stsName, Namespace: namespace}, sts)
		if sts.Status.ReadyReplicas == sts.Status.Replicas {
			return nil
		}
		time.Sleep(time.Second)
	}
	return err
}
