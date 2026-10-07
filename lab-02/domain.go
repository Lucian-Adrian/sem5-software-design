package main

import (
	"errors"
	"fmt"
	"strings"
)

const maxGroupCapacity = 20

var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrGroupNotFound   = errors.New("group not found")
	ErrProfileNotFound = errors.New("profile not found")
	ErrGroupFull       = errors.New("group is full")
	ErrBudgetTooLow    = errors.New("monthly budget is below the group's cost")
)

// Group keeps membership rules beside its data. Lowercase fields are package-private.
type Group struct {
	id               string
	name             string
	maxMonthlyCost   int
	capacity         int
	memberProfileIDs []string
}

// Profile keeps the personal budget inside the domain model. API responses omit it.
type Profile struct {
	id            string
	name          string
	monthlyBudget int
}

type GroupInput struct {
	Name           string `json:"name"`
	MaxMonthlyCost int    `json:"maxMonthlyCost"`
	Capacity       int    `json:"capacity"`
}

type ProfileInput struct {
	Name          string `json:"name"`
	MonthlyBudget int    `json:"monthlyBudget"`
}

type GroupView struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	MaxMonthlyCost int    `json:"maxMonthlyCost"`
	Capacity       int    `json:"capacity"`
	MemberCount    int    `json:"memberCount"`
}

type ProfileView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type GroupDetail struct {
	Group   GroupView     `json:"group"`
	Members []ProfileView `json:"members"`
}

func newGroup(id string, input GroupInput) (Group, error) {
	input, err := normalizeGroupInput(input)
	if err != nil {
		return Group{}, err
	}

	return Group{
		id:             id,
		name:           input.Name,
		maxMonthlyCost: input.MaxMonthlyCost,
		capacity:       input.Capacity,
	}, nil
}

func normalizeGroupInput(input GroupInput) (GroupInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return GroupInput{}, fmt.Errorf("%w: group name is required", ErrInvalidInput)
	}
	if input.MaxMonthlyCost <= 0 {
		return GroupInput{}, fmt.Errorf("%w: maxMonthlyCost must be greater than zero", ErrInvalidInput)
	}
	if input.Capacity <= 0 {
		return GroupInput{}, fmt.Errorf("%w: capacity must be greater than zero", ErrInvalidInput)
	}
	if input.Capacity > maxGroupCapacity {
		return GroupInput{}, fmt.Errorf("%w: capacity cannot exceed %d people in a roommate group", ErrInvalidInput, maxGroupCapacity)
	}
	return input, nil
}

func newProfile(id string, input ProfileInput) (Profile, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Profile{}, fmt.Errorf("%w: profile name is required", ErrInvalidInput)
	}
	if input.MonthlyBudget < 0 {
		return Profile{}, fmt.Errorf("%w: monthlyBudget cannot be negative", ErrInvalidInput)
	}

	return Profile{id: id, name: name, monthlyBudget: input.MonthlyBudget}, nil
}

func (group Group) view() GroupView {
	return GroupView{
		ID:             group.id,
		Name:           group.name,
		MaxMonthlyCost: group.maxMonthlyCost,
		Capacity:       group.capacity,
		MemberCount:    len(group.memberProfileIDs),
	}
}

func (profile Profile) view() ProfileView {
	// Public member details deliberately omit the personal budget.
	return ProfileView{ID: profile.id, Name: profile.name}
}

func (group *Group) addMember(profile Profile) error {
	// Repeating a completed join must succeed even when the group is now full.
	for _, memberID := range group.memberProfileIDs {
		if memberID == profile.id {
			return nil
		}
	}

	// Reject a new member before changing the membership list.
	if profile.monthlyBudget < group.maxMonthlyCost {
		return ErrBudgetTooLow
	}
	if len(group.memberProfileIDs) >= group.capacity {
		return ErrGroupFull
	}

	// All checks passed; membership changes exactly once.
	group.memberProfileIDs = append(group.memberProfileIDs, profile.id)
	return nil
}
