package main

import (
	"encoding/json"
	"errors"
	"os"
)

// creationalDemoResult collects the core pattern examples for JSON output.
// Its fields retain the existing names consumed by the demonstration.
type creationalDemoResult struct {
	BuilderGroup GroupView           `json:"builderGroup"`
	PresetGroups []presetDemoResult  `json:"presetGroups"`
	Prototype    prototypeDemoResult `json:"prototype"`
}

type presetDemoResult struct {
	Family           string      `json:"family"`
	Group            GroupDetail `json:"group"`
	SuggestedProfile ProfileView `json:"suggestedProfile"`
}

type prototypeDemoResult struct {
	SourceGroupID     string    `json:"sourceGroupId"`
	SourceMemberCount int       `json:"sourceMemberCount"`
	Copy              GroupView `json:"copy"`
}

// runCreationalDemo builds the Builder, factory-family, and Prototype examples.
// It encodes their saved results as indented JSON on standard output.
func runCreationalDemo(groups *GroupService, profiles *ProfileService) error {
	builderGroup, err := createBuilderExample(groups)
	if err != nil {
		return err
	}

	presetGroups, budgetGroup, err := createPresetExamples(groups, profiles)
	if err != nil {
		return err
	}

	prototype, err := createPrototypeExample(groups, budgetGroup)
	if err != nil {
		return err
	}

	result := creationalDemoResult{
		BuilderGroup: builderGroup,
		PresetGroups: presetGroups,
		Prototype:    prototype,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// createBuilderExample builds and validates one standalone group input.
// It saves the input through GroupService, which returns the assigned ID.
func createBuilderExample(groups *GroupService) (GroupView, error) {
	input, err := NewGroupBuilder().
		WithName("Builder example home").
		WithMaxMonthlyCost(650).
		WithCapacity(3).
		Build()
	if err != nil {
		return GroupView{}, err
	}
	return groups.CreateGroup(input)
}

// createPresetExamples builds, saves, and joins each family's related inputs.
// It returns the joined Budget group for the Prototype example.
// A later error leaves any earlier in-memory writes in place.
func createPresetExamples(groups *GroupService, profiles *ProfileService) ([]presetDemoResult, GroupDetail, error) {
	families := []struct {
		name    string
		factory OnboardingPresetFactory
	}{
		{name: "budget", factory: BudgetPresetFactory{}},
		{name: "comfort", factory: ComfortPresetFactory{}},
	}

	results := make([]presetDemoResult, 0, len(families))
	var budgetGroup GroupDetail
	for _, family := range families {
		preset, err := buildOnboardingPreset(family.factory)
		if err != nil {
			return nil, GroupDetail{}, err
		}

		group, err := groups.CreateGroup(preset.Group)
		if err != nil {
			return nil, GroupDetail{}, err
		}

		profile, err := profiles.CreateProfile(preset.SuggestedProfile)
		if err != nil {
			return nil, GroupDetail{}, err
		}

		joined, err := groups.JoinGroup(group.ID, profile.ID)
		if err != nil {
			return nil, GroupDetail{}, err
		}
		results = append(results, presetDemoResult{
			Family:           family.name,
			Group:            joined,
			SuggestedProfile: profile,
		})

		if family.name == "budget" {
			budgetGroup = joined
		}
	}
	return results, budgetGroup, nil
}

// createPrototypeExample reconstructs and snapshots the joined Budget group.
// It saves a copy with the same settings and a fresh ID but no members.
func createPrototypeExample(groups *GroupService, sourceDetail GroupDetail) (prototypeDemoResult, error) {
	source, err := groupFromDetail(sourceDetail)
	if err != nil {
		return prototypeDemoResult{}, err
	}

	prototype, err := NewGroupPrototype(source)
	if err != nil {
		return prototypeDemoResult{}, err
	}

	copyDraft, err := prototype.CloneAsDraft("Budget shared home copy")
	if err != nil {
		return prototypeDemoResult{}, err
	}

	copyGroup, err := groups.CreateGroup(copyDraft.input())
	if err != nil {
		return prototypeDemoResult{}, err
	}
	return prototypeDemoResult{
		SourceGroupID:     sourceDetail.Group.ID,
		SourceMemberCount: len(sourceDetail.Members),
		Copy:              copyGroup,
	}, nil
}

// groupFromDetail reconstructs a domain group from the service response view.
// It checks that the restored member IDs match the reported member count.
func groupFromDetail(detail GroupDetail) (Group, error) {
	group, err := newGroup(detail.Group.ID, GroupInput{
		Name:           detail.Group.Name,
		MaxMonthlyCost: detail.Group.MaxMonthlyCost,
		Capacity:       detail.Group.Capacity,
	})
	if err != nil {
		return Group{}, err
	}

	for _, member := range detail.Members {
		group.memberProfileIDs = append(group.memberProfileIDs, member.ID)
	}
	if len(group.memberProfileIDs) != detail.Group.MemberCount {
		return Group{}, errors.New("group detail member count does not match its members")
	}
	return group, nil
}
