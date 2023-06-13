package repo_manager_restore

import (
	"context"
	"fmt"
	"time"

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// restoreDatabaseData scales down the pods and runs a pg_restore
func (r *RepoManagerRestoreReconciler) restoreDatabaseData(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore, backupDir string, pod *corev1.Pod) error {
	log := r.RawLogger
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

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version
	if ansible, _ := r.isAnsibleBackup(ctx, pulpRestore, backupDir, pod); ansible {
		log.V(1).Info("Restoring from ansible db backup ...")
		psqlRestore := fmt.Sprintf("psql -U %v -h %v -d %v -p %v", string(pgConfig.Data["username"]), string(pgConfig.Data["host"]), string(pgConfig.Data["database"]), string(pgConfig.Data["port"]))
		execCmd = []string{"sh", "-c", fmt.Sprintf("cat %v/pulp.db | PGPASSWORD=%v %v", backupDir, string(pgConfig.Data["password"]), psqlRestore)}
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
func (r *RepoManagerRestoreReconciler) waitDBReady(ctx context.Context, namespace, stsName string) error {
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
