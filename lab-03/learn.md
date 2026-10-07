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

## Reconstruct the dispatcher before adding more patterns

The course asks for three behavioral patterns and knowledge of the remaining list. For this project, the useful design question is which part changes: a selection rule, a requested operation, or a reaction to success. Strategy handles the rule, Command carries the operation, and Observer supplies the reactions. A direct service call is sufficient when no common invoker or subscribers are needed.

### Run one small example using the actual types

The existing [behavioral test file](behavioral_test.go) now contains `ExampleCommandDispatcher_Subscribe`. It uses the real store, service, command, dispatcher and logs. From this lab folder:

```bash
go test -run '^ExampleCommandDispatcher_Subscribe$' -count=1 -v
```

Go recognizes an `Example...` function and compares its printed output with the final `Output:` comment block. The command reports pass/fail; it captures the example's output for comparison. Those comments are a Go tooling convention, not another application implementation. The example uses `panic(err)` only to fail immediately on unexpected setup/execution errors; the application still returns domain errors normally. Ordinary `go run .` excludes this test file.

The example's checked output is:

```text
after subscribe: members=1 first=1 late=0
after direct call: members=1 first=1 late=0
after dispatch: members=1 first=2 late=1
```

One successful dispatch joins Alex and notifies the first log. Registering a second log afterward does not replay the old event: `Subscribe` only appends a listener. Calling `command.Execute()` directly makes a successful repeat join through the service, but bypasses the dispatcher entirely. Finally, `dispatcher.Run(command)` executes another successful repeat and notifies both current listeners. Membership remains one throughout these three printed observations. The logs describe which successful calls passed through their dispatcher after subscription, not every change or request in the whole application.

This is why a command object is not automatically an event system. `JoinCommand.Execute` knows its receiver and arguments. `CommandDispatcher.Run` owns the notification step. Python has the same distinction: calling an object's `execute()` method directly does not also call some other object's dispatcher wrapper.

### Rebuild Run from its contract

First read the declarations of `GroupCommand`, `CommandResult` and `CompletionObserver` in [behavioral.go](behavioral.go). Then hide the implementation of `Run`. Its job can be reconstructed without reimplementing HTTP or storage.

Pseudocode:

```text
Reject a missing command.
Execute the command once.
If it fails, return its result and error without a success notification.
Copy the successful group's ID and member count into an event.
Deliver that event to each currently registered observer.
Return the successful result to the caller.
```

Reconstruct it in stages in a separate scratch package or the Go Playground. Start with small result structs and a stub command returning a known result; this isolates the dispatcher before adding real storage. Write `Execute() (GroupDetail, error)` as the contract. Add the error branch, then the event value and observer loop. Finally compare against the real `Run`, and connect the actual `JoinCommand` through the existing example. Keep scratch declarations outside this package to avoid redeclaring its types.

The central source excerpt is:

```go
result, err := command.Execute()
if err != nil {
    return result, err
}
event := CommandResult{GroupID: result.Group.ID, MemberCount: result.Group.MemberCount}
for _, observer := range dispatcher.observers {
    observer.OnCompleted(event)
}
return result, nil
```

The first line obtains two return values. The `if` stops the success path on an error; moving it below the callbacks would report success for a rejected command. The struct literal copies only the public event fields. `range` supplies an index and observer; `_` discards the unused index. Calling `OnCompleted` invokes the particular implementation behind each interface. `return result, nil` tells the caller execution succeeded. The full method also checks `command == nil` before this excerpt. Receiving a matching method set is a type check; preserving the meaning of success and failure is the behavioral contract.

### Connect the same operation across Labs 0–3

| Source | State or boundary it owns | What the later layer must preserve |
| --- | --- | --- |
| [Lab 0 addMember](../lab-00/domain.go) | Budget, capacity and duplicate membership rules | Commands delegate the join rather than inventing different rules. |
| [Lab 1 Builder.Build](../lab-01/patterns.go) | A validated creation draft | A valid draft is not yet a saved group with an assigned ID. |
| [Lab 2 OnboardingFacade.Onboard](../lab-02/structural.go) | The order of separate profile, group and join operations | A convenient method does not turn those writes into one transaction. |
| [Lab 3 CommandDispatcher.Run](behavioral.go) | Execution followed by success notifications | The notification is a later operation and does not supply rollback or durable delivery. |

The common question is: what changed, what has merely been copied, and what would remain after a later step fails? An object boundary is a way to allocate responsibilities. A transaction boundary and a process boundary have different consequences. Compare the [existing system-design explanation](../system-design.md) after the in-process trace is clear.

### Change the conditions before reading the answers

These are transfer questions beyond the earlier budget/capacity cases. State the result, the source statement responsible, and one limit of the conclusion.

1. Create and subscribe the late log before the first dispatch. Predict all three output lines. Does `Execute()` directly become observable?
2. Create a second dispatcher using the same `GroupService`, but subscribe a log only to the first. Run a successful join through the second. Which state is shared and which log changes?
3. With the original pointer receiver on `CompletionLog.OnCompleted`, pass `CompletionLog{}` rather than `&CompletionLog{}` to `Subscribe`. Does the program compile, or does it fail while notifying?
4. Suppose the entire process stops immediately after a successful join and before notification. What survives in this actual in-memory program? How would the answer change if only the store were replaced with durable database storage?

### Reason through the changed conditions

1. After the first dispatch, both logs contain one event. The direct call leaves both at one. The final dispatch raises both to two, with membership still one. Earlier registration changes the listeners present in the notification loop, not the execution path of a direct command call.
2. Membership is shared because both service paths reach the same store. The first dispatcher's log is unchanged: each dispatcher owns its own observer slice, and no global publisher connects them. A common database or mutex would not automatically share the in-process subscriber list either.
3. It fails to compile at the interface assignment. `OnCompleted` belongs to the pointer's method set, so `*CompletionLog` satisfies `CompletionObserver` while a `CompletionLog` value does not. A value stored in an interface is not automatically made addressable so Go can obtain its pointer. This differs from a typed nil pointer, which can satisfy the method set but still be unusable at runtime.
4. In the current program, the process-local store and both logs disappear on process termination; neither membership nor event delivery survives restart. With a durable database store, committed membership could survive while the notification is lost. A proposed transactional outbox would record the notification intent in the same database transaction as membership, then deliver it later with retry and duplicate-handling rules. An in-process observer list alone provides none of that durability.

No independent reconstruction has been observed. Use a fresh case after correcting a gap, rather than counting an agent-run example as personal mastery.

## Optional study: State and Chain of Responsibility

The three assessed implementations remain Strategy, Command and Observer. The other behavioral patterns are theory to understand. The two cases in [study/main.go](study/main.go) are optional executable study, not extra required implementations or additions to the backend's HTTP workflow. They run in a separate package so their small teaching types cannot replace the assessed types.

From this lab folder, run both cases with one command:

```bash
go run ./study
```

Read `runStateCase` and `runChainCase` first to see the inputs, then follow the methods they call. This program uses only the standard library. The State case uses names as stand-ins for profile IDs and a fixed two-place group. The chain uses already-formed budget/capacity snapshots. Neither case implements authentication, persistent storage or concurrent joins.

### State starts with a lifecycle problem

Suppose a roommate group has one spare place, but its organizer temporarily closes admission. A free place alone no longer determines whether a new person may join. Reopening the group should permit joining again, while retaining its existing members. The same operation now needs different behavior according to the group's lifecycle.

`StudyGroup` holds its members, capacity and current `GroupState`. The interface requires a `Join` method. `OpenState.Join` checks capacity and adds a member; `ClosedState.Join` rejects new membership without changing the list. `Close` replaces the current state with `ClosedState{}`, and `Reopen` restores `OpenState{}`. The group remains the same object with the same member list throughout these transitions.

Pseudocode:

```text
Normalize the proposed member name and reject a blank name.
If that member is already present, succeed without changing membership.
Otherwise ask the current lifecycle state to handle the join.
Open state checks capacity before adding; closed state refuses the new member.
```

The delegation point is this actual line from `StudyGroup.Join`:

```go
return group.state.Join(group, member)
```

The receiver `group` is comparable to Python's `self`. `group.state` holds a value satisfying the interface, and the dot selects its `Join` method. The first argument passes the group that may be changed. A pointer to that group lets the open-state method append to the caller's membership list. The returned error travels back to the caller; a refused join does not append a member. `newStudyGroup` supplies the initial open state, so the example does not attempt to call a method through a missing state.

The State case prints:

```text
State case
open, join Alex: ok, members=1
closed, join Pat: group is closed, members=1
closed, repeat Alex: ok, members=1
reopened, join Pat: ok, members=2
```

The closed-state refusal concerns a **new membership**. Alex's repeat succeeds because duplicate detection occurs before state delegation. That preserves the same idempotency idea as the assessed [Group.addMember](domain.go). If closing were supposed to reject even existing-member retries, the operation's contract and check order would have to change deliberately.

### State and Strategy share a mechanism but answer different questions

Both designs can store interchangeable objects behind an interface. The pattern name comes from the relationship and reason for the change. The existing [AvailableGroups.Matches](behavioral.go) is a Strategy selected by a caller to filter snapshots. The optional `StudyGroup` selects its join behavior through the lifecycle state changed by `Close` and `Reopen`.

For an equivalent snapshot with cost 500, budget 550, one member and capacity two, the existing `AvailableGroups` returns true. Its `GroupView` input has no lifecycle field, so it cannot see that this optional study group is closed. A recommendation rule and permission to mutate the actual group are separate responsibilities. Integrating lifecycle into the real backend would require an explicit domain/API change and authoritative join validation; these study types have not done that integration.

For two stable behaviors, a private `closed` boolean and an `if` inside `Join` can be clearer and shorter. State earns the extra types when lifecycle-specific operations or transitions grow and keeping those behaviors together helps. It does not become necessary simply because a value has two possible settings.

### Reconstruct State from a blank file

After reading the case, hide `study/main.go`. In an isolated scratch package, define a group with capacity, members and a current-state interface. Define open and closed state types. Rebuild `Join`, `Close` and `Reopen` in that order. Preserve name validation and duplicate success before delegation. Use the four printed observations above as predictions, then compare your behavior with the reference. Explain which method actually changes the member slice and which methods replace the state object.

Before making classes, also write the boolean-and-conditional version. Explain what the state objects improve for a proposed third state, such as suspended admission, and what additional code they cost. Neither implementation alone establishes that you understand the trade-off.

### Chain of Responsibility starts with several admission checks

A separate admission request may fail because the person cannot afford the cost or because the group has no place. We want independent checks whose order can change, and a rule that stops the request at the first failure. A linked chain gives each handler one check and a possible next handler.

This is the validation form of Chain of Responsibility: a passing handler forwards the request, while a failing handler ends processing. Other forms may stop because one handler has fully dealt with a request. Our version succeeds only after all configured checks pass.

`AdmissionRequest` holds the budget, monthly cost, member count and capacity snapshot. An `AdmissionCheck` holds its name, its check function and a pointer to the next handler. The `check func(AdmissionRequest) error` field stores a function as a value; Python can likewise store a function on an object and call it later. `next *AdmissionCheck` points to another node of the same type, or is `nil` at the end.

Pseudocode:

```text
Print which handler is checking this request.
Run this handler's one rule.
On failure, return the error immediately.
Otherwise forward to the next handler if it exists.
If there is no next handler, all configured checks passed.
```

The forwarding part of `AdmissionCheck.Handle` is:

```go
if err := handler.check(request); err != nil {
    return err
}
if handler.next != nil {
    return handler.next.Handle(request)
}
return nil
```

The first `if` calls the stored check function and receives an error. Returning there prevents the successor from running. `handler.next != nil` asks whether another node exists. Calling its `Handle` passes the same request value onward and returns that node's result. No handler adds a group member; `nil` here means eligibility checks passed, not that space was reserved or joining completed.

The chain is assembled as budget → capacity. It prints:

```text
Chain case
both rules fail:
check budget
checks: budget too low
budget fixed:
check budget
check capacity
checks: group is full
one place free:
check budget
check capacity
checks: ok
```

At first, budget 400 is below cost 500, so capacity is not inspected even though that check would also fail. Raising the budget to 550 allows forwarding, where two members in a two-place group fail the capacity check. Reducing the snapshot's member count to one makes both checks pass. The printed handler names establish the executed path; the scenario changes one field at a time.

### A chain is different from the command dispatcher

The existing [CommandDispatcher.Run](behavioral.go) receives one `GroupCommand`, executes it, and sends a successful result to registered observers. Its listeners are not successive approval gates. The optional chain instead forwards one request through validation handlers; its first failure prevents later checks. A command could use such a chain before performing its operation, but these examples have not changed the assessed command implementation.

For two fixed checks, sequential `if` statements are usually enough. A chain becomes useful when checks need independent replacement, reuse or ordering. It also introduces links and a termination rule that must be kept valid. The example is a finite chain with no cycle and initialized check functions; linking a node back to itself would not create useful validation.

Snapshot checks cannot protect the last free place against another concurrent request. The authoritative check and membership update must still occur together, as they do under the lock in the actual [MemoryStore.JoinGroup](store.go). Adding a linked chain does not add a transaction, rollback or a durable queue.

### Reconstruct the chain from a blank file

Hide the reference and write `AdmissionRequest`, two ordinary check functions, and one handler type containing a function and a next pointer. Rebuild `Handle` so it checks once, stops on failure and otherwise forwards. Construct the two-node chain and reproduce the three scenarios above. Before running it, predict which handler names print in each scenario. Then swap the order and explain which results can change without changing the final acceptance/rejection decision.

### Changed requirements to solve before reading the reasoning

1. The organizer manually closes a group. A later requirement also closes admission automatically when it fills and reopens it after a departure. Would a single closed state plus "reopen after every departure" preserve the organizer's intent? What extra distinction is needed?
2. A budget check and capacity check both fail. The interface now promises to show every failure to the user, rather than the first failure. Can the existing handler merely change its error message to satisfy this requirement?
3. A new handler reserves a place before forwarding to a deposit check. The deposit check fails. Does this chain release the reservation, and which design boundary must now be specified?

### Reason through the changed requirements

1. No. A manually closed group and a group blocked only by fullness represent different reasons for refusal. Keep manual closure as separate state/data, or introduce distinct closed states and transition rules. A departure can remove the capacity restriction while leaving manual closure intact. Reopening a still-full group also cannot create a spare place: the open-state capacity rule must remain. Defend which facts belong to lifecycle and which can be derived from membership.
2. No. The current `return err` prevents later handlers from inspecting the request, so their errors are unknown. Define a collection result and an aggregation contract, run all independent checks, and decide which checks depend on earlier success. Simply ignoring an early failure and then returning nil would incorrectly report success. The application's public error contract and handler control flow both change; first-failure order is a meaningful behavior choice.
3. No. The current forwarding method returns an error without undoing work. The study checks are deliberately read-only; adding reservation introduces mutation and failure recovery. Either keep preliminary checks read-only and validate/mutate in one authoritative transaction, or explicitly design rollback/compensation for the side effect. Process crashes and concurrent requests make an informal "decrement the count afterward" insufficient. The existing dispatcher and observer loop do not solve that problem either.

These are prepared study cases and reasoning prompts. No independent reconstruction or defense by the owner has been observed.

## Next practical rehearsal and real feedback

Allow a proposed 15 minutes: five reading and tracing, three explaining the demo, four answering one changed-input and one adjacent-concept question, three correcting the first gap. Repeat later with a fresh case. Record only observed ability as independent, assisted or unresolved.

Use the [shared feedback record](../../semester-info/defense-preparation/README.md#observe-and-incorporate-real-feedback) after watching classmates or presenting. Keep exact professor questions separate from inferred expectations. Correct the relevant explanation here and add a regression check only when feedback exposes a real behavior requirement. No defense result has been recorded for this checkpoint.
