package main

import (
	"bytes"
	"encoding/json"
	"io"
)

// structuralDemoResult groups the visible results from the adapter, decorator,
// and facade demonstrations.
type structuralDemoResult struct {
	AdaptedGroup GroupView        `json:"adaptedGroup"`
	AuditLog     string           `json:"auditLog"`
	Onboarding   OnboardingResult `json:"onboarding"`
}

// runStructuralDemo connects the three patterns around one in-memory store.
// It prints their successful results and captured group-write outcomes as JSON.
func runStructuralDemo(output io.Writer) error {
	// The demo uses a local buffer so its decorated writes appear in the result.
	// The same store backs the services and the writer being decorated.
	store := NewMemoryStore()
	var auditLog bytes.Buffer
	auditedWriter := NewAuditedGroupWriter(store, &auditLog)
	groups := NewGroupService(store, auditedWriter)
	profiles := NewProfileService(store)

	// Translate the whole-group total before sending the input to GroupService.
	// For this example, 1501 shared by three people becomes a ceiling of 501.
	adapter := NewLegacyOfferAdapter(groups)
	adaptedGroup, err := adapter.CreateGroup(LegacyRoommateOffer{
		Name:          "Central apartment",
		MonthlyTotal:  1501,
		RoommateLimit: 3,
	})
	if err != nil {
		return err
	}

	// The facade validates first, then creates the profile, group, and membership.
	// Its result keeps each view that was returned by a completed write.
	onboardingFacade := NewOnboardingFacade(profiles, groups)
	onboarding, err := onboardingFacade.Onboard(OnboardingInput{
		Profile: ProfileInput{Name: "Alex", MonthlyBudget: 600},
		Group:   GroupInput{Name: "Quiet shared flat", MaxMonthlyCost: 550, Capacity: 2},
	})
	if err != nil {
		return err
	}

	result := structuralDemoResult{
		AdaptedGroup: adaptedGroup,
		AuditLog:     auditLog.String(),
		Onboarding:   onboarding,
	}
	// Encode only after every example succeeds so the JSON result is complete.
	// The audit buffer contains only group writes routed through the decorator.
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
