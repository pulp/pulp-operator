package repo_manager_backup

import (
	"context"
	"errors"

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
)

// createBackupDir creates the directory to store the backup
func (r *RepoManagerBackupReconciler) createBackupDir(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := r.RawLogger
	backupPod := pulpBackup.Name + "-backup-manager"

	log.Info("Creating backup folder ...")
	execCmd := []string{"mkdir", "-p", backupDir}
	_, err := controllers.ContainerExec(r, pod, execCmd, backupPod, pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to create backup folder")
		return err
	}
	return nil
}

// checkRequiredFields will verify if all required fields are provided
func checkRequiredFields(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup) error {
	if len(pulpBackup.Spec.DeploymentName) == 0 {
		return errors.New("error! deployment_name not provided")
	}
	return nil
}

// getDeploymentName returns the deployment_name
func getDeploymentName(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup) string {
	return pulpBackup.Spec.DeploymentName
}

// getDeploymentType returns the deployment_type
func getDeploymentType(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup) string {
	deploymentType := pulpBackup.Spec.DeploymentType
	if len(pulpBackup.Spec.DeploymentType) == 0 {
		deploymentType = "pulp"
	}
	return deploymentType
}

// getAdminPasswordSecret returns the admin_password_secret if provided, if not will return the default one based
// on deployment_name
func getAdminPasswordSecret(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup) string {
	adminPasswordSecret := pulpBackup.Spec.AdminPasswordSecret
	if len(adminPasswordSecret) == 0 {
		adminPasswordSecret = getDeploymentName(ctx, pulpBackup) + "-admin-password"
	}
	return adminPasswordSecret
}

// getPulpSecretKey returns the pulp_secret_key if provided, if not will return
// the default one based on deployment_name
func getPulpSecretKey(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup) string {
	adminPasswordSecret := pulpBackup.Spec.PulpSecretKey
	if len(adminPasswordSecret) == 0 {
		adminPasswordSecret = getDeploymentName(ctx, pulpBackup) + "-secret-key"
	}
	return adminPasswordSecret
}

// getBackupPVC returns the backup_pvc if provided, if not will return the default one based
// on deployment_name
func getBackupPVC(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup) string {
	backupPVC := pulpBackup.Spec.BackupPVC
	if len(backupPVC) == 0 {
		backupPVC = pulpBackup.Name + "-backup-claim"
	}
	return backupPVC
}

// getBackupPVCNamespace returns the backup_pvc_namespace if provided, if not will return the default one based
// on deployment_name
func getBackupPVCNamespace(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup) string {
	backupPVCNamespace := pulpBackup.Spec.BackupPVCNamespace
	if len(backupPVCNamespace) == 0 {
		backupPVCNamespace = pulpBackup.Namespace
	}
	return backupPVCNamespace
}

// getBackupDir returns the backup path based on the provided timestamp
func getBackupDir(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup, timestamp string) string {
	return "/backups/openshift-backup-" + timestamp
}

// getPostgresCfgSecret returns the name of the secret with postgres configuration
func getPostgresCfgSecret(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup) string {
	postgresCfgSecret := pulpBackup.Spec.PostgresConfigurationSecret
	if len(postgresCfgSecret) == 0 {
		postgresCfgSecret = getDeploymentName(ctx, pulpBackup) + "-postgres-configuration"
	}
	return postgresCfgSecret
}

// getDBFieldsEncryption returns the db_fields_encryption_secret
func getDBFieldsEncryption(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup, pulp *repomanagerpulpprojectorgv1beta2.Pulp) string {
	return pulp.Status.DBFieldsEncryptionSecret
}

// getContainerTokenSecret returns the container_token_secret
func getContainerTokenSecret(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup, pulp *repomanagerpulpprojectorgv1beta2.Pulp) string {
	return pulp.Status.ContainerTokenSecret
}

// setStatusFields will populate all the related status but conditions[].
func (r *RepoManagerBackupReconciler) setStatusFields(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup, timestamp string) error {
	pulpBackup.Status.AdminPasswordSecret = getAdminPasswordSecret(ctx, pulpBackup)
	pulpBackup.Status.BackupClaim = getBackupPVC(ctx, pulpBackup)
	pulpBackup.Status.BackupDirectory = getBackupDir(ctx, pulpBackup, timestamp)
	pulpBackup.Status.BackupNamespace = getBackupPVCNamespace(ctx, pulpBackup)
	pulpBackup.Status.DeploymentName = getDeploymentName(ctx, pulpBackup)
	if err := r.Status().Update(ctx, pulpBackup); err != nil {
		return err
	}
	return nil
}
