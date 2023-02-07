package repo_manager_backup

import (
	"context"

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// backupDatabase runs a pg_dump inside backup-manager pod and store it in backup PVC
func (r *RepoManagerBackupReconciler) backupDatabase(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := r.RawLogger
	backupFile := "pulp.db"

	log.Info("Starting database backup process ...")
	execCmd := []string{"touch", backupDir + "/" + backupFile}
	_, err := controllers.ContainerExec(r, pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to create pulp.db backup file")
		return err
	}

	execCmd = []string{"chmod", "0600", backupDir + "/" + backupFile}
	_, err = controllers.ContainerExec(r, pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to modify backup file permissions")
		return err
	}

	pgConfig := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.PostgresConfigurationSecret, Namespace: pulpBackup.Namespace}, pgConfig)
	if err != nil {
		log.Error(err, "Failed to find postgres-configuration secret")
		return err
	}
	execCmd = []string{
		"pg_dump", "--clean", "--create", "-Ft",
		"-d", "postgresql://" + string(pgConfig.Data["username"]) + ":" + string(pgConfig.Data["password"]) + "@" + string(pgConfig.Data["host"]) + ":" + string(pgConfig.Data["port"]) + "/" + string(pgConfig.Data["database"]),
		"-f", backupDir + "/" + backupFile,
	}

	_, err = controllers.ContainerExec(r, pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to run pg_dump")
		return err
	}

	log.Info("Database Backup finished!")
	return nil
}
