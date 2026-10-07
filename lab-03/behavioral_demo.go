package main

import (
	"encoding/json"
	"errors"
	"io"
)

// runBehavioralDemo builds a fresh example and writes its results as JSON.
// The fixed inputs show filtering, one rejection, a join, and a successful retry.
func runBehavioralDemo(output io.Writer) error {
	store := NewMemoryStore()
	groups := NewGroupService(store, store)
	profiles := NewProfileService(store)
	alex, err := profiles.CreateProfile(ProfileInput{Name: "Alex", MonthlyBudget: 550})
	if err != nil {
		return err
	}
	pat, err := profiles.CreateProfile(ProfileInput{Name: "Pat", MonthlyBudget: 400})
	if err != nil {
		return err
	}
	group, err := groups.CreateGroup(GroupInput{Name: "Shared flat", MaxMonthlyCost: 500, Capacity: 1})
	if err != nil {
		return err
	}
	before, err := groups.ListGroups(AvailableGroups{MonthlyBudget: 550})
	if err != nil {
		return err
	}
	log := &CompletionLog{}
	dispatcher := &CommandDispatcher{}
	dispatcher.Subscribe(log)
	_, patError := dispatcher.Run(JoinCommand{Groups: groups, GroupID: group.ID, ProfileID: pat.ID})
	if !errors.Is(patError, ErrBudgetTooLow) {
		return errors.New("expected budget rejection")
	}
	joined, err := dispatcher.Run(JoinCommand{Groups: groups, GroupID: group.ID, ProfileID: alex.ID})
	if err != nil {
		return err
	}
	if _, err = dispatcher.Run(JoinCommand{Groups: groups, GroupID: group.ID, ProfileID: alex.ID}); err != nil {
		return err
	}
	after, err := groups.ListGroups(AvailableGroups{MonthlyBudget: 550})
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	result := struct {
		AvailableBefore []GroupView     `json:"availableBefore"`
		Joined          GroupDetail     `json:"joined"`
		AvailableAfter  []GroupView     `json:"availableAfter"`
		RejectedBudget  bool            `json:"rejectedBudget"`
		Completions     []CommandResult `json:"successfulInvocations"`
	}{
		AvailableBefore: before,
		Joined:          joined,
		AvailableAfter:  after,
		RejectedBudget:  true,
		Completions:     log.Events,
	}
	return encoder.Encode(result)
}
