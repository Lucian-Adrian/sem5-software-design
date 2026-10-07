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

func runCreationalDemo(groups *GroupService, profiles *ProfileService) error {
	builderInput, err := NewGroupBuilder().
		WithName("Builder example home").
		WithMaxMonthlyCost(650).
		WithCapacity(3).
		Build()
	if err != nil {
		return err
	}
	builderGroup, err := groups.CreateGroup(builderInput)
	if err != nil {
		return err
	}

	families := []struct {
		name    string
		factory OnboardingPresetFactory
	}{
		{name: "budget", factory: BudgetPresetFactory{}},
		{name: "comfort", factory: ComfortPresetFactory{}},
	}

	result := creationalDemoResult{BuilderGroup: builderGroup}
	var budgetGroup GroupDetail
	for _, family := range families {
		preset, createErr := buildOnboardingPreset(family.factory)
		if createErr != nil {
			return createErr
		}
		group, createErr := groups.CreateGroup(preset.Group)
		if createErr != nil {
			return createErr
		}
		profile, createErr := profiles.CreateProfile(preset.SuggestedProfile)
		if createErr != nil {
			return createErr
		}
		joined, joinErr := groups.JoinGroup(group.ID, profile.ID)
		if joinErr != nil {
			return joinErr
		}
		result.PresetGroups = append(result.PresetGroups, presetDemoResult{
			Family:           family.name,
			Group:            joined,
			SuggestedProfile: profile,
		})
		if family.name == "budget" {
			budgetGroup = joined
		}
	}

	source, err := groupFromDetail(budgetGroup)
	if err != nil {
		return err
	}
	prototype, err := NewGroupPrototype(source)
	if err != nil {
		return err
	}
	copyDraft, err := prototype.CloneAsDraft("Budget shared home copy")
	if err != nil {
		return err
	}
	copyGroup, err := groups.CreateGroup(copyDraft.input())
	if err != nil {
		return err
	}
	result.Prototype = prototypeDemoResult{
		SourceGroupID:     budgetGroup.Group.ID,
		SourceMemberCount: len(budgetGroup.Members),
		Copy:              copyGroup,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

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
