package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPlanStruct(t *testing.T) {
	now := time.Now()
	plan := Plan{
		ID:          1,
		Name:        "Premium",
		Description: "Premium plan with unlimited access",
		RPM:         1000,
		RPD:         50000,
		IsDefault:   false,
		CreatedAt:   now,
	}

	if plan.ID != 1 {
		t.Errorf("Expected ID 1, got %d", plan.ID)
	}
	if plan.Name != "Premium" {
		t.Errorf("Expected Name 'Premium', got '%s'", plan.Name)
	}
	if plan.Description != "Premium plan with unlimited access" {
		t.Errorf("Expected Description 'Premium plan with unlimited access', got '%s'", plan.Description)
	}
	if plan.RPM != 1000 {
		t.Errorf("Expected RPM 1000, got %d", plan.RPM)
	}
	if plan.RPD != 50000 {
		t.Errorf("Expected RPD 50000, got %d", plan.RPD)
	}
	if plan.IsDefault {
		t.Error("Expected IsDefault to be false")
	}
}

func TestPlanJSONSerialization(t *testing.T) {
	now := time.Now()
	plan := Plan{
		ID:          1,
		Name:        "Basic",
		Description: "Basic plan",
		RPM:         100,
		RPD:         1000,
		IsDefault:   true,
		CreatedAt:   now,
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("Failed to marshal Plan: %v", err)
	}

	jsonStr := string(data)
	if !contains(jsonStr, "Basic") {
		t.Error("Name should be serialized")
	}
	if !contains(jsonStr, "is_default") {
		t.Error("is_default should be serialized")
	}
}

func TestPlanCreate(t *testing.T) {
	create := PlanCreate{
		Name:        "Enterprise",
		Description: "Enterprise plan",
		RPM:         -1, // unlimited
		RPD:         -1, // unlimited
		IsDefault:   false,
	}

	if create.Name != "Enterprise" {
		t.Errorf("Expected Name 'Enterprise', got '%s'", create.Name)
	}
	if create.RPM != -1 {
		t.Errorf("Expected RPM -1 (unlimited), got %d", create.RPM)
	}
	if create.RPD != -1 {
		t.Errorf("Expected RPD -1 (unlimited), got %d", create.RPD)
	}
}

func TestPlanUpdate(t *testing.T) {
	name := "Updated Plan"
	description := "Updated description"
	rpm := 2000
	rpd := 100000
	isDefault := true

	update := PlanUpdate{
		Name:        &name,
		Description: &description,
		RPM:         &rpm,
		RPD:         &rpd,
		IsDefault:   &isDefault,
	}

	if update.Name == nil || *update.Name != "Updated Plan" {
		t.Error("Expected Name to be 'Updated Plan'")
	}
	if update.Description == nil || *update.Description != "Updated description" {
		t.Error("Expected Description to be 'Updated description'")
	}
	if update.RPM == nil || *update.RPM != 2000 {
		t.Error("Expected RPM to be 2000")
	}
	if update.RPD == nil || *update.RPD != 100000 {
		t.Error("Expected RPD to be 100000")
	}
	if update.IsDefault == nil || !*update.IsDefault {
		t.Error("Expected IsDefault to be true")
	}
}

func TestPlanUpdatePartial(t *testing.T) {
	name := "Partial Update"
	update := PlanUpdate{
		Name: &name,
	}

	if update.Name == nil || *update.Name != "Partial Update" {
		t.Error("Expected Name to be 'Partial Update'")
	}
	if update.Description != nil {
		t.Error("Expected Description to be nil")
	}
	if update.RPM != nil {
		t.Error("Expected RPM to be nil")
	}
	if update.RPD != nil {
		t.Error("Expected RPD to be nil")
	}
	if update.IsDefault != nil {
		t.Error("Expected IsDefault to be nil")
	}
}

func TestPlanWithUnlimitedRates(t *testing.T) {
	plan := Plan{
		ID:          1,
		Name:        "Unlimited",
		Description: "Unlimited access",
		RPM:         -1,
		RPD:         -1,
		IsDefault:   false,
	}

	if plan.RPM != -1 {
		t.Errorf("Expected RPM -1, got %d", plan.RPM)
	}
	if plan.RPD != -1 {
		t.Errorf("Expected RPD -1, got %d", plan.RPD)
	}
}
