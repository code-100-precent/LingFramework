package auth

import (
	"errors"
	"fmt"
	"sync"
)

// Permission represents a permission
type Permission struct {
	Resource string // Resource name (e.g., "user", "article")
	Action   string // Action name (e.g., "read", "write", "delete")
}

// String returns string representation of permission
func (p Permission) String() string {
	return fmt.Sprintf("%s:%s", p.Resource, p.Action)
}

// Role represents a role with permissions
type Role struct {
	Name        string       // Role name
	Permissions []Permission // Permissions granted to this role
}

// RBAC represents Role-Based Access Control manager
type RBAC struct {
	mu        sync.RWMutex
	roles     map[string]*Role
	userRoles map[uint][]string // userID -> roles
}

// NewRBAC creates a new RBAC manager
func NewRBAC() *RBAC {
	return &RBAC{
		roles:     make(map[string]*Role),
		userRoles: make(map[uint][]string),
	}
}

// AddRole adds a role with permissions
func (r *RBAC) AddRole(roleName string, permissions []Permission) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.roles[roleName] = &Role{
		Name:        roleName,
		Permissions: permissions,
	}
}

// GetRole retrieves a role by name
func (r *RBAC) GetRole(roleName string) (*Role, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	role, ok := r.roles[roleName]
	return role, ok
}

// AssignRole assigns a role to a user
func (r *RBAC) AssignRole(userID uint, roleName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if role exists
	if _, exists := r.roles[roleName]; !exists {
		return fmt.Errorf("role %s does not exist", roleName)
	}

	// Check if user already has this role
	roles := r.userRoles[userID]
	for _, role := range roles {
		if role == roleName {
			return nil // Already assigned
		}
	}

	r.userRoles[userID] = append(r.userRoles[userID], roleName)
	return nil
}

// RemoveRole removes a role from a user
func (r *RBAC) RemoveRole(userID uint, roleName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	roles := r.userRoles[userID]
	newRoles := make([]string, 0, len(roles))
	for _, role := range roles {
		if role != roleName {
			newRoles = append(newRoles, role)
		}
	}
	r.userRoles[userID] = newRoles
}

// GetUserRoles retrieves all roles for a user
func (r *RBAC) GetUserRoles(userID uint) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.userRoles[userID]
}

// HasPermission checks if a user has a specific permission
func (r *RBAC) HasPermission(userID uint, resource, action string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roles := r.userRoles[userID]
	for _, roleName := range roles {
		role, ok := r.roles[roleName]
		if !ok {
			continue
		}

		for _, perm := range role.Permissions {
			if perm.Resource == resource && perm.Action == action {
				return true
			}
		}
	}

	return false
}

// HasAnyPermission checks if a user has any of the specified permissions
func (r *RBAC) HasAnyPermission(userID uint, permissions []Permission) bool {
	for _, perm := range permissions {
		if r.HasPermission(userID, perm.Resource, perm.Action) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if a user has all of the specified permissions
func (r *RBAC) HasAllPermissions(userID uint, permissions []Permission) bool {
	for _, perm := range permissions {
		if !r.HasPermission(userID, perm.Resource, perm.Action) {
			return false
		}
	}
	return true
}

// CheckPermission checks permission and returns error if not allowed
func (r *RBAC) CheckPermission(userID uint, resource, action string) error {
	if !r.HasPermission(userID, resource, action) {
		return fmt.Errorf("user %d does not have permission %s:%s", userID, resource, action)
	}
	return nil
}

// ListRoles returns all available roles
func (r *RBAC) ListRoles() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roles := make([]string, 0, len(r.roles))
	for name := range r.roles {
		roles = append(roles, name)
	}
	return roles
}

// DeleteRole deletes a role (but doesn't remove it from users)
func (r *RBAC) DeleteRole(roleName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.roles[roleName]; !exists {
		return errors.New("role does not exist")
	}

	delete(r.roles, roleName)
	return nil
}

// ClearUserRoles removes all roles from a user
func (r *RBAC) ClearUserRoles(userID uint) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.userRoles, userID)
}

// DefaultPermissions returns commonly used permissions
var DefaultPermissions = struct {
	User   UserPermissions
	Admin  AdminPermissions
	Common CommonPermissions
}{
	User: UserPermissions{
		ReadOwn:   Permission{Resource: "user", Action: "read:own"},
		UpdateOwn: Permission{Resource: "user", Action: "update:own"},
	},
	Admin: AdminPermissions{
		ReadAll:    Permission{Resource: "user", Action: "read:all"},
		UpdateAll:  Permission{Resource: "user", Action: "update:all"},
		DeleteAll:  Permission{Resource: "user", Action: "delete:all"},
		CreateUser: Permission{Resource: "user", Action: "create"},
	},
	Common: CommonPermissions{
		Read:   Permission{Resource: "*", Action: "read"},
		Write:  Permission{Resource: "*", Action: "write"},
		Delete: Permission{Resource: "*", Action: "delete"},
	},
}

// UserPermissions common user permissions
type UserPermissions struct {
	ReadOwn   Permission
	UpdateOwn Permission
}

// AdminPermissions common admin permissions
type AdminPermissions struct {
	ReadAll    Permission
	UpdateAll  Permission
	DeleteAll  Permission
	CreateUser Permission
}

// CommonPermissions wildcard permissions
type CommonPermissions struct {
	Read   Permission
	Write  Permission
	Delete Permission
}
