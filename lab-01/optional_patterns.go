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

// DefaultOnboardingCatalog returns the one catalog shared across this process.
// sync.Once initializes it only on its first call, including concurrent calls.
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

// DefaultGroup returns a copy of the catalog's fixed starter group settings.
// Callers can change the returned value without changing the catalog fields.
func (catalog *OnboardingCatalog) DefaultGroup() GroupInput {
	return catalog.defaultGroup
}

// DefaultProfile returns a copy of the catalog's fixed profile suggestion.
// Callers can change the returned value without changing the catalog fields.
func (catalog *OnboardingCatalog) DefaultProfile() ProfileInput {
	return catalog.defaultProfile
}

// GroupDraftCreator supplies the group product for a shared onboarding flow.
type GroupDraftCreator interface {
	CreateGroupDraft() (GroupInput, error)
}

// StudentGroupCreator makes the group draft used for a student household.
type StudentGroupCreator struct{}

// CreateGroupDraft builds the group input for the student household example.
// It returns a validated draft for the shared workflow to save.
func (StudentGroupCreator) CreateGroupDraft() (GroupInput, error) {
	return NewGroupBuilder().
		WithName("Student shared home").
		WithMaxMonthlyCost(650).
		WithCapacity(3).
		Build()
}

// QuietGroupCreator makes a smaller group draft for a quieter household.
type QuietGroupCreator struct{}

// CreateGroupDraft builds the group input for the quieter household example.
// It returns a validated draft for the shared workflow to save.
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

// NewGroupOnboardingWorkflow connects the shared flow to the Domi services.
// It uses the process catalog for the profile input in each run.
func NewGroupOnboardingWorkflow(groups *GroupService, profiles *ProfileService) GroupOnboardingWorkflow {
	return GroupOnboardingWorkflow{
		groups:   groups,
		profiles: profiles,
		catalog:  DefaultOnboardingCatalog(),
	}
}

// Run asks the creator for a group draft, then saves and joins both records.
// It returns an error immediately if any creation or join step fails.
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

// NewBufferPool sets the maximum number of reset buffers retained for reuse.
// A negative limit is treated as zero.
func NewBufferPool(maxIdle int) *BufferPool {
	if maxIdle < 0 {
		maxIdle = 0
	}
	return &BufferPool{maxIdle: maxIdle}
}

// Acquire takes the most recently released buffer when one is available.
// It creates a new empty buffer when the idle list is empty.
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

// Release clears the buffer and retains it only while the idle limit allows.
// A nil buffer is ignored.
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

// IdleCount reports the number of reset buffers held for reuse.
// It does not count buffers currently borrowed from the pool.
func (pool *BufferPool) IdleCount() int {
	return len(pool.idle)
}
