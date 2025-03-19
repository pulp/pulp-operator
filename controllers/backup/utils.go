package repo_manager_backup

import (
	"context"
	"errors"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
)

// createBackupDir creates the directory to store the backup
func (r *RepoManagerBackupReconciler) createBackupDir(ctx context.Context, pulpBackup *pulpv1.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := r.RawLogger
	backupPod := pulpBackup.Name + "-backup-manager"

	log.Info("Creating backup folder ...")
	execCmd := []string{"mkdir", "-p", backupDir}
	_, err := controllers.ContainerExec(ctx, r, pod, execCmd, backupPod, pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to create backup folder")
		return err
	}
	return nil
}

// checkRequiredFields will verify if all required fields are provided
func checkRequiredFields(pulpBackup *pulpv1.PulpBackup) error {
	if len(pulpBackup.Spec.DeploymentName) == 0 {
		return errors.New("error! deployment_name not provided")
	}
	return nil
}

// getDeploymentName returns the deployment_name
func getDeploymentName(pulpBackup *pulpv1.PulpBackup) string {
	return pulpBackup.Spec.DeploymentName
}

// getAdminPasswordSecret returns the admin_password_secret if provided, if not will return the default one based
// on deployment_name
func getAdminPasswordSecret(pulpBackup *pulpv1.PulpBackup) string {
	adminPasswordSecret := pulpBackup.Spec.AdminPasswordSecret
	if len(adminPasswordSecret) == 0 {
		adminPasswordSecret = getDeploymentName(pulpBackup) + "-admin-password"
	}
	return adminPasswordSecret
}

// getPulpSecretKey returns the pulp_secret_key if provided, if not will return
// the default one based on deployment_name
func getPulpSecretKey(pulpBackup *pulpv1.PulpBackup) string {
	adminPasswordSecret := pulpBackup.Spec.PulpSecretKey
	if len(adminPasswordSecret) == 0 {
		adminPasswordSecret = getDeploymentName(pulpBackup) + "-secret-key"
	}
	return adminPasswordSecret
}

// getBackupPVC returns the backup_pvc if provided, if not will return the default one based
// on deployment_name
func getBackupPVC(pulpBackup *pulpv1.PulpBackup) string {
	backupPVC := pulpBackup.Spec.BackupPVC
	if len(backupPVC) == 0 {
		backupPVC = pulpBackup.Name + "-backup-claim"
	}
	return backupPVC
}

// getBackupPVCNamespace returns the backup_pvc_namespace if provided, if not will return the default one based
// on deployment_name
func getBackupPVCNamespace(pulpBackup *pulpv1.PulpBackup) string {
	backupPVCNamespace := pulpBackup.Spec.BackupPVCNamespace
	if len(backupPVCNamespace) == 0 {
		backupPVCNamespace = pulpBackup.Namespace
	}
	return backupPVCNamespace
}

// getBackupDir returns the backup path based on the provided timestamp
func getBackupDir(timestamp string) string {
	return "/backups/openshift-backup-" + timestamp
}

// getPostgresCfgSecret returns the name of the secret with postgres configuration
func getPostgresCfgSecret(pulpBackup *pulpv1.PulpBackup) string {
	postgresCfgSecret := pulpBackup.Spec.PostgresConfigurationSecret
	if len(postgresCfgSecret) == 0 {
		postgresCfgSecret = getDeploymentName(pulpBackup) + "-postgres-configuration"
	}
	return postgresCfgSecret
}

// getDBFieldsEncryption returns the db_fields_encryption_secret
func getDBFieldsEncryption(pulp *pulpv1.Pulp) string {
	return pulp.Status.DBFieldsEncryptionSecret
}

// getContainerTokenSecret returns the container_token_secret
func getContainerTokenSecret(pulp *pulpv1.Pulp) string {
	return pulp.Status.ContainerTokenSecret
}

// setStatusFields will populate all the related status but conditions[].
func (r *RepoManagerBackupReconciler) setStatusFields(ctx context.Context, pulpBackup *pulpv1.PulpBackup, timestamp string) error {
	pulpBackup.Status.AdminPasswordSecret = getAdminPasswordSecret(pulpBackup)
	pulpBackup.Status.BackupClaim = getBackupPVC(pulpBackup)
	pulpBackup.Status.BackupDirectory = getBackupDir(timestamp)
	pulpBackup.Status.BackupNamespace = getBackupPVCNamespace(pulpBackup)
	pulpBackup.Status.DeploymentName = getDeploymentName(pulpBackup)
	if err := r.Status().Update(ctx, pulpBackup); err != nil {
		return err
	}
	return nil
}
