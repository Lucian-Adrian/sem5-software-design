package main

import "testing"

func TestDefaultOnboardingCatalogIsSharedAndReturnsCopies(t *testing.T) {
	first := DefaultOnboardingCatalog()
	second := DefaultOnboardingCatalog()
	if first != second {
		t.Fatal("expected both calls to return the same catalog")
	}

	group := first.DefaultGroup()
	profile := first.DefaultProfile()
	group.Name = "changed outside the catalog"
	profile.MonthlyBudget = 1
	if got := second.DefaultGroup().Name; got != "Catalog starter home" {
		t.Fatalf("catalog group changed through a returned copy: %q", got)
	}
	if got := second.DefaultProfile().MonthlyBudget; got != 800 {
		t.Fatalf("catalog profile changed through a returned copy: %d", got)
	}
}

func TestGroupOnboardingWorkflowUsesCreatorHookAndJoinsProfile(t *testing.T) {
	store := NewMemoryStore()
	workflow := NewGroupOnboardingWorkflow(NewGroupService(store, store), NewProfileService(store))
	creator := &recordingGroupDraftCreator{input: GroupInput{
		Name:           "Custom household",
		MaxMonthlyCost: 500,
		Capacity:       2,
	}}

	result, err := workflow.Run(creator)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !creator.called {
		t.Fatal("workflow did not call the creator hook")
	}
	if result.Group.Group.Name != "Custom household" {
		t.Fatalf("saved group name = %q", result.Group.Group.Name)
	}
	if result.Group.Group.MemberCount != 1 || len(result.Group.Members) != 1 {
		t.Fatalf("joined group = %#v", result.Group)
	}
	if result.Group.Members[0].ID != result.Profile.ID {
		t.Fatalf("joined member ID = %q, profile ID = %q", result.Group.Members[0].ID, result.Profile.ID)
	}
}

func TestConcreteGroupCreatorsSupplyDifferentValidDrafts(t *testing.T) {
	creators := []struct {
		creator GroupDraftCreator
		name    string
	}{
		{creator: StudentGroupCreator{}, name: "Student shared home"},
		{creator: QuietGroupCreator{}, name: "Quiet shared home"},
	}
	for _, example := range creators {
		input, err := example.creator.CreateGroupDraft()
		if err != nil {
			t.Fatalf("CreateGroupDraft() error = %v", err)
		}
		if input.Name != example.name {
			t.Fatalf("group name = %q, want %q", input.Name, example.name)
		}
	}
}

func TestDemoModeSelectionRejectsMultipleModes(t *testing.T) {
	cases := []struct {
		name          string
		demo          bool
		patterns      bool
		extraPatterns bool
		wantInvalid   bool
	}{
		{name: "no mode", wantInvalid: false},
		{name: "demo", demo: true, wantInvalid: false},
		{name: "patterns", patterns: true, wantInvalid: false},
		{name: "extra patterns", extraPatterns: true, wantInvalid: false},
		{name: "demo and patterns", demo: true, patterns: true, wantInvalid: true},
		{name: "demo and extra patterns", demo: true, extraPatterns: true, wantInvalid: true},
		{name: "both pattern modes", patterns: true, extraPatterns: true, wantInvalid: true},
		{name: "all modes", demo: true, patterns: true, extraPatterns: true, wantInvalid: true},
	}
	for _, example := range cases {
		t.Run(example.name, func(t *testing.T) {
			if got := invalidDemoModeSelection(example.demo, example.patterns, example.extraPatterns); got != example.wantInvalid {
				t.Fatalf("invalidDemoModeSelection() = %t, want %t", got, example.wantInvalid)
			}
		})
	}
}

func TestBufferPoolResetsReusesAndBoundsIdleBuffers(t *testing.T) {
	pool := NewBufferPool(1)
	first := pool.Acquire()
	first.WriteString("temporary data")
	pool.Release(first)
	if pool.IdleCount() != 1 {
		t.Fatalf("idle count after release = %d, want 1", pool.IdleCount())
	}

	reused := pool.Acquire()
	if reused != first {
		t.Fatal("expected pool to return the released buffer")
	}
	if reused.Len() != 0 {
		t.Fatalf("reused buffer length = %d, want 0", reused.Len())
	}

	second := pool.Acquire()
	pool.Release(second)
	pool.Release(reused)
	if pool.IdleCount() != 1 {
		t.Fatalf("idle count exceeded its bound: %d", pool.IdleCount())
	}
	if len(reused.Bytes()) != 0 {
		t.Fatal("released buffer should be reset")
	}
}

type recordingGroupDraftCreator struct {
	called bool
	input  GroupInput
}

func (creator *recordingGroupDraftCreator) CreateGroupDraft() (GroupInput, error) {
	creator.called = true
	return creator.input, nil
}
