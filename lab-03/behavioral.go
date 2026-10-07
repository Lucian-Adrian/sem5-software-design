package main

import "fmt"

// AvailableGroups holds the search budget for one availability rule.
// GroupService can use it through the existing GroupFilter interface.
type AvailableGroups struct{ MonthlyBudget int }

// Matches returns true when the group fits the budget and has a free place.
// The store checks capacity again when someone tries to join.
func (filter AvailableGroups) Matches(group GroupView) bool {
	return group.MaxMonthlyCost <= filter.MonthlyBudget &&
		group.MemberCount < group.Capacity
}

// GroupCommand describes one request that a dispatcher can run.
// A command stores the receiver and values needed for that request.
type GroupCommand interface {
	// Execute asks the receiver to perform the request.
	// It returns the operation result or the receiver's error.
	Execute() (GroupDetail, error)
}

// JoinCommand stores the service and IDs for one join request.
// The existing service and store still enforce membership rules.
type JoinCommand struct {
	Groups    *GroupService
	GroupID   string
	ProfileID string
}

// Execute sends this request to the configured group service.
// A missing service returns an input error before the join is attempted.
func (command JoinCommand) Execute() (GroupDetail, error) {
	if command.Groups == nil {
		return GroupDetail{}, fmt.Errorf("%w: join command needs a group service", ErrInvalidInput)
	}
	return command.Groups.JoinGroup(command.GroupID, command.ProfileID)
}

// CommandResult is a public snapshot of one successful request.
// It contains no private budgets or member names.
type CommandResult struct {
	GroupID     string `json:"groupId"`
	MemberCount int    `json:"memberCount"`
}

// CompletionObserver receives a result after a command succeeds.
// Each implementation decides what to do with the result.
type CompletionObserver interface {
	// OnCompleted receives the public result of a successful command.
	// The dispatcher calls observers in subscription order.
	OnCompleted(CommandResult)
}

// CommandDispatcher runs commands and sends successful results to observers.
// It calls them synchronously in the order they subscribed.
type CommandDispatcher struct {
	observers []CompletionObserver
}

// Subscribe adds an observer to the end of the listener list.
// A nil interface is ignored; callers must not pass a typed nil observer.
func (dispatcher *CommandDispatcher) Subscribe(observer CompletionObserver) {
	if observer != nil {
		dispatcher.observers = append(dispatcher.observers, observer)
	}
}

// Run executes one command and returns its error without notifying observers.
// After success, it sends the public result to every observer.
func (dispatcher *CommandDispatcher) Run(command GroupCommand) (GroupDetail, error) {
	if command == nil {
		return GroupDetail{}, fmt.Errorf("%w: command is required", ErrInvalidInput)
	}
	result, err := command.Execute()
	if err != nil {
		return result, err
	}
	event := CommandResult{GroupID: result.Group.ID, MemberCount: result.Group.MemberCount}
	for _, observer := range dispatcher.observers {
		observer.OnCompleted(event)
	}
	return result, nil
}

// CompletionLog stores the results delivered to it.
// The demo uses it as a simple in-memory observer.
type CompletionLog struct{ Events []CommandResult }

// OnCompleted saves a result without reading or changing group storage.
// The dispatcher decides when this method is called.
func (log *CompletionLog) OnCompleted(event CommandResult) {
	log.Events = append(log.Events, event)
}
