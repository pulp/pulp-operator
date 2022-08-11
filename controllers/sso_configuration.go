package controllers

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

	settings := map[string]string{}
	for _, key := range requiredKeys {

		// Check to make sure all required sso config values are set before proceeding
		if secret.Data[key] == nil {
			return fmt.Errorf("secret %v is missing required configuration data (%v)", pulp.Spec.SSOSecret, key)
		}
		settings[strings.ToUpper(key)] = string(secret.Data[key])
	}

	for _, key := range optionalKeys {
		if secret.Data[key] != nil {
			settings[strings.ToUpper(key)] = string(secret.Data[key])
		}
	}

	// Inject SSO settings into pulp_settings
	for key := range settings {
		*pulpSettings = *pulpSettings + fmt.Sprintf("%v = \"%v\"\n", key, settings[key])
	}

	return nil
}
