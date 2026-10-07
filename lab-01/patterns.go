package main

// GroupBuilder gathers the required group settings before validating them as
// one draft. The returned input can then go through the normal service path.
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

// OnboardingPreset is one family of related group and profile suggestions.
type OnboardingPreset struct {
	Group            GroupInput   `json:"group"`
	SuggestedProfile ProfileInput `json:"suggestedProfile"`
}

// OnboardingPresetFactory creates the related products for one housing style.
type OnboardingPresetFactory interface {
	CreateGroupDraft() (GroupInput, error)
	CreateSuggestedProfile() (ProfileInput, error)
}

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

// GroupPrototype keeps a snapshot of a group that can seed a fresh draft.
type GroupPrototype struct {
	template Group
}

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

// CloneAsDraft copies the group's settings while clearing its identity and
// membership. The store assigns a new ID when the caller saves the draft.
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

func (group Group) input() GroupInput {
	return GroupInput{
		Name:           group.name,
		MaxMonthlyCost: group.maxMonthlyCost,
		Capacity:       group.capacity,
	}
}
