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
	guardedKinds = []struct {
		name               string
		kustomizeResources []string
		helmResource       string
	}{
		{name: "nexus", kustomizeResources: []string{"nexuses"}, helmResource: "nexuses"},
		{name: "nexusrepository", kustomizeResources: []string{"nexusrepositories"}, helmResource: "nexusrepositories"},
		{name: "nexusblobstore", kustomizeResources: []string{"nexusblobstores"}, helmResource: "nexusblobstores"},
	}

	// Kinds without a Secret reference to a spec address: admin and viewer roles, no editor.
	otherKinds = map[string]string{
		"nexuscleanuppolicy": "nexuscleanuppolicies",
		"nexusrole":          "nexusroles",
		"nexusscript":        "nexusscripts",
		"nexususer":          "nexususers",
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

// renderTemplate drops template control lines and replaces inline expressions with a placeholder.
func renderTemplate(t *testing.T, name string) []rbacv1.ClusterRole {
	t.Helper()

	lines := strings.Split(readFile(t, filepath.Join(templateDir, name)), "\n")
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

// assertHelperRoles checks the roles that target a kind: one admin with explicit verbs, one read-only viewer, no other.
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

func TestKustomizeOtherKindsHaveAdminAndViewerOnly(t *testing.T) {
	t.Parallel()

	files := kustomizeResourceFiles(t)

	for kind, resource := range otherKinds {
		require.Contains(t, files, kind+"_admin_role.yaml")
		require.Contains(t, files, kind+"_viewer_role.yaml")
		require.NotContains(t, files, kind+"_editor_role.yaml")
		require.NoFileExists(t, filepath.Join(rbacDir, kind+"_editor_role.yaml"))

		viewers := decodeRoles(t, readFile(t, filepath.Join(rbacDir, kind+"_viewer_role.yaml")))
		require.Len(t, viewers, 1)
		require.Equal(t, "ClusterRole", viewers[0].Kind)
		require.Equal(t, viewerVerbs, verbsOn(viewers[0], resource), "viewer verbs of %s", kind)
	}
}

func TestKustomizeViewerRolesAreClusterRoles(t *testing.T) {
	t.Parallel()

	for _, file := range kustomizeResourceFiles(t) {
		if !strings.HasSuffix(file, "_viewer_role.yaml") && !strings.HasSuffix(file, "_admin_role.yaml") {
			continue
		}

		for _, role := range decodeRoles(t, readFile(t, filepath.Join(rbacDir, file))) {
			require.Equal(t, "ClusterRole", role.Kind, file)
		}
	}
}

func TestHelmHelperRoles(t *testing.T) {
	t.Parallel()

	roles := renderTemplate(t, "helper_roles.yaml")
	require.Len(t, roles, 2*(len(guardedKinds)+len(otherKinds)))

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

	for kind, resource := range otherKinds {
		prefix := "edp-X-X-" + kind

		var viewer *rbacv1.ClusterRole

		names := sets.New[string]()

		for i, role := range roles {
			if strings.HasSuffix(role.Name, "-"+kind+"-admin") || strings.HasSuffix(role.Name, "-"+kind+"-viewer") {
				names.Insert(role.Name)

				if role.Name == prefix+"-viewer" {
					viewer = &roles[i]
				}
			}
		}

		require.Equal(t, sets.New(prefix+"-admin", prefix+"-viewer"), names, "admin and viewer only for %s", kind)
		require.NotNil(t, viewer)
		require.Equal(t, viewerVerbs, verbsOn(*viewer, resource), "viewer verbs of %s", kind)
	}

	for _, role := range roles {
		require.Equal(t, "ClusterRole", role.Kind)
		require.NotContains(t, role.Name, "editor")
	}
}

// The helper roles are not optional: no template condition and no value may switch them off.
func TestHelmHelperRolesAlwaysRendered(t *testing.T) {
	t.Parallel()

	content := readFile(t, filepath.Join(templateDir, "helper_roles.yaml"))
	require.NotContains(t, content, "{{- if")
	require.NotContains(t, content, "{{ if")
	require.NotContains(t, content, ".Values.helperRoles")
	require.NotContains(t, readFile(t, valuesFile), "helperRoles")
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

func operatorRole(t *testing.T, roles []rbacv1.ClusterRole) rbacv1.ClusterRole {
	t.Helper()

	require.Len(t, roles, 1)

	return roles[0]
}

// The operator reconciles the guarded kinds but never creates or deletes them.
func TestOperatorRoleCannotCreateOrDeleteGuardedKinds(t *testing.T) {
	t.Parallel()

	generated := operatorRole(t, decodeRoles(t, readFile(t, filepath.Join(rbacDir, "role.yaml"))))
	chart := operatorRole(t, renderTemplate(t, "role.yaml"))

	for _, kind := range guardedKinds {
		resource := kind.helmResource
		resources := map[string][]string{
			"main":       {resource},
			"status":     {resource + "/status"},
			"finalizers": {resource + "/finalizers"},
		}

		for source, role := range map[string]rbacv1.ClusterRole{"controller-gen": generated, "helm": chart} {
			main := verbsOn(role, resources["main"]...)
			require.False(t, main.HasAny("create", "delete", "deletecollection", "*"), "%s %s", source, resource)
			require.True(t, main.HasAll("get", "list", "watch", "update", "patch"), "%s %s", source, resource)
			require.True(t, verbsOn(role, resources["status"]...).HasAll("update", "patch"), "%s %s status", source, resource)
			require.True(t, verbsOn(role, resources["finalizers"]...).Has("update"), "%s %s finalizers", source, resource)
		}

		for name, res := range resources {
			require.Equal(t, verbsOn(generated, res...), verbsOn(chart, res...), "%s %s", resource, name)
		}
	}
}

func TestOperatorRoleKeepsCreateAndDeleteOnOtherKinds(t *testing.T) {
	t.Parallel()

	generated := operatorRole(t, decodeRoles(t, readFile(t, filepath.Join(rbacDir, "role.yaml"))))
	chart := operatorRole(t, renderTemplate(t, "role.yaml"))

	for _, resource := range []string{"nexuscleanuppolicies", "nexusroles", "nexusscripts", "nexususers"} {
		require.True(t, verbsOn(generated, resource).HasAll("create", "delete"), "controller-gen %s", resource)
		require.True(t, verbsOn(chart, resource).HasAll("create", "delete"), "helm %s", resource)
	}
}
