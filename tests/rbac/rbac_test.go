package rbac

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	rbacDir     = "../../config/rbac"
	templateDir = "../../deploy-templates/templates"
	valuesFile  = "../../deploy-templates/values.yaml"
)

var (
	adminVerbs  = sets.New("create", "delete", "get", "list", "patch", "update", "watch")
	viewerVerbs = sets.New("get", "list", "watch")

	// Kinds whose spec pairs a Secret reference with an author-controlled address.
	// The scaffolded Kustomize Nexus roles name the resource "nexus", the other roles use the plural.
	guardedKinds = []struct {
		name               string
		kustomizeResources []string
		helmResource       string
	}{
		{name: "nexus", kustomizeResources: []string{"nexus", "nexuses"}, helmResource: "nexuses"},
		{name: "nexusrepository", kustomizeResources: []string{"nexusrepositories"}, helmResource: "nexusrepositories"},
		{name: "nexusblobstore", kustomizeResources: []string{"nexusblobstores"}, helmResource: "nexusblobstores"},
	}

	templateExpr = regexp.MustCompile(`\{\{.*?\}\}`)
)

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)

	return string(data)
}

func decodeRoles(t *testing.T, content string) []rbacv1.ClusterRole {
	t.Helper()

	var roles []rbacv1.ClusterRole

	dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(content)), 4096)

	for {
		var role rbacv1.ClusterRole

		err := dec.Decode(&role)
		if errors.Is(err, io.EOF) {
			return roles
		}

		require.NoError(t, err)

		if role.Kind != "" {
			roles = append(roles, role)
		}
	}
}

// renderHelperRoles drops template control lines and replaces inline expressions with a placeholder.
func renderHelperRoles(t *testing.T) []rbacv1.ClusterRole {
	t.Helper()

	lines := strings.Split(readFile(t, filepath.Join(templateDir, "helper_roles.yaml")), "\n")
	kept := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "{{") {
			continue
		}

		kept = append(kept, templateExpr.ReplaceAllString(line, "X"))
	}

	return decodeRoles(t, strings.Join(kept, "\n"))
}

func kustomizeResourceFiles(t *testing.T) []string {
	t.Helper()

	var resources struct {
		Resources []string `json:"resources"`
	}

	require.NoError(t, yaml.Unmarshal([]byte(readFile(t, filepath.Join(rbacDir, "kustomization.yaml"))), &resources))

	return resources.Resources
}

func verbsOn(role rbacv1.ClusterRole, resources ...string) sets.Set[string] {
	verbs := sets.New[string]()

	for _, rule := range role.Rules {
		if sets.New(rule.Resources...).HasAny(resources...) {
			verbs.Insert(rule.Verbs...)
		}
	}

	return verbs
}

// assertHelperRoles checks the roles that target a kind: one admin with explicit verbs, one viewer, no other.
func assertHelperRoles(t *testing.T, roles []rbacv1.ClusterRole, adminName, viewerName string, resources []string) {
	t.Helper()

	var creators []string

	for _, role := range roles {
		for key := range role.Labels {
			require.NotContains(t, key, "aggregate-to-", "role %s must not aggregate into built-in roles", role.Name)
		}

		for _, rule := range role.Rules {
			require.NotContains(t, rule.Verbs, "*", "role %s", role.Name)
			require.NotContains(t, rule.Resources, "*", "role %s", role.Name)
		}

		if verbsOn(role, resources...).Has("create") {
			creators = append(creators, role.Name)
		}
	}

	require.Equal(t, []string{adminName}, creators, "only the admin role may create the kind")

	for _, role := range roles {
		statusResources := make([]string, 0, len(resources))
		for _, r := range resources {
			statusResources = append(statusResources, r+"/status")
		}

		switch role.Name {
		case adminName:
			require.Equal(t, adminVerbs, verbsOn(role, resources...), "admin verbs of %s", role.Name)
			require.Equal(t, sets.New("get"), verbsOn(role, statusResources...), "admin status verbs of %s", role.Name)
		case viewerName:
			require.Equal(t, viewerVerbs, verbsOn(role, resources...), "viewer verbs of %s", role.Name)
			require.Equal(t, sets.New("get"), verbsOn(role, statusResources...), "viewer status verbs of %s", role.Name)
		default:
			require.Failf(t, "unexpected helper role", "%s", role.Name)
		}
	}
}

func TestKustomizeHelperRoles(t *testing.T) {
	t.Parallel()

	files := kustomizeResourceFiles(t)

	for _, kind := range guardedKinds {
		t.Run(kind.name, func(t *testing.T) {
			t.Parallel()

			require.NotContains(t, files, kind.name+"_editor_role.yaml")
			require.NoFileExists(t, filepath.Join(rbacDir, kind.name+"_editor_role.yaml"))

			var roles []rbacv1.ClusterRole

			for _, file := range files {
				if !strings.HasPrefix(file, kind.name+"_") {
					continue
				}

				roles = append(roles, decodeRoles(t, readFile(t, filepath.Join(rbacDir, file)))...)
			}

			require.Len(t, roles, 2, "admin and viewer only")
			assertHelperRoles(t, roles, kind.name+"-admin-role", kind.name+"-viewer-role", kind.kustomizeResources)
		})
	}
}

func TestKustomizeOtherKindsKeepEditorRole(t *testing.T) {
	t.Parallel()

	files := kustomizeResourceFiles(t)

	for _, kind := range []string{"nexuscleanuppolicy", "nexusrole", "nexusscript", "nexususer"} {
		require.Contains(t, files, kind+"_admin_role.yaml")
		require.Contains(t, files, kind+"_editor_role.yaml")
		require.Contains(t, files, kind+"_viewer_role.yaml")
	}
}

func TestHelmHelperRoles(t *testing.T) {
	t.Parallel()

	roles := renderHelperRoles(t)
	require.Len(t, roles, 2*len(guardedKinds))

	for _, kind := range guardedKinds {
		var kindRoles []rbacv1.ClusterRole

		for _, role := range roles {
			if strings.HasSuffix(role.Name, "-"+kind.name+"-admin") || strings.HasSuffix(role.Name, "-"+kind.name+"-viewer") {
				kindRoles = append(kindRoles, role)
			}
		}

		require.Len(t, kindRoles, 2, kind.name)
		prefix := "edp-X-X-" + kind.name
		assertHelperRoles(t, kindRoles, prefix+"-admin", prefix+"-viewer", []string{kind.helmResource})
	}

	for _, role := range roles {
		require.Equal(t, "ClusterRole", role.Kind)
	}
}

func TestHelmHelperRolesSwitch(t *testing.T) {
	t.Parallel()

	template := strings.TrimSpace(readFile(t, filepath.Join(templateDir, "helper_roles.yaml")))
	require.True(t, strings.HasPrefix(template, "{{- if .Values.helperRoles.enabled }}"))
	require.True(t, strings.HasSuffix(template, "{{- end }}"))

	var values struct {
		HelperRoles struct {
			Enabled bool `json:"enabled"`
		} `json:"helperRoles"`
	}

	require.NoError(t, yaml.Unmarshal([]byte(readFile(t, valuesFile)), &values))
	require.True(t, values.HelperRoles.Enabled)
}

func TestHelperRolesAreNotBound(t *testing.T) {
	t.Parallel()

	helperName := regexp.MustCompile(`(?m)^\s*name:.*-(admin|viewer)(-role)?\s*$`)

	bindings := []string{
		filepath.Join(rbacDir, "role_binding.yaml"),
		filepath.Join(rbacDir, "leader_election_role_binding.yaml"),
		filepath.Join(rbacDir, "metrics_auth_role_binding.yaml"),
	}

	templates, err := filepath.Glob(filepath.Join(templateDir, "*.yaml"))
	require.NoError(t, err)

	for _, file := range append(bindings, templates...) {
		content := readFile(t, file)

		if filepath.Base(file) == "helper_roles.yaml" {
			require.NotContains(t, content, "RoleBinding", file)

			continue
		}

		if !strings.Contains(content, "RoleBinding") {
			continue
		}

		require.NotRegexp(t, helperName, content, "%s must not bind a helper role", file)
	}
}
