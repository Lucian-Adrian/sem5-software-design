package main

import (
	"errors"
	"fmt"
	"strings"
)

// StudyGroup owns its current lifecycle state and member names.
// Names stand in for profile IDs in this isolated two-place example.
type StudyGroup struct {
	capacity int
	members  []string
	state    GroupState
}

// GroupState supplies the join behavior for the group's current lifecycle.
// The group changes this collaborator when it closes or reopens.
type GroupState interface {
	Join(group *StudyGroup, member string) error
}

type OpenState struct{}
type ClosedState struct{}

// newStudyGroup creates a valid, open group with two places.
// The fixed capacity keeps the example focused on lifecycle behavior.
func newStudyGroup() *StudyGroup {
	return &StudyGroup{capacity: 2, state: OpenState{}}
}

// Join accepts a new member only while a place remains available.
// Membership changes here, after the open state's capacity check.
func (OpenState) Join(group *StudyGroup, member string) error {
	if len(group.members) >= group.capacity {
		return errors.New("group is full")
	}
	group.members = append(group.members, member)
	return nil
}

// Join rejects a new member while the group is closed.
// It leaves existing membership untouched.
func (ClosedState) Join(group *StudyGroup, member string) error {
	return errors.New("group is closed")
}

// Join validates the name and treats an existing membership as a successful retry.
// Only a new membership request reaches the current state's behavior.
func (group *StudyGroup) Join(member string) error {
	member = strings.TrimSpace(member)
	if member == "" {
		return errors.New("member name is required")
	}
	for _, existingMember := range group.members {
		if existingMember == member {
			return nil
		}
	}
	return group.state.Join(group, member)
}

// Close changes the behavior of future new-membership requests.
// Closing does not delete members who already joined.
func (group *StudyGroup) Close() {
	group.state = ClosedState{}
}

// Reopen restores open-state behavior without replacing the group itself.
// A reopened group must still enforce its capacity limit.
func (group *StudyGroup) Reopen() {
	group.state = OpenState{}
}

// AdmissionRequest contains a read-only snapshot for the separate chain case.
// These checks decide eligibility; they do not create membership or reserve space.
type AdmissionRequest struct {
	MonthlyBudget int
	MonthlyCost   int
	MemberCount   int
	Capacity      int
}

// AdmissionCheck is one linked handler with one rule and an optional successor.
// A handler forwards the request only after its own rule succeeds.
type AdmissionCheck struct {
	name  string
	check func(AdmissionRequest) error
	next  *AdmissionCheck
}

// Handle prints its rule, stops on failure, or forwards to the next handler.
// Success at the end means every configured check accepted this snapshot.
func (handler *AdmissionCheck) Handle(request AdmissionRequest) error {
	fmt.Printf("check %s\n", handler.name)
	if err := handler.check(request); err != nil {
		return err
	}
	if handler.next != nil {
		return handler.next.Handle(request)
	}
	return nil
}

// checkBudget accepts equality between the person's budget and the cost.
// A failure stops the chain before its next rule runs.
func checkBudget(request AdmissionRequest) error {
	if request.MonthlyBudget < request.MonthlyCost {
		return errors.New("budget too low")
	}
	return nil
}

// checkCapacity accepts the snapshot only when at least one place remains.
// A later real join must recheck capacity together with its storage update.
func checkCapacity(request AdmissionRequest) error {
	if request.MemberCount >= request.Capacity {
		return errors.New("group is full")
	}
	return nil
}

// resultText makes both successful and rejected operations visible in the trace.
// A nil error means success; other errors supply their explanatory text.
func resultText(err error) string {
	if err != nil {
		return err.Error()
	}
	return "ok"
}

// runStateCase keeps one group while its lifecycle changes from open to closed.
// The repeated join illustrates the duplicate rule before state delegation.
func runStateCase() {
	fmt.Println("State case")
	group := newStudyGroup()
	err := group.Join("Alex")
	fmt.Printf("open, join Alex: %s, members=%d\n", resultText(err), len(group.members))
	group.Close()
	err = group.Join("Pat")
	fmt.Printf("closed, join Pat: %s, members=%d\n", resultText(err), len(group.members))
	err = group.Join("Alex")
	fmt.Printf("closed, repeat Alex: %s, members=%d\n", resultText(err), len(group.members))
	group.Reopen()
	err = group.Join("Pat")
	fmt.Printf("reopened, join Pat: %s, members=%d\n", resultText(err), len(group.members))
}

// runChainCase changes one request field at a time through the same linked rules.
// The printed checks reveal which handler stops or forwards each request.
func runChainCase() {
	fmt.Println("Chain case")
	capacity := &AdmissionCheck{name: "capacity", check: checkCapacity}
	budget := &AdmissionCheck{name: "budget", check: checkBudget, next: capacity}
	request := AdmissionRequest{MonthlyBudget: 400, MonthlyCost: 500, MemberCount: 2, Capacity: 2}
	fmt.Println("both rules fail:")
	err := budget.Handle(request)
	fmt.Printf("checks: %s\n", resultText(err))
	request.MonthlyBudget = 550
	fmt.Println("budget fixed:")
	err = budget.Handle(request)
	fmt.Printf("checks: %s\n", resultText(err))
	request.MemberCount = 1
	fmt.Println("one place free:")
	err = budget.Handle(request)
	fmt.Printf("checks: %s\n", resultText(err))
}

// main runs two optional study cases in one process.
// This package is separate from Lab 3's assessed executable and pattern types.
func main() {
	runStateCase()
	runChainCase()
}
