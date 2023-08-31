/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package repo_manager

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pulp/pulp-operator/controllers"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ldapSettings reads the contents from ldapSecret, formats it, and append to pulpSettings
func ldapSettings(resources controllers.FunctionResources, pulpSettings *string) {

	pulp := resources.Pulp
	log := controllers.CustomZapLogger()
	if len(pulp.Spec.LDAP.Config) == 0 {
		return
	}

	settings := `
#### LDAP SETTINGS ####
import ldap
from django_auth_ldap.config import LDAPSearch, PosixGroupType

AUTHENTICATION_BACKENDS = [
  "django_auth_ldap.backend.LDAPBackend",
  "django.contrib.auth.backends.ModelBackend",
  "pulpcore.backends.ObjectRolePermissionBackend",
]
`

	secret := &corev1.Secret{}
	if err := resources.Get(resources.Context, types.NamespacedName{Name: pulp.Spec.LDAP.Config, Namespace: pulp.Namespace}, secret); err != nil {
		log.Error("Error trying to read " + pulp.Spec.LDAP.Config + " Secret")
		return
	}

	// regex to validate the keys from Secret
	r, _ := regexp.Compile("(?i)^AUTH_LDAP_.*$")

	// we need to sort the keys to avoid triggering a reconciliation because
	// when we iterate through golang maps the output are not ordered and it can
	// mislead the controller to think the data are different
	for _, k := range sortKeys(secret.Data) {
		if !r.MatchString(k) { // ignore secret key if it is not in LDAP_AUTH_*
			log.Warn("The key \"" + k + "\" from Secret \"" + pulp.Spec.LDAP.Config + "\" is invalid, ignoring it ...")
			continue
		}

		configValue := string(secret.Data[k])
		// Kubernetes' Secret data is a map[string]string field. Because of that, I could not find a
		// generic way to handle each AUTH_LDAP_ configuration, since they will all be stored as
		// a string in the Secret.
		switch strings.ToUpper(k) {
		// these fields should not be defined as strings (don't add double quotes in it)
		case "AUTH_LDAP_CACHE_TIMEOUT", "AUTH_LDAP_GROUP_SEARCH", "AUTH_LDAP_USER_SEARCH", "AUTH_LDAP_GROUP_TYPE", "AUTH_LDAP_GLOBAL_OPTIONS", "AUTH_LDAP_CONNECTION_OPTIONS":
			settings += fmt.Sprintf("%v = %v\n", strings.ToUpper(k), configValue)
		//these fields are boolean, we need to capitalize them
		case "AUTH_LDAP_MIRROR_GROUPS", "AUTH_LDAP_ALWAYS_UPDATE_USER", "AUTH_LDAP_FIND_GROUP_PERMS", "AUTH_LDAP_START_TLS":
			settings += fmt.Sprintf("%v = %v\n", strings.ToUpper(k), cases.Title(language.English, cases.Compact).String(configValue))
		default:
			settings += fmt.Sprintf("%v = \"%v\"\n", strings.ToUpper(k), configValue)
		}
	}

	settings += "#### END OF LDAP SETTINGS ####\n\n"
	*pulpSettings = *pulpSettings + settings
}
