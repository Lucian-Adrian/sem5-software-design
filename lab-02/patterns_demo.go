package main

import (
	"encoding/json"
	"errors"
	"os"
)

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

// runCreationalDemo builds each example, then prints the combined JSON result.
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

// createBuilderExample validates a standalone draft and saves it through the service.
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

// createPresetExamples creates and joins each family's suggested profile:
// 1. build and validate the related product pair;
// 2. save both products through their services;
// 3. join the profile and keep the Budget group for the prototype example.
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

// createPrototypeExample snapshots the joined Budget group and saves a clean copy.
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

// groupFromDetail reconstructs domain state and checks the response's member count.
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
