package main

import (
	"sort"
	"strconv"
	"sync"
)

// MemoryStore is the in-memory implementation of the storage contracts.
type MemoryStore struct {
	mutex         sync.RWMutex
	groups        map[string]Group
	profiles      map[string]Profile
	nextGroupID   int
	nextProfileID int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		groups:   make(map[string]Group),
		profiles: make(map[string]Profile),
	}
}

func (store *MemoryStore) CreateGroup(input GroupInput) (GroupView, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// Validate the candidate record before consuming its ID.
	id := formatID("group", store.nextGroupID+1)
	group, err := newGroup(id, input)
	if err != nil {
		return GroupView{}, err
	}
	store.nextGroupID++
	store.groups[id] = group
	return group.view(), nil
}

func (store *MemoryStore) CreateProfile(input ProfileInput) (ProfileView, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// Invalid input must not consume a profile ID.
	id := formatID("profile", store.nextProfileID+1)
	profile, err := newProfile(id, input)
	if err != nil {
		return ProfileView{}, err
	}
	store.nextProfileID++
	store.profiles[id] = profile
	return profile.view(), nil
}

func (store *MemoryStore) ListGroups() ([]GroupView, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	groups := make([]GroupView, 0, len(store.groups))
	for _, group := range store.groups {
		groups = append(groups, group.view())
	}

	// Map iteration has no stable order; sort views for predictable output.
	sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })
	return groups, nil
}

func (store *MemoryStore) GetGroup(id string) (GroupDetail, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	group, found := store.groups[id]
	if !found {
		return GroupDetail{}, ErrGroupNotFound
	}
	return store.groupDetail(group), nil
}

// JoinGroup checks the profile, affordability and capacity while holding one
// lock. Concurrent joins therefore cannot overfill the same group.
func (store *MemoryStore) JoinGroup(groupID, profileID string) (GroupDetail, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// Find both records before attempting a membership change.
	group, found := store.groups[groupID]
	if !found {
		return GroupDetail{}, ErrGroupNotFound
	}
	profile, found := store.profiles[profileID]
	if !found {
		return GroupDetail{}, ErrProfileNotFound
	}

	// Keep the domain check and save inside the same lock.
	if err := group.addMember(profile); err != nil {
		return GroupDetail{}, err
	}

	// Map lookup returned a group value, so write the changed value back.
	store.groups[groupID] = group
	return store.groupDetail(group), nil
}

func (store *MemoryStore) groupDetail(group Group) GroupDetail {
	members := make([]ProfileView, 0, len(group.memberProfileIDs))
	// Build response views instead of returning stored profile objects.
	for _, profileID := range group.memberProfileIDs {
		members = append(members, store.profiles[profileID].view())
	}
	return GroupDetail{Group: group.view(), Members: members}
}

func formatID(prefix string, number int) string {
	return prefix + "-" + strconv.Itoa(number)
}
