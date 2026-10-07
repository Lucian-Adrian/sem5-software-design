package main

import (
	"sort"
	"strconv"
	"sync"
)

// MemoryStore is the in-memory implementation of the storage contracts.
type MemoryStore struct {
	mu            sync.RWMutex
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
	store.mu.Lock()
	defer store.mu.Unlock()

	store.nextGroupID++
	id := formatID("group", store.nextGroupID)
	group, err := newGroup(id, input)
	if err != nil {
		store.nextGroupID--
		return GroupView{}, err
	}
	store.groups[id] = group
	return group.view(), nil
}

func (store *MemoryStore) CreateProfile(input ProfileInput) (ProfileView, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.nextProfileID++
	id := formatID("profile", store.nextProfileID)
	profile, err := newProfile(id, input)
	if err != nil {
		store.nextProfileID--
		return ProfileView{}, err
	}
	store.profiles[id] = profile
	return profile.view(), nil
}

func (store *MemoryStore) ListGroups() ([]GroupView, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	groups := make([]GroupView, 0, len(store.groups))
	for _, group := range store.groups {
		groups = append(groups, group.view())
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })
	return groups, nil
}

func (store *MemoryStore) GetGroup(id string) (GroupDetail, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	group, found := store.groups[id]
	if !found {
		return GroupDetail{}, ErrGroupNotFound
	}
	return store.groupDetail(group), nil
}

// JoinGroup checks the profile, affordability and capacity while holding one
// lock. Concurrent joins therefore cannot overfill the same group.
func (store *MemoryStore) JoinGroup(groupID, profileID string) (GroupDetail, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	group, found := store.groups[groupID]
	if !found {
		return GroupDetail{}, ErrGroupNotFound
	}
	profile, found := store.profiles[profileID]
	if !found {
		return GroupDetail{}, ErrProfileNotFound
	}
	if err := group.addMember(profile); err != nil {
		return GroupDetail{}, err
	}
	store.groups[groupID] = group
	return store.groupDetail(group), nil
}

func (store *MemoryStore) groupDetail(group Group) GroupDetail {
	members := make([]ProfileView, 0, len(group.memberProfileIDs))
	for _, profileID := range group.memberProfileIDs {
		members = append(members, store.profiles[profileID].view())
	}
	return GroupDetail{Group: group.view(), Members: members}
}

func formatID(prefix string, number int) string {
	return prefix + "-" + strconv.Itoa(number)
}
