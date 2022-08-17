package pulp

import (
	"context"
	"fmt"
	"strings"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// ssoConfig sets the configurations needed to authenticate pulp through keycloak
func (r *PulpReconciler) ssoConfig(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, pulpSettings *string) error {
	log := ctrllog.FromContext(ctx)

	// Check for specified sso configuration secret
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Spec.SSOSecret, Namespace: pulp.Namespace}, secret)
	if err != nil {
		log.Error(err, "Failed to find "+pulp.Spec.SSOSecret+" secret")
		return err
	}

	// Check sso data format
	if secret.Data == nil {
		return fmt.Errorf("cannot read the data for secret %v", pulp.Spec.SSOSecret)
	}

	requiredKeys := []string{
		"social_auth_keycloak_key", "social_auth_keycloak_secret", "social_auth_keycloak_public_key",
		"keycloak_host", "keycloak_protocol", "keycloak_port", "keycloak_realm",
	}

	optionalKeys := []string{
		"keycloak_admin_role", "keycloak_group_token_claim", "keycloak_role_token_claim", "keycloak_host_loopback",
	}

	// retrieve mandatory keys from sso_secret
	settings, err := r.retrieveSecretData(ctx, pulp.Spec.SSOSecret, pulp.Namespace, true, requiredKeys...)
	if err != nil {
		return err
	}

	// retrieve optional keys from sso_secret
	optionalSettings, err := r.retrieveSecretData(ctx, pulp.Spec.SSOSecret, pulp.Namespace, false, optionalKeys...)
	if err != nil {
		return err
	}

	// merge required + optional keys
	for key, value := range optionalSettings {
		settings[key] = value
	}

	// Inject SSO settings into pulp_settings
	for key := range settings {
		*pulpSettings = *pulpSettings + fmt.Sprintf("%v = \"%v\"\n", strings.ToUpper(key), settings[key])
	}

	return nil
}
