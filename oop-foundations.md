# Rebuild OOP for TMPS

You want to write a correct object, explain its rules, and choose a pattern when a requirement makes it useful. Start with the Domi roommate group you already use in the labs. Alex has a monthly budget of 550; a shared flat costs 500 per person and has one place. Pat has budget 400. Creating, inspecting and changing this group will give each OOP term a concrete meaning.

This guide develops prerequisites before the existing lab guides. Its [standalone practice source](oop-practice.go) deliberately leaves out HTTP, storage and concurrency so you can reconstruct the object and its selection rules. The actual lab checkpoints remain the reference for their assessed implementation. The course's retained requirement record asks for SOLID followed by three implemented patterns from each family and knowledge of the remaining lists; see [the dated requirements](../semester-info/defense-preparation/README.md#sources-versions-and-unresolved-rules) and the course [README](README.md#assessment-preparation). Go acceptance and current assessment allocation still need confirmation. This guide does not establish a defense or a grade.

## Ten-minute orientation

This is a map for starting deeper study, with reading and correction included. It cannot establish mastery.

| Time | Action |
| --- | --- |
| 0–2 minutes | Read the next section. Point to the group's identity, state and behavior. |
| 2–5 | Read construction and encapsulation. Explain why a blank rename cannot change the stored name. |
| 5–7 | Follow the worked trace below. Explain why a full group remains affordable. |
| 7–8 | Without looking, describe the difference between `alias := group` and `copy := *group`. |
| 8–10 | Check that explanation in the ownership section, repair the first gap, then explain a fresh case: would changing the copied name change the original? |

Continue with the staged route after this orientation. If pointers or functions remain unclear, spend a full session on those before adding patterns.

## An object has state, identity and behavior

State is the data that can vary at a particular moment. Here it includes the group's name, cost, capacity and member IDs. Behavior is the set of operations that inspect or change that state: report a view, rename the group, or attempt a join. Putting the join rules beside the state makes it easier to keep every caller consistent. Otherwise, an HTTP handler might check capacity while a command-line caller forgets it.

Identity answers which domain entity this is across changes. `group-1` remains the same group when its name changes or Alex joins. Two different groups can have equal names, costs and capacities. Those equal settings do not make them the same entity. Conversely, two snapshots with the same ID but different member counts can describe the same entity at different times. Domain identity is a design decision; it is different from a memory address.

A type describes permitted values and operations. A `Group` value is one instance. In Java or C++, you may have learned a class declaration that combines fields and methods. Go declares a `struct` and receiver methods separately:

```go
type Group struct {
    id       string
    name     string
    cost     int
    capacity int
    members  []string
}

func (group Group) Name() string { return group.name }
```

`group Group` is the receiver, the particular value receiving this method call. It fills the role often called `this` or `self`. `func Name(group Group) string` could compute the same answer as a free function; receiver syntax associates behavior with a type and lets it satisfy interfaces.

A `GroupView` has a different purpose. It is a public snapshot with cost, capacity and member count. It does not own membership or provide permission to change it. This distinction keeps private details and mutation rules out of a response type. See the actual [Lab 0 domain types](lab-00/domain.go), `Group.view`, `Profile.view`, and [structs describing data stages](lab-00/learn.md#structs-describe-distinct-stages-of-data). `Profile.view` deliberately omits the person's private budget.

## Construction establishes a valid starting state

An invariant is a condition the object promises to preserve across its supported operations. For this practice group, the name and ID must be nonblank, cost positive, capacity from 1 to 20, and membership no greater than capacity. A profile needs a nonblank ID and a nonnegative budget. A valid zero-budget profile cannot join a positive-cost group. Distinguish a valid profile from an eligible join.

A constructor initializes an instance so callers can begin using it under those assumptions. Go has no special constructor syntax. `NewGroup` is an ordinary function named by convention; its return type and body define what it does:

```go
func NewGroup(id, name string, cost, capacity int) (*Group, error)
```

It trims names, checks inputs, and returns a pointer plus `nil` on success. Failure returns `nil` plus an error. Read the entire call as two results:

```go
group, err := NewGroup("group-1", " Shared flat ", 500, 1)
if err != nil {
    return err
}
```

`:=` declares local variables. `err != nil` asks whether construction failed; returning prevents the invalid result from reaching later calls. `&Group{...}` takes the address of the initialized value. The compiler/runtime arrange its valid lifetime even though it was made inside a function. Go does not require the caller to free that address.

`new(Group)` has a different meaning: it allocates a zero-initialized Group and returns its pointer. It does not call `NewGroup`. `Group{}` also makes a zero value, with empty strings, zero integers and a nil member slice. Those are legal Go values but they do not satisfy this domain's creation rules. Constructors are a package convention, not a language guarantee that no invalid value can exist. Because all practice code is in one package, it could bypass the constructor. In a larger design, an unexported representation plus an exported constructor can reduce that opportunity outside the owning package; zero/nil behavior still needs a contract.

In [Lab 0 `newGroup` and `newProfile`](lab-00/domain.go), construction returns values rather than pointers. The store assigns IDs; `newGroup` validates name, cost and capacity, while `newProfile` validates name and budget. The practice accepts IDs directly and adds nonblank-ID checks because it has no store. These are deliberate simplifications, not new lab behavior. Compare [constructor validation](lab-00/learn.md#constructors-return-validation-errors).

A `New...` name does not by itself demonstrate the GoF Factory Method pattern. A plain function that validates and returns one concrete value is already useful. Factory Method addresses a further question: how a workflow lets a chosen creator vary which product it constructs. Keep the ordinary construction mechanism clear before adding that variation.

## Encapsulation, getters and setters

Encapsulation keeps a representation behind operations with meaningful contracts. It helps because callers can request a valid transition without learning every internal rule. Merely making a field inaccessible does not ensure the methods preserve the rules.

A getter reads a fact. `Name()` returns the name; the conventional Go name omits `Get`. A view can expose several safe facts at once. Neither needs to expose the internal membership slice. Returning that slice directly would copy a slice descriptor while leaving access to its backing array, allowing callers to overwrite member IDs. A copy of simple counts avoids that path. Go naming and access conventions are described in [Effective Go: getters](https://go.dev/doc/effective_go#Getters).

A setter changes a property. An unrestricted `SetMembers` or `SetMemberCount` would let callers create a member count inconsistent with the list or exceed capacity. Prefer `Join(profile)` because the intended action carries rules. `Rename(name)` is a useful setter-shaped operation: trim first, reject an empty name, then assign. If rejected, the old name must remain unchanged. Validation after assignment would violate that failure contract.

The order in `Join` is intentional:

1. Check whether this profile ID already belongs to the group. If it does, succeed without changing membership.
2. Reject an insufficient budget.
3. Reject a full group.
4. Append the new ID exactly once.

This gives repeated successful joins idempotent membership behavior: repeating the request has the same membership result as doing it once. If the full check came before the duplicate check, Alex's repeat would fail after filling the final place. See [the actual join rule](lab-00/learn.md#the-actual-join-rule) and `Group.addMember` in [domain.go](lab-00/domain.go).

Lowercase Go fields are unexported outside the package, not private to a struct. Every file in the lab's `package main` can access them. Encapsulation is therefore partly the package design and partly the supported operation contract. It is also separate from authentication: an unexported field does not decide which web user may join.

## Mutation, copies and lifetime

A value receiver gets a copy of its receiver. Reading a small value such as a view or name fits that model. A pointer receiver receives an address through which it can change the original value:

```go
func (group *Group) Rename(name string) error
```

The receiver still passes by value: Go copies the pointer, and that copied pointer refers to the same Group. `group.name = name` follows that pointer to change the field. Go can automatically take the address for many method calls on addressable variables, but this convenience does not erase the distinction between value and pointer method sets. A `*T` has both `T` and `*T` receiver methods; `T` does not gain the pointer-only methods for interface satisfaction. See [the Go specification](https://go.dev/ref/spec#Method_sets).

In the practice, `group` already holds a pointer. `alias := group` copies that address. Renaming through `alias` changes what `group.Name()` reads. It creates a second reference to one object, not a second group.

`copy := *group` copies the struct fields. Assigning `copy.name = "Other"` changes the copy's string field and leaves the original name alone. But `members` is a slice, so both structs may refer to the same backing array. Replacing an existing element through one can affect the other; appending may or may not allocate another array. Do not describe this as an independent deep copy. The actual [Lab 1 prototype](lab-01/patterns.go), `NewGroupPrototype`, explicitly copies the member slice, while `CloneAsDraft` clears identity and members for the new record. Study [slices and copying](lab-01/learn.md#slices-and-copying).

Ownership means who is responsible for an object's use and cleanup. Go pointers do not enforce unique ownership. Here `run` creates the group and uses it for this demonstration; aliases must follow the same rules. Its views are snapshots. Once the program cannot reach the group, its memory may be reclaimed by garbage collection. Reclamation timing is not a business event, and a group going out of a local scope does not automatically remove a domain record from a map.

A destructor, sometimes remembered as a “destroyer”, is a special lifetime operation in some languages. C++ destructors can run deterministically when a local object leaves scope or an owning object is destroyed; that supports releasing resources at a known point. Go has no C++-style deterministic destructor. Its garbage collector manages memory, while files, sockets and other resources need an explicit lifecycle. Do not rely on eventual GC or a finalizer to promptly close them.

`Close() error` is a method convention captured by [Go's `io.Closer`](https://pkg.go.dev/io#Closer). It can release an external resource and report failure; it does not mean “delete this struct's memory”. `defer file.Close()` arranges a call when the current function returns, rather than when a loop iteration or nested block ends. Register it only after a successful open. For an output file whose completion matters, handle both write and close errors; a deferred call that ignores the return value cannot establish successful completion. Repeated-close behavior depends on the concrete resource's contract.

The practice owns no file or socket, so it needs no invented `Destroy` or `Close`. In the actual [Lab 2 audit decorator](lab-02/structural.go), `NewAuditedGroupWriter` receives an `io.Writer`. That is a borrowed output capability. The decorator does not open it and cannot assume it can close it, since `io.Writer` only promises `Write`. Whoever opens a file and passes it in remains responsible for its lifetime unless an explicit contract transfers ownership. This is a useful system-design question before choosing an abstraction.

## Interfaces, polymorphism and composition

A full group still costs 500. Alex can afford it even when there is no place. We therefore need two precise questions: “does the cost fit?” and “does the cost fit and is a place free?” This useful variation motivates a common selection contract:

```go
type GroupFilter interface {
    Matches(GroupView) bool
}
```

`AffordableGroups{Budget: 550}` and `AvailableGroups{Budget: 550}` have the required method. `CountMatches` accepts either through `GroupFilter` and calls `Matches` without knowing the concrete rule. That is polymorphism: a caller uses a common contract while the concrete value supplies behavior. Go uses implicit method-set satisfaction, not an `implements` declaration or class inheritance.

The signature checks shape, while the semantic contract defines meaning: given a valid snapshot, answer whether it belongs in this rule's selection without changing domain membership. A method with the right signature that panics for ordinary valid input would violate that expected behavior. Types and tests help establish different parts of the contract; a compiling interface alone does not prove substitutability.

`CountMatches` expects a usable non-nil filter. Unlike actual `GroupService.ListGroups`, it does not supply an all-groups default for nil. Keep that precondition explicit. A typed nil pointer inside an interface also differs from a nil interface; passing one is not an automatic “no filter” signal.

Composition assembles collaborators instead of inheriting their implementation. In the actual lab, `GroupService` contains reader and writer interfaces. A decorator contains another writer. The service calls collaborators rather than becoming a subclass of a store. Go embedding can promote fields and methods, but it does not create Java-style class inheritance or virtual overriding.

```text
practice run
  -> NewGroup(...) -> one mutable Group
  -> Group.View() -> a snapshot
  -> CountMatches(snapshot, filter)
       -> concrete filter.Matches(snapshot)
  -> Group.Join(profile) -> validation -> possible membership change

actual Lab 0 join
  -> GroupService.JoinGroup -> GroupWriter interface
  -> MemoryStore.JoinGroup -> lock/read/check/change/save -> unlock
```

The diagram separates observation from mutation. A filter's `true` result reserves nothing. In the real store, a concurrent request can change membership after listing, so [MemoryStore.JoinGroup](lab-00/store.go) protects check-and-update with its lock. It copies a group out of its map, changes that local value, then writes it back. A pointer receiver on the local copy does not persist a map entry by itself. Follow [value and pointer receivers](lab-00/learn.md#value-and-pointer-receivers) and [mutex reasoning](lab-00/learn.md#why-memorystore-uses-a-mutex).

## Run and explain one complete trace

On Dev1, run only the practice file, using the native Go toolchain:

```sh
cd /d/edu/faf/sem5/software-design
go run oop-practice.go
```

The file is self-contained and uses the standard library; no new server, module or installed dependency is needed. Read it in this order: `Group`, `NewGroup`, `Name`/`Rename`, `View`, `Join`, the filters, `CountMatches`, then `run`. `main` only prints a failure to stderr and exits with status 1 if initialization or a required operation fails. `fmt.Printf` uses `%d` for integers, `%t` for booleans and `%v` for an error's displayed value. A nil error prints `<nil>`; the demonstrated rejection errors have the messages below.

```text
created: Shared flat, members=0
before: affordable=1 available=1
Pat: budget too low, members=0
Alex: ok, members=1
Alex again: ok, members=1
after: affordable=1 available=0
snapshot before: members=0
rename blank: name is required, name=Shared flat
through alias: name=Shared home same pointer=true
```

Construction trims the name. Both filters initially accept the only snapshot because `500 <= 550` and `0 < 1`. Pat fails the budget check without mutation. Alex succeeds, then repeats without duplication. Affordability remains true, while availability becomes false because `1 < 1` is false. The earlier `before` view remains at zero members: it holds copied scalar facts, not a live connection. A blank rename fails before assignment. Finally, the alias and original variable point to the same group, so renaming through the alias changes the original's observed name.

This run checks a worked example, not every possible input. The practice assumes profiles and groups were created successfully, has no registered profile store, and is single-threaded. It does not demonstrate concurrent joins, persistence, HTTP authorization, notification delivery or learner understanding.

## SOLID describes the reasons for these boundaries

Use the principle names after you can explain the concrete change they protect. None requires splitting every method into another interface.

| Principle | Reason in this workflow | Actual source and deeper defense |
| --- | --- | --- |
| Single Responsibility | Rent and membership rules change for domain reasons; JSON/status mapping changes for transport reasons. Separating them keeps one kind of change from spreading. It does not mean one method per type. | [domain.go](lab-00/domain.go), [http.go](lab-00/http.go); [SRP](lab-00/learn.md#single-responsibility-principle) |
| Open/Closed | Add an availability filter while keeping the existing listing loop. This boundary is closed to routine policy edits, while the implementation still evolves when its requirements change. | `GroupFilter`, `GroupService.ListGroups` in [service.go](lab-00/service.go), `AvailableGroups` in [behavioral.go](lab-03/behavioral.go); [OCP](lab-00/learn.md#openclosed-principle) |
| Liskov Substitution | A caller's valid operations must remain valid when another implementation is supplied. Matching method signatures is insufficient. A replacement filter must preserve the selection contract. | `AllGroups`, `AffordableGroups` in [service.go](lab-00/service.go); [LSP](lab-00/learn.md#liskov-substitution-principle) |
| Interface Segregation | A client that only reads groups should not depend on unrelated write methods. Small capability contracts describe what it needs. | `GroupReader`, `GroupWriter`, `ProfileWriter` in [service.go](lab-00/service.go); [ISP](lab-00/learn.md#interface-segregation-principle) |
| Dependency Inversion | Application policy depends on reader/writer abstractions. Startup selects the concrete store and injects it. Passing a dependency is injection; depending on the abstraction is the design principle. | `NewGroupService` in [service.go](lab-00/service.go), wiring in [main.go](lab-00/main.go); [DIP](lab-00/learn.md#dependency-inversion-principle) |

## Patterns solve a further problem

The practice needs a constructor, a few domain methods and replaceable selection rules. It does not need to implement every pattern at once. Use the existing course implementations for the next steps. For each one, name the changing requirement, identify participants, trace a call, then defend the simpler alternative.

| Requirement and why the pattern fits | Exact implemented participants | Existing explanation |
| --- | --- | --- |
| Build a draft through several configuration steps, validating the completed input: Builder. No identity or stored record is created just by configuring the builder. | `GroupBuilder`, `WithName`, `WithCapacity`, `Build` in [patterns.go](lab-01/patterns.go) | [Builder versus a literal](lab-01/learn.md#builder-compared-with-a-struct-literal), [reconstruct it](lab-01/learn.md#write-the-builder-yourself) |
| Choose a matching group draft and profile suggestion as a family: Abstract Factory. One constructor returning one product does not express that family. | `OnboardingPresetFactory`, `BudgetPresetFactory`, `ComfortPresetFactory`, `buildOnboardingPreset` in [patterns.go](lab-01/patterns.go) | [family reasoning](lab-01/learn.md#abstract-factory-compared-with-a-function-or-factory-method) |
| Reuse a configured group's settings while giving a new listing no old identity or members: Prototype. Struct assignment alone can share slice backing storage. | `NewGroupPrototype`, `CloneAsDraft` in [patterns.go](lab-01/patterns.go) | [copying and identity](lab-01/learn.md#prototype-compared-with-copying-fields-at-the-call-site) |
| Import a feed that supplies total rent instead of cost per person: Adapter. The boundary translates representation before delegation. Total 1501 across three people rounds upward to 501. | `LegacyOfferAdapter`, `groupInputFromLegacyOffer` in [structural.go](lab-02/structural.go) | [conversion contract](lab-02/learn.md#adapter-convert-before-crossing-the-service-boundary) |
| Add auditing while preserving a writer's operations and returned results: Decorator. It wraps the same contract rather than translating a foreign one. | `AuditedGroupWriter`, `next GroupWriter` in [structural.go](lab-02/structural.go) | [delegation and best-effort logging](lab-02/learn.md#decorator-wrap-the-same-writer-contract) |
| Offer one onboarding operation coordinating services: Facade. Fewer calls for the client do not create an atomic transaction. A later failed step can leave earlier writes. | `OnboardingFacade.Onboard` in [structural.go](lab-02/structural.go) | [partial completion](lab-02/learn.md#facade-sequence-calls-without-promising-a-transaction) |
| Vary inclusion behavior for the same listing loop: Strategy. Affordability and availability answer different questions. | `AvailableGroups.Matches`, plus `GroupFilter` and `ListGroups`, in [behavioral.go](lab-03/behavioral.go) and [service.go](lab-03/service.go) | [state and decision](lab-03/learn.md#start-with-state-and-a-decision) |
| Carry a join request as a value that a common invoker can execute: Command. Storing receiver and IDs does not automatically add undo, persistence or a background queue. | `GroupCommand`, `JoinCommand`, `CommandDispatcher.Run` in [behavioral.go](lab-03/behavioral.go) | [request as a value](lab-03/learn.md#turn-a-request-into-a-value) |
| Notify registered listeners after successful invocation: Observer. This lab's callbacks are immediate and ordered, not durable asynchronous delivery. A repeat join produces another success event although membership is unchanged. | `CompletionObserver`, `CompletionLog`, `Subscribe`, `Run` in [behavioral.go](lab-03/behavioral.go) | [notification and event meaning](lab-03/learn.md#notify-listeners-after-success) |

The retained course lists extend beyond these nine. After the foundations, use [remaining creational examples](lab-01/learn.md#singleton-one-process-wide-catalog), [the four other structural patterns](lab-02/learn.md#the-remaining-structural-patterns-in-else), and [other behavioral distinctions](lab-03/learn.md#distinguish-the-other-behavioral-patterns). Study these from the existing guides rather than memorizing another catalogue here. UML class diagrams show static participants and relationships; sequence diagrams show the messages during one scenario. Trace both, then redraw them from memory.

System design grows from these contracts. Ask who owns the state, who may change it, which operation is atomic, how an error leaves the state, and which data crosses a boundary. If the store becomes remote, failures and concurrent requests matter. If observer effects must survive a crash, an in-memory callback is insufficient. Those are further requirements to analyze after the local object is clear; adding service boxes alone does not solve them.

## A staged route with practice and repair

These are proposed sessions, including initial explanation, coding and correction. They are not a promise of a completion date. Repeat a stage when the fresh exercise still needs help; shorten reading when reconstruction is already independent. Later retrieval is additional study time.

| Stage | Proposed time and work | Concrete output before moving on |
| --- | --- | --- |
| 1. Object and construction | 45 minutes: 15 read through construction, 10 trace the fields, 15 rebuild the constructor, 5 repair and retry a changed invalid input. | From a blank file, create a group and explain successful and rejected construction. |
| 2. Encapsulation and ownership | 50 minutes: 15 read mutation/lifetime, 20 implement `Rename` and `Join`, 10 investigate a copy/alias case, 5 explain cleanup responsibility. | Reject a bad rename without mutation; explain the repeat join and the slice-copy limit. |
| 3. Interfaces and SOLID | 50 minutes: 15 trace the filters and real service, 20 rebuild selection, 10 handle a changed rule, 5 repair. | Add a rule without editing the loop; justify each boundary in Lab 0 with an actual change. |
| 4. Construction patterns | 60 minutes: 20 read Lab 1, 20 reconstruct Builder without copying, 10 sketch the factory and prototype, 10 explain when a literal is sufficient. | A working small Builder, a family trace, and an explanation of fresh identity and slice ownership. |
| 5. Structural patterns | 60 minutes: 20 study Lab 2, 20 build a small rent adapter, 10 draw wrapper/facade call order, 10 repair a failed-step scenario. | Explain adapter versus decorator and why a facade does not imply rollback. |
| 6. Behavioral patterns | 60 minutes: 20 study Lab 3, 20 write a command and listener, 10 predict repeat/failure events, 10 repair. | Distinguish a policy, an action and a reaction in your own code. |
| 7. Mixed design and retrieval | 45 minutes on a later day: 10 reconstruct key contracts without notes, 20 solve an unfamiliar mixed case, 10 draw/explain it, 5 inspect and repair. | Choose a pattern or a simpler alternative with reasons, then adapt one changed requirement. |

These sessions total 370 minutes, about six hours ten minutes, spread across several days as useful. Any repeated stage adds time. An examiner's actual feedback can change priorities. A useful checkpoint is whether you can write, trace, explain a nearby distinction and adapt a fresh case without consulting the answer. Reading completion and a successful agent-run program leave that ability unobserved.

## Reconstruction prompts

Do these after the relevant explanation. Hide the checking section. Work in a separate scratch file; keep the reference example available for later comparison, not beside you while reconstructing.

1. From a blank file, declare the minimum `Group` state and write `NewGroup`. Decide the result for blank name, cost zero, capacity zero, capacity 21 and a space-padded valid name. Explain why `new(Group)` cannot replace the validator.
2. Write `Rename`. Predict the old and new name after a blank input. Changed condition: a valid name must be at most 60 Unicode characters. Where do you check, and why is byte length potentially wrong?
3. Write `Join` without copying. Start with a one-place group and Pat/Alex. Predict the result of Pat, Alex, Alex. Changed condition: a new person Jo with budget 600 tries after Alex. Which check rejects Jo and what remains unchanged?
4. Given `alias := group`, rename through the alias. Then try `copy := *group` and rename the copy. Explain the difference. Changed condition: `copy.members[0] = "other"` after Alex has joined. Why is the result different from assigning `copy.name`?
5. Rebuild `GroupFilter` and `CountMatches`. Add a new `HasSpace` rule that ignores budget. Changed condition: cost 700, budget 550, zero members, capacity one. Which of affordable, available and has-space select it?
6. One caller creates exactly one three-field input once. Decide whether to add Builder, Abstract Factory or neither. Defend the smallest readable choice. Changed condition: two onboarding families must each provide a compatible group draft and profile suggestion. What new reason now supports a factory family?
7. Design a `ChangeCapacity` operation. A group already has three members. Reject capacity two, allow three, reject zero and 21. State whether rejected operations mutate state. Why does a raw setter weaken the object contract?
8. You open an output file and pass it to the audit decorator. Who closes it? Can the decorator close every `io.Writer`? Changed condition: the output is a shared buffer used again after the operation. What ownership assumption would an automatic close violate?
9. Draw an actual Lab 0 join sequence and a Lab 3 successful command sequence. Mark the state change and event callbacks. Changed condition: the second observer panics. Does membership roll back? Does the third observer run?
10. A pattern name is forbidden in your first sentence. Explain why you chose the design for importing total rent, auditing writes, and changing the listing rule. Then give each pattern name and one alternative you rejected with a concrete reason.

## Hints and checking answers

Try each exercise before reading its entry. If a hint was needed, treat that attempt as assisted and check a new case after repair.

1. Hint: separate legal Go values from domain-valid groups. Check all inputs before returning the initialized pointer. Blank/zero/out-of-range inputs fail; trimming makes the padded valid name usable. `new(Group)` makes zero fields without validation. A constructor is also different from a store operation that assigns an ID and persists a record.
2. Hint: preserve the old state until every check passes. The blank rename returns an error and retains the old name. Apply the length check to the normalized candidate before assignment; `len(string)` counts bytes, while `utf8.RuneCountInString` counts Unicode code points. If the rule means user-perceived characters, code points may still be insufficient; define that requirement before implementing it.
3. Hint: distinguish “already belongs” from “new join”. Pat fails budget, Alex joins, Alex repeats successfully with one member. Jo passes budget and fails capacity; no append occurs. Matching the cost exactly is permitted. Moving the full check ahead of duplicate detection changes the repeat-success contract.
4. Hint: follow what is copied. Alias and original share one address. A struct copy has its own scalar fields, so a name assignment affects only the copy. Its slice descriptor initially refers to the same backing array, so replacing an existing member can affect the original. To isolate members, copy their elements to a new slice. Neither operation alone assigns a new domain ID.
5. Hint: keep the signature and vary only the predicate. `HasSpace.Matches` returns `MemberCount < Capacity`. Only has-space selects the 700-cost changed case. CountMatches remains unchanged. A filter is a read-time decision; it cannot guarantee the group remains available for a later join.
6. Hint: name the variation before choosing machinery. Use an input literal plus the existing validator for the one-off case. A named helper may help reuse; it need not be a GoF pattern. Multiple related products that vary together give Abstract Factory a reason. Builder addresses construction steps and options, which is a different problem.
7. Hint: the proposed capacity must respect both range and current membership. Check `newCapacity >= len(members)`, `newCapacity >= 1`, and `newCapacity <= 20` before assignment. Capacity three is full but valid. All rejection cases preserve the old capacity. A domain method makes those conditions part of the change contract.
8. Hint: inspect what the interface promises. The opener/caller owns the file unless the contract explicitly transfers ownership; it must handle successful completion including close errors. `io.Writer` has no Close method. Closing borrowed/shared output would make subsequent users fail and assume ownership that was never granted. The current decorator performs best-effort writes and leaves resource lifecycle to its caller.
9. Hint: locate the write before the callback loop. Lab 0 is client -> service -> writer/store -> domain -> save under lock -> result. Lab 3 adds dispatcher -> command -> service before successful callbacks. In the current synchronous dispatcher, a panic interrupts delivery to later observers; membership was already saved and is not rolled back. Returning a join error instead causes no success callbacks. A second successful identical join yields another completion event but one member.
10. Hint: use the mismatch or variation as the reason. Importing total rent requires representation conversion, so Adapter fits. Auditing adds behavior around the same writer contract, so Decorator fits. Varying selection behind the same listing loop is Strategy. A direct conversion helper or direct call can be enough when there is only one fixed use and no useful substitutable boundary. More participants are a cost to justify.

## Useful examiner distinctions

These are anticipated follow-ups from the current source, not reported questions from a defense. Answer first, then use the corresponding section above or linked lab guide to check.

- Type versus instance; domain ID versus pointer identity; entity versus snapshot.
- Constructor function versus struct literal versus `new`; creation versus persistence; constructor versus GoF Factory Method.
- Encapsulation versus private visibility; a getter versus a safe snapshot; setter versus validated domain action.
- Value receiver versus pointer receiver; struct copy versus deep copy; method-call convenience versus interface method sets.
- Memory lifetime versus resource lifetime; GC versus deterministic destructor; a borrowed writer versus an owned file; `defer` versus end of a block.
- Interface shape versus behavioral contract; dependency injection versus dependency inversion; composition versus embedding and class inheritance.
- Strategy versus Command; Adapter versus Decorator; Facade versus transaction; successful invocation versus a newly added member.
- Demonstrating a run versus independently reconstructing and adapting the code.

For a live coaching session, start with one explanation and one fresh changed case. Locate the first missing connection, repair it with a small trace, and try another case. Record only the performance actually observed. No independent learner response has been assessed by producing this guide.

Continue with [System design through one Domi join](system-design.md) to study concurrent callers, durable transactions, request retries and event delivery.
