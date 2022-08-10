package controllers

import (
	"context"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// backupDatabase runs a pg_dump inside backup-manager pod and store it in backup PVC
func (r *PulpBackupReconciler) backupDatabase(ctx context.Context, pulpBackup *repomanagerv1alpha1.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := ctrllog.FromContext(ctx)
	backupFile := "pulp.db"

	log.Info("Starting database backup process ...")
	execCmd := []string{"touch", backupDir + "/" + backupFile}
	_, err := r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to create pulp.db backup file")
		return err
	}

	execCmd = []string{"chmod", "0600", backupDir + "/" + backupFile}
	_, err = r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to modify backup file permissions")
		return err
	}

	// [WIP] GET POSTGRES CREDENTIALS AND HOST
	// [WIP] GET POSTGRES CREDENTIALS AND HOST
	// [WIP] GET POSTGRES CREDENTIALS AND HOST
	postgresHost := "pulp-database-svc"
	postgresUser := "pulp"
	postgresDB := "pulp"
	postgresPort := "5432"
	postgresPwd := "pXs5dKOL9dtKWAWIzBWNO8GkAh3QI9Go"
	execCmd = []string{
		"pg_dump", "--clean", "--create",
		"-d", "postgresql://" + postgresUser + ":" + postgresPwd + "@" + postgresHost + ":" + postgresPort + "/" + postgresDB,
		"-f", backupDir + "/" + backupFile,
	}

	_, err = r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to run pg_dump")
		return err
	}

	log.Info("Database Backup finished!")
	return nil
}
