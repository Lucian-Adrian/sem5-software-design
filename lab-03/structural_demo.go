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

// runStructuralDemo composes the patterns around one store:
// 1. Wrap group writes so the demo can show outcome records.
// 2. Adapt a legacy offer into a group.
// 3. Onboard a profile through the existing services.
// 4. Encode the values and records as indented JSON.
func runStructuralDemo(output io.Writer) error {
	store := NewMemoryStore()
	var auditLog bytes.Buffer
	auditedWriter := NewAuditedGroupWriter(store, &auditLog)
	groups := NewGroupService(store, auditedWriter)
	profiles := NewProfileService(store)

	adapter := NewLegacyOfferAdapter(groups)
	adaptedGroup, err := adapter.CreateGroup(LegacyRoommateOffer{
		Name:          "Central apartment",
		MonthlyTotal:  1501,
		RoommateLimit: 3,
	})
	if err != nil {
		return err
	}

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
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
