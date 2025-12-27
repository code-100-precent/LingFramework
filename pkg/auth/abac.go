package auth

import (
	"fmt"
	"sync"
	"time"
)

// Attribute represents an attribute in ABAC
type Attribute struct {
	Key   string
	Value interface{}
}

// String returns string representation of attribute
func (a Attribute) String() string {
	return fmt.Sprintf("%s:%v", a.Key, a.Value)
}

// Policy represents an ABAC policy
type Policy struct {
	ID          string
	Name        string
	Description string
	Subjects    []Attribute // Who (user attributes)
	Resources   []Attribute // What (resource attributes)
	Actions     []string    // Actions allowed
	Effect      string      // "allow" or "deny"
	Conditions  []Condition // Additional conditions
	Priority    int         // Higher priority policies are evaluated first
}

// Condition represents a condition that must be met
type Condition struct {
	Attribute string
	Operator  string // "eq", "ne", "gt", "lt", "ge", "le", "in", "contains"
	Value     interface{}
}

// EvaluateCondition evaluates a condition against attributes
func (c *Condition) EvaluateCondition(attributes map[string]interface{}) bool {
	attrValue, exists := attributes[c.Attribute]
	if !exists {
		return false
	}

	switch c.Operator {
	case "eq":
		return attrValue == c.Value
	case "ne":
		return attrValue != c.Value
	case "gt":
		return compareValues(attrValue, c.Value) > 0
	case "lt":
		return compareValues(attrValue, c.Value) < 0
	case "ge":
		return compareValues(attrValue, c.Value) >= 0
	case "le":
		return compareValues(attrValue, c.Value) <= 0
	case "in":
		if list, ok := c.Value.([]interface{}); ok {
			for _, v := range list {
				if attrValue == v {
					return true
				}
			}
		}
		return false
	case "contains":
		if str, ok := attrValue.(string); ok {
			if valStr, ok := c.Value.(string); ok {
				return contains(str, valStr)
			}
		}
		return false
	default:
		return false
	}
}

// compareValues compares two values
func compareValues(a, b interface{}) int {
	switch aVal := a.(type) {
	case int:
		if bVal, ok := b.(int); ok {
			if aVal > bVal {
				return 1
			} else if aVal < bVal {
				return -1
			}
			return 0
		}
	case int64:
		if bVal, ok := b.(int64); ok {
			if aVal > bVal {
				return 1
			} else if aVal < bVal {
				return -1
			}
			return 0
		}
	case float64:
		if bVal, ok := b.(float64); ok {
			if aVal > bVal {
				return 1
			} else if aVal < bVal {
				return -1
			}
			return 0
		}
	case string:
		if bVal, ok := b.(string); ok {
			if aVal > bVal {
				return 1
			} else if aVal < bVal {
				return -1
			}
			return 0
		}
	case time.Time:
		if bVal, ok := b.(time.Time); ok {
			if aVal.After(bVal) {
				return 1
			} else if aVal.Before(bVal) {
				return -1
			}
			return 0
		}
	}
	return 0
}

// contains checks if a string contains a substring
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 ||
		(len(str) > len(substr) && (str[:len(substr)] == substr ||
			str[len(str)-len(substr):] == substr ||
			containsMiddle(str, substr))))
}

func containsMiddle(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ABAC represents Attribute-Based Access Control manager
type ABAC struct {
	mu       sync.RWMutex
	policies []*Policy
}

// NewABAC creates a new ABAC manager
func NewABAC() *ABAC {
	return &ABAC{
		policies: make([]*Policy, 0),
	}
}

// AddPolicy adds a policy
func (a *ABAC) AddPolicy(policy *Policy) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Insert policy based on priority (higher priority first)
	inserted := false
	for i, p := range a.policies {
		if policy.Priority > p.Priority {
			a.policies = append(a.policies[:i], append([]*Policy{policy}, a.policies[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		a.policies = append(a.policies, policy)
	}
}

// RemovePolicy removes a policy by ID
func (a *ABAC) RemovePolicy(policyID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, policy := range a.policies {
		if policy.ID == policyID {
			a.policies = append(a.policies[:i], a.policies[i+1:]...)
			return
		}
	}
}

// GetPolicy retrieves a policy by ID
func (a *ABAC) GetPolicy(policyID string) (*Policy, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, policy := range a.policies {
		if policy.ID == policyID {
			return policy, true
		}
	}
	return nil, false
}

// ListPolicies returns all policies
func (a *ABAC) ListPolicies() []*Policy {
	a.mu.RLock()
	defer a.mu.RUnlock()

	policies := make([]*Policy, len(a.policies))
	copy(policies, a.policies)
	return policies
}

// CheckAccess checks if access is allowed based on attributes
func (a *ABAC) CheckAccess(subjectAttrs map[string]interface{}, resourceAttrs map[string]interface{}, action string) (bool, *Policy) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Combine all attributes for condition evaluation
	allAttrs := make(map[string]interface{})
	for k, v := range subjectAttrs {
		allAttrs["subject."+k] = v
	}
	for k, v := range resourceAttrs {
		allAttrs["resource."+k] = v
	}

	// Evaluate policies in priority order
	for _, policy := range a.policies {
		if a.matchesPolicy(subjectAttrs, resourceAttrs, action, allAttrs, policy) {
			return policy.Effect == "allow", policy
		}
	}

	// Default deny if no policy matches
	return false, nil
}

// matchesPolicy checks if attributes match a policy
func (a *ABAC) matchesPolicy(subjectAttrs, resourceAttrs map[string]interface{}, action string, allAttrs map[string]interface{}, policy *Policy) bool {
	// Check if action matches
	actionMatched := false
	for _, act := range policy.Actions {
		if act == "*" || act == action {
			actionMatched = true
			break
		}
	}
	if !actionMatched {
		return false
	}

	// Check subject attributes
	if !a.matchAttributes(subjectAttrs, policy.Subjects) {
		return false
	}

	// Check resource attributes
	if !a.matchAttributes(resourceAttrs, policy.Resources) {
		return false
	}

	// Check conditions
	for _, condition := range policy.Conditions {
		if !condition.EvaluateCondition(allAttrs) {
			return false
		}
	}

	return true
}

// matchAttributes checks if attributes match policy attributes
func (a *ABAC) matchAttributes(attrs map[string]interface{}, policyAttrs []Attribute) bool {
	if len(policyAttrs) == 0 {
		return true // Empty policy attributes means match all
	}

	for _, policyAttr := range policyAttrs {
		if policyAttr.Key == "*" {
			continue // Wildcard matches all
		}

		value, exists := attrs[policyAttr.Key]
		if !exists {
			return false
		}

		if policyAttr.Value != "*" && value != policyAttr.Value {
			return false
		}
	}

	return true
}

// CheckAccessWithError checks access and returns error if denied
func (a *ABAC) CheckAccessWithError(subjectAttrs map[string]interface{}, resourceAttrs map[string]interface{}, action string) error {
	allowed, policy := a.CheckAccess(subjectAttrs, resourceAttrs, action)
	if !allowed {
		if policy != nil {
			return fmt.Errorf("access denied by policy %s: %s", policy.ID, policy.Name)
		}
		return fmt.Errorf("access denied: no matching policy")
	}
	return nil
}
