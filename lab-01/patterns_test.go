package main

import (
	"errors"
	"testing"
)

// TestGroupBuilderRejectsInvalidDraft checks both sides of Build's contract.
// Invalid capacity returns a domain error, while valid input is normalized.
func TestGroupBuilderRejectsInvalidDraft(t *testing.T) {
	_, err := NewGroupBuilder().
		WithName("Too many roommates").
		WithMaxMonthlyCost(700).
		WithCapacity(maxGroupCapacity + 1).
		Build()
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Build() error = %v, want ErrInvalidInput", err)
	}

	input, err := NewGroupBuilder().
		WithName("  Sunny home  ").
		WithMaxMonthlyCost(700).
		WithCapacity(3).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if input.Name != "Sunny home" || input.MaxMonthlyCost != 700 || input.Capacity != 3 {
		t.Fatalf("Build() = %+v, want normalized valid group input", input)
	}
}

// TestGroupPrototypeClonesSettingsWithoutIdentityOrMembers checks snapshot isolation.
// It also verifies that a saved draft receives a fresh ID and starts without members.
func TestGroupPrototypeClonesSettingsWithoutIdentityOrMembers(t *testing.T) {
	source, err := newGroup("group-template", GroupInput{
		Name:           "Budget home",
		MaxMonthlyCost: 450,
		Capacity:       4,
	})
	if err != nil {
		t.Fatal(err)
	}
	source.memberProfileIDs = []string{"profile-original"}

	prototype, err := NewGroupPrototype(source)
	if err != nil {
		t.Fatal(err)
	}
	source.memberProfileIDs[0] = "profile-changed-after-snapshot"

	clone, err := prototype.CloneAsDraft("  Copied home  ")
	if err != nil {
		t.Fatalf("CloneAsDraft() error = %v", err)
	}
	if clone.id != "" {
		t.Fatalf("clone ID = %q, want no ID before saving", clone.id)
	}
	if clone.name != "Copied home" || clone.maxMonthlyCost != 450 || clone.capacity != 4 {
		t.Fatalf("clone settings = %+v, want copied and renamed settings", clone)
	}
	if len(clone.memberProfileIDs) != 0 {
		t.Fatalf("clone has %d members, want none", len(clone.memberProfileIDs))
	}

	clone.memberProfileIDs = append(clone.memberProfileIDs, "profile-copy-only")
	if prototype.template.memberProfileIDs[0] != "profile-original" {
		t.Fatalf("prototype members = %v, want its isolated snapshot", prototype.template.memberProfileIDs)
	}

	store := NewMemoryStore()
	saved, err := NewGroupService(store, store).CreateGroup(clone.input())
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}
	if saved.ID == source.id || saved.ID == "" || saved.MemberCount != 0 {
		t.Fatalf("saved clone = %+v, want a fresh ID and no members", saved)
	}
}

// TestOnboardingPresetFactoriesKeepFamiliesConsistent checks both concrete families.
// Each must return valid, affordable inputs with a distinct group name.
func TestOnboardingPresetFactoriesKeepFamiliesConsistent(t *testing.T) {
	families := []struct {
		name    string
		factory OnboardingPresetFactory
	}{
		{name: "budget", factory: BudgetPresetFactory{}},
		{name: "comfort", factory: ComfortPresetFactory{}},
	}

	groupNames := make(map[string]bool)
	for _, family := range families {
		t.Run(family.name, func(t *testing.T) {
			preset, err := buildOnboardingPreset(family.factory)
			if err != nil {
				t.Fatalf("CreatePreset() error = %v", err)
			}
			if _, err := normalizeGroupInput(preset.Group); err != nil {
				t.Fatalf("preset group is invalid: %v", err)
			}
			if _, err := newProfile("suggested", preset.SuggestedProfile); err != nil {
				t.Fatalf("suggested profile is invalid: %v", err)
			}
			if preset.SuggestedProfile.MonthlyBudget < preset.Group.MaxMonthlyCost {
				t.Fatalf("profile budget %d is below group cost %d", preset.SuggestedProfile.MonthlyBudget, preset.Group.MaxMonthlyCost)
			}
			groupNames[preset.Group.Name] = true
		})
	}
	if len(groupNames) != len(families) {
		t.Fatalf("factories created %d distinct group names, want %d", len(groupNames), len(families))
	}
}
