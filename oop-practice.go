// Run alone: go run oop-practice.go. This is practice, not a lab checkpoint.
package main

import (
	"fmt"
	"os"
	"strings"
)

type Profile struct {
	id     string
	budget int
}

// NewProfile validates the identity and budget before returning a person.
// On failure, the caller receives an empty value and an error to check.
func NewProfile(id string, budget int) (Profile, error) {
	id = strings.TrimSpace(id)
	if id == "" || budget < 0 {
		return Profile{}, fmt.Errorf("profile needs an ID and a nonnegative budget")
	}
	return Profile{id: id, budget: budget}, nil
}

type Group struct {
	id       string
	name     string
	cost     int
	capacity int
	members  []string
}

// NewGroup checks every setting before constructing the shared group.
// The pointer lets later methods update this same group's state.
func NewGroup(id, name string, cost, capacity int) (*Group, error) {
	id, name = strings.TrimSpace(id), strings.TrimSpace(name)
	if id == "" || name == "" || cost <= 0 || capacity < 1 || capacity > 20 {
		return nil, fmt.Errorf("group needs ID, name, positive cost and capacity 1..20")
	}
	return &Group{id: id, name: name, cost: cost, capacity: capacity}, nil
}

// Name exposes one fact without exposing a way to replace all internal state.
// Reading the name does not need to change the caller's group.
func (group Group) Name() string {
	return group.name
}

// Rename validates before assigning, so rejection preserves the old name.
// The pointer receiver applies a successful change to the caller's group.
func (group *Group) Rename(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	group.name = name
	return nil
}

type GroupView struct {
	ID          string
	Cost        int
	Capacity    int
	MemberCount int
}

// View copies the public facts into a snapshot without exposing member storage.
// Later joins change the group, while this returned member count stays the same.
func (group Group) View() GroupView {
	return GroupView{
		ID:          group.id,
		Cost:        group.cost,
		Capacity:    group.capacity,
		MemberCount: len(group.members),
	}
}

// Join checks existing membership, budget and capacity before adding a person.
// A repeated successful join returns success without adding a duplicate ID.
func (group *Group) Join(profile Profile) error {
	// A repeated successful join stays successful, including after the group fills.
	for _, memberID := range group.members {
		if memberID == profile.id {
			return nil
		}
	}
	if profile.budget < group.cost {
		return fmt.Errorf("budget too low")
	}
	if len(group.members) >= group.capacity {
		return fmt.Errorf("group full")
	}
	group.members = append(group.members, profile.id)
	return nil
}

// Filters select snapshots; they cannot reserve places or perform joins.
type GroupFilter interface {
	Matches(GroupView) bool
}

type AffordableGroups struct{ Budget int }

// Matches accepts a snapshot when the rent fits the configured budget.
// It answers affordability only, so even a full group may match.
func (filter AffordableGroups) Matches(group GroupView) bool {
	return group.Cost <= filter.Budget
}

type AvailableGroups struct{ Budget int }

// Matches requires both affordable rent and an unoccupied place.
// It reads a snapshot, so a successful match cannot reserve that place.
func (filter AvailableGroups) Matches(group GroupView) bool {
	return group.Cost <= filter.Budget && group.MemberCount < group.Capacity
}

// CountMatches asks the supplied rule to select each group snapshot.
// Different rules work through the same interface without changing this loop.
func CountMatches(groups []GroupView, filter GroupFilter) int {
	count := 0
	for _, group := range groups {
		if filter.Matches(group) {
			count++
		}
	}
	return count
}

// run follows one group through rejected, successful and repeated operations.
// Each printed result exposes a state change or a value that remains unchanged.
func run() error {
	group, err := NewGroup("group-1", " Shared flat ", 500, 1)
	if err != nil {
		return err
	}
	alex, err := NewProfile("alex", 550)
	if err != nil {
		return err
	}
	pat, err := NewProfile("pat", 400)
	if err != nil {
		return err
	}
	affordable := AffordableGroups{Budget: 550}
	available := AvailableGroups{Budget: 550}
	before := group.View()
	fmt.Printf("created: %s, members=%d\n", group.Name(), before.MemberCount)
	fmt.Printf("before: affordable=%d available=%d\n",
		CountMatches([]GroupView{before}, affordable), CountMatches([]GroupView{before}, available))
	fmt.Printf("Pat: %v, members=%d\n", group.Join(pat), group.View().MemberCount)
	if err := group.Join(alex); err != nil {
		return err
	}
	fmt.Printf("Alex: ok, members=%d\n", group.View().MemberCount)
	if err := group.Join(alex); err != nil {
		return err
	}
	fmt.Printf("Alex again: ok, members=%d\n", group.View().MemberCount)
	after := group.View()
	fmt.Printf("after: affordable=%d available=%d\n",
		CountMatches([]GroupView{after}, affordable), CountMatches([]GroupView{after}, available))
	fmt.Printf("snapshot before: members=%d\n", before.MemberCount)
	err = group.Rename("   ")
	fmt.Printf("rename blank: %v, name=%s\n", err, group.Name())
	alias := group
	if err := alias.Rename("Shared home"); err != nil {
		return err
	}
	fmt.Printf("through alias: name=%s same pointer=%t\n", group.Name(), alias == group)
	return nil
}

// main runs the practice and sends any unexpected failure to standard error.
// Exit status 1 lets a terminal or another program detect that failure.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
