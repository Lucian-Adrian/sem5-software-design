# Explain and defend Lab 3

Read the [report](report.md) beside [behavioral.go](behavioral.go). This guide develops the mechanism and then provides practice. A prepared script is support for learning; independent explanation has not yet been observed.

## Start with state and a decision

A group has a cost per person, a capacity and a current member count. A profile has a private monthly budget. Alex's budget is 550. Shared flat costs 500 and has one place. Pat's budget is 400. These example values isolate two different reasons a join can fail: insufficient budget and lack of capacity.

Before joining, `500 <= 550` and `0 < 1`, so Shared flat is available to a person searching with budget 550. After Alex joins, `500 <= 550` remains true but `1 < 1` is false. The group is still affordable but no longer available. An object that knows how to answer this yes/no question is a selection strategy.

The `GroupFilter` interface in `service.go` requires a `Matches(GroupView) bool` method. It defines what the service may ask, not how every filter must decide. `AvailableGroups` holds a budget and implements that method. `AffordableGroups` ignores capacity; `AllGroups` always includes the record. Go checks interface satisfaction from method sets without an `implements` declaration. A struct is a description of fields; a value such as `AvailableGroups{MonthlyBudget:550}` is a particular configured rule.

![Change the rule without changing the listing loop](assets/strategy.svg)

`ListGroups` reads views and calls `Matches` on each one. It retains those returning true. A caller changes behavior by supplying another implementation. That is Strategy: selection is delegated to an interchangeable rule. It also illustrates Open/Closed from Lab 0. These classifications describe different aspects of the same design; discovering the Strategy interpretation does not require replacing working code.

### A bridge from beginner Python

If you know basic Python, think of `AvailableGroups` as a small object that stores a budget and has a method that checks one group:

```python
class AvailableGroups:
    def __init__(self, monthly_budget):
        self.monthly_budget = monthly_budget

    def matches(self, group):
        return group.max_monthly_cost <= self.monthly_budget and group.member_count < group.capacity
```

Go uses a `struct` for the stored fields and a method with an explicit receiver instead of Python's `self`:

```go
type AvailableGroups struct{ MonthlyBudget int }

// Matches checks the budget and place count for one group.
// The store checks capacity again when someone tries to join.
func (filter AvailableGroups) Matches(group GroupView) bool {
    return group.MaxMonthlyCost <= filter.MonthlyBudget &&
        group.MemberCount < group.Capacity
}
```

`AvailableGroups{MonthlyBudget: 550}` creates a value with that budget. It satisfies `GroupFilter` because it has the required `Matches(GroupView) bool` method; Go needs no `implements` declaration. A Python programmer might use a base class or protocol for this contract, but Go checks the method shape directly. Calls that can fail also return an error value, so code handles `result, err := ...` instead of using an exception for ordinary domain failures. The `&CompletionLog{}` expression later gives a pointer so `OnCompleted` can update the log's stored event slice.

A listing result cannot reserve a place. Another request could fill the group between listing and joining. The store therefore checks membership under its lock on every join. This is the connection between a read-time recommendation and a write-time invariant. The dispatcher is single-threaded in this demo, but the inherited HTTP service can receive concurrent requests and relies on its store's locking.

## Turn a request into a value

Normally a client can call `groups.JoinGroup(groupID, profileID)` directly. That call is sufficient when no generic invoker is needed. Here the demo instead creates a `JoinCommand` containing the same receiver and arguments. The command's `Execute` method makes the call later when requested by the dispatcher.

```text
JoinCommand value = receiver + group ID + profile ID
CommandDispatcher.Run(value)
  -> value.Execute()
  -> GroupService.JoinGroup(ids)
  -> MemoryStore.JoinGroup(ids), under the store lock
  -> domain validation and membership update
```

`GroupCommand` is the command interface; `JoinCommand` is a concrete command; `CommandDispatcher` is the invoker; `GroupService` is the receiver. These names describe roles. A Go type can have more than one role, so the dispatcher can also be an Observer subject. The command owns no group data and duplicates no domain rule. A nil service is rejected before delegation. A missing profile or full group is rejected by the actual domain/storage path.

![Invoker, request and receiver](assets/command.svg)

Command does not mean every request has undo. Undo would need a correct domain operation and rules for later changes; this backend has no leave operation. Nor does a command value by itself provide persistence, a background worker or safe retries across process restarts. This lab uses an immediate synchronous call.

Strategy and Command both hide behavior behind a contract, but their purpose differs here. `Matches` answers a policy question without modifying the group. `Execute` represents a requested join that can modify membership. Naming both interfaces `Run` would not erase that distinction.

## Notify listeners after success

An observer reacts to a subject's event without the subject hard-coding each reaction. The demo creates a `CompletionLog` and subscribes it to the dispatcher. `Run` first executes the command. If an error is returned, it passes that error to the caller and sends no success event. If execution succeeds, it creates a small event value and calls each registered observer.

![Successful results fan out to subscribed listeners](assets/observer.svg)

The event carries only group ID and member count. It is a snapshot of a successful invocation, not permission to mutate storage. The log stores those values in a slice. The subject knows only `CompletionObserver.OnCompleted`, so another observer can be added without changing its success loop. Subscription is done before execution; unsubscribe and concurrent registration are not implemented.

Notice the event name. The store first checks whether Alex is already a member. A second join succeeds without adding Alex again. Both calls completed successfully, so the log receives two events with member count one. Calling this event `MemberAdded` would incorrectly imply two insertions. An observer that sends mail would need its own duplicate-handling rule or a different event emitted only for a new insertion. This lab sends no messages.

Callbacks run immediately on the caller's execution path, in subscription order. A slow callback delays the return. A panic interrupts delivery to later callbacks after membership is already saved; the dispatcher does not recover or roll back. Asynchronous pub/sub and a durable transactional outbox address other requirements, but are not present here. The in-memory log and observer list are also not protected against concurrent access. Do not infer their thread safety from the store's mutex.

## Reconstruct the complete demonstration

1. Create fresh storage, group/profile services, Alex, Pat and the one-place group.
2. Apply `AvailableGroups{550}`: one group matches.
3. Register the completion log.
4. Execute Pat's join: budget rejection; no event.
5. Execute Alex's join: one member; first completion event.
6. Execute Alex's join again: still one member; second completion event.
7. Apply the same strategy again: no group matches because capacity is full.

![Calls for a successful execution](assets/sequence.svg)

Run `go run . --behavioral` and point to those fields in its JSON output. To inspect all new checks, run `go test -count=1 -run 'TestAvailableStrategyBoundary|TestCommandObserverOutcomes' -v`. The full `go test -count=1 ./...` also checks the inherited backend and pattern examples. A test passing is evidence about exercised code, not evidence that you can explain it without notes.

## Follow the source during a presentation

Start at `behavioral_demo.go`. `runBehavioralDemo` creates fresh storage and sample people, calls the available-group rule, wires a command dispatcher to two kinds of work, then writes one JSON result. This file provides the concrete story, not the reusable policy.

Move to `behavioral.go` to explain the reusable parts in the order the demo reaches them: `AvailableGroups.Matches` selects a group; `JoinCommand.Execute` delegates the request to `GroupService`; `CommandDispatcher.Run` returns errors before notifying; and `CompletionLog.OnCompleted` records each successful result. The service and store still own the existing join contract and membership rules.

Finish at `behavioral_test.go`. `TestAvailableStrategyBoundary` checks budget equality and what happens when capacity is used. `TestCommandObserverOutcomes` shows that a failed request sends no event, while two successful invocations reach both observers and still leave only one member. Tests check those cases directly; they do not generate the presentation JSON.

## Distinguish the other behavioral patterns

The ELSE list also asks for knowledge of the remaining patterns. Use their mechanism, then relate it to Domi rather than adding code just to collect names.

| Pattern | Mechanism and a possible Domi use | Nearest distinction |
|---|---|---|
| Chain of Responsibility | A request moves through handlers, each deciding to handle or pass it on; a moderation pipeline could use this. | Observer normally fans an event out to listeners; a chain has a forwarding path and may stop. |
| Interpreter | Classes/operations represent grammar rules and evaluate sentences; a small saved-search language could qualify. | Parsing any JSON request is not automatically a grammar-interpreter pattern. ELSE marks this good to know. |
| Iterator | A traversal contract exposes successive elements while hiding collection representation; a paged group collection could expose traversal. | A strategy decides inclusion; an iterator decides how traversal proceeds. A plain loop need not be advertised as a separate implemented pattern. |
| Mediator | A central object coordinates participants so they avoid direct many-to-many references; a more complex approval conversation could use one. | A facade offers a simpler entry point; a mediator manages participant interactions. |
| Memento | A caretaker stores an opaque snapshot supplied by the originator for later restoration; draft editing could use this. | Prototype creates a copy for use as another object; Memento supports restoring prior state. |
| State | A context delegates behavior to its current state and transitions change permitted operations; pending/approved/closed groups could use this. | Strategy is usually selected as a policy; State behavior changes with the context's lifecycle. Current groups implement no approval lifecycle. |
| Template Method | A base algorithm fixes operation order while extension hooks supply some steps. | Strategy uses composition with a replaceable object. Go can express composition and callbacks; this lab does not simulate an inheritance hierarchy. |
| Visitor | An operation is separated from an element hierarchy, with dispatch selecting the operation for each element type. | Adding operations becomes easier, while adding element types can require visitor changes. Domi's few simple records do not currently justify it. |

These are theory illustrations and possible extensions, not claims about implemented features.

## A short oral route

“My third lab extends the Domi roommate backend with three behavioral patterns. Strategy selects affordable groups with free places through the existing filter contract. Command packages a join request with its service and IDs so one dispatcher can execute it. Observer lets subscribed listeners react after successful execution. In the demo, Pat cannot afford the group, Alex joins, and Alex's repeated request succeeds without adding another member. That produces two successful-invocation events but one membership. Availability is only a snapshot, so the store still checks the rules under its lock. Notification is synchronous and not transactional.”

Then point to one diagram and trace the real call into storage. Explain one limitation rather than claiming a production feature.

## Questions to answer before reading the checks

1. Change Alex's budget from 550 to 500, then 499. What changes in filtering and joining?
2. Change capacity from 1 to 2. Which JSON field changes after Alex's first and repeated joins?
3. Replace `AvailableGroups` with `AffordableGroups`. Does the full group disappear? Why?
4. Subscribe two logs. Pat fails, Alex joins, Alex retries. How many events does each log receive, and why?
5. A listener panics after membership is saved. Has joining been undone? What can a later listener assume?
6. Why is Command different from Strategy here? Why is Observer different from a Decorator around `GroupWriter`?
7. If a group becomes full after listing, which component must reject a new join? Does a filter guarantee availability?
8. Does the store's mutex make `CompletionLog.Events` safe for concurrent writers? Explain the protected state.
9. Which pattern fits a future pending/approved group lifecycle? Which fits saving an editable draft's undo history?
10. Could a typed nil observer satisfy the interface and still panic when called? What does the constructor/caller need to ensure?

### Checking answers

1. At 500 both budget comparisons permit equality; at 499 the group is not selected and joining is rejected. A search budget is an input to the filter; joining uses the stored profile budget, so they must not be conflated if a caller supplies different values.
2. After the first join there is still one free place, so `availableAfter` contains the group. A repeat adds no member and leaves that result unchanged. Each successful invocation still emits a completion.
3. It remains in an affordability-only result: cost is within budget even when no place is free.
4. Each receives two events, in order, because only the two successful invocations notify. The group has one member. Multiple subscriptions are separate registrations; no deduplication is promised.
5. No rollback occurs. Later listeners may not be called. The store mutation precedes the notification loop.
6. Strategy answers a selection-policy question; Command represents a requested operation with a receiver and arguments. The earlier Decorator wraps the writer contract and augments its calls; Observer maintains a list of subscribers and delivers an event to each after successful dispatch. Neither implies durable delivery.
7. `MemoryStore.JoinGroup` checks the domain rule while holding its lock. Listing provides no reservation.
8. No. The store lock protects its maps and membership changes. It does not surround dispatcher callbacks or append operations on the log slice.
9. State for lifecycle-dependent behavior; Memento for restoring a previous snapshot. These are proposed applications, not existing Domi features.
10. Yes. An interface holding a nil pointer is not the same as a nil interface. The composition code must supply initialized observers/commands; the demo does so. The simple nil checks are not a universal defense against every invalid custom implementation.

## Next practical rehearsal and real feedback

Allow a proposed 15 minutes: five reading and tracing, three explaining the demo, four answering one changed-input and one adjacent-concept question, three correcting the first gap. Repeat later with a fresh case. Record only observed ability as independent, assisted or unresolved.

Use the [shared feedback record](../../semester-info/defense-preparation/README.md#observe-and-incorporate-real-feedback) after watching classmates or presenting. Keep exact professor questions separate from inferred expectations. Correct the relevant explanation here and add a regression check only when feedback exposes a real behavior requirement. No defense result has been recorded for this checkpoint.
