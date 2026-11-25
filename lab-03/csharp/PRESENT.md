# Lab 3 defense: behavioral patterns in Domi (C#)

Read this in the 10 minutes before your defense. The deeper study guide with pseudocode, traces and Socratic Q&A is [learn.md](learn.md).

## 1. What Istrati is testing

Grading: 1.5 per pattern (Strategy, Command, Observer), 1 justification, 1 interaction, 1 SOLID/clean code, 1 tests, **1.5 defense + live change request**. Speak in this order for every pattern: **problem → where in Domi → mechanism → why not simpler → cost**. Stop after the cost.

## 2. Opening (30 seconds)

> "Domi matches roommates to shared flats. Lab 1 gave `GroupBuilder`, Lab 2 gave `Adapter`, `Decorator`, `Facade` and `Proxy` around `GroupService`. Lab 3 adds three behavioral patterns—**Strategy**, **Command** and **Observer**—that connect along one path without putting rules or callbacks inside `GroupService`:"

```text
GroupMatchingWorkflow
  -> GroupMatcher (Strategy: runtime-swappable filter)
  -> GroupCommandInvoker + JoinGroupCommand (Command: queue + LIFO undo)
  -> AdminOnlyGroupWriter (Proxy) -> TimedGroupWriter -> AuditedGroupWriter (Decorators) -> GroupService
  -> GroupActivityNotifier (Observer: runtime subscribe/unsubscribe) -> Observers
```

## 2b. "What changed in your app when you implemented these?"

Istrati asks this. Answer **before → after** for Domi.

**Before Lab 3** (end of Lab 2): Domi could import listings, onboard in one call, log, time and guard writes. But it could only *create* and *join*. Choosing a group was up to the caller, a join could not be taken back, and nothing else in the app learned that a join happened.

**Changes to existing code (small, say them first):**
- `IGroupWriter` gained `LeaveGroup` and `DeleteGroup` (`Domain/Services.cs:39-40`), so a command has something to undo with. `GroupService` implements them.
- The Lab 2 decorators forward and log the two new methods, and the proxy makes `DeleteGroup` admin-only.
- Everything else is new classes. `Group.Join` and its rules did not change.

| Pattern | Before | After: what Domi can do now |
|---|---|---|
| **Strategy** | No matching: the caller had to pick a group itself. | Domi **suggests groups** for a person, and the rule can be **swapped at runtime**: affordable with space, affordable with 50 left over in the budget, or small groups only (`matcher.SetStrategy(...)`). |
| **Command** | A join was an immediate method call, gone once it ran. | A join or create is an **object**: it can be **queued** and run later, and the last successful action can be **undone** (join → leave, create → delete). A failed command leaves no undo entry. |
| **Observer** | Nobody else heard about a join. | Every successful action is **broadcast**: a log, a "group is full" alert and a history record react, and they can **subscribe or unsubscribe at runtime**. The invoker doesn't know who listens. |

**The new single path:** workflow → matcher (Strategy) → command (Command) → proxy → decorators → `GroupService` → notifier (Observer). The demo shows it: Vlad matched and joined, the studio filled, Ion's join refused with no undo entry and no event, then Elena's join undone.

**30-second spoken version:**
> "Before, Domi could only create and join, the caller chose the group itself, nothing could be undone, and nothing else heard about a join. I changed existing code in one place: I added `LeaveGroup` and `DeleteGroup` to `IGroupWriter`, so commands can be undone. Then I added new classes. Strategy lets Domi suggest groups with a rule I can swap at runtime. Command turns each action into an object I can queue and undo. Observer broadcasts every successful action to listeners that subscribe at runtime. The join rules are still in `Group.Join`."

## 3. The three cards

### Strategy: `Patterns/Strategy/IGroupMatchingStrategy.cs`, `GroupMatcher.cs`

- **Problem.** Different users filter groups differently (basic budget, budget reserve, small household). An `if/else` inside `GroupMatcher` forces edits whenever a rule is added.
- **Where.** `GroupMatcher` (Context), `IGroupMatchingStrategy` (Strategy), `AffordableWithSpaceStrategy`, `BudgetBufferStrategy`, `SmallGroupStrategy` (ConcreteStrategies).
- **Mechanism.** `GroupMatcher` holds `IGroupMatchingStrategy Strategy` and delegates `FindMatches` via `Strategy.Matches(group, profile)`. `SetStrategy(...)` swaps rules at runtime. Zero `if/else` in `GroupMatcher`.
- **Why not simpler.** Conditionals inside `GroupMatcher` violate Open/Closed; separate classes also validate their own parameters (e.g., `reserveAmount >= 0`).
- **Cost.** One extra class per rule; overkill if there is only one fixed filter.

```text
interface IGroupMatchingStrategy { bool Matches(Group g, Profile p); }
class GroupMatcher { IGroupMatchingStrategy Strategy; SetStrategy(s); FindMatches(...) => ...; }
```

### Command: `Patterns/Command/IGroupCommand.cs`, `JoinGroupCommand.cs`, `CreateGroupCommand.cs`, `GroupCommandInvoker.cs`

- **Problem.** Group writes need queueing, undo history, and uniform dispatch without leaking domain/security logic out of `GroupService` and `AdminOnlyGroupWriter`.
- **Where.** `IGroupCommand` (Command), `JoinGroupCommand` & `CreateGroupCommand` (ConcreteCommands), `GroupCommandInvoker` (Invoker), `IGroupWriter` (Receiver).
- **Mechanism.** Commands hold only parameters + `IGroupWriter`. `GroupCommandInvoker` supports `Enqueue`/`RunQueued` (FIFO), `Run`, and `UndoLast` (LIFO `Stack<IGroupCommand>`). Failed commands throw before `_history.Push` and before `notifier.Notify`. `UndoLast` peeks before popping and routes `LeaveGroup`/`DeleteGroup` through `IGroupWriter`.
- **Why not simpler.** A direct method call cannot be queued, stacked, or undone uniformly. Keeping `Leave`/`DeleteGroup` on `IGroupWriter` ensures Proxy and Decorators still run on undo.
- **Cost.** Extra command classes and undo preconditions (e.g., `CreateGroupCommand.Undo` refuses to delete a non-empty group).

```text
interface IGroupCommand { GroupActivityEvent Execute(); GroupActivityEvent Undo(); }
class GroupCommandInvoker { Queue q; Stack history; Run(c) { e=c.Execute(); history.Push(c); notify(e); } }
```

### Observer: `Patterns/Observer/IGroupActivityObserver.cs`, `GroupActivityNotifier.cs`, `ActivityObservers.cs`

- **Problem.** Logs, full-group alerts, and history tracking should react to group changes and undos without coupling `GroupCommandInvoker` to concrete output classes.
- **Where.** `IGroupActivitySubject` (Subject), `GroupActivityNotifier` (ConcreteSubject), `IGroupActivityObserver` (Observer), `ActivityLogObserver`, `FullGroupAlertObserver`, `ActivityHistoryObserver`.
- **Mechanism.** `GroupActivityNotifier` manages `_observers` with deduplicated `Subscribe` and runtime `Unsubscribe`, and iterates `_observers.ToList()` (snapshot) when broadcasting immutable `GroupActivityEvent` records.
- **Why not simpler.** Calling `Console.WriteLine` or alerts inside the invoker couples core orchestration to every output channel.
- **Cost.** Notification order follows subscription order; un-unsubscribed observers stay referenced (memory leak if long-lived subject holds short-lived listeners).

```text
interface IGroupActivityObserver { void OnGroupChanged(GroupActivityEvent e); }
class GroupActivityNotifier { List<IGroupActivityObserver> _obs; Notify(e) { foreach(o in _obs.ToList()) o.OnGroupChanged(e); } }
```

## 4. Live change requests: what to type

**"Add a new Strategy: `MaxCostStrategy(int cap)` (matches groups costing at most `cap`, regardless of profile)."**
Write one new class in `Patterns/Strategy/IGroupMatchingStrategy.cs` (or a new file), wire it in `Program.cs` with `matcher.SetStrategy(new MaxCostStrategy(500))`. Touch nothing in `GroupMatcher`:

```csharp
public sealed class MaxCostStrategy(int maxCost) : IGroupMatchingStrategy
{
    public string Name => $"Max cost <= {maxCost}";
    public bool Matches(Group group, Profile profile) =>
        group.MaxMonthlyCost <= maxCost && group.MemberIds.Count < group.Capacity;
}
```

**"Combine two strategies with AND (Composite + Strategy)."**
Add one `AllOfStrategy(params IGroupMatchingStrategy[] rules) : IGroupMatchingStrategy`:

```csharp
public sealed class AllOfStrategy(params IGroupMatchingStrategy[] rules) : IGroupMatchingStrategy
{
    public string Name => string.Join(" AND ", rules.Select(r => r.Name));
    public bool Matches(Group group, Profile profile) =>
        rules.All(rule => rule.Matches(group, profile));
}
```

**"Add a new Command (e.g., `LeaveGroupCommand` that rejoins on Undo)."**
Add one class implementing `IGroupCommand` in `Patterns/Command/` (`Execute` calls `receiver.LeaveGroup`, `Undo` calls `receiver.JoinGroup`). `GroupCommandInvoker` does not change.

**"Add a new Observer (e.g., counts total joins across all groups)."**
Add one class implementing `IGroupActivityObserver` and call `notifier.Subscribe(counter)`:

```csharp
public sealed class JoinCountObserver : IGroupActivityObserver
{
    public int NetJoins { get; private set; }
    public void OnGroupChanged(GroupActivityEvent activity)
    {
        if (activity.Action == "Joined") NetJoins++;
        else if (activity.Action == "UndidJoin") NetJoins--;
    }
}
```

**"Undo the last N commands at once."**
Add `UndoLast(int count)` on `GroupCommandInvoker` calling `UndoLast()` in a loop while `HistoryCount > 0`.

After any live edit, run `dotnet test` and add one xUnit test in `tests/Domi.Behavioral.Tests/BehavioralPatternTests.cs`.

## 5. Hard questions, short answers

1. **Strategy vs State?** Strategy is chosen/swapped by the client; State transitions internally between states.
2. **Where does business logic live in Command?** In the receiver (`Group` and `GroupService`), NOT in the command.
3. **Why does `GroupCommandInvoker` call `Peek()` before `Pop()`?** If `Undo()` throws, the command stays on the stack instead of disappearing silently.
4. **Why `_observers.ToList()` in `Notify`?** So an observer unsubscribing during notification doesn't crash the `foreach`.
5. **Where is SOLID?**
   - S: Strategy filters, Command executes/undoes, Observer notifies, `Group` owns rules.
   - O: New strategy/command/observer = one new class; context/invoker/subject unchanged.
   - L: Any `IGroupMatchingStrategy`, `IGroupCommand`, `IGroupActivityObserver`, or `IGroupWriter` substitutes cleanly.
   - I: Small, single-purpose interfaces (`IGroupMatchingStrategy`, `IGroupCommand`, `IGroupActivityObserver`, `IGroupActivitySubject`, `IGroupWriter`).
   - D: Workflow, matcher, commands, invoker and notifier depend only on abstractions; only `Program.cs` constructs concrete classes.
6. **Remaining 8 patterns in one line each?** Chain of Responsibility (pass along handlers), State (behavior per internal state), Memento (opaque state snapshot), Mediator (central coordinator), Template Method (fixed skeleton + overridable steps), Iterator (custom traversal), Visitor (double dispatch `Accept`/`Visit`), Interpreter (grammar evaluation tree, doesn't count as 1 of the 3).
