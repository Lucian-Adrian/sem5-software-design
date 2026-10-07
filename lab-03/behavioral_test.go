package main

import (
	"errors"
	"testing"
)

// TestAvailableStrategyBoundary checks budget equality and the capacity rule.
// The store rejects a join after the listing rule hides a full group.
func TestAvailableStrategyBoundary(t *testing.T) {
	store := NewMemoryStore()
	groups := NewGroupService(store, store)
	profiles := NewProfileService(store)

	group, err := groups.CreateGroup(GroupInput{
		Name:           "Shared flat",
		MaxMonthlyCost: 500,
		Capacity:       1,
	})
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}

	equalBudget, err := groups.ListGroups(AvailableGroups{MonthlyBudget: 500})
	if err != nil {
		t.Fatalf("ListGroups() at budget equality: %v", err)
	}
	if len(equalBudget) != 1 {
		t.Fatalf("groups at budget equality = %d, want 1", len(equalBudget))
	}

	belowBudget, err := groups.ListGroups(AvailableGroups{MonthlyBudget: 499})
	if err != nil {
		t.Fatalf("ListGroups() below budget: %v", err)
	}
	if len(belowBudget) != 0 {
		t.Fatalf("groups below budget = %d, want 0", len(belowBudget))
	}

	alex, err := profiles.CreateProfile(ProfileInput{Name: "Alex", MonthlyBudget: 500})
	if err != nil {
		t.Fatalf("CreateProfile(Alex) error = %v", err)
	}
	joined, err := groups.JoinGroup(group.ID, alex.ID)
	if err != nil {
		t.Fatalf("JoinGroup(Alex) error = %v", err)
	}
	if joined.Group.MemberCount != 1 {
		t.Fatalf("joined member count = %d, want 1", joined.Group.MemberCount)
	}

	fullGroup, err := groups.ListGroups(AvailableGroups{MonthlyBudget: 500})
	if err != nil {
		t.Fatalf("ListGroups() after join: %v", err)
	}
	if len(fullGroup) != 0 {
		t.Fatalf("available groups after join = %d, want 0", len(fullGroup))
	}

	pat, err := profiles.CreateProfile(ProfileInput{Name: "Pat", MonthlyBudget: 500})
	if err != nil {
		t.Fatalf("CreateProfile(Pat) error = %v", err)
	}
	if _, err := groups.JoinGroup(group.ID, pat.ID); !errors.Is(err, ErrGroupFull) {
		t.Fatalf("JoinGroup(Pat) error = %v, want ErrGroupFull", err)
	}
}

// TestCommandObserverOutcomes checks failures, repeated joins and listeners.
// Each listener gets events only for successful command executions.
func TestCommandObserverOutcomes(t *testing.T) {
	store := NewMemoryStore()
	groups := NewGroupService(store, store)
	profiles := NewProfileService(store)

	group, err := groups.CreateGroup(GroupInput{
		Name:           "Shared flat",
		MaxMonthlyCost: 500,
		Capacity:       1,
	})
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}
	pat, err := profiles.CreateProfile(ProfileInput{Name: "Pat", MonthlyBudget: 400})
	if err != nil {
		t.Fatalf("CreateProfile(Pat) error = %v", err)
	}
	alex, err := profiles.CreateProfile(ProfileInput{Name: "Alex", MonthlyBudget: 550})
	if err != nil {
		t.Fatalf("CreateProfile(Alex) error = %v", err)
	}

	firstLog := &CompletionLog{}
	secondLog := &CompletionLog{}
	dispatcher := &CommandDispatcher{}
	dispatcher.Subscribe(firstLog)
	dispatcher.Subscribe(secondLog)

	_, err = dispatcher.Run(JoinCommand{Groups: groups, GroupID: group.ID, ProfileID: pat.ID})
	if !errors.Is(err, ErrBudgetTooLow) {
		t.Fatalf("Run(Pat) error = %v, want ErrBudgetTooLow", err)
	}
	if len(firstLog.Events) != 0 || len(secondLog.Events) != 0 {
		t.Fatal("failed command sent a completion event")
	}

	firstJoin, err := dispatcher.Run(JoinCommand{Groups: groups, GroupID: group.ID, ProfileID: alex.ID})
	if err != nil {
		t.Fatalf("Run(Alex) error = %v", err)
	}
	if firstJoin.Group.MemberCount != 1 {
		t.Fatalf("first member count = %d, want 1", firstJoin.Group.MemberCount)
	}

	repeatedJoin, err := dispatcher.Run(JoinCommand{Groups: groups, GroupID: group.ID, ProfileID: alex.ID})
	if err != nil {
		t.Fatalf("Run(Alex retry) error = %v", err)
	}
	if repeatedJoin.Group.MemberCount != 1 {
		t.Fatalf("member count after retry = %d, want 1", repeatedJoin.Group.MemberCount)
	}

	for _, log := range []*CompletionLog{firstLog, secondLog} {
		if len(log.Events) != 2 {
			t.Fatalf("observer event count = %d, want 2", len(log.Events))
		}
		for _, event := range log.Events {
			if event.GroupID != group.ID || event.MemberCount != 1 {
				t.Fatalf("observer event = %+v, want group %q with one member", event, group.ID)
			}
		}
	}
}
