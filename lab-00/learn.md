# Learn Lab 0 through its actual Go source

This guide traces one real Domi workflow through the code in [domain.go](domain.go), [service.go](service.go), [store.go](store.go), [http.go](http.go), and [main.go](main.go). The [report](report.md) gives the professor-facing account. The current application creates profiles and roommate groups, lists all or affordable groups, shows group details, and joins a profile directly. City, preferences, pending join requests, and owner approval are not part of this Lab 0 implementation.

## Start with Alex's example

The command-line demonstration in [main.go](main.go) creates Alex with a monthly budget of 550. It creates two groups:

| `Group` | Monthly cost per person | `Capacity` |
| --- | ---: | ---: |
| Sunny apartment | 700 | 3 |
| Shared flat | 500 | 2 |

The store assigns profile-1 to Alex, group-1 to Sunny apartment, and group-2 to Shared flat. The `AffordableGroups` filter with `MonthlyBudget` 550 keeps Shared flat because 500 is at most 550. It excludes Sunny apartment because 700 is greater than 550. Alex can join Shared flat because the profile exists, the budget covers the cost, and one of its two places is available.

This one example connects the domain rule, filtering, storage, service interfaces, HTTP response, and all five SOLID principles. The same values appear in the actual `--demo` output.

## Find each responsibility in the source

| Source | Actual responsibility |
| --- | --- |
| [domain.go](domain.go) | `Group` and `Profile` data, `GroupInput` and `ProfileInput`, validation, view conversion, and `Group.addMember` |
| [service.go](service.go) | `GroupReader`, `GroupWriter`, `ProfileWriter`, `GroupFilter`, `GroupService`, and `ProfileService` |
| [store.go](store.go) | `MemoryStore` maps, ID assignment, locks, and implementation of the storage interfaces |
| [http.go](http.go) | Routes, JSON decoding, calls to services, JSON responses, and HTTP error mapping |
| [main.go](main.go) | Construction of `MemoryStore` and services, server startup, and the `--demo` path |

When Alex joins Shared flat, [http.go](http.go) reads the group ID from the route and profileId from the JSON body. The handler calls `GroupService`.`JoinGroup`. The service delegates to `GroupWriter`. `MemoryStore.JoinGroup` finds the records and calls `Group.addMember`. If the join succeeds, the store saves the changed group and builds a `GroupDetail` response.

The HTTP routes are POST /profiles, POST /groups, GET /groups, GET /groups/{id}, and POST /groups/{id}/join. The budget filter is selected with GET /groups?budget=550.

## Go foundations in this implementation

### Structs describe distinct stages of data

A struct groups named fields. `GroupInput` describes the values needed to create a group: `Name`, `MaxMonthlyCost`, and `Capacity`. It has no ID or members because it is input, not a stored record. `Group` is the domain record, with its ID, settings, and `memberProfileIDs`. `ProfileInput` carries a name and budget, while `Profile` is the stored domain value.

The code uses response structs for a different purpose. `GroupView` exposes a group's ID, name, cost, capacity, and member count. `ProfileView` exposes only a profile's ID and name. `GroupDetail` combines a `GroupView` with the member ProfileViews. Keeping these types distinct lets the code validate incoming data and control what it returns.

### Constructors return validation errors

`newGroup` trims the group name and rejects a blank name, a cost at or below zero, or a capacity outside 1 through 20. `newProfile` trims the profile name and rejects a blank name or a negative monthly budget. These functions return both a domain value and an error, so `MemoryStore` can stop before saving invalid records. A zero profile budget is allowed as input, but it cannot cover any valid group cost because group costs must be positive.

### Value and pointer receivers

A method is a function associated with a receiver type. `Group.view` has a value receiver because it reads a `Group` and constructs a summary. `Profile.view` is also read-only. `Group.addMember` has a pointer receiver because it appends to the group's membership.

The difference matters when `MemoryStore.JoinGroup` looks up a map value. The assignment group, found := store.groups[groupID] copies the `Group` value out of the map. The store calls group.addMember(profile) on this local variable, then writes the updated value back with store.groups[groupID] = group. The pointer receiver lets the method mutate that addressable local value; it does not make a map lookup return a reference to the stored map entry.

A pointer receiver is useful when a method must mutate the receiver or when copying the value would be undesirable. A value receiver suits a small value when the method can work on a copy, as `Group.view` does. Choosing a pointer receiver does not make every caller's value persistent by itself. The caller still has to save the changed value in the appropriate place.

The write-back step in [store.go](store.go) happens only after the membership method succeeds:

```go
if err := group.addMember(profile); err != nil {
    return GroupDetail{}, err
}
store.groups[groupID] = group
```

The `group` variable is the local value read from the map. The method changes that value, and the assignment records the changed membership in storage. Returning on error prevents a rejected join from being saved.

### Lowercase fields are package-unexported, not private to a struct

`Group`'s fields begin with lowercase letters: id, name, maxMonthlyCost, capacity, and `memberProfileIDs`. Go treats these identifiers as unexported outside their package. All files declared as `package main` in this lab can still access them. This is a package boundary, not a rule that only methods on `Group` may touch the fields.

`GroupInput` and `GroupView` use exported fields because other packages and the JSON encoder need access to them. The struct tags on their fields define JSON names such as maxMonthlyCost and memberCount. The API decodes request bodies into input types and encodes response views. It returns `ProfileView` instead of `Profile`, so a member's monthly budget is not included in group details.

### Interfaces specify behavior contracts

An interface lists method signatures a caller needs. `GroupFilter` asks for one method:

```go
type `GroupFilter` interface {
    `Matches`(group `GroupView`) bool
}
```

`AllGroups` satisfies that interface with `Matches`(`GroupView`) bool, which always returns true. `AffordableGroups` has a `MonthlyBudget` field and implements the same method by comparing the group's `MaxMonthlyCost` with that budget. Go checks interface satisfaction implicitly from a type's method set. No implements keyword is written.

The compiler checks that the method signature matches. It cannot prove that `Matches` actually answers the intended question. That semantic promise is part of the contract: a filter returns true exactly when its rule says the group belongs in the result. Both current filters honor that contract even though they return different sets.

`MemoryStore` also satisfies three interfaces: `GroupReader`, `GroupWriter`, and `ProfileWriter`. Its methods have pointer receivers, so `*MemoryStore` is the implementation passed to the services. A value `MemoryStore` would not have those pointer-receiver methods in its method set.

### Errors and HTTP status codes belong at different boundaries

Domain and service functions return Go errors. For example, `Group.addMember` returns `ErrBudgetTooLow` or `ErrGroupFull`. `GroupService`.`JoinGroup` passes a returned error to the HTTP handler without converting it to a status code. The HTTP layer translates service errors to transport responses.

| Go error | HTTP response |
| --- | --- |
| `ErrInvalidInput` | 400 Bad Request |
| `ErrGroupNotFound` or `ErrProfileNotFound` | 404 Not Found |
| `ErrGroupFull` | 409 Conflict |
| `ErrBudgetTooLow` | 422 Unprocessable Entity |
| An unrecognized internal error | 500 Internal Server Error |

The JSON error body has error and message fields. This separation lets the domain describe what failed without depending on HTTP. The same domain result could be returned by another caller, such as a command-line operation, without inventing an HTTP status there.

### Composition provides collaboration without class inheritance

Go has no class inheritance syntax. `GroupService` contains a `GroupReader` and a `GroupWriter`, and `ProfileService` contains a `ProfileWriter`. These fields are collaborators supplied through the service constructors. `MemoryStore` implements the interfaces, so the services can call the required behavior without accessing its maps directly.

In main.go, one `*MemoryStore` is created and passed to NewGroupService and NewProfileService. This is dependency injection: main selects the concrete implementation and supplies it to the services. The services use composition because they hold and call their dependencies. A type can satisfy several interfaces without being a subclass of any of them.

## The actual join rule

The membership rule lives in `Group.addMember` in [domain.go](domain.go). In order, it:

1. Checks whether the profile ID is already in `memberProfileIDs`. If so, it returns success without changing the group.
2. Rejects a new member whose monthlyBudget is less than maxMonthlyCost with `ErrBudgetTooLow`.
3. Rejects a new member when the number of member IDs is already at least capacity with `ErrGroupFull`.
4. Appends the profile ID and returns success.

The order makes repeated joins idempotent. Suppose Shared flat is full and already contains Alex. Repeating Alex's join finds Alex's ID in the membership slice at step 1 and succeeds without adding another entry. The capacity check is not applied to an existing member.

For a new, existing profile who is not already a member, with budget 400 and a place still available, Shared flat's cost of 500 causes `ErrBudgetTooLow`. The method returns before appending the ID. For a new profile with budget 550 when the group already has two members, the affordability check succeeds and the capacity check returns `ErrGroupFull`. Again, the ID is not appended. If both checks would fail, budget is checked first, so `ErrBudgetTooLow` is returned before the capacity result.

These are business rules. They answer whether the join is allowed. The mutex in `MemoryStore` answers a different question: how to keep concurrent operations from corrupting or overfilling shared state.

## Why `MemoryStore` uses a mutex

`MemoryStore` protects its maps and ID counters with sync.RWMutex. Read-only operations such as `ListGroups` and `GetGroup` use RLock. Creation and joining use Lock because they modify data.

`JoinGroup` holds the write lock while it looks up the group and profile, calls `Group.addMember`, writes the updated group back to the map, and prepares the response. The lock makes this read-check-write sequence one operation relative to other store calls.

Imagine a group with capacity 2 and one current member. Two eligible profiles try to join at nearly the same time. Without a lock around the whole sequence, both calls could observe one remaining place and both pass the capacity check before either update is saved. With the lock, one call completes first. The next call then observes a full group and receives `ErrGroupFull`.

The mutex does not decide whether a profile can afford the rent, whether a group has a place, or whether a duplicate join should succeed. `Group.addMember` decides those rules. The mutex ensures that concurrent store operations apply the rules to a consistent state.

## SOLID traced through Alex's workflow

SOLID is a set of design questions, not a requirement to make every function indirect. Each principle below connects to types and behavior in the Lab 0 source.

### Single Responsibility Principle

A component should have one focused reason to change. `Group.addMember` owns membership decisions. `MemoryStore` owns maps, IDs, and synchronization. `GroupService` coordinates the application operation through interfaces. The HTTP API decodes requests and chooses response codes.

If the maximum capacity changes, domain validation and membership behavior need review. If the error response format changes, http.go changes. Keeping those jobs separate means a transport edit does not rewrite the cost comparison in `Group.addMember`.

### Open/Closed Principle

A stable operation should allow a real variation to be added without rewriting its central algorithm. `GroupService.ListGroups` fetches group views and asks the supplied `GroupFilter` whether to keep each one. `AllGroups` and `AffordableGroups` provide two rules through that same contract.

With Alex's budget of 550, `AffordableGroups` includes Shared flat at 500 and excludes Sunny apartment at 700. A future OpenPlaces filter could include groups whose `MemberCount` is below `Capacity` without changing the selection loop. The caller would still need to choose and pass that filter; OCP does not mean that no file can ever change.

### Liskov Substitution Principle

A value used through an interface must keep the behavior callers expect from that interface. `ListGroups` can use `AllGroups` or `AffordableGroups` as a `GroupFilter`. Both accept a `GroupView` and return a decision. `AllGroups` includes Sunny apartment and Shared flat; `AffordableGroups` includes only Shared flat for Alex. The result sets differ because their rules differ, but each is a valid inclusion predicate for the service.

Go's compiler confirms the method signatures, not the full behavior. A filter that panicked on a valid empty group or silently ignored its budget would violate the caller's expectations even though it compiled.

### Interface Segregation Principle

A caller should not depend on operations it does not need. `GroupReader` contains listing and detail reads. `GroupWriter` contains group creation and joining. `ProfileWriter` contains profile creation. `ProfileService` needs only `ProfileWriter`; it has no reason to depend on group listing or membership operations. `GroupService` receives the group read and write contracts because its operations use both.

`MemoryStore` implements all three interfaces, but each service sees only its declared capabilities.

### Dependency Inversion Principle

High-level application behavior should depend on stable abstractions rather than choosing low-level storage details. `GroupService` stores `GroupReader` and `GroupWriter` fields. `ProfileService` stores a `ProfileWriter` field. Neither service constructs `MemoryStore` or reaches into its maps.

The main function creates `MemoryStore` and passes it into the constructors. Replacing memory storage with another implementation would require that implementation to honor the same results, errors, and membership guarantees. The constructors are the injection point; the interfaces are the dependency inversion boundary.

## Oral prompts

Try answering these from the source before reading the answer section.

1. Trace Alex's profile with budget 550 through the affordable listing. Why does Shared flat appear and Sunny apartment not appear?
2. What is the difference between `GroupInput`, `Group`, `GroupView`, and `GroupDetail`?
3. Why does `Group.addMember` have a pointer receiver while `Group.view` has a value receiver?
4. `MemoryStore.JoinGroup` reads a `Group` from a map into a local variable. Why must it write the changed value back?
5. What happens when Alex repeats the join after Shared flat is full?
6. In what order does `Group.addMember` check a new member's budget and the group's capacity?
7. Which methods make `*MemoryStore` satisfy `GroupReader`, `GroupWriter`, and `ProfileWriter`?
8. How does `GroupFilter` let `ListGroups` share one selection loop between `AllGroups` and `AffordableGroups`?
9. What can the compiler prove about an interface implementation, and what must the tests or code review establish?
10. Why does `MemoryStore` need a lock around the complete join operation? Which rule is still decided by `Group.addMember`?
11. Why does the API return `ProfileView` for group members instead of `Profile`?
12. Where does `ErrGroupFull` become an HTTP status, and why is that mapping outside `Group.addMember`?
13. Give one concrete example of each SOLID principle in the actual Lab 0 files.
14. Why does passing `MemoryStore` to service constructors count as composition and dependency injection rather than inheritance?

## Answers

1. [main.go](main.go) creates Alex with budget 550, Sunny apartment at 700, and Shared flat at 500. [service.go](service.go) defines `AffordableGroups.Matches` as `MaxMonthlyCost` <= `MonthlyBudget`. The filter includes 500 and excludes 700.
2. [domain.go](domain.go) defines `GroupInput` as validated creation data, `Group` as the domain value with identity and membership, `GroupView` as the public summary, and `GroupDetail` as that summary plus member ProfileViews.
3. [domain.go](domain.go) uses a pointer receiver for `Group.addMember` because it changes `memberProfileIDs`. `Group.view` only reads fields and creates a new summary, so its value receiver is sufficient.
4. A map lookup copies the `Group` value. [store.go](store.go) calls addMember on the local copy and assigns that changed copy back to store.groups[groupID].
5. [Group.addMember in domain.go](domain.go) finds Alex's profile ID before checking capacity. It returns nil without appending, so the repeated join leaves the membership count unchanged even when the group is full.
6. [Group.addMember in domain.go](domain.go) checks for an existing member first. For a new member it compares `MonthlyBudget` with `MaxMonthlyCost`, then compares the current member count with `Capacity`, then appends the ID.
7. [service.go](service.go) defines the three contracts. [store.go](store.go) provides `ListGroups` and `GetGroup` for `GroupReader`, `CreateGroup` and `JoinGroup` for `GroupWriter`, and `CreateProfile` for `ProfileWriter`. The pointer receiver methods are visible on the pointer returned by NewMemoryStore.
8. [service.go](service.go) sets `AllGroups` when the filter is nil, fetches GroupViews once, then calls `Matches` for each view. New filter values can supply another decision rule while that loop stays the same.
9. [service.go](service.go) shows the method signatures. The compiler checks that required methods and signatures match. It cannot prove that `AffordableGroups` compares the right fields or that a replacement obeys the intended behavior; those semantics need reasoning and tests.
10. [store.go](store.go) holds the mutex across record lookup, membership check, mutation, and map write-back, making that sequence atomic relative to other store calls. [domain.go](domain.go) shows that `Group.addMember` still decides duplicate, budget, and capacity rules.
11. [domain.go](domain.go) defines `ProfileView` with only ID and `Name`. `Profile` contains `MonthlyBudget`, but the group response needs member identity and display name, not the private budget.
12. [http.go](http.go) maps `ErrGroupFull` to HTTP 409 Conflict in writeServiceError. The domain method returns a domain error so it does not depend on HTTP.
13. In [domain.go](domain.go), `Group.addMember` owns membership decisions (SRP). In [service.go](service.go), `GroupFilter` lets `ListGroups` accept new rules without rewriting its loop (OCP). `AllGroups` and `AffordableGroups` keep the `Matches` contract (LSP). `GroupReader`, `GroupWriter`, and `ProfileWriter` keep service dependencies narrow (ISP). `GroupService` depends on those interfaces, while main.go supplies `MemoryStore` (DIP).
14. [GroupService and ProfileService in service.go](service.go) hold interfaces in fields and call those collaborators. [main.go](main.go) supplies one concrete `*MemoryStore` through constructor arguments. No type inherits fields or methods from another type.

## Optional reference: the 52-concept catalog

Use the map as a speaking checklist. Each Domi example is a possible design choice, not a claim that the current code already contains it.

### Principles and object-oriented vocabulary

| No. | Concept | Meaning and Domi connection |
| ---: | --- | --- |
| 1 | SOLID | Five checks for maintainable object-oriented design. In Domi, use them to reason about profile, browse, and join changes; avoid adding abstractions just to tick names. |
| 2 | SRP | One focused responsibility. Keep a group's membership rule with membership behavior; keep screen rendering elsewhere. |
| 3 | OCP | Extend a stable behavior through a seam instead of editing it for each variant. Add an optional ranking rule without rewriting the browse flow when ranking actually varies. |
| 4 | LSP | A replacement must keep the behavior its callers rely on. A test group finder should honor the same filter expectations as a real one. |
| 5 | ISP | A caller should not depend on methods it does not use. Browsing reads group data; requesting to join writes a request. |
| 6 | DIP | Important policy depends on an abstraction, while storage or other details implement it. A join service uses `JoinRequestStore`, not a particular database type. |
| 7 | DRY | Keep one authoritative copy of a rule or fact. Share the capacity rule if several callers must follow it; do not merge unrelated code just because two lines look similar. |
| 8 | KISS | Prefer the easiest design that correctly serves the current journey. A simple filter function can beat a configurable matching framework. |
| 9 | YAGNI | Do not build speculative features. Skip a plugin system until Domi has real alternative matching policies to support. |
| 10 | Encapsulation | Hide internal state and expose safe operations. Keep membership changes behind a method that checks duplicates and capacity. |
| 11 | Abstraction | Expose the needed behavior while hiding implementation details. The browse use case asks for matching groups, not SQL statements. |
| 12 | Inheritance | In class languages, a subclass inherits or overrides a superclass's behavior. Go has no class inheritance; use interfaces for behavior contracts and struct composition for reused data or methods. |
| 13 | Polymorphism | One caller works with different values through the same behavior. `BrowseService` can use any valid `GroupFinder`. |
| 14 | Coupling | How much one part knows about another. If the join flow names every database detail, a storage change affects business logic. |
| 15 | Loose Coupling | Keep dependencies limited to the needed contract. Injecting a small request-store interface lets the join flow ignore storage mechanics, with a small cost in indirection. |
| 16 | Cohesion | How strongly a type's data and operations belong together. A group membership rule belongs with group membership state. |
| 17 | High Cohesion | Keep each module focused on closely related work. A profile type should manage profile data rather than send notifications. |
| 18 | Composition over Inheritance | Build behavior from contained values and collaborators. A group card can contain location data and use a formatter rather than depend on a subclass chain. |
| 19 | GRASP | Nine responsibility-assignment guidelines. **Information Expert:** the group checks its own capacity. **Creator:** the component that owns a request collection may create join requests. **Controller:** a join use case receives the UI's request and coordinates the steps. **Low Coupling:** that use case depends on a small store interface. **High Cohesion:** profile rules stay with profile behavior. **Polymorphism:** alternative ranking rules share one contract. **Pure Fabrication:** a notification sender may be a service created to keep delivery work out of domain types. **Indirection:** a small interface separates join logic from storage. **Protected Variations:** keep a boundary around a change-prone database or ranking policy. These are lenses for discussing responsibility, not mandatory classes. |

### Pattern vocabulary and GoF families

| No. | Concept | Meaning and Domi connection |
| ---: | --- | --- |
| 20 | Design Patterns | Named approaches to recurring design problems, with known trade-offs. A matching strategy can help when ranking rules genuinely vary; a pattern name alone is not a reason to add code. |
| 21 | GoF | The Gang of Four catalog describes 23 classic patterns. This checklist names 22 individual patterns and omits Interpreter, which evaluates a small language such as user-authored filter expressions. It is useful vocabulary for later coursework, not a checklist for the first Domi flow. |
| 22 | Creational | Patterns about choosing or coordinating object creation. A Domi group setup may need a creation policy if groups have several validated forms. |
| 23 | Structural | Patterns about how parts are composed. An adapter can connect an existing search client to Domi's group-search contract. |
| 24 | Behavioral | Patterns about communication and changing behavior. A strategy could rank groups by rent or location. |

### The 23 GoF patterns

The first three rows below explain GoF categories; these rows cover the remaining individual pattern names. All examples are possible future Domi uses.

| No. | Pattern | Mechanism, Domi example, and trade-off |
| ---: | --- | --- |
| 25 | Singleton | Keep one shared instance. A process-wide read-only configuration could be shared, but a global store hides dependencies and makes tests harder. |
| 26 | Factory Method | In GoF form, a creator superclass defines a creation operation and subclasses override which product is made. A Go constructor-like function is not this pattern. Go has no superclass override mechanism, so use an interface plus a supplied function or factory value if creation genuinely varies. |
| 27 | Abstract Factory | Create a matching family of related objects. Domi could pair a profile store and group store from the same in-memory or database setup; the extra family abstraction only helps if several products must stay compatible. |
| 28 | Builder | Assemble a complex object step by step. It could construct a validated search query from optional city, budget, and move-in filters; a builder is too much for a simple struct literal. |
| 29 | Prototype | Create a new object by copying a configured example. A group owner might start from a saved group template; copying nested mutable values needs care. |
| 30 | Adapter | Translate one interface into another. Wrap an existing housing-search client so it satisfies `GroupFinder`; this isolates incompatible method names or data formats. |
| 31 | Bridge | Separate a high-level operation from an implementation that can vary independently. Domi might vary search behavior and storage backend as separate axes; this adds indirection and is unnecessary if only one axis varies. |
| 32 | Composite | Treat one item and a tree of items through the same interface. A housing community could contain nested roommate groups; avoid the tree if Domi only lists flat groups. |
| 33 | Decorator | Wrap an object to add behavior while keeping the same interface. A group finder can be wrapped with caching or logging; multiple wrappers can make the call path harder to follow. |
| 34 | Facade | Offer a simpler operation over several collaborators. A `RequestToJoin` use case could coordinate validation, saving, and notification; keep it focused so it does not become a catch-all service. |
| 35 | Flyweight | Share common immutable data instead of repeating it. Many profiles could refer to shared city metadata; measure memory pressure before adding this complexity. |
| 36 | Proxy | Stand in for another object to control access or defer work. A lazy remote group finder might fetch details on demand; the extra hop can complicate failures and debugging. |
| 37 | Strategy | Make one part of an algorithm replaceable. Choose a rent-first or compatibility-first group ranking rule; pass a small function or interface only when rules need to vary. |
| 38 | Observer | Notify interested listeners when an event occurs. A group may notify its owner when someone requests to join; event ordering and duplicate delivery need clear handling. |
| 39 | Command | Represent an action as a value with an execution method. A `JoinCommand` can carry profile and group IDs so the request can be queued or logged; direct method calls are simpler without those needs. |
| 40 | State | Change behavior based on an object's current state. A join request can be pending, accepted, or declined; an enum and a few checks may be enough before separate state types are justified. |
| 41 | Template Method | A superclass fixes an algorithm outline and lets subclasses override steps. Go has no class inheritance for that mechanism. Use a plain function with injected steps or compose helpers when useful; do not disguise a function call as a literal GoF implementation. |
| 42 | Iterator | Visit elements without exposing how they are stored. Go slices already work with `for range`; a custom iterator is useful for lazy or paginated results. |
| 43 | Chain of Responsibility | Pass a request through ordered handlers until one handles or rejects it. Join checks could be separated into ordered policies, but a simple sequence of validation calls is easier until policies need independent assembly. |
| 44 | Mediator | Coordinate several components through one collaborator. A join use case can coordinate group lookup, request storage, and notification; keep its job narrow to avoid a god object. |
| 45 | Memento | Save a snapshot that can later restore prior state. A user could restore earlier search filters; protect any personal data included in the snapshot. |
| 46 | Visitor | Add operations over a stable family of element types without putting every operation on those types. Domi might calculate export summaries for several group variants; adding a new variant makes visitors harder to maintain. |

### Implementation, diagrams, and code health

| No. | Concept | Meaning and Domi connection |
| ---: | --- | --- |
| 47 | Dependency Injection | Supply a collaborator from outside instead of constructing a concrete dependency inside a use case. Pass a `JoinRequestStore` when making a join service; a constructor parameter is often enough in Go. |
| 48 | UML | A visual notation for communicating a design. Use a class diagram for types and relationships, and a sequence diagram for messages during the profile-to-request journey. |
| 49 | UML Class | Shows static types, fields, operations, and associations. A Domi diagram can show that a join request refers to one profile and one group. |
| 50 | UML Sequence | Shows participants and calls in time order. Draw the user browsing results, selecting a group, and submitting a join request. |
| 51 | Refactoring | Change internal structure while keeping externally observable behavior. Extract a duplicated group-capacity rule, then check that the same requests still succeed or fail. |
| 52 | AntiPatterns | Common designs that look convenient but cause recurring trouble. A god service that browses, validates, stores, and notifies makes the Domi flow hard to test and explain. |

The catalog is a reference list of possible design terms. Its hypothetical examples do not describe additional Lab 0 code.

The following Factory Method and Command examples are supplementary GoF explanations, not actual Lab 0 source. Lab 0 does not define JoinRequest, NewJoinRequest, JoinCommand, or a Command interface.

## Supplementary: GoF mechanisms in separate examples

### A factory function is useful, but not GoF Factory Method

A Go function can make a valid request and hide the initial status rule:

```go
package main

import "errors"

type JoinRequest struct {
	ProfileID string
	GroupID   string
	Status    string
}

func NewJoinRequest(profileID, groupID string) (JoinRequest, error) {
	if profileID == "" || groupID == "" {
		return JoinRequest{}, errors.New("profile ID and group ID are required")
	}
	return JoinRequest{
		ProfileID: profileID,
		GroupID:   groupID,
		Status:    "pending",
	}, nil
}
```

This is an ordinary factory function. It does not require inheritance or a subclass overriding a creator method. The GoF Factory Method has that creator/subclass structure in class-based languages. In Go, a function value or interface-based creator may express the same need to vary creation, but explain the language-specific mechanism accurately.

### A Command can carry the join action

```go
package main

type JoinRequester interface {
	RequestJoin(profileID, groupID string) error
}

type Command interface {
	Execute() error
}

type JoinCommand struct {
	requester JoinRequester
	profileID string
	groupID   string
}

func (command JoinCommand) Execute() error {
	return command.requester.RequestJoin(command.profileID, command.groupID)
}
```

The command stores the values needed to perform the operation and exposes one execution method. That can help with queues or audit logs. For a normal immediate call from a handler, invoking `RequestJoin` directly is simpler.

## Go references

- [A Tour of Go: Methods and interfaces](https://go.dev/tour/methods)
- [Go language specification: struct types, embedding, and method sets](https://go.dev/ref/spec#Struct_types)
- [Effective Go: embedding](https://go.dev/doc/effective_go#embedding)
- [Go FAQ: why there is no type inheritance](https://go.dev/doc/faq#Why_is_there_no_type_inheritance)
