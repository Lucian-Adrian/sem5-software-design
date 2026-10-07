package main

// GroupBuilder assembles one group input:
// 1. collect the name, cost, and capacity;
// 2. validate the completed draft before returning it.
type GroupBuilder struct {
	input GroupInput
}

// NewGroupBuilder starts an empty group input draft.
// Add settings with With... methods, then call Build to validate them together.
func NewGroupBuilder() GroupBuilder {
	return GroupBuilder{}
}

// WithName records the group's name in this builder value.
// The returned copy carries the name into the next setter or Build call.
func (builder GroupBuilder) WithName(name string) GroupBuilder {
	builder.input.Name = name
	return builder
}

// WithMaxMonthlyCost records the group's monthly cost in this builder value.
// It returns the updated copy so the next call receives the setting.
func (builder GroupBuilder) WithMaxMonthlyCost(amount int) GroupBuilder {
	builder.input.MaxMonthlyCost = amount
	return builder
}

// WithCapacity records the maximum number of people in this builder value.
// It returns the updated copy so the next call receives the setting.
func (builder GroupBuilder) WithCapacity(people int) GroupBuilder {
	builder.input.Capacity = people
	return builder
}

// Build applies the same normalization rules used at the domain boundary.
// It returns a validated input draft; a service must still save it as a record.
func (builder GroupBuilder) Build() (GroupInput, error) {
	return normalizeGroupInput(builder.input)
}

// OnboardingPreset keeps a family's related creation inputs together.
// The service creates the stored records later and assigns their IDs.
type OnboardingPreset struct {
	Group            GroupInput   `json:"group"`
	SuggestedProfile ProfileInput `json:"suggestedProfile"`
}

// OnboardingPresetFactory creates both inputs needed by one onboarding family.
// buildOnboardingPreset checks that the returned pair is valid and affordable.
type OnboardingPresetFactory interface {
	CreateGroupDraft() (GroupInput, error)
	CreateSuggestedProfile() (ProfileInput, error)
}

// BudgetPresetFactory creates the lower-cost onboarding family.
// Its methods return inputs; the demo saves the group and profile through services.
type BudgetPresetFactory struct{}

// CreateGroupDraft builds and validates the Budget family's group input.
// The builder returns input data; the group service later saves the record.
func (BudgetPresetFactory) CreateGroupDraft() (GroupInput, error) {
	return NewGroupBuilder().
		WithName("Budget shared home").
		WithMaxMonthlyCost(450).
		WithCapacity(4).
		Build()
}

// CreateSuggestedProfile validates the Budget family's profile suggestion.
// It returns an input without an ID for the profile service to save.
func (BudgetPresetFactory) CreateSuggestedProfile() (ProfileInput, error) {
	return buildProfileSuggestion("Budget renter", 500)
}

// ComfortPresetFactory creates the higher-cost onboarding family.
// Its methods return inputs; the demo saves the group and profile through services.
type ComfortPresetFactory struct{}

// CreateGroupDraft builds and validates the Comfort family's group input.
// The builder returns input data; the group service later saves the record.
func (ComfortPresetFactory) CreateGroupDraft() (GroupInput, error) {
	return NewGroupBuilder().
		WithName("Comfort shared home").
		WithMaxMonthlyCost(700).
		WithCapacity(2).
		Build()
}

// CreateSuggestedProfile validates the Comfort family's profile suggestion.
// It returns an input without an ID for the profile service to save.
func (ComfortPresetFactory) CreateSuggestedProfile() (ProfileInput, error) {
	return buildProfileSuggestion("Comfort renter", 900)
}

// buildOnboardingPreset requests and validates both inputs from one factory.
// It returns the pair only when the profile budget covers the group cost.
func buildOnboardingPreset(factory OnboardingPresetFactory) (OnboardingPreset, error) {
	group, err := factory.CreateGroupDraft()
	if err != nil {
		return OnboardingPreset{}, err
	}

	profile, err := factory.CreateSuggestedProfile()
	if err != nil {
		return OnboardingPreset{}, err
	}

	profileModel, err := newProfile("suggested-profile", profile)
	if err != nil {
		return OnboardingPreset{}, err
	}
	if profileModel.monthlyBudget < group.MaxMonthlyCost {
		return OnboardingPreset{}, ErrBudgetTooLow
	}

	group, err = normalizeGroupInput(group)
	if err != nil {
		return OnboardingPreset{}, err
	}
	return OnboardingPreset{Group: group, SuggestedProfile: profile}, nil
}

// buildProfileSuggestion applies the domain's name and budget checks to a suggestion.
// It returns input data, leaving identity assignment to the service and store.
func buildProfileSuggestion(name string, monthlyBudget int) (ProfileInput, error) {
	profile, err := newProfile("suggested-profile", ProfileInput{
		Name:          name,
		MonthlyBudget: monthlyBudget,
	})
	if err != nil {
		return ProfileInput{}, err
	}
	return ProfileInput{Name: profile.name, MonthlyBudget: profile.monthlyBudget}, nil
}

// GroupPrototype stores an isolated snapshot that can seed another group draft.
// Its clone keeps the settings but receives no identity or memberships.
type GroupPrototype struct {
	template Group
}

// NewGroupPrototype validates the source settings and captures its current state.
// It copies member IDs into a new slice so later source changes do not alter the snapshot.
func NewGroupPrototype(source Group) (GroupPrototype, error) {
	input, err := normalizeGroupInput(source.input())
	if err != nil {
		return GroupPrototype{}, err
	}

	snapshot := source
	snapshot.name = input.Name
	snapshot.memberProfileIDs = append([]string(nil), source.memberProfileIDs...)
	return GroupPrototype{template: snapshot}, nil
}

// CloneAsDraft copies group settings, then removes identity and membership.
// The store assigns a new ID when the caller saves the returned draft.
func (prototype GroupPrototype) CloneAsDraft(name string) (Group, error) {
	input, err := normalizeGroupInput(GroupInput{
		Name:           name,
		MaxMonthlyCost: prototype.template.maxMonthlyCost,
		Capacity:       prototype.template.capacity,
	})
	if err != nil {
		return Group{}, err
	}

	clone := prototype.template
	clone.id = ""
	clone.name = input.Name
	clone.memberProfileIDs = make([]string, 0)
	return clone, nil
}

// input converts a saved group into a creation request with reusable settings.
// It leaves out the identity and membership that belong to the original record.
func (group Group) input() GroupInput {
	return GroupInput{
		Name:           group.name,
		MaxMonthlyCost: group.maxMonthlyCost,
		Capacity:       group.capacity,
	}
}
