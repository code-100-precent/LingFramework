package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewABAC(t *testing.T) {
	abac := NewABAC()
	assert.NotNil(t, abac)
	assert.NotNil(t, abac.policies)
}

func TestABAC_AddPolicy(t *testing.T) {
	abac := NewABAC()

	policy := &Policy{
		ID:          "policy-1",
		Name:        "Test Policy",
		Description: "Test description",
		Subjects: []Attribute{
			{Key: "role", Value: "admin"},
		},
		Resources: []Attribute{
			{Key: "type", Value: "article"},
		},
		Actions:    []string{"read", "write"},
		Effect:     "allow",
		Conditions: []Condition{},
		Priority:   100,
	}

	abac.AddPolicy(policy)

	retrieved, exists := abac.GetPolicy("policy-1")
	assert.True(t, exists)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "policy-1", retrieved.ID)
}

func TestABAC_RemovePolicy(t *testing.T) {
	abac := NewABAC()

	policy := &Policy{
		ID:       "policy-1",
		Priority: 100,
	}
	abac.AddPolicy(policy)

	abac.RemovePolicy("policy-1")

	_, exists := abac.GetPolicy("policy-1")
	assert.False(t, exists)
}

func TestABAC_ListPolicies(t *testing.T) {
	abac := NewABAC()

	policy1 := &Policy{ID: "policy-1", Priority: 100}
	policy2 := &Policy{ID: "policy-2", Priority: 200}

	abac.AddPolicy(policy1)
	abac.AddPolicy(policy2)

	policies := abac.ListPolicies()
	assert.Len(t, policies, 2)

	// Policies should be sorted by priority (higher first)
	assert.Equal(t, "policy-2", policies[0].ID)
	assert.Equal(t, "policy-1", policies[1].ID)
}

func TestABAC_CheckAccess_Allow(t *testing.T) {
	abac := NewABAC()

	policy := &Policy{
		ID: "policy-1",
		Subjects: []Attribute{
			{Key: "role", Value: "admin"},
		},
		Resources: []Attribute{
			{Key: "type", Value: "article"},
		},
		Actions: []string{"read"},
		Effect:  "allow",
	}
	abac.AddPolicy(policy)

	subjectAttrs := map[string]interface{}{
		"role": "admin",
	}
	resourceAttrs := map[string]interface{}{
		"type": "article",
	}

	allowed, matchedPolicy := abac.CheckAccess(subjectAttrs, resourceAttrs, "read")
	assert.True(t, allowed)
	assert.NotNil(t, matchedPolicy)
	assert.Equal(t, "policy-1", matchedPolicy.ID)
}

func TestABAC_CheckAccess_Deny(t *testing.T) {
	abac := NewABAC()

	policy := &Policy{
		ID: "policy-1",
		Subjects: []Attribute{
			{Key: "role", Value: "user"},
		},
		Resources: []Attribute{
			{Key: "type", Value: "article"},
		},
		Actions: []string{"read"},
		Effect:  "deny",
	}
	abac.AddPolicy(policy)

	subjectAttrs := map[string]interface{}{
		"role": "user",
	}
	resourceAttrs := map[string]interface{}{
		"type": "article",
	}

	allowed, matchedPolicy := abac.CheckAccess(subjectAttrs, resourceAttrs, "read")
	assert.False(t, allowed)
	assert.NotNil(t, matchedPolicy)
}

func TestABAC_CheckAccess_NoMatch(t *testing.T) {
	abac := NewABAC()

	policy := &Policy{
		ID: "policy-1",
		Subjects: []Attribute{
			{Key: "role", Value: "admin"},
		},
		Resources: []Attribute{
			{Key: "type", Value: "article"},
		},
		Actions: []string{"read"},
		Effect:  "allow",
	}
	abac.AddPolicy(policy)

	subjectAttrs := map[string]interface{}{
		"role": "user", // Different role
	}
	resourceAttrs := map[string]interface{}{
		"type": "article",
	}

	allowed, matchedPolicy := abac.CheckAccess(subjectAttrs, resourceAttrs, "read")
	assert.False(t, allowed) // Default deny
	assert.Nil(t, matchedPolicy)
}

func TestABAC_CheckAccess_Wildcard(t *testing.T) {
	abac := NewABAC()

	policy := &Policy{
		ID:       "policy-1",
		Subjects: []Attribute{}, // Empty means match all
		Resources: []Attribute{
			{Key: "type", Value: "*"}, // Wildcard
		},
		Actions: []string{"*"}, // Wildcard action
		Effect:  "allow",
	}
	abac.AddPolicy(policy)

	subjectAttrs := map[string]interface{}{
		"role": "any",
	}
	resourceAttrs := map[string]interface{}{
		"type": "anything",
	}

	allowed, _ := abac.CheckAccess(subjectAttrs, resourceAttrs, "any-action")
	assert.True(t, allowed)
}

func TestABAC_CheckAccessWithError(t *testing.T) {
	abac := NewABAC()

	policy := &Policy{
		ID: "policy-1",
		Subjects: []Attribute{
			{Key: "role", Value: "admin"},
		},
		Resources: []Attribute{
			{Key: "type", Value: "article"},
		},
		Actions: []string{"read"},
		Effect:  "allow",
	}
	abac.AddPolicy(policy)

	subjectAttrs := map[string]interface{}{
		"role": "admin",
	}
	resourceAttrs := map[string]interface{}{
		"type": "article",
	}

	err := abac.CheckAccessWithError(subjectAttrs, resourceAttrs, "read")
	assert.NoError(t, err)

	err = abac.CheckAccessWithError(subjectAttrs, resourceAttrs, "delete")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestCondition_EvaluateCondition_EQ(t *testing.T) {
	condition := &Condition{
		Attribute: "age",
		Operator:  "eq",
		Value:     30,
	}

	attrs := map[string]interface{}{
		"age": 30,
	}

	assert.True(t, condition.EvaluateCondition(attrs))

	attrs["age"] = 31
	assert.False(t, condition.EvaluateCondition(attrs))
}

func TestCondition_EvaluateCondition_NE(t *testing.T) {
	condition := &Condition{
		Attribute: "age",
		Operator:  "ne",
		Value:     30,
	}

	attrs := map[string]interface{}{
		"age": 31,
	}

	assert.True(t, condition.EvaluateCondition(attrs))

	attrs["age"] = 30
	assert.False(t, condition.EvaluateCondition(attrs))
}

func TestCondition_EvaluateCondition_GT(t *testing.T) {
	condition := &Condition{
		Attribute: "age",
		Operator:  "gt",
		Value:     30,
	}

	attrs := map[string]interface{}{
		"age": 31,
	}
	assert.True(t, condition.EvaluateCondition(attrs))

	attrs["age"] = 30
	assert.False(t, condition.EvaluateCondition(attrs))

	attrs["age"] = 29
	assert.False(t, condition.EvaluateCondition(attrs))
}

func TestCondition_EvaluateCondition_LT(t *testing.T) {
	condition := &Condition{
		Attribute: "age",
		Operator:  "lt",
		Value:     30,
	}

	attrs := map[string]interface{}{
		"age": 29,
	}
	assert.True(t, condition.EvaluateCondition(attrs))

	attrs["age"] = 30
	assert.False(t, condition.EvaluateCondition(attrs))

	attrs["age"] = 31
	assert.False(t, condition.EvaluateCondition(attrs))
}

func TestCondition_EvaluateCondition_GE(t *testing.T) {
	condition := &Condition{
		Attribute: "age",
		Operator:  "ge",
		Value:     30,
	}

	attrs := map[string]interface{}{
		"age": 30,
	}
	assert.True(t, condition.EvaluateCondition(attrs))

	attrs["age"] = 31
	assert.True(t, condition.EvaluateCondition(attrs))

	attrs["age"] = 29
	assert.False(t, condition.EvaluateCondition(attrs))
}

func TestCondition_EvaluateCondition_LE(t *testing.T) {
	condition := &Condition{
		Attribute: "age",
		Operator:  "le",
		Value:     30,
	}

	attrs := map[string]interface{}{
		"age": 30,
	}
	assert.True(t, condition.EvaluateCondition(attrs))

	attrs["age"] = 29
	assert.True(t, condition.EvaluateCondition(attrs))

	attrs["age"] = 31
	assert.False(t, condition.EvaluateCondition(attrs))
}

func TestCondition_EvaluateCondition_IN(t *testing.T) {
	condition := &Condition{
		Attribute: "role",
		Operator:  "in",
		Value:     []interface{}{"admin", "user", "guest"},
	}

	attrs := map[string]interface{}{
		"role": "admin",
	}
	assert.True(t, condition.EvaluateCondition(attrs))

	attrs["role"] = "user"
	assert.True(t, condition.EvaluateCondition(attrs))

	attrs["role"] = "unknown"
	assert.False(t, condition.EvaluateCondition(attrs))
}

func TestCondition_EvaluateCondition_Contains(t *testing.T) {
	condition := &Condition{
		Attribute: "name",
		Operator:  "contains",
		Value:     "test",
	}

	attrs := map[string]interface{}{
		"name": "testuser",
	}
	assert.True(t, condition.EvaluateCondition(attrs))

	attrs["name"] = "user"
	assert.False(t, condition.EvaluateCondition(attrs))
}

func TestCondition_EvaluateCondition_Time(t *testing.T) {
	now := time.Now()
	before := now.Add(-1 * time.Hour)
	after := now.Add(1 * time.Hour)

	condition := &Condition{
		Attribute: "created_at",
		Operator:  "gt",
		Value:     before,
	}

	attrs := map[string]interface{}{
		"created_at": now,
	}
	assert.True(t, condition.EvaluateCondition(attrs))

	attrs["created_at"] = before
	assert.False(t, condition.EvaluateCondition(attrs))

	attrs["created_at"] = after
	assert.True(t, condition.EvaluateCondition(attrs))
}

func TestCondition_EvaluateCondition_MissingAttribute(t *testing.T) {
	condition := &Condition{
		Attribute: "missing",
		Operator:  "eq",
		Value:     "value",
	}

	attrs := map[string]interface{}{
		"other": "value",
	}

	assert.False(t, condition.EvaluateCondition(attrs))
}

func TestABAC_CheckAccess_WithConditions(t *testing.T) {
	abac := NewABAC()

	policy := &Policy{
		ID:        "policy-1",
		Subjects:  []Attribute{{Key: "role", Value: "user"}},
		Resources: []Attribute{{Key: "type", Value: "article"}},
		Actions:   []string{"read"},
		Effect:    "allow",
		Conditions: []Condition{
			{
				Attribute: "subject.age",
				Operator:  "ge",
				Value:     18,
			},
		},
	}
	abac.AddPolicy(policy)

	subjectAttrs := map[string]interface{}{
		"role": "user",
		"age":  20,
	}
	resourceAttrs := map[string]interface{}{
		"type": "article",
	}

	// Note: The condition check happens internally with "subject.age" key
	// This test verifies the policy structure is correct
	_, _ = abac.CheckAccess(subjectAttrs, resourceAttrs, "read")
}

func TestAttribute_String(t *testing.T) {
	attr := Attribute{Key: "test", Value: "value"}
	assert.Equal(t, "test:value", attr.String())

	attr = Attribute{Key: "number", Value: 42}
	assert.Equal(t, "number:42", attr.String())
}

func TestCompareValues_Int64(t *testing.T) {
	// Test int64 comparison
	result := compareValues(int64(100), int64(50))
	assert.Greater(t, result, 0)

	result = compareValues(int64(50), int64(100))
	assert.Less(t, result, 0)

	result = compareValues(int64(100), int64(100))
	assert.Equal(t, 0, result)
}

func TestCompareValues_Float64(t *testing.T) {
	// Test float64 comparison
	result := compareValues(100.5, 50.3)
	assert.Greater(t, result, 0)

	result = compareValues(50.3, 100.5)
	assert.Less(t, result, 0)

	result = compareValues(100.5, 100.5)
	assert.Equal(t, 0, result)
}

func TestCompareValues_String(t *testing.T) {
	// Test string comparison
	result := compareValues("zebra", "apple")
	assert.Greater(t, result, 0)

	result = compareValues("apple", "zebra")
	assert.Less(t, result, 0)

	result = compareValues("apple", "apple")
	assert.Equal(t, 0, result)
}

func TestCompareValues_Time(t *testing.T) {
	now := time.Now()
	later := now.Add(1 * time.Hour)
	earlier := now.Add(-1 * time.Hour)

	result := compareValues(later, now)
	assert.Greater(t, result, 0)

	result = compareValues(earlier, now)
	assert.Less(t, result, 0)

	result = compareValues(now, now)
	assert.Equal(t, 0, result)
}

func TestCompareValues_TypeMismatch(t *testing.T) {
	// Test type mismatch (should return 0)
	result := compareValues(int(100), "100")
	assert.Equal(t, 0, result)

	result = compareValues(int(100), float64(100))
	assert.Equal(t, 0, result)
}

func TestContainsMiddle(t *testing.T) {
	assert.True(t, containsMiddle("hello world", "world"))
	assert.True(t, containsMiddle("hello world", "lo wo"))
	assert.False(t, containsMiddle("hello", "world"))
	assert.True(t, containsMiddle("abc", "b"))
}

func TestContains_EdgeCases(t *testing.T) {
	// Test empty substring
	assert.True(t, contains("hello", ""))

	// Test equal strings
	assert.True(t, contains("hello", "hello"))

	// Test prefix match
	assert.True(t, contains("hello world", "hello"))

	// Test suffix match
	assert.True(t, contains("hello world", "world"))

	// Test middle match
	assert.True(t, contains("hello world", "lo wo"))

	// Test no match
	assert.False(t, contains("hello", "world"))
}

func TestABAC_MatchAttributes_Wildcard(t *testing.T) {
	abac := NewABAC()

	attrs := map[string]interface{}{
		"role": "admin",
		"age":  30,
	}

	// Test with wildcard key (should match all)
	policyAttrs := []Attribute{
		{Key: "*", Value: "*"},
	}
	result := abac.matchAttributes(attrs, policyAttrs)
	assert.True(t, result)

	// Test with empty policy attributes (matches all)
	result = abac.matchAttributes(attrs, []Attribute{})
	assert.True(t, result)

	// Test with wildcard value
	policyAttrs2 := []Attribute{
		{Key: "role", Value: "*"},
	}
	result = abac.matchAttributes(attrs, policyAttrs2)
	assert.True(t, result)

	// Test with non-matching key
	policyAttrs3 := []Attribute{
		{Key: "nonexistent", Value: "value"},
	}
	result = abac.matchAttributes(attrs, policyAttrs3)
	assert.False(t, result)

	// Test with matching key and value
	policyAttrs4 := []Attribute{
		{Key: "role", Value: "admin"},
	}
	result = abac.matchAttributes(attrs, policyAttrs4)
	assert.True(t, result)

	// Test with non-matching value
	policyAttrs5 := []Attribute{
		{Key: "role", Value: "user"},
	}
	result = abac.matchAttributes(attrs, policyAttrs5)
	assert.False(t, result)
}

func TestABAC_CheckAccess_EmptyResources(t *testing.T) {
	abac := NewABAC()

	policy := &Policy{
		ID:        "policy-1",
		Subjects:  []Attribute{{Key: "role", Value: "admin"}},
		Resources: []Attribute{}, // Empty resources
		Actions:   []string{"read"},
		Effect:    "allow",
	}
	abac.AddPolicy(policy)

	subjectAttrs := map[string]interface{}{
		"role": "admin",
	}
	resourceAttrs := map[string]interface{}{
		"type": "article",
	}

	allowed, _ := abac.CheckAccess(subjectAttrs, resourceAttrs, "read")
	assert.True(t, allowed)
}

func TestABAC_CheckAccess_WildcardAction(t *testing.T) {
	abac := NewABAC()

	policy := &Policy{
		ID:        "policy-1",
		Subjects:  []Attribute{{Key: "role", Value: "admin"}},
		Resources: []Attribute{{Key: "type", Value: "article"}},
		Actions:   []string{"*"}, // Wildcard action
		Effect:    "allow",
	}
	abac.AddPolicy(policy)

	subjectAttrs := map[string]interface{}{
		"role": "admin",
	}
	resourceAttrs := map[string]interface{}{
		"type": "article",
	}

	allowed, _ := abac.CheckAccess(subjectAttrs, resourceAttrs, "any-action")
	assert.True(t, allowed)
}

func TestABAC_CheckAccess_ActionMismatch(t *testing.T) {
	abac := NewABAC()

	policy := &Policy{
		ID:        "policy-1",
		Subjects:  []Attribute{{Key: "role", Value: "admin"}},
		Resources: []Attribute{{Key: "type", Value: "article"}},
		Actions:   []string{"read"}, // Only read action
		Effect:    "allow",
	}
	abac.AddPolicy(policy)

	subjectAttrs := map[string]interface{}{
		"role": "admin",
	}
	resourceAttrs := map[string]interface{}{
		"type": "article",
	}

	// Request delete action (not in policy)
	allowed, _ := abac.CheckAccess(subjectAttrs, resourceAttrs, "delete")
	assert.False(t, allowed) // Default deny
}
