package main

// GroupBuilder assembles one group input:
// 1. collect the name, cost, and capacity;
// 2. validate the completed draft before returning it.
type GroupBuilder struct {
	input GroupInput
}

func NewGroupBuilder() GroupBuilder {
	return GroupBuilder{}
}

func (builder GroupBuilder) WithName(name string) GroupBuilder {
	builder.input.Name = name
	return builder
}

func (builder GroupBuilder) WithMaxMonthlyCost(amount int) GroupBuilder {
	builder.input.MaxMonthlyCost = amount
	return builder
}

func (builder GroupBuilder) WithCapacity(people int) GroupBuilder {
	builder.input.Capacity = people
	return builder
}

func (builder GroupBuilder) Build() (GroupInput, error) {
	return normalizeGroupInput(builder.input)
}

// OnboardingPreset keeps a family's related creation inputs together.
type OnboardingPreset struct {
	Group            GroupInput   `json:"group"`
	SuggestedProfile ProfileInput `json:"suggestedProfile"`
}

// OnboardingPresetFactory creates both products needed by one onboarding family.
type OnboardingPresetFactory interface {
	CreateGroupDraft() (GroupInput, error)
	CreateSuggestedProfile() (ProfileInput, error)
}

// BudgetPresetFactory creates a lower-cost group and a profile that can afford it.
type BudgetPresetFactory struct{}

func (BudgetPresetFactory) CreateGroupDraft() (GroupInput, error) {
	return NewGroupBuilder().
		WithName("Budget shared home").
		WithMaxMonthlyCost(450).
		WithCapacity(4).
		Build()
}

func (BudgetPresetFactory) CreateSuggestedProfile() (ProfileInput, error) {
	return buildProfileSuggestion("Budget renter", 500)
}

// ComfortPresetFactory creates a higher-cost group and a compatible suggestion.
type ComfortPresetFactory struct{}

func (ComfortPresetFactory) CreateGroupDraft() (GroupInput, error) {
	return NewGroupBuilder().
		WithName("Comfort shared home").
		WithMaxMonthlyCost(700).
		WithCapacity(2).
		Build()
}

func (ComfortPresetFactory) CreateSuggestedProfile() (ProfileInput, error) {
	return buildProfileSuggestion("Comfort renter", 900)
}

// buildOnboardingPreset follows one family contract:
// 1. request both product inputs;
// 2. validate the profile and check its budget against the group cost;
// 3. normalize the group and return the compatible pair.
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
type GroupPrototype struct {
	template Group
}

// NewGroupPrototype validates the source and copies its slice-backed membership.
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

// input exposes the group's creation settings to the existing service path.
func (group Group) input() GroupInput {
	return GroupInput{
		Name:           group.name,
		MaxMonthlyCost: group.maxMonthlyCost,
		Capacity:       group.capacity,
	}
}
