package main

import (
	"bytes"
	"encoding/json"
	"io"
)

type structuralDemoResult struct {
	AdaptedGroup GroupView        `json:"adaptedGroup"`
	AuditLog     string           `json:"auditLog"`
	Onboarding   OnboardingResult `json:"onboarding"`
}

func runStructuralDemo(output io.Writer) error {
	store := NewMemoryStore()
	var auditLog bytes.Buffer
	groupWriter := NewAuditedGroupWriter(store, &auditLog)
	groups := NewGroupService(store, groupWriter)
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

	onboarding, err := NewOnboardingFacade(profiles, groups).Onboard(OnboardingInput{
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
