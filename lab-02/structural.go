package main

import (
	"fmt"
	"io"
)

// LegacyRoommateOffer is the older feed's representation of a group: one
// whole monthly cost and the number of roommates who share it.
type LegacyRoommateOffer struct {
	Name          string
	MonthlyTotal  int
	RoommateLimit int
}

// LegacyOfferAdapter translates a legacy offer into the current group service
// input. The service remains responsible for domain validation and storage.
type LegacyOfferAdapter struct {
	groups *GroupService
}

// NewLegacyOfferAdapter stores the existing group service for converted offers.
// Creation still follows the service's usual validation and storage path.
func NewLegacyOfferAdapter(groups *GroupService) *LegacyOfferAdapter {
	return &LegacyOfferAdapter{groups: groups}
}

// CreateGroup converts the offer, then delegates creation through the group
// service so the existing domain rules still apply.
func (adapter *LegacyOfferAdapter) CreateGroup(offer LegacyRoommateOffer) (GroupView, error) {
	input, err := groupInputFromLegacyOffer(offer)
	if err != nil {
		return GroupView{}, err
	}
	return adapter.groups.CreateGroup(input)
}

// groupInputFromLegacyOffer converts a positive total to a whole-unit ceiling.
// It checks capacity before division and leaves name validation to the service.
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

	// A remainder means the shared whole-unit limit must round up by one.
	// Quotient and remainder avoid adding to a total near the largest int value.
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

// AuditedGroupWriter decorates group mutations with records that contain only
// the operation and outcome while preserving the wrapped writer's returned
// value and error.
type AuditedGroupWriter struct {
	next   GroupWriter
	output io.Writer
}

// NewAuditedGroupWriter wraps a writer and sends records to output. A nil
// output discards records without changing the wrapped operation.
func NewAuditedGroupWriter(next GroupWriter, output io.Writer) *AuditedGroupWriter {
	if output == nil {
		output = io.Discard
	}
	return &AuditedGroupWriter{next: next, output: output}
}

// CreateGroup delegates creation, records its outcome, and returns the same
// view and error supplied by the wrapped writer.
func (writer *AuditedGroupWriter) CreateGroup(input GroupInput) (GroupView, error) {
	group, err := writer.next.CreateGroup(input)
	writer.record("CreateGroup", err)
	return group, err
}

// JoinGroup delegates membership, records its outcome, and returns the same
// detail and error supplied by the wrapped writer.
func (writer *AuditedGroupWriter) JoinGroup(groupID, profileID string) (GroupDetail, error) {
	group, err := writer.next.JoinGroup(groupID, profileID)
	writer.record("JoinGroup", err)
	return group, err
}

// record maps the returned error to a success or error outcome and attempts
// one line of output. Logging failure is ignored by design.
func (writer *AuditedGroupWriter) record(operation string, err error) {
	// Keep the record small and leave the wrapped result as the source of truth.
	// A failed output write is best effort and must not replace the operation.
	result := "success"
	if err != nil {
		result = "error"
	}
	_, _ = fmt.Fprintf(writer.output, "operation=%s result=%s\n", operation, result)
}

// OnboardingInput groups the profile and group data needed for one onboarding
// operation.
type OnboardingInput struct {
	Profile ProfileInput `json:"profile"`
	Group   GroupInput   `json:"group"`
}

// OnboardingResult returns the views from completed steps. JoinedGroup stays
// nil when an error occurs before membership succeeds.
type OnboardingResult struct {
	Profile     ProfileView  `json:"profile"`
	Group       GroupView    `json:"group"`
	JoinedGroup *GroupDetail `json:"joinedGroup,omitempty"`
}

// OnboardingFacade coordinates the existing profile and group services. It
// validates known constraints before writing, then reports completed steps
// if a later call fails. The sequence is not transactional.
type OnboardingFacade struct {
	profiles *ProfileService
	groups   *GroupService
}

// NewOnboardingFacade keeps the existing services behind one onboarding call.
// Their storage and validation behavior remain owned by the services.
func NewOnboardingFacade(profiles *ProfileService, groups *GroupService) *OnboardingFacade {
	return &OnboardingFacade{profiles: profiles, groups: groups}
}

// Onboard validates profile, group, and affordability constraints first. It
// then creates the profile, creates the group, and joins them in that order.
// Each returned view records a completed step if a later call fails.
func (facade *OnboardingFacade) Onboard(input OnboardingInput) (OnboardingResult, error) {
	// Validate with temporary IDs before any service call can write a record.
	// These domain constructors apply the same field rules as normal creation.
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

	// Save each view after its write so a later error reports completed work.
	// The three calls remain separate writes and do not roll back one another.
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
