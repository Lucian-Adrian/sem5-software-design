# Learn and defend Lab 3 (C#)

> **Read this before your defense (2 December 2026; ELSE due 1 December 23:59).**
> Grading (handout): 3 × 1.5 for patterns (Strategy, Command, Observer), 1 justification, 1 interaction, 1 SOLID/clean code, 1 tests, **1.5 for the defense and a live change request**. "Copied or AI-generated code that you cannot explain at the defense will be graded as not done."
>
> **How this builds on Lab 2:** All four Lab 2 structural patterns (`LegacyCsvListingAdapter`, `TimedGroupWriter`, `AuditedGroupWriter`, `OnboardingFacade`, `AdminOnlyGroupWriter`) and Lab 1 `GroupBuilder` stay active. Strategy picks which group to join, Command executes and undoes writes through `IGroupWriter` (Proxy → Decorators → `GroupService`), and Observer announces every successful write or undo.

Order: the one-sentence idea, then pseudocode, trace, Socratic questions, cost, and the remaining patterns. If you can draw the UML box diagram and write the pseudocode from memory, you understand the pattern.

## The one idea behind all behavioral patterns

> Behavioral patterns decide **how objects assign responsibilities and talk to each other at runtime**, instead of hard-coding algorithms, method calls or callbacks inside one class.

Every pattern in this lab separates *who decides* from *who does the work*:

| Pattern | What it separates | How they connect at runtime |
|---|---|---|
| Strategy | **which rule/algorithm runs** from the context that needs it | `GroupMatcher` holds `IGroupMatchingStrategy` and calls `SetStrategy(...)` |
| Command | **when/how a request runs (queue, undo)** from the receiver that executes it | `GroupCommandInvoker` runs/undoes `IGroupCommand` over `IGroupWriter` |
| Observer | **who produces an event** from who reacts to it | `GroupActivityNotifier` broadcasts `GroupActivityEvent` to `IGroupActivityObserver`s |

If you remember this table, you can justify why each pattern exists instead of `if/else` or direct calls.

## C# you need, compared with Python

| C# | Python equivalent | Where |
|---|---|---|
| `interface IGroupMatchingStrategy` | `Protocol` / ABC with `Matches(...)` | `Patterns/Strategy/` |
| `Stack<IGroupCommand>`, `Queue<IGroupCommand>` | `list` used as stack / `collections.deque` | `GroupCommandInvoker.cs` |
| `public sealed record GroupActivityEvent(...)` | `@dataclass(frozen=True)` | `IGroupActivityObserver.cs` |
| `_observers.ToList()` | `list(self._observers)` snapshot copy | `GroupActivityNotifier.cs` |
| `groups.Where(g => Strategy.Matches(g, p)).ToList()` | `[g for g in groups if strategy.matches(g, p)]` | `GroupMatcher.cs` |

## 1. Strategy

**Problem.** Users filter roommate groups with different rules: any affordable group with space, affordable with a monthly savings reserve, or small groups only. Putting `if (mode == ...)` inside `GroupMatcher` forces you to edit `GroupMatcher` for every new filter.

**Where in Domi.** `Patterns/Strategy/IGroupMatchingStrategy.cs` (`IGroupMatchingStrategy`, `AffordableWithSpaceStrategy`, `BudgetBufferStrategy`, `SmallGroupStrategy`) and `Patterns/Strategy/GroupMatcher.cs` (`GroupMatcher` context).

**Pseudocode**

```
interface MatchingStrategy:
    matches(group, profile) -> bool

class AffordableWithSpace implements MatchingStrategy:
    matches(g, p): return g.cost <= p.budget and len(g.members) < g.capacity

class BudgetBuffer(reserve) implements MatchingStrategy:
    matches(g, p): return g.cost <= (p.budget - reserve) and len(g.members) < g.capacity

class GroupMatcher:
    strategy: MatchingStrategy
    set_strategy(s): self.strategy = s
    find_matches(groups, profile):
        return [g for g in groups if self.strategy.matches(g, profile)]
```

**Trace for Vlad (`budget = 550`) over `Riscani flat (500, 0/3)` and `Botanica studio (501, 0/3)`:**
1. `matcher = new GroupMatcher(new AffordableWithSpaceStrategy())` → `FindMatches` checks `500 <= 550` (true) and `501 <= 550` (true) → returns 2 groups.
2. `matcher.SetStrategy(new BudgetBufferStrategy(50))` at runtime → `FindMatches` checks `500 <= 500` (true) and `501 <= 500` (false) → returns only `Riscani flat`.
3. `GroupMatcher` itself never runs an `if/else` on strategy type.

**Socratic questions** (answer aloud first, then read the answer)

1. *Why not just pass a `Func<Group, Profile, bool>` or write an `if/else` in `GroupMatcher`?*
   An `if/else` breaks Open/Closed: adding a new rule edits `GroupMatcher`. An interface (`IGroupMatchingStrategy`) gives each rule a name (`Name`), constructor validation (e.g., `reserveAmount >= 0`), unit testability, and a clear OOP role.
2. *What is the difference between Strategy and State? Their UML diagrams look almost identical.*
   **Intent and transitions.** In Strategy, the client chooses and swaps the algorithm from outside (`matcher.SetStrategy(...)`), and strategies do not know about each other. In State, the object's internal state classes know about neighbor states and trigger transitions themselves.
3. *Why keep `SetStrategy` on `GroupMatcher` instead of creating a new `GroupMatcher` each time?*
   So the caller (`GroupMatchingWorkflow`) keeps one `GroupMatcher` dependency while behavior changes at runtime.
4. *Cost / when NOT to use Strategy?*
   Adds one interface and one class per rule. Overkill if there is only one fixed rule that never changes, or if callers need to inspect internal fields of every variant.

## 2. Command

**Problem.** Writing to groups (`JoinGroup`, `CreateGroup`) needs queueing (`Enqueue`/`RunQueued`), undo (`UndoLast`), and execution history without moving domain validation or security checks out of `Group`, `GroupService`, or `AdminOnlyGroupWriter`.

**Where in Domi.** `Patterns/Command/`: `IGroupCommand`, `JoinGroupCommand`, `CreateGroupCommand`, `GroupCommandInvoker`. Receiver is `IGroupWriter` (which at runtime is `AdminOnlyGroupWriter` → `TimedGroupWriter` → `AuditedGroupWriter` → `GroupService`).

**Pseudocode**

```
interface GroupCommand:
    execute() -> Event
    undo() -> Event

class JoinGroupCommand(receiver, group_id, profile_id):
    executed = false
    execute():
        g = receiver.join_group(group_id, profile_id)   # domain & proxy check here
        executed = true
        return Event("Joined", g)
    undo():
        if not executed: error
        g = receiver.leave_group(group_id, profile_id)
        executed = false
        return Event("UndidJoin", g)

class GroupCommandInvoker(notifier):
    queue = Queue()
    history = Stack()
    enqueue(cmd): queue.push(cmd)
    run(cmd):
        ev = cmd.execute()      # if this throws, nothing is pushed or notified
        history.push(cmd)
        notifier.notify(ev)
        return ev
    undo_last():
        cmd = history.peek()
        ev = cmd.undo()
        history.pop()
        notifier.notify(ev)
        return ev
```

**Trace for failed join vs successful join + undo:**
1. `invoker.Run(new JoinGroupCommand(guarded, studioId, ion.Id))` where Ion's budget is 300 < 500:
   - `JoinGroupCommand.Execute()` calls `guarded.JoinGroup(...)` → Proxy → Timer → Audit → `GroupService` → `Group.Join` throws `DomainException`.
   - `AuditedGroupWriter` logs `[audit] JoinGroup error` and rethrows; `TimedGroupWriter` logs time in `finally`.
   - `GroupCommandInvoker.Run` aborts BEFORE `_history.Push(command)` and BEFORE `notifier.Notify(activity)`! History size and observer count stay unchanged.
2. `invoker.Run(new JoinGroupCommand(guarded, studioId, elena.Id))` succeeds → pushed onto `_history` → `notifier.Notify` fires `Joined`.
3. `invoker.UndoLast()` peeks `JoinGroupCommand`, calls `Undo()` → `guarded.LeaveGroup(studioId, elena.Id)` → `Group.Leave` removes Elena → pops `_history` → `notifier.Notify` fires `UndidJoin`.

**Socratic questions**

1. *Why put `Leave(Profile)` in `Group` instead of having `JoinGroupCommand` remove the ID from a list?*
   If the command manipulated lists directly, business logic would leak out of the receiver/domain (`Group`), and `Undo()` would bypass the Lab 2 Proxy and Audit/Timer Decorators!
2. *Why `Peek()` before `Pop()` in `UndoLast()`?*
   If `command.Undo()` throws (for example, `CreateGroupCommand.Undo()` when the group still has members, or when the user lost admin permission), keeping the command on `_history` avoids losing the undo entry silently.
3. *Why not just store a list of lambdas `Action execute, Action undo`?*
   Separate `IGroupCommand` classes carry `Description`, prevent double-execution/double-undo (`_executed`, `_createdGroupId`), hold typed parameters, and are easy to inspect, serialize, or test.
4. *Cost / when NOT to use Command?*
   Extra class per action and extra state to keep `Undo()` valid (for example, `CreateGroupCommand.Undo()` fails if members joined after creation unless composite undo is used). Don't use Command if you never need undo, queue, history, or deferred execution.

## 3. Observer

**Problem.** Multiple features care when a group changes or becomes full (activity log, full-group alert, event history), and listeners must be able to attach and detach at runtime without `GroupCommandInvoker` or `GroupService` knowing about them.

**Where in Domi.** `Patterns/Observer/`: `IGroupActivitySubject`, `GroupActivityNotifier`, `IGroupActivityObserver`, `GroupActivityEvent`, `ActivityLogObserver`, `FullGroupAlertObserver`, `ActivityHistoryObserver`.

**Pseudocode**

```
interface Observer:
    on_group_changed(event)

class GroupActivityNotifier:
    observers = []
    subscribe(obs):
        if obs not in observers: observers.append(obs)
    unsubscribe(obs):
        observers.remove(obs)
    notify(event):
        for obs in list(observers):    # snapshot copy
            obs.on_group_changed(event)
```

**Trace for `Center studio` (capacity 1) becoming full, then unsubscribing `logObserver`, then `UndoLast()`:**
1. Three observers subscribed: `logObserver`, `alertObserver`, `historyObserver`.
2. `JoinGroupCommand` adds Elena (`1/1`) → `Notify` loops over snapshot → `logObserver` prints `[observer:log] Joined...`, `alertObserver` prints `[observer:alert] ... is now FULL (1/1)`, `historyObserver` stores event.
3. `notifier.Unsubscribe(logObserver)` → `SubscriberCount` drops from 3 to 2.
4. `invoker.UndoLast()` removes Elena (`0/1`) → `Notify` delivers `UndidJoin` to `alertObserver` ("has a free spot again") and `historyObserver`; `logObserver` prints nothing.

**Socratic questions**

1. *Why iterate over `_observers.ToList()` in `Notify`?*
   If an observer calls `Unsubscribe` while inside `OnGroupChanged`, iterating directly over `_observers` would throw `InvalidOperationException` ("Collection was modified"). A snapshot copy makes runtime unsubscription during notification safe.
2. *How do you prevent endless update loops in Observer?*
   Observers receive an immutable `GroupActivityEvent` record and never write back to `GroupService` or call `GroupCommandInvoker.Run` inside `OnGroupChanged`.
3. *Why use `IGroupActivitySubject` + `IGroupActivityObserver` instead of C# `event Action<GroupActivityEvent>`?*
   In an OOP patterns course, explicit `Subject` and `Observer` interfaces clearly show the GoF structure (`Subscribe`, `Unsubscribe`, `Notify`, `OnGroupChanged`) and work across languages without language-specific delegate syntax.
4. *Cost / when NOT to use Observer?*
   Notification order depends on subscription order, and forgetting to `Unsubscribe` short-lived observers from a long-lived subject causes memory leaks (lapsed listener problem). Also debugging can feel indirect because `Notify` hides who reacts.

## The remaining behavioral patterns

Know the problem, the idea, and one Domi example for each:

1. **Chain of Responsibility.** *Problem:* A join request needs several checks (auth, rate limit, budget) and any check can reject or pass onward. *Idea:* Handler interface with `next` reference; handle or call `next.Handle(req)`. *Mistake to avoid:* hard-coding chain order inside handlers or letting requests fall off without a defined result.
2. **State.** *Problem:* A `Group` behaves differently when `Open`, `Full`, or `Closed`. *Idea:* Context holds `IGroupState` and delegates `Join`/`Close` to it; states trigger valid transitions. *Mistake to avoid:* `switch (statusEnum)` inside `Group`.
3. **Memento.** *Problem:* Save and restore a `Group`'s state without exposing private `_memberIds`. *Idea:* Originator (`Group`) returns an opaque `GroupMemento`; Caretaker stores it without reading inside and passes it back to `Restore(m)`. *Mistake to avoid:* exposing mutable lists inside the memento.
4. **Mediator.** *Problem:* Matcher, waitlist, notifications and chat calling each other directly form a tangled web. *Idea:* All colleagues talk only to `IDomiMediator`, which coordinates them. *Mistake to avoid:* turning the mediator into a giant god class holding all domain rules.
5. **Template Method.** *Problem:* CSV, JSON and XML importers share the same steps (`Read -> Validate -> Save`) with different parsing. *Idea:* Base class defines a non-virtual `Import()` skeleton calling abstract/virtual `ReadRows()` and `ParseRow()`. *Mistake to avoid:* making the template method itself overridable.
6. **Iterator.** *Problem:* Traverse groups in a custom order (cheapest first, nearly full first) without exposing `InMemoryStore`'s dictionary. *Idea:* Separate `IGroupIterator` (`HasNext`, `Next`) tracks traversal state. *Mistake to avoid:* just returning the raw internal list.
7. **Visitor.** *Problem:* Add new reports/exports (`RentSummaryVisitor`, `JsonExportVisitor`) across stable domain elements (`Profile`, `Group`) without modifying them. *Idea:* Double dispatch: `element.Accept(visitor)` calls `visitor.VisitGroup(this)`. *Mistake to avoid:* `if (x is Group)` type checks instead of `Accept`/`Visit`, or using Visitor when the element hierarchy changes constantly.
8. **Interpreter (good to know, does not count as one of the 3).** *Problem:* Evaluate user filter rules like `cost <= 500 AND capacity <= 3`. *Idea:* Represent grammar rules as expression classes implementing `Interpret(Group)`.

## Rebuild from a blank file (20-minute self-test)

1. Write `interface INumberRule { bool Keep(int n); }` and two strategies (`EvenRule`, `GreaterThanRule(int min)`), plus `NumberFilter` context with `SetRule(INumberRule)`.
2. Write `Counter` receiver (`Add(int)`, `Subtract(int)`), `ICounterCommand` (`Execute()`, `Undo()`), `AddCommand`, and `CommandRunner` with `Stack<ICounterCommand>` undo.
3. Write `IObserver` (`OnChanged(int value)`), `CounterSubject` (`Subscribe`, `Unsubscribe`, `Notify`), and two observers (console + history list).
4. Wire them: filter a list with Strategy, add each kept number with Command (notifying Observers), then `Unsubscribe` one observer and `UndoLast()`.

## Three-minute presentation script

1. "Domi matches people to shared-rent groups. Lab 1 built `GroupInput` with `GroupBuilder`; Lab 2 wrapped `GroupService` with `Adapter`, `Decorator`, `Facade`, and `Proxy`. In Lab 3 I added **Strategy**, **Command**, and **Observer** around the unchanged core."
2. Run `dotnet run --project src/Domi.Behavioral`:
   - Section 1: "Lab 2's Adapter + Builder and Facade populate Domi through the Proxy and Decorators."
   - Section 2: "`GroupMatcher` swaps strategies at runtime (`AffordableWithSpaceStrategy` → `BudgetBufferStrategy(50)` → `SmallGroupStrategy(2)`), with zero `if/else` in `GroupMatcher`."
   - Section 3–4: "`GroupMatchingWorkflow` connects Strategy → Command (`JoinGroupCommand` / queued `CreateGroupCommand`) → Lab 2 Proxy & Decorators → Observer (`ActivityLogObserver` and `FullGroupAlertObserver`)."
   - Section 5–6: "A failing command does not enter undo history or trigger observers. Then we unsubscribe `logObserver` at runtime and call `UndoLast()`: `LeaveGroup` flows through Proxy and Decorators, and only the remaining observers are notified."
3. Run `dotnet test`: "25 tests pass—4+ per Lab 3 pattern, end-to-end interaction, and all 12 Lab 2 tests."
