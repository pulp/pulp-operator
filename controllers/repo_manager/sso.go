package repo_manager

import (
	"fmt"
	"strings"

	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ssoConfig sets the configurations needed to authenticate pulp through keycloak
func ssoConfig(resource controllers.FunctionResources, pulpSettings *string) error {

	log := resource.Logger
	client := resource.Client
	pulp := resource.Pulp
	ctx := resource.Context

	if len(pulp.Spec.SSOSecret) == 0 {
		return nil
	}

	// Check for specified sso configuration secret
	secret := &corev1.Secret{}
	err := client.Get(ctx, types.NamespacedName{Name: pulp.Spec.SSOSecret, Namespace: pulp.Namespace}, secret)
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
	settings, err := controllers.RetrieveSecretData(ctx, pulp.Spec.SSOSecret, pulp.Namespace, true, client, requiredKeys...)
	if err != nil {
		return err
	}

	// retrieve optional keys from sso_secret
	optionalSettings, err := controllers.RetrieveSecretData(ctx, pulp.Spec.SSOSecret, pulp.Namespace, false, client, optionalKeys...)
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
