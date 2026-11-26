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
	"testing"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
)

func migrationSettingsFor(spec pulpv1.PulpSpec) string {
	settings := ""
	needsMigrationSetting(controllers.FunctionResources{Pulp: &pulpv1.Pulp{Spec: spec}}, &settings, map[string]struct{}{})
	return settings
}

// A false setting must not hide the ones after it, whatever the map iteration order.
func TestNeedsMigrationSettingKeepsTrueSettings(t *testing.T) {
	spec := pulpv1.PulpSpec{RedirectToObjectStorage: true, HideGuardedDistributions: false}
	want := "REDIRECT_TO_OBJECT_STORAGE = True\n"
	for i := 0; i < 100; i++ {
		if got := migrationSettingsFor(spec); got != want {
			t.Fatalf("run %d: settings = %q, want %q", i, got, want)
		}
	}
}

// settings.py is compared between reconciles, so the output must be stable.
func TestNeedsMigrationSettingIsDeterministic(t *testing.T) {
	spec := pulpv1.PulpSpec{RedirectToObjectStorage: true, HideGuardedDistributions: true}
	first := migrationSettingsFor(spec)
	for i := 0; i < 100; i++ {
		if got := migrationSettingsFor(spec); got != first {
			t.Fatalf("run %d: settings = %q, first run gave %q", i, got, first)
		}
	}
}
