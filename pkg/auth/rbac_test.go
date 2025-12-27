package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRBAC_AddRole(t *testing.T) {
	rbac := NewRBAC()

	permissions := []Permission{
		{Resource: "user", Action: "read"},
		{Resource: "user", Action: "write"},
	}

	rbac.AddRole("admin", permissions)

	role, exists := rbac.GetRole("admin")
	assert.True(t, exists)
	assert.NotNil(t, role)
	assert.Equal(t, "admin", role.Name)
	assert.Len(t, role.Permissions, 2)
}

func TestRBAC_AssignRole(t *testing.T) {
	rbac := NewRBAC()

	permissions := []Permission{
		{Resource: "user", Action: "read"},
	}
	rbac.AddRole("user", permissions)

	err := rbac.AssignRole(1, "user")
	assert.NoError(t, err)

	roles := rbac.GetUserRoles(1)
	assert.Contains(t, roles, "user")
}

func TestRBAC_AssignRole_NonExistent(t *testing.T) {
	rbac := NewRBAC()

	err := rbac.AssignRole(1, "nonexistent")
	assert.Error(t, err)
}

func TestRBAC_HasPermission(t *testing.T) {
	rbac := NewRBAC()

	permissions := []Permission{
		{Resource: "user", Action: "read"},
		{Resource: "user", Action: "write"},
	}
	rbac.AddRole("admin", permissions)
	rbac.AssignRole(1, "admin")

	assert.True(t, rbac.HasPermission(1, "user", "read"))
	assert.True(t, rbac.HasPermission(1, "user", "write"))
	assert.False(t, rbac.HasPermission(1, "user", "delete"))
}

func TestRBAC_RemoveRole(t *testing.T) {
	rbac := NewRBAC()

	permissions := []Permission{
		{Resource: "user", Action: "read"},
	}
	rbac.AddRole("user", permissions)
	rbac.AssignRole(1, "user")

	rbac.RemoveRole(1, "user")

	roles := rbac.GetUserRoles(1)
	assert.NotContains(t, roles, "user")
}

func TestRBAC_CheckPermission(t *testing.T) {
	rbac := NewRBAC()

	permissions := []Permission{
		{Resource: "user", Action: "read"},
	}
	rbac.AddRole("user", permissions)
	rbac.AssignRole(1, "user")

	err := rbac.CheckPermission(1, "user", "read")
	assert.NoError(t, err)

	err = rbac.CheckPermission(1, "user", "delete")
	assert.Error(t, err)
}

func TestRBAC_HasAnyPermission(t *testing.T) {
	rbac := NewRBAC()

	permissions := []Permission{
		{Resource: "user", Action: "read"},
	}
	rbac.AddRole("user", permissions)
	rbac.AssignRole(1, "user")

	checkPermissions := []Permission{
		{Resource: "user", Action: "read"},
		{Resource: "user", Action: "delete"},
	}

	assert.True(t, rbac.HasAnyPermission(1, checkPermissions))
}

func TestRBAC_HasAllPermissions(t *testing.T) {
	rbac := NewRBAC()

	permissions := []Permission{
		{Resource: "user", Action: "read"},
		{Resource: "user", Action: "write"},
	}
	rbac.AddRole("admin", permissions)
	rbac.AssignRole(1, "admin")

	checkPermissions := []Permission{
		{Resource: "user", Action: "read"},
		{Resource: "user", Action: "write"},
	}

	assert.True(t, rbac.HasAllPermissions(1, checkPermissions))

	checkPermissions = []Permission{
		{Resource: "user", Action: "read"},
		{Resource: "user", Action: "delete"},
	}

	assert.False(t, rbac.HasAllPermissions(1, checkPermissions))
}
