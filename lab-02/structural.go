package main

import (
	"fmt"
	"io"
)

// LegacyRoommateOffer uses the older total-cost shape. MonthlyTotal is the
// whole group's monthly housing cost, and RoommateLimit is the intended size.
type LegacyRoommateOffer struct {
	Name          string
	MonthlyTotal  int
	RoommateLimit int
}

// LegacyOfferAdapter converts the older offer shape into the current group
// service contract. Each member's listed cost rounds up to a whole currency
// unit so the converted limit does not understate the shared total.
type LegacyOfferAdapter struct {
	groups *GroupService
}

func NewLegacyOfferAdapter(groups *GroupService) *LegacyOfferAdapter {
	return &LegacyOfferAdapter{groups: groups}
}

func (adapter *LegacyOfferAdapter) CreateGroup(offer LegacyRoommateOffer) (GroupView, error) {
	input, err := groupInputFromLegacyOffer(offer)
	if err != nil {
		return GroupView{}, err
	}
	return adapter.groups.CreateGroup(input)
}

func groupInputFromLegacyOffer(offer LegacyRoommateOffer) (GroupInput, error) {
	if offer.MonthlyTotal <= 0 {
		return GroupInput{}, fmt.Errorf("%w: monthly total must be greater than zero", ErrInvalidInput)
	}
	if offer.RoommateLimit <= 0 {
		return GroupInput{}, fmt.Errorf("%w: roommate limit must be greater than zero", ErrInvalidInput)
	}
	if offer.RoommateLimit > maxGroupCapacity {
		return GroupInput{}, fmt.Errorf("%w: roommate limit cannot exceed %d people", ErrInvalidInput, maxGroupCapacity)
	}

	perPersonCost := offer.MonthlyTotal / offer.RoommateLimit
	if offer.MonthlyTotal%offer.RoommateLimit != 0 {
		perPersonCost++
	}

	return GroupInput{
		Name:           offer.Name,
		MaxMonthlyCost: perPersonCost,
		Capacity:       offer.RoommateLimit,
	}, nil
}

// AuditedGroupWriter decorates group mutations with outcome-only audit lines.
// The log is best-effort: an output failure never changes the domain result.
type AuditedGroupWriter struct {
	next   GroupWriter
	output io.Writer
}

func NewAuditedGroupWriter(next GroupWriter, output io.Writer) *AuditedGroupWriter {
	if output == nil {
		output = io.Discard
	}
	return &AuditedGroupWriter{next: next, output: output}
}

func (writer *AuditedGroupWriter) CreateGroup(input GroupInput) (GroupView, error) {
	group, err := writer.next.CreateGroup(input)
	writer.record("CreateGroup", err)
	return group, err
}

func (writer *AuditedGroupWriter) JoinGroup(groupID, profileID string) (GroupDetail, error) {
	group, err := writer.next.JoinGroup(groupID, profileID)
	writer.record("JoinGroup", err)
	return group, err
}

func (writer *AuditedGroupWriter) record(operation string, err error) {
	result := "success"
	if err != nil {
		result = "error"
	}
	_, _ = fmt.Fprintf(writer.output, "operation=%s result=%s\n", operation, result)
}

type OnboardingInput struct {
	Profile ProfileInput `json:"profile"`
	Group   GroupInput   `json:"group"`
}

type OnboardingResult struct {
	Profile     ProfileView  `json:"profile"`
	Group       GroupView    `json:"group"`
	JoinedGroup *GroupDetail `json:"joinedGroup,omitempty"`
}

// OnboardingFacade coordinates the existing profile and group services. It
// validates all known constraints before writing, then reports completed
// steps if a later service call fails. The three writes are not transactional.
type OnboardingFacade struct {
	profiles *ProfileService
	groups   *GroupService
}

func NewOnboardingFacade(profiles *ProfileService, groups *GroupService) *OnboardingFacade {
	return &OnboardingFacade{profiles: profiles, groups: groups}
}

func (facade *OnboardingFacade) Onboard(input OnboardingInput) (OnboardingResult, error) {
	profile, err := newProfile("validation", input.Profile)
	if err != nil {
		return OnboardingResult{}, err
	}
	group, err := newGroup("validation", input.Group)
	if err != nil {
		return OnboardingResult{}, err
	}
	if profile.monthlyBudget < group.maxMonthlyCost {
		return OnboardingResult{}, fmt.Errorf("%w: monthly budget is below the group's cost", ErrBudgetTooLow)
	}

	var result OnboardingResult
	result.Profile, err = facade.profiles.CreateProfile(input.Profile)
	if err != nil {
		return result, err
	}
	result.Group, err = facade.groups.CreateGroup(input.Group)
	if err != nil {
		return result, err
	}
	joined, err := facade.groups.JoinGroup(result.Group.ID, result.Profile.ID)
	if err != nil {
		return result, err
	}
	result.JoinedGroup = &joined
	return result, nil
}
