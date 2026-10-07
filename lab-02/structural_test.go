package main

import (
	"bytes"
	"errors"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestLegacyOfferAdapterRoundsUpWithoutOverflow(t *testing.T) {
	store := NewMemoryStore()
	adapter := NewLegacyOfferAdapter(NewGroupService(store, store))

	group, err := adapter.CreateGroup(LegacyRoommateOffer{
		Name:          "Shared apartment",
		MonthlyTotal:  1000,
		RoommateLimit: 3,
	})
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}
	if group.MaxMonthlyCost != 334 || group.Capacity != 3 {
		t.Fatalf("adapted group cost/capacity = %d/%d, want 334/3", group.MaxMonthlyCost, group.Capacity)
	}

	maximumCost := math.MaxInt/2 + 1
	input, err := groupInputFromLegacyOffer(LegacyRoommateOffer{
		Name:          "Large total",
		MonthlyTotal:  math.MaxInt,
		RoommateLimit: 2,
	})
	if err != nil {
		t.Fatalf("maximum representable total was rejected: %v", err)
	}
	if input.MaxMonthlyCost != maximumCost {
		t.Fatalf("rounded maximum cost = %d, want %d", input.MaxMonthlyCost, maximumCost)
	}
}

func TestLegacyOfferAdapterRejectsInvalidTotalsAndCapacity(t *testing.T) {
	tests := []struct {
		name  string
		offer LegacyRoommateOffer
	}{
		{name: "zero total", offer: LegacyRoommateOffer{Name: "Room", MonthlyTotal: 0, RoommateLimit: 1}},
		{name: "negative total", offer: LegacyRoommateOffer{Name: "Room", MonthlyTotal: -1, RoommateLimit: 1}},
		{name: "zero roommate limit", offer: LegacyRoommateOffer{Name: "Room", MonthlyTotal: 100, RoommateLimit: 0}},
		{name: "limit above domain maximum", offer: LegacyRoommateOffer{Name: "Room", MonthlyTotal: 100, RoommateLimit: maxGroupCapacity + 1}},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := groupInputFromLegacyOffer(testCase.offer)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("groupInputFromLegacyOffer() error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestAuditedGroupWriterPreservesResultsAndDoesNotLogPersonalData(t *testing.T) {
	store := NewMemoryStore()
	var auditLog bytes.Buffer
	writer := NewAuditedGroupWriter(store, &auditLog)
	group, err := writer.CreateGroup(GroupInput{Name: "Hidden name", MaxMonthlyCost: 400, Capacity: 2})
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}
	profile, err := store.CreateProfile(ProfileInput{Name: "Hidden profile", MonthlyBudget: 987654})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	joined, err := writer.JoinGroup(group.ID, profile.ID)
	if err != nil {
		t.Fatalf("JoinGroup() error = %v", err)
	}
	if joined.Group.MemberCount != 1 || len(joined.Members) != 1 {
		t.Fatalf("JoinGroup() result = %+v, want one member", joined)
	}

	logText := auditLog.String()
	if logText != "operation=CreateGroup result=success\noperation=JoinGroup result=success\n" {
		t.Fatalf("audit log = %q, want outcome-only records", logText)
	}
	for _, privateValue := range []string{"987654", "Hidden profile", "Hidden name"} {
		if strings.Contains(logText, privateValue) {
			t.Errorf("audit log contains private value %q", privateValue)
		}
	}
}

func TestAuditedGroupWriterReturnsUnderlyingErrorAndIgnoresLogFailure(t *testing.T) {
	wantDetail := GroupDetail{Group: GroupView{ID: "group-kept"}}
	wantErr := errors.New("store unavailable")
	wrapped := &fixedJoinWriter{detail: wantDetail, err: wantErr}
	writer := NewAuditedGroupWriter(wrapped, failingAuditWriter{})

	detail, err := writer.JoinGroup("group-1", "profile-1")
	if err != wantErr {
		t.Fatalf("JoinGroup() error = %v, want exact underlying error %v", err, wantErr)
	}
	if !reflect.DeepEqual(detail, wantDetail) {
		t.Fatalf("JoinGroup() result = %+v, want underlying result %+v", detail, wantDetail)
	}
}

func TestOnboardingFacadeReturnsCreatedViews(t *testing.T) {
	store := NewMemoryStore()
	facade := NewOnboardingFacade(NewProfileService(store), NewGroupService(store, store))

	result, err := facade.Onboard(OnboardingInput{
		Profile: ProfileInput{Name: "Alex", MonthlyBudget: 600},
		Group:   GroupInput{Name: "Quiet flat", MaxMonthlyCost: 550, Capacity: 2},
	})
	if err != nil {
		t.Fatalf("Onboard() error = %v", err)
	}
	if result.Profile.ID != "profile-1" || result.Group.ID != "group-1" {
		t.Fatalf("onboarding profile/group = %q/%q", result.Profile.ID, result.Group.ID)
	}
	if result.JoinedGroup == nil || result.JoinedGroup.Group.MemberCount != 1 || len(result.JoinedGroup.Members) != 1 {
		t.Fatalf("onboarding joined group = %+v, want one joined profile", result.JoinedGroup)
	}
}

func TestOnboardingFacadeValidatesBeforeWritingAndReportsPartialFailure(t *testing.T) {
	t.Run("budget rejected before writes", func(t *testing.T) {
		store := NewMemoryStore()
		facade := NewOnboardingFacade(NewProfileService(store), NewGroupService(store, store))
		_, err := facade.Onboard(OnboardingInput{
			Profile: ProfileInput{Name: "Alex", MonthlyBudget: 400},
			Group:   GroupInput{Name: "Costly flat", MaxMonthlyCost: 500, Capacity: 2},
		})
		if !errors.Is(err, ErrBudgetTooLow) {
			t.Fatalf("Onboard() error = %v, want ErrBudgetTooLow", err)
		}
		if store.nextProfileID != 0 || store.nextGroupID != 0 {
			t.Fatalf("IDs allocated before validation completed: profiles=%d groups=%d", store.nextProfileID, store.nextGroupID)
		}
	})

	t.Run("group creation failure exposes completed profile step", func(t *testing.T) {
		store := NewMemoryStore()
		groupWriter := &failingCreateGroupWriter{GroupWriter: store, err: errors.New("group storage unavailable")}
		groups := NewGroupService(store, groupWriter)
		facade := NewOnboardingFacade(NewProfileService(store), groups)
		result, err := facade.Onboard(OnboardingInput{
			Profile: ProfileInput{Name: "Alex", MonthlyBudget: 600},
			Group:   GroupInput{Name: "Quiet flat", MaxMonthlyCost: 550, Capacity: 2},
		})
		if err == nil {
			t.Fatal("Onboard() error = nil, want group creation error")
		}
		if result.Profile.ID != "profile-1" || result.Group.ID != "" || result.JoinedGroup != nil {
			t.Fatalf("partial onboarding result = %+v, want created profile only", result)
		}
		if store.nextProfileID != 1 || store.nextGroupID != 0 {
			t.Fatalf("stored IDs = profiles:%d groups:%d, want 1/0", store.nextProfileID, store.nextGroupID)
		}
	})
}

func TestOnboardingFacadeReturnsPartialViewsWhenJoinFails(t *testing.T) {
	store := NewMemoryStore()
	joinErr := errors.New("membership storage unavailable")
	groupWriter := &failingJoinGroupWriter{GroupWriter: store, err: joinErr}
	groups := NewGroupService(store, groupWriter)
	facade := NewOnboardingFacade(NewProfileService(store), groups)

	result, err := facade.Onboard(OnboardingInput{
		Profile: ProfileInput{Name: "Alex", MonthlyBudget: 600},
		Group:   GroupInput{Name: "Quiet flat", MaxMonthlyCost: 550, Capacity: 2},
	})
	if err != joinErr {
		t.Fatalf("Onboard() error = %v, want exact join error %v", err, joinErr)
	}
	if result.Profile.ID != "profile-1" || result.Group.ID != "group-1" || result.JoinedGroup != nil {
		t.Fatalf("partial onboarding result = %+v, want profile and group without joined group", result)
	}
}

type fixedJoinWriter struct {
	GroupWriter
	detail GroupDetail
	err    error
}

func (writer *fixedJoinWriter) JoinGroup(string, string) (GroupDetail, error) {
	return writer.detail, writer.err
}

type failingCreateGroupWriter struct {
	GroupWriter
	err error
}

func (writer *failingCreateGroupWriter) CreateGroup(GroupInput) (GroupView, error) {
	return GroupView{}, writer.err
}

type failingJoinGroupWriter struct {
	GroupWriter
	err error
}

func (writer *failingJoinGroupWriter) JoinGroup(string, string) (GroupDetail, error) {
	return GroupDetail{}, writer.err
}

type failingAuditWriter struct{}

func (failingAuditWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}
