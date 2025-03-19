package repo_manager_backup

import (
	"bytes"
	"context"
	"fmt"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/printers"
)

// configMapType contains all the information needed to make the backup of the ConfigMap
type configMapType struct {

	// name of the key that will be used to store the configmap name
	name string

	// PulpBackup instance
	pulpBackup *pulpv1.PulpBackup

	// path of where the backup will be stored (PVC mount point)
	backupDir string

	// name of the backup file
	backupFile string

	// name of the configmap that will be copied
	configMapName string

	// backup-manager pod
	pod *corev1.Pod
}

// backupConfigMap makes a copy of the ConfigMaps used by Pulp components
func (r *RepoManagerBackupReconciler) backupConfigMap(ctx context.Context, pulpBackup *pulpv1.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := r.RawLogger
	deploymentName := getDeploymentName(pulpBackup)

	// we are considering that pulp CR instance is running in the same namespace as pulpbackup and
	// that there is only a single instance of pulp CR available
	// we could also let users pass the name of pulp instance
	pulp := &pulpv1.Pulp{}
	if err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: pulpBackup.Namespace}, pulp); err != nil {
		log.Error(err, "Failed to get Pulp")
		return err
	}

	custom_pulp_settings := pulp.Spec.CustomPulpSettings
	if custom_pulp_settings == "" {
		return nil
	}

	// CUSTOM PULP SETTINGS
	if err := r.createConfigMapBackupFile(ctx, configMapType{"custom_pulp_settings", pulpBackup, backupDir, "custom_pulp_settings.yaml", custom_pulp_settings, pod}); err != nil {
		return err
	}
	log.Info("custom_pulp_settings ConfigMap backup finished")

	return nil
}

// createConfigMapBackupFile stores a copy of the ConfigMaps in YAML format.
func (r *RepoManagerBackupReconciler) createConfigMapBackupFile(ctx context.Context, configMapType configMapType) error {
	log := r.RawLogger
	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMapType.configMapName, Namespace: configMapType.pulpBackup.Namespace}, configMap)
	if err != nil {
		log.Error(err, "Error trying to find "+configMapType.configMapName+" configmap")
		return err
	}

	configMapYaml := new(bytes.Buffer)
	ymlPrinter := printers.YAMLPrinter{}
	ymlPrinter.PrintObj(configMap, configMapYaml)

	execCmd := []string{
		"bash", "-c", fmt.Sprintf("cat<<EOF> %s/%s \n%sEOF", configMapType.backupDir, configMapType.backupFile, configMapYaml.String()),
	}
	_, err = controllers.ContainerExec(ctx, r, configMapType.pod, execCmd, configMapType.pulpBackup.Name+"-backup-manager", configMapType.pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup "+configMapType.configMapName+" configmap")
		return err
	}

	log.Info("ConfigMap " + configMapType.configMapName + " backup finished")
	return nil
}
