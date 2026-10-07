package main

import (
	"bytes"
	"sync"
)

// OnboardingCatalog keeps fixed process defaults for optional examples.
// Its fields stay private and its accessors return value copies. A singleton
// suits immutable defaults, not per-user settings or mutable application state.
type OnboardingCatalog struct {
	defaultGroup   GroupInput
	defaultProfile ProfileInput
}

var (
	defaultOnboardingCatalogOnce sync.Once
	defaultOnboardingCatalog     *OnboardingCatalog
)

// DefaultOnboardingCatalog returns one process-wide catalog of fixed defaults.
func DefaultOnboardingCatalog() *OnboardingCatalog {
	defaultOnboardingCatalogOnce.Do(func() {
		defaultOnboardingCatalog = &OnboardingCatalog{
			defaultGroup: GroupInput{
				Name:           "Catalog starter home",
				MaxMonthlyCost: 600,
				Capacity:       2,
			},
			defaultProfile: ProfileInput{
				Name:          "New renter",
				MonthlyBudget: 800,
			},
		}
	})
	return defaultOnboardingCatalog
}

// DefaultGroup returns a copy of the catalog's starter group input.
func (catalog *OnboardingCatalog) DefaultGroup() GroupInput {
	return catalog.defaultGroup
}

// DefaultProfile returns a copy of the catalog's suggested profile input.
func (catalog *OnboardingCatalog) DefaultProfile() ProfileInput {
	return catalog.defaultProfile
}

// GroupDraftCreator supplies the group product for a shared onboarding flow.
type GroupDraftCreator interface {
	CreateGroupDraft() (GroupInput, error)
}

// StudentGroupCreator makes the group draft used for a student household.
type StudentGroupCreator struct{}

func (StudentGroupCreator) CreateGroupDraft() (GroupInput, error) {
	return NewGroupBuilder().
		WithName("Student shared home").
		WithMaxMonthlyCost(650).
		WithCapacity(3).
		Build()
}

// QuietGroupCreator makes a smaller group draft for a quieter household.
type QuietGroupCreator struct{}

func (QuietGroupCreator) CreateGroupDraft() (GroupInput, error) {
	return NewGroupBuilder().
		WithName("Quiet shared home").
		WithMaxMonthlyCost(700).
		WithCapacity(2).
		Build()
}

// GroupOnboardingResult contains the saved group and the profile that joined it.
type GroupOnboardingResult struct {
	Group   GroupDetail
	Profile ProfileView
}

// GroupOnboardingWorkflow shares the steps around the creator's group hook.
type GroupOnboardingWorkflow struct {
	groups   *GroupService
	profiles *ProfileService
	catalog  *OnboardingCatalog
}

// NewGroupOnboardingWorkflow connects the workflow to the Domi services.
func NewGroupOnboardingWorkflow(groups *GroupService, profiles *ProfileService) GroupOnboardingWorkflow {
	return GroupOnboardingWorkflow{
		groups:   groups,
		profiles: profiles,
		catalog:  DefaultOnboardingCatalog(),
	}
}

// Run asks the creator for one group, saves the default profile and joins it.
func (workflow GroupOnboardingWorkflow) Run(creator GroupDraftCreator) (GroupOnboardingResult, error) {
	groupInput, err := creator.CreateGroupDraft()
	if err != nil {
		return GroupOnboardingResult{}, err
	}

	group, err := workflow.groups.CreateGroup(groupInput)
	if err != nil {
		return GroupOnboardingResult{}, err
	}

	profile, err := workflow.profiles.CreateProfile(workflow.catalog.DefaultProfile())
	if err != nil {
		return GroupOnboardingResult{}, err
	}

	joined, err := workflow.groups.JoinGroup(group.ID, profile.ID)
	if err != nil {
		return GroupOnboardingResult{}, err
	}
	return GroupOnboardingResult{Group: joined, Profile: profile}, nil
}

// BufferPool reuses a bounded number of temporary buffers in one goroutine.
type BufferPool struct {
	idle    []*bytes.Buffer
	maxIdle int
}

// NewBufferPool creates a pool with a fixed limit on retained idle buffers.
func NewBufferPool(maxIdle int) *BufferPool {
	if maxIdle < 0 {
		maxIdle = 0
	}
	return &BufferPool{maxIdle: maxIdle}
}

// Acquire returns an idle buffer or creates a new one when the pool is empty.
func (pool *BufferPool) Acquire() *bytes.Buffer {
	last := len(pool.idle) - 1
	if last < 0 {
		return &bytes.Buffer{}
	}

	buffer := pool.idle[last]
	pool.idle[last] = nil
	pool.idle = pool.idle[:last]
	return buffer
}

// Release resets a buffer before retaining it, and drops it when the idle limit is full.
func (pool *BufferPool) Release(buffer *bytes.Buffer) {
	if buffer == nil {
		return
	}
	buffer.Reset()
	if len(pool.idle) >= pool.maxIdle {
		return
	}
	pool.idle = append(pool.idle, buffer)
}

// IdleCount reports how many reset buffers the pool currently retains.
func (pool *BufferPool) IdleCount() int {
	return len(pool.idle)
}
