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

Istrati asks this. Do not define the pattern. Show **the code before, the code after, and the line in the demo output that proves it**. All code below is real (`src/Domi.Behavioral`), and all output is from `evidence/demo-output.txt`.

### Step 0: what I had to change in old code (say this first)

At the end of Lab 2, `IGroupWriter` had only `CreateGroup` and `JoinGroup`. That left nothing to undo a join or a create with. So I added two methods:

```csharp
public interface IGroupWriter
{
    Group CreateGroup(GroupInput input);
    Group JoinGroup(string groupId, string profileId);
    Group LeaveGroup(string groupId, string profileId);   // NEW: undo of a join
    Group DeleteGroup(string groupId);                    // NEW: undo of a create (only if empty)
}
```

As a result, four places changed:
- `GroupService` implements the two new methods.
- The domain got `Group.Leave(profile)` and `InMemoryStore.RemoveGroup(id)`.
- The Lab 2 decorators forward and log them (`[audit] LeaveGroup ok`, `[timer] LeaveGroup took ...`).
- The proxy lets anyone leave, but only admins delete.

`Group.Join` and its rules did not change. Everything else is new classes.

### Strategy: Domi can now suggest groups, and the rule can be swapped while running

- **Before:** Domi had no matching at all. To join, the caller had to already know the group id. Writing matching as `if (mode == "budget") ... else if (mode == "small") ...` inside one class would mean editing that class for every new rule.
- **After:** each rule is a class, and `GroupMatcher` only asks the current rule:

```csharp
public interface IGroupMatchingStrategy { string Name { get; } bool Matches(Group g, Profile p); }

public sealed class BudgetBufferStrategy(int reserve) : IGroupMatchingStrategy   // keeps "reserve" money left
{
    public bool Matches(Group g, Profile p) =>
        g.MaxMonthlyCost <= p.MonthlyBudget - reserve && g.MemberIds.Count < g.Capacity;
}

public sealed class GroupMatcher(IGroupMatchingStrategy initial)
{
    public void SetStrategy(IGroupMatchingStrategy s) => Strategy = s;           // swap at runtime
    public IReadOnlyList<Group> FindMatches(IEnumerable<Group> groups, Profile p) =>
        groups.Where(g => Strategy.Matches(g, p)).ToList();                       // no if/else
}
```

- **Proof in the demo (section 2):** the same person and the same groups give a different answer after `SetStrategy`:
  ```text
  [Affordable with space] for Vlad (budget=550): 2 match(es)     <- Riscani 500 and Botanica 501
  [Budget buffer (>= 50 left)] for Vlad (budget=550): 1 match(es) <- only Riscani: 550 - 50 = 500
  [Small group (<= 2 people)] for Ana (budget=700): 1 match(es)   <- only Ana's flat (capacity 2)
  ```

### Command: a join is now an object that can be queued and undone

- **Before:** a join was a method call, `guarded.JoinGroup(groupId, profileId)`. Once it ran, it left no trace, so it could not be queued for later, kept in a history, or taken back.
- **After:** the call is wrapped in an object that also knows how to reverse itself:

```csharp
public sealed class JoinGroupCommand(IGroupWriter receiver, string groupId, string profileId) : IGroupCommand
{
    public GroupActivityEvent Execute() => ... receiver.JoinGroup(groupId, profileId) ...   // "Joined"
    public GroupActivityEvent Undo()    => ... receiver.LeaveGroup(groupId, profileId) ...  // "UndidJoin"
}

public sealed class GroupCommandInvoker(IGroupActivitySubject notifier)
{
    Queue<IGroupCommand> _queue;  Stack<IGroupCommand> _history;
    public GroupActivityEvent Run(IGroupCommand c)
    {
        var e = c.Execute();      // if this throws, the next two lines never run
        _history.Push(c);         // only successful commands can be undone
        notifier.Notify(e);       // only successful commands are announced
        return e;
    }
    public GroupActivityEvent UndoLast() { var c = _history.Peek(); var e = c.Undo(); _history.Pop(); notifier.Notify(e); return e; }
}
```

The receiver is the `guarded` writer from Lab 2, so every command and every undo still goes through the proxy and the decorators.

- **Proof in the demo (sections 4–6):**
  ```text
  [observer:log] Created Center studio [group-8] members=0/1        <- queued CreateGroupCommand, run later
  [observer:log] Joined Center studio [group-8] members=1/1 ...     <- Elena's JoinGroupCommand
  [audit] JoinGroup error: Center studio is full                    <- Ion's command fails...
    undo stack size stays 3, observer events=3                       <- ...no undo entry, no event
  [audit] LeaveGroup ok -> group-8                                  <- UndoLast: Elena's join reversed
    undone action=UndidJoin on Center studio, members=0/1
  ```

### Observer: other parts of the app now react to a join without the invoker knowing them

- **Before:** after a join, nothing else in Domi found out. To add a "group is full" alert, I would have had to put `Console.WriteLine` and `if (full)` checks inside the code that does the join.
- **After:** the invoker only calls `notifier.Notify(event)`. Whoever subscribed reacts, and listeners can be added or removed while the program runs:

```csharp
var notifier = new GroupActivityNotifier();
notifier.Subscribe(new ActivityLogObserver(log));       // prints [observer:log] ...
notifier.Subscribe(new FullGroupAlertObserver(log));    // prints "is now FULL" / "has a free spot again"
notifier.Subscribe(historyObserver);                     // keeps a list of events
...
notifier.Unsubscribe(logObserver);                       // at runtime
```

- **Proof in the demo:**
  ```text
  [observer:log] Joined Center studio [group-8] members=1/1 profile=profile-7
  [observer:alert] Center studio [group-8] is now FULL (1/1)           <- alert reacted on its own
  subscribers after unsubscribe: 2
  [observer:alert] Center studio [group-8] has a free spot again (0/1) <- after undo: no [observer:log] line any more
  observer history count: 4                                             <- Joined, Created, Joined, UndidJoin
  ```

### How the three connect: one real call path

`GroupMatchingWorkflow.JoinBestMatch(catalog, vlad)` does three things:
- asks the **Strategy** for matches,
- wraps the first match in a **Command** and runs it through the invoker: proxy → decorators → `GroupService` → `Group.Join`,
- lets the invoker **notify the observers**.

Demo section 3 shows that whole path in four lines:
```text
[audit] JoinGroup ok -> group-1
[timer] JoinGroup took 0.016 ms
[observer:log] Joined Riscani flat [group-1] members=1/3 profile=profile-5
  workflow matched and executed: Joined -> Riscani flat
```

### 30-second spoken version

> "At the end of Lab 2, Domi could create and join, but the caller had to pick the group itself, nothing could be undone, and nothing else heard about a join. In old code I only extended `IGroupWriter` with `LeaveGroup` and `DeleteGroup`, plus `Group.Leave`, so commands have something to undo with. **Strategy:** `GroupMatcher` asks a swappable rule object. For Vlad with 550, 'affordable' gives 2 groups, and 'keep 50 left' gives only Riscani. **Command:** a join is a `JoinGroupCommand` with `Execute` and `Undo`. The invoker keeps a history, so a failed join leaves no undo entry, and `UndoLast` calls `LeaveGroup`. **Observer:** the invoker only calls `Notify`. The log, the 'group is full' alert and the history react on their own, and I unsubscribed the log while the program was running. The join rules are still in `Group.Join`."

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
