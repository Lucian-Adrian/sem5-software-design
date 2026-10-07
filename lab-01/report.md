# Lab 1: Creational patterns in the Domi backend

**Student:** Lucian-Adrian Gavril, FAF-241 | **Language:** Go | **Revision:** 7 October 2026

## Introduction

Domi helps people find roommates and form groups to rent a home together. The backend carried forward from Lab 0 creates profiles and groups, checks whether a profile can afford a group's monthly cost, and stores records through application services. This laboratory examines three ways to decide how new objects are assembled: Builder prepares a group input in stages, Abstract Factory creates a related group and profile suggestion, and Prototype derives a clean group draft from an existing group.

The [ELSE Laboratory work #1 brief](https://else.fcim.utm.md/mod/assign/view.php?id=27825), titled "Creational Design Patterns," requires three patterns from Singleton, Builder, Prototype, Object Pooling, Factory Method, and Abstract Factory, and asks students to know the remaining patterns. The main `--patterns` demonstration presents Builder, Prototype, and Abstract Factory as the selected three. A separate `--extra-patterns` demonstration shows Singleton, Factory Method, and Object Pooling for comparison. These local examples illustrate construction and reuse choices; they do not claim production use or measured performance. The normal application does not need process-wide storage or a pool for its small values.

The examples use whole EUR per person per month. A group has a name, a maximum monthly cost, and a capacity from 1 to 20. A profile has a name and a monthly budget. These values make the domain rules concrete; they are example data, not measurements of a deployed service. The application is a local prototype with an in-memory store.

## Go types, objects, and the program

Object-oriented programming is one way to organize a program around objects that combine state with operations. State is the data held by a value. An operation is behavior that reads or changes that data. In Domi, a group has state such as its capacity and member IDs, and the domain method that adds a member checks whether the change is allowed. Go supports this style through types, methods, and interfaces. It does not require class inheritance; types can be composed by storing values of other types and can satisfy interfaces by providing the required methods. The [Go FAQ](https://go.dev/doc/faq#Is_Go_an_object-oriented_language) describes this approach.

A struct declares named fields. A type declaration such as `type Group struct { ... }` defines the shape of a group value, while each created value is one particular group. A Go variable holds a value of its declared type. A stored `Group` differs from a `GroupInput`: the former represents a domain record with an identity and membership; the latter represents only the settings requested by a caller.

```text
Keep the stored group record, creation input, and public response as separate shapes.
The record has its ID and member IDs.
The input has only the requested settings.
The response exposes an ID and a member count.
```

```go
type Group struct {
    id               string
    name             string
    maxMonthlyCost   int
    capacity         int
    memberProfileIDs []string
}

type GroupInput struct {
    Name           string `json:"name"`
    MaxMonthlyCost int    `json:"maxMonthlyCost"`
    Capacity       int    `json:"capacity"`
}

type GroupView struct {
    ID             string `json:"id"`
    Name           string `json:"name"`
    MaxMonthlyCost int    `json:"maxMonthlyCost"`
    Capacity       int    `json:"capacity"`
    MemberCount    int    `json:"memberCount"`
}
```

The code uses Go's `string` and `int` types. The `[]string` notation means a slice, a growable sequence of strings. Here it stores member IDs rather than whole profile records. The `GroupInput` has exported fields, whose names begin with capital letters, so callers in other packages can set them. Its struct tags map Go field names to JSON names. The fields in `Group` begin with lowercase letters and are unexported outside the package. All lab files use `package main`, so other files in this package, including the store, can access them.

The separation between input, domain record, and view serves different purposes. A request should not choose the ID assigned by the store. A domain group needs member IDs so membership rules can operate on them. A public view needs enough information to display the group, but can omit internal state. The same distinction appears with `ProfileInput`, `Profile`, and `ProfileView`. The profile model retains the monthly budget for eligibility checks while its response view exposes only its ID and name.

Each file starts with a package declaration. The package name `main` means Go builds these files together into an executable. The module file, `go.mod`, names the module `domi-lab-01` and declares Go 1.22. The project uses standard-library packages, not a separate application framework. Running `go run . --patterns` from the lab directory compiles the package in that directory and passes the `--patterns` option to its `main` function. The dot means "this package"; running only `main.go` would leave out declarations in the other files.

Methods attach behavior to a receiver type. In `func (builder GroupBuilder) WithName(name string) GroupBuilder`, the part in parentheses names the receiver value, `builder`, and its type, `GroupBuilder`. The method accepts a string named `name` and returns another `GroupBuilder`. A function or method can return several values, so `(GroupInput, error)` means one input and one possible error. Go code normally checks that error explicitly. The short declaration `:=` introduces local variables and assigns values; `err != nil` checks whether an error was returned. A `nil` error represents success.

A struct literal constructs a value by naming its fields, for example the factory can provide a `GroupInput` with a name, cost, and capacity. A method call uses a dot between a value and method, as in `builder.WithName(...)`. When the returned builder is followed by another dot and method call, the result of the first call becomes the receiver for the next. This ordinary method-call syntax is the basis of the builder chain.

The domain method `Group.addMember` uses a pointer receiver, written `*Group`, because it changes the group's membership slice. A pointer refers to a particular value, so the method can update that value rather than only a temporary copy. The store's map lookup still produces a group value copy, so `JoinGroup` writes the changed group back to the map after the method succeeds. The builder uses value receivers for a different reason: each setting returns the updated builder value for the next call.

Go's concrete types and interfaces supply the polymorphism used in the factory. An interface lists the operations a caller needs, not the data fields of a concrete type. A type satisfies an interface implicitly if it has methods with the required names and signatures. There is no separate `implements` declaration. This lets the demonstration work with multiple factory families through one interface value. These language conventions follow the style described in [Effective Go](https://go.dev/doc/effective_go).

## Builder: assemble and validate a group input

Builder separates the construction of a value from the code that uses it. It is useful when the caller sets related fields in distinct steps and wants a final validation point. In this implementation, the product is a `GroupInput`, which is a request to create a group. It is not yet the stored `Group`.

```text
Start with an empty group input.
Set its name, monthly cost, and capacity on successive builder values.
When all settings are present, normalize and validate the input.
```

```go
type GroupBuilder struct {
    input GroupInput
}

func NewGroupBuilder() GroupBuilder {
    return GroupBuilder{}
}

func (builder GroupBuilder) WithName(name string) GroupBuilder {
    builder.input.Name = name
    return builder
}

func (builder GroupBuilder) WithMaxMonthlyCost(amount int) GroupBuilder {
    builder.input.MaxMonthlyCost = amount
    return builder
}

func (builder GroupBuilder) WithCapacity(people int) GroupBuilder {
    builder.input.Capacity = people
    return builder
}

func (builder GroupBuilder) Build() (GroupInput, error) {
    return normalizeGroupInput(builder.input)
}
```

The receiver on each `With...` method is a value receiver, written as `(builder GroupBuilder)`. Calling the method copies the builder struct into the receiver. The method changes the receiver's `input` and returns that updated builder value. The next call receives the returned value. This is why the methods can be chained without a pointer receiver. In this builder the input contains strings and integers, so copying the builder copies those values. A value receiver is a shallow copy in Go; it would not independently copy a slice or map stored inside a future version of the builder.

The factory uses the chain this way:

```text
Create a fresh builder.
Set the family's group name, cost, and capacity in sequence.
Build one validated group input and return its error if invalid.
```

```go
func (BudgetPresetFactory) CreateGroupDraft() (GroupInput, error) {
    return NewGroupBuilder().
        WithName("Budget shared home").
        WithMaxMonthlyCost(450).
        WithCapacity(4).
        Build()
}
```

The dot begins each continued method call. `NewGroupBuilder()` returns a zero-valued builder. `WithName` returns the builder containing the name; the next call receives it, adds the cost, and returns it again. `Build` is called on the final value. The Budget family uses a cost of 450 and capacity 4; the Comfort family follows the same sequence with cost 700 and capacity 2. These values are in EUR per person per month.

The builder does not make an invalid value impossible merely because it has a `Build` method. Callers can construct `GroupInput` directly, and the HTTP handler also decodes request data into input types. Therefore validation must remain at the domain and persistence boundaries. The builder is a named, reusable path to a valid input, not a security boundary or a substitute for storage validation.

The shared validator trims whitespace from names, rejects a blank name, requires a positive monthly cost, and restricts capacity to 1 through 20. It returns the normalized input on success. On failure it returns a zero-valued input and an error that wraps `ErrInvalidInput` with a specific reason.

```text
Trim surrounding spaces from the proposed name.
Reject a blank name, nonpositive cost, or capacity outside 1 to 20.
Return the normalized input only after all checks pass.
```

```go
func normalizeGroupInput(input GroupInput) (GroupInput, error) {
    input.Name = strings.TrimSpace(input.Name)
    if input.Name == "" {
        return GroupInput{}, fmt.Errorf("%w: group name is required", ErrInvalidInput)
    }
    if input.MaxMonthlyCost <= 0 {
        return GroupInput{}, fmt.Errorf("%w: maxMonthlyCost must be greater than zero", ErrInvalidInput)
    }
    if input.Capacity <= 0 {
        return GroupInput{}, fmt.Errorf("%w: capacity must be greater than zero", ErrInvalidInput)
    }
    if input.Capacity > maxGroupCapacity {
        return GroupInput{}, fmt.Errorf("%w: capacity cannot exceed %d people in a roommate group", ErrInvalidInput, maxGroupCapacity)
    }
    return input, nil
}
```

The `if` statements reject one condition at a time and return immediately, so later code runs only when earlier checks pass. The format verb `%w` wraps the shared error value while adding detail. Callers can use `errors.Is` to identify the category even if a more specific message has been attached. Returning `(GroupInput{}, err)` on failure avoids presenting a partially valid input as a successful result.

A struct literal is shorter for a single fixed input at one call site. Builder adds value here because the standalone example and both preset families use the same staged construction and validator. If group inputs had few fields and one construction site, direct literals would be simpler. A pattern is useful when its named responsibility clarifies the code, not simply because it can be applied.

### Builder UML

The builder holds a GroupInput, returns updated builder values from its setting methods, and returns validated input from Build. Storage and ID assignment happen after this construction step.

![Builder class diagram](assets/builder.svg)

[Editable Isomorph source](assets/builder.isx) | [PNG export](assets/builder.png)

## Abstract Factory: create a compatible family

Abstract Factory provides one contract for creating a family of related products. A family here consists of a group input and a suggested profile input. They are related by a domain constraint: the profile's monthly budget must cover the group's maximum monthly cost. The products are input values; the factory does not write to the store.

```text
Require each factory family to produce both kinds of input.
Let the caller request the group and profile through the same contract.
```

```go
type OnboardingPresetFactory interface {
    CreateGroupDraft() (GroupInput, error)
    CreateSuggestedProfile() (ProfileInput, error)
}

type BudgetPresetFactory struct{}

func (BudgetPresetFactory) CreateGroupDraft() (GroupInput, error) {
    return NewGroupBuilder().
        WithName("Budget shared home").
        WithMaxMonthlyCost(450).
        WithCapacity(4).
        Build()
}

func (BudgetPresetFactory) CreateSuggestedProfile() (ProfileInput, error) {
    return buildProfileSuggestion("Budget renter", 500)
}

type ComfortPresetFactory struct{}

func (ComfortPresetFactory) CreateGroupDraft() (GroupInput, error) {
    return NewGroupBuilder().
        WithName("Comfort shared home").
        WithMaxMonthlyCost(700).
        WithCapacity(2).
        Build()
}

func (ComfortPresetFactory) CreateSuggestedProfile() (ProfileInput, error) {
    return buildProfileSuggestion("Comfort renter", 900)
}
```

Both concrete factories satisfy the interface because they provide the two matching methods. Their struct types are empty because their methods use fixed example settings and need no per-instance fields. The Budget family pairs a 450-cost group with a 500-budget profile. The Comfort family pairs a 700-cost group with a 900-budget profile. The concrete values differ, but callers can ask either factory for the same two kinds of product.

The shared interface describes available operations, not the compatibility rule. An interface can verify that a factory has methods returning `GroupInput` and `ProfileInput`; it cannot state that one returned budget must be at least the other returned cost. `buildOnboardingPreset` checks that relationship after asking for both products. It validates the profile using the domain constructor, then compares its normalized monthly budget with the group cost. If the budget is lower, it returns `ErrBudgetTooLow` instead of a successful preset. It normalizes the group before returning the pair as an `OnboardingPreset`.

```text
Ask one family for both proposed products.
Validate the profile and compare its budget with the group cost.
Return a preset only when the pair is valid and affordable.
```

```go
profileModel, err := newProfile("suggested-profile", profile)
if err != nil {
    return OnboardingPreset{}, err
}
if profileModel.monthlyBudget < group.MaxMonthlyCost {
    return OnboardingPreset{}, ErrBudgetTooLow
}

group, err = normalizeGroupInput(group)
if err != nil {
    return OnboardingPreset{}, err
}
return OnboardingPreset{Group: group, SuggestedProfile: profile}, nil
```

The ID string `suggested-profile` passed to `newProfile` is used only to reuse domain validation. This temporary model is not stored. The function returns the profile input, and the store later assigns the profile's real ID. Likewise, normalizing the group here produces an input but does not create a stored record. This keeps construction and validation separate from persistence.

The demonstration stores the families in a slice of anonymous structs. Each element has a display name and a factory stored through the interface type. It then loops over the elements and runs the same creation logic for each family, without branching on whether the current concrete value is Budget or Comfort.

```text
Keep each display name beside its factory.
Place different concrete factories in the same interface-typed field.
Iterate over the families with one shared creation path.
```

```go
families := []struct {
    name    string
    factory OnboardingPresetFactory
}{
    {name: "budget", factory: BudgetPresetFactory{}},
    {name: "comfort", factory: ComfortPresetFactory{}},
}
```

The slice is a list of values; the anonymous struct declares the element shape locally instead of adding a separate named type. Each `factory` field can hold either concrete factory because both satisfy `OnboardingPresetFactory`. The loop passes that interface value to the same function. This is polymorphism through a common contract, implemented with composition and interface satisfaction rather than subclassing.

Factory Method is related but has a different emphasis. A factory method generally lets a creator vary the construction of one product type. Abstract Factory coordinates several product-creation methods as a family. Domi needs the pair because the group and profile suggestion must be chosen together and checked for compatibility. If there were one permanent preset with no choice of family, two ordinary functions could be simpler than an interface.

### Abstract Factory UML

Both concrete factories satisfy OnboardingPresetFactory and create the same two product types. The shared preset helper checks affordability before the services save and join the products.

![Abstract Factory class diagram](assets/abstract-factory.svg)

[Editable Isomorph source](assets/abstract-factory.isx) | [PNG export](assets/abstract-factory.png)

## Validation, services, and storage

Creation has three distinct stages. A builder or factory prepares an input. A domain constructor checks the values and creates a domain record. An application service delegates the operation to a storage interface, and the store assigns identity and saves the record. These stages answer different questions: are the requested settings valid, what is the domain value, and can this record be persisted here?

The builder's `Build` calls the same group input normalizer used by the domain constructor. That gives the caller an early error before attempting storage. The store repeats validation when creating a group because callers can bypass the builder. For example, the HTTP handler decodes a `GroupInput` directly, and other service callers may create input literals. If storage trusted only the builder, invalid values could enter through those paths.

```text
Lock the store while creating the record.
Choose the next candidate ID and validate the input.
Only after validation succeeds, advance the counter and save.
Return a view of the stored record.
```

```go
func (store *MemoryStore) CreateGroup(input GroupInput) (GroupView, error) {
    store.mutex.Lock()
    defer store.mutex.Unlock()

    id := formatID("group", store.nextGroupID+1)
    group, err := newGroup(id, input)
    if err != nil {
        return GroupView{}, err
    }
    store.nextGroupID++
    store.groups[id] = group
    return group.view(), nil
}
```

The service calls its writer interface; the store implements that interface. The store calculates a candidate ID such as `group-1` but does not increment the counter until validation succeeds. Invalid input therefore does not consume an ID. A successful call saves the `Group` in a map keyed by ID, then returns a `GroupView`. The input had no identity, and the response contains the assigned one.

The `MemoryStore` is in-memory: its maps exist only while the process runs. The mutex protects access to those maps and ID counters. The new creational patterns do not add an alternate save path. The group and profile services still delegate creation and joins to the same store contracts used by the rest of the application. This keeps IDs, membership checks, and persistence responsibilities in one place.

## Prototype: copy settings without copying identity

Prototype creates a new object by starting from an existing object's state. A group listing can provide the settings for another listing, but copying a complete record would also copy its identity and member IDs. The prototype therefore distinguishes reusable configuration from state that belongs only to one group.

A Go assignment such as `snapshot := source` copies the struct value. Its string and integer fields are copied as values. The member field is a slice, however. A slice contains a pointer to an underlying array plus its length and capacity. Copying a slice value copies that descriptor, not the elements. The source and copied struct would still refer to the same member-ID array. Mutating an element through either slice could then be observed through the other.

```text
Validate the source group's reusable settings.
Copy the source record, then copy its member-ID elements into a new slice.
Store the resulting value as the prototype snapshot.
```

```go
func NewGroupPrototype(source Group) (GroupPrototype, error) {
    input, err := normalizeGroupInput(source.input())
    if err != nil {
        return GroupPrototype{}, err
    }

    snapshot := source
    snapshot.name = input.Name
    snapshot.memberProfileIDs = append([]string(nil), source.memberProfileIDs...)
    return GroupPrototype{template: snapshot}, nil
}
```

The expression `append([]string(nil), source.memberProfileIDs...)` creates a destination slice and appends each source member ID. The three dots expand the source slice into individual arguments to `append`. For a nonempty source, the new slice has its own backing array, so later changes to source membership do not alter the prototype snapshot. The name is set to its normalized version. The snapshot keeps its source ID and copied membership because it represents the source state from which drafts can be derived.

Copying a prototype into a draft uses a different membership rule. The prototype preserves a snapshot, but each new group must begin as a distinct record with no members. The clone method validates the requested new name together with the stored cost and capacity, copies the template, clears its ID, applies the normalized name, and replaces the member slice with an empty one.

```text
Validate the new name and the copied group settings.
Copy the snapshot, clear its old ID, and apply the new name.
Replace the member list with an empty slice and return the unsaved value.
```

```go
func (prototype GroupPrototype) CloneAsDraft(name string) (Group, error) {
    input, err := normalizeGroupInput(GroupInput{
        Name:           name,
        MaxMonthlyCost: prototype.template.maxMonthlyCost,
        Capacity:       prototype.template.capacity,
    })
    if err != nil {
        return Group{}, err
    }

    clone := prototype.template
    clone.id = ""
    clone.name = input.Name
    clone.memberProfileIDs = make([]string, 0)
    return clone, nil
}
```

The draft keeps the cost and capacity but does not keep the source identity or membership. The slice replacement matters even though the clone is a shallow copy of the template: after `clone := prototype.template`, the clone initially shares the template's slice descriptor and backing array. Assigning a new empty slice to the clone means it no longer points at that member array. The prototype's captured state remains intact.

An empty slice has length zero, so the resulting view reports `memberCount: 0`. Whether the empty slice is `nil` or non-`nil` does not change this count; here `make([]string, 0)` creates a non-nil empty slice. The store does not save this returned value as-is. The demo converts it to a `GroupInput` with the `input()` method, which contains only name, cost, and capacity. `GroupService.CreateGroup` then uses the normal store path to validate, assign a new ID, and save it.

This reset prevents two identity errors. Reusing the source ID could overwrite the original map entry, because IDs are storage keys. Reusing the source member IDs would make the copied listing appear occupied by the source group's members. A prototype centralizes the field-copy and reset rules, so each caller does not have to remember them.

The demonstration first reconstructs a domain `Group` from the joined Budget group's response. It appends the response members' IDs and checks that the number reconstructed agrees with `MemberCount`. The prototype then snapshots that joined state, and `CloneAsDraft` removes membership for the new listing. This check ensures that the data used for cloning is internally consistent before the example proceeds.

### Prototype UML

GroupPrototype holds a snapshot of Group. CloneAsDraft derives a new group value with copied settings, a blank ID, and no members; the service and store later assign the new identity.

![Prototype class diagram](assets/prototype.svg)

[Editable Isomorph source](assets/prototype.isx) | [PNG export](assets/prototype.png)

## Optional demonstrations of the other three patterns

The separate `--extra-patterns` mode demonstrates the three patterns outside the presentation's core selection. It uses the same application services and in-memory store, but starts with an empty store for its own process. The examples keep separate concerns visible: a process-wide default catalog, a method hook that varies one product, and a bounded pool for temporary byte buffers.

### Singleton: a catalog of fixed defaults

`OnboardingCatalog` holds a default group input and a default profile input. `DefaultOnboardingCatalog` initializes one pointer through `sync.Once` and returns the same pointer on later calls. Its accessors return input values, so callers can use the settings without receiving the stored group or profile. In the demo, `createCatalogExample` reads both defaults, creates records through `GroupService` and `ProfileService`, and joins the profile to the group. The singleton is the catalog of defaults, not the store or the saved records.

The catalog holds fixed defaults, not mutable application state. The store remains an ordinary dependency constructed in `main` and passed to services.

```text
Initialize the catalog once.
Return the shared pointer to the caller.
```

```go
func DefaultOnboardingCatalog() *OnboardingCatalog {
    defaultOnboardingCatalogOnce.Do(func() {
        defaultOnboardingCatalog = &OnboardingCatalog{
            defaultGroup: GroupInput{
                Name:           "Catalog starter home",
                MaxMonthlyCost: 600,
                Capacity:       2,
            },
            defaultProfile: ProfileInput{
                Name:          "New renter",
                MonthlyBudget: 800,
            },
        }
    })
    return defaultOnboardingCatalog
}
```

`sync.Once` runs the initializer once even if callers arrive concurrently, and the demo's pointer comparison confirms the same instance is returned. The fields are private to other packages and accessors return copies, but files within `package main` can still change them. Immutability is a package convention here.

#### Singleton UML

The shared catalog contains fixed default inputs. Its accessor and sync.Once live at package level, while DefaultGroup and DefaultProfile are methods returning value copies. The diagram distinguishes this initialization mechanism from saved domain records.

![Singleton class diagram](assets/singleton.svg)

[Editable Isomorph source](assets/singleton.isx) | [PNG export](assets/singleton.png)

### Factory Method: vary one product through an interface hook

`GroupDraftCreator` specifies one creation method for one `GroupInput`. `StudentGroupCreator` and `QuietGroupCreator` each implement that method and build their distinct values through `GroupBuilder.Build`. `GroupOnboardingWorkflow.Run` receives a creator interface, saves the returned group, creates the catalog's default profile, then joins it. This moves the variable group choice behind one hook while keeping the rest of onboarding in the workflow.

The profile remains the same across both creators, so this example varies one product. Abstract Factory coordinates multiple products as a family, as the core `OnboardingPresetFactory` does with group and profile inputs. A direct function would be simpler for one creator and no shared workflow.

The classic Factory Method arrangement often puts a factory method on a creator base class and lets subclasses override it. Go has no class inheritance or method overriding. This example uses composition: the workflow accepts a `GroupDraftCreator` interface, and concrete types supply different method results. The relationship is still a factory hook called by a common creator workflow, but its Go implementation uses interface satisfaction instead of subclassing.

```text
Ask the selected creator for one group draft.
Save the group, create the default profile, then join them through the services.
```

```go
func (workflow GroupOnboardingWorkflow) Run(creator GroupDraftCreator) (GroupOnboardingResult, error) {
    groupInput, err := creator.CreateGroupDraft()
    if err != nil {
        return GroupOnboardingResult{}, err
    }

    group, err := workflow.groups.CreateGroup(groupInput)
    if err != nil {
        return GroupOnboardingResult{}, err
    }

    profile, err := workflow.profiles.CreateProfile(workflow.catalog.DefaultProfile())
    if err != nil {
        return GroupOnboardingResult{}, err
    }

    joined, err := workflow.groups.JoinGroup(group.ID, profile.ID)
    if err != nil {
        return GroupOnboardingResult{}, err
    }
    return GroupOnboardingResult{Group: joined, Profile: profile}, nil
}
```

The interface varies only draft construction; validation, persistence, and membership stay in the shared service path.

#### Factory Method UML

The common workflow receives GroupDraftCreator. StudentGroupCreator and QuietGroupCreator provide that creation hook through interface satisfaction; the workflow then uses the existing services. These Go types do not inherit from a creator base class.

![Factory Method class diagram](assets/factory-method.svg)

[Editable Isomorph source](assets/factory-method.isx) | [PNG export](assets/factory-method.png)

### Object Pooling: reuse bounded temporary buffers

`BufferPool` retains up to its configured number of idle `bytes.Buffer` values. `Acquire` returns one from the idle slice or allocates a new buffer if none is available. `Release` resets the buffer and retains it only while the idle limit has room. In the demo, the limit is one. A first group summary is written, then its buffer is released and acquired again; the second write uses the same pointer after `Reset` cleared its old contents.

The `Reset` call clears logical contents but retains backing capacity and does not securely erase old bytes. The example establishes reuse and reset behavior only; it does not measure a performance improvement. The pool's configured limit applies to idle buffers, not buffers currently in use.

```text
Reset the released buffer.
Discard it when the idle pool is full; otherwise retain it for reuse.
```

```go
func (pool *BufferPool) Release(buffer *bytes.Buffer) {
    if buffer == nil {
        return
    }
    buffer.Reset()
    if len(pool.idle) >= pool.maxIdle {
        return
    }
    pool.idle = append(pool.idle, buffer)
}
```

For an empty store, the program saves the catalog group and the two Factory Method groups, each with one joined profile. It then formats the catalog group through the buffer that held the Quiet group's summary. This excerpt shows the pool fields reported by the actual JSON result.

```text
Report the summary text.
Record whether the same buffer returned empty and how many buffers remain idle.
```

```json
{
  "summary": "Catalog starter home has 1 member(s)",
  "reused": true,
  "resetOnReuse": true,
  "idleBuffers": 1
}
```

From the lab directory, run the optional demonstration with:

```text
Build and run the package in the current directory.
Select the optional Singleton, Factory Method, and Object Pool examples.
```

```sh
go run . --extra-patterns
```

The JSON sections are `singleton`, `factoryMethod`, and `objectPool`; the output includes group IDs, names, settings, and members created through the ordinary services. Because the store is new for each invocation, the IDs are examples for one process, not persistent identifiers.

#### Object Pooling UML

BufferPool retains idle bytes.Buffer pointers. Acquire lends one buffer to a caller and Release resets and optionally retains it. The idle limit does not limit the number of buffers currently in use.

![Object Pooling class diagram](assets/object-pool.svg)

[Editable Isomorph source](assets/object-pool.isx) | [PNG export](assets/object-pool.png)

## One execution from inputs to saved records

The `--patterns` option follows a separate command-line branch from `--demo` and `--extra-patterns`. Each one-shot mode uses the same in-memory store and service constructors but creates different sample data. `main` rejects a selection that combines more than one of the three demo modes; with no demo flag, it starts the HTTP server.

```text
After parsing options and constructing the services, inspect the patterns flag.
When it is set, run the creational demonstration with those services.
If the demonstration fails, log the error and exit unsuccessfully.
Otherwise return from main after printing the result.
```

```go
if *patterns {
    if err := runCreationalDemo(groups, profiles); err != nil {
        log.Fatal(err)
    }
    return
}
```

The call to `flag.Bool` returns a pointer to a Boolean option value, so `*patterns` reads whether the option was supplied. The services keep interface-typed dependencies: `GroupService` receives a reader and writer, and `ProfileService` receives a profile writer. The concrete store satisfies those interfaces by implementing their methods. Both services refer to the same store value, so groups and profiles created through them share the same maps and ID counters.

The demonstration creates its values in this order:

| Step | Operation | State or result |
| --- | --- | --- |
| 1 | Create the store and services | Empty group and profile maps with independent counters. |
| 2 | Build and save the standalone example | `group-1`, cost 650, capacity 3, no members. |
| 3 | Build the Budget family and save both inputs | `group-2` and `profile-1`; the profile joins the group. |
| 4 | Build the Comfort family and save both inputs | `group-3` and `profile-2`; the profile joins the group. |
| 5 | Reconstruct the joined Budget group and snapshot it | Source `group-2` has one member in the captured state. |
| 6 | Clone the snapshot and save the new input | `group-4`, same cost and capacity as Budget, no members. |
| 7 | Encode the result | One indented JSON object is written to standard output. |

The Budget profile can afford the 450 monthly cost because 500 is greater than 450. The Comfort profile can afford the 700 cost because 900 is greater than 700. Each profile is created by `ProfileService.CreateProfile` and joined using the group and profile IDs. The join passes through the existing service and storage path, so the member count in each preset result reflects a real in-memory membership change in this execution.

The prototype uses the joined Budget group rather than the original empty group input. The output therefore makes the reset visible: its source count is one and its copy count is zero. Its name differs, while cost and capacity remain 450 and 4. Since each invocation creates a new store, IDs start again from `group-1` and `profile-1`. They are not persistent identifiers.

From the lab directory, this command runs the demonstration. Go builds for the current machine when using `go run`, so this source command works on Windows, macOS, or Linux when Go is installed. A separately built executable targets an operating system and architecture; a Windows `.exe` is not directly runnable on Linux.

```text
Build and run the Go package in the current directory.
Pass the patterns option to print the pattern examples as JSON.
```

```sh
go run . --patterns
```

For a new process with an empty memory store, the demonstration prints the following JSON. The IDs reflect the creation order above; the data is produced by the application rather than being a persisted database snapshot.

```text
Create one Builder group.
Create and join the Budget and Comfort family products.
Clone the joined Budget group's settings into a new empty group.
Return the source and copy information as indented JSON.
```

```json
{
  "builderGroup": {
    "id": "group-1",
    "name": "Builder example home",
    "maxMonthlyCost": 650,
    "capacity": 3,
    "memberCount": 0
  },
  "presetGroups": [
    {
      "family": "budget",
      "group": {
        "group": {
          "id": "group-2",
          "name": "Budget shared home",
          "maxMonthlyCost": 450,
          "capacity": 4,
          "memberCount": 1
        },
        "members": [
          {
            "id": "profile-1",
            "name": "Budget renter"
          }
        ]
      },
      "suggestedProfile": {
        "id": "profile-1",
        "name": "Budget renter"
      }
    },
    {
      "family": "comfort",
      "group": {
        "group": {
          "id": "group-3",
          "name": "Comfort shared home",
          "maxMonthlyCost": 700,
          "capacity": 2,
          "memberCount": 1
        },
        "members": [
          {
            "id": "profile-2",
            "name": "Comfort renter"
          }
        ]
      },
      "suggestedProfile": {
        "id": "profile-2",
        "name": "Comfort renter"
      }
    }
  ],
  "prototype": {
    "sourceGroupId": "group-2",
    "sourceMemberCount": 1,
    "copy": {
      "id": "group-4",
      "name": "Budget shared home copy",
      "maxMonthlyCost": 450,
      "capacity": 4,
      "memberCount": 0
    }
  }
}
```

The outer result groups the standalone builder result, both preset results, and the prototype result. Each preset contains a group detail and a suggested profile view. The repeated profile data is intentional: the detail shows current group membership, while the separate field shows the profile product returned by its creation service. The prototype section reports its source ID and original member count beside the saved copy. These JSON keys come from struct tags in the result types.

## Source responsibilities

| Source | Responsibility |
| --- | --- |
| [domain.go](domain.go) | Domain records, input types, validation, membership rules, and response views |
| [patterns.go](patterns.go) | Builder, core Abstract Factory interface and families, and group Prototype |
| [patterns_demo.go](patterns_demo.go) | Core examples, service calls, result assembly, and JSON output |
| [optional_patterns.go](optional_patterns.go) | Optional Singleton catalog, Factory Method hook and workflow, and bounded BufferPool |
| [optional_patterns_demo.go](optional_patterns_demo.go) | Optional examples, service-backed records, buffer reuse, and JSON output |
| [main.go](main.go) | HTTP server, baseline demo, and the `--demo`, `--patterns`, and `--extra-patterns` modes |
| [service.go](service.go) | Application services and storage interfaces |
| [store.go](store.go) | In-memory persistence, synchronization, and ID assignment |
| [patterns_test.go](patterns_test.go) | Checks for Builder validation, Prototype isolation, and core factory-family consistency |
| [optional_patterns_test.go](optional_patterns_test.go) | Checks for singleton identity and copies, creator-hook workflow, mode selection, and buffer reuse bounds |

The core sequence of calls is `main -> runCreationalDemo -> createBuilderExample and createPresetExamples -> createPrototypeExample -> GroupService and ProfileService -> MemoryStore`. The optional mode calls `runExtraPatternsDemo`, which creates catalog records and then invokes the shared onboarding workflow with each creator before demonstrating the buffer pool. In both modes, pattern functions prepare inputs or drafts, services expose use-case operations, and the store assigns IDs and changes membership.

## Pattern methods, integration, and trade-offs

The six patterns solve different creation problems. The first three are the selected core of the laboratory report; the last three appear in the separate optional mode to make the other brief topics concrete.

| Pattern | Source and methods | How it joins the application | Strength in this example | Limitation in this example |
| --- | --- | --- | --- | --- |
| Builder | [patterns.go](patterns.go): `NewGroupBuilder`, `WithName`, `WithMaxMonthlyCost`, `WithCapacity`, `Build` | Produces a normalized `GroupInput`. The factory methods and standalone example pass that input to `GroupService.CreateGroup`; the store still validates and assigns the ID. | Reuses one readable, staged construction path and whole-input validator across multiple callers. | It is extra API for a small input, and callers can bypass it with a struct literal. Validation must still exist at the service/store boundary. |
| Abstract Factory | [patterns.go](patterns.go): `OnboardingPresetFactory`, `BudgetPresetFactory`, `ComfortPresetFactory`, `buildOnboardingPreset`; [patterns_demo.go](patterns_demo.go): `createPresetExamples` | A family supplies group and profile inputs. `buildOnboardingPreset` checks their relationship, then the services save them and join the profile to its group. | The loop can handle related product families through one contract while keeping family values together. | Method signatures cannot guarantee affordability; the helper must check it. The interface adds little if only one family exists. |
| Prototype | [patterns.go](patterns.go): `NewGroupPrototype`, `CloneAsDraft`; [patterns_demo.go](patterns_demo.go): `createPrototypeExample` | Captures the joined Budget group, copies its slice for the snapshot, clears identity and membership on the draft, then saves the draft through `GroupService`. | Centralizes which settings to preserve and which record-specific fields to reset. | New or reference-valued `Group` fields require an explicit copy/reset decision. For one copy of two fields, direct construction could be simpler. |
| Singleton | [optional_patterns.go](optional_patterns.go): `DefaultOnboardingCatalog`, `DefaultGroup`, `DefaultProfile`; [optional_patterns_demo.go](optional_patterns_demo.go): `createCatalogExample` | Supplies fixed input values that the normal services turn into independent saved records. It is not a singleton store. | A single process-wide source keeps default values consistent and makes instance reuse observable. | The shared dependency is global and can complicate tests or per-user configuration. Its immutability is a package convention, not enforced against other files in `package main`. |
| Factory Method | [optional_patterns.go](optional_patterns.go): `GroupDraftCreator`, `StudentGroupCreator.CreateGroupDraft`, `QuietGroupCreator.CreateGroupDraft`, `GroupOnboardingWorkflow.Run`; [optional_patterns_demo.go](optional_patterns_demo.go): `runExtraPatternsDemo` | The interface hook supplies one group input. The workflow saves it, saves the default profile, and joins them through the existing services. | One product varies while the common workflow remains in one method. | It adds a hook and workflow for two fixed cases. The creator selects only the group; the profile stays the catalog default. |
| Object Pooling | [optional_patterns.go](optional_patterns.go): `NewBufferPool`, `Acquire`, `Release`, `IdleCount`; [optional_patterns_demo.go](optional_patterns_demo.go): `createPooledSummary` | Temporarily formats group data in a `bytes.Buffer`, releases and resets it, then reacquires the same buffer for the next summary. | The bounded idle slice makes reuse and reset deterministic in this demonstration. | The pool has no synchronization, bounds only idle count rather than active buffers or bytes, and has no measured performance benefit for these small strings. |

The Singleton's `sync.Once` and pointer live in package-level variables because Go has no class-level storage. Its accessors return value copies, and the package uses those unexported fields as immutable defaults by convention. `BufferPool.Release` calls `Reset`, which empties the logical contents while retaining the buffer's backing capacity. It is not secure erasure and does not return that capacity to the allocator. `GroupOnboardingWorkflow.Run` saves in sequence; if a later profile or join step fails, earlier saved records are not rolled back even though the returned result is zero-valued.

## UML class diagrams in Isomorph

Each implemented pattern has a separate native Isomorph class diagram beside its explanation. The diagrams show focused relationships rather than combining all six patterns into one large graph. Interface realization means that a Go type supplies the required methods; it does not imply class inheritance. Field relationships show the collaborators or values a type holds. Some surrounding validation and transport code is omitted to keep each pattern readable. The method labels may omit the additional Go error return or pointer markers; the source excerpts give the full signatures. The pool diagram uses BytesBuffer as a diagram identifier for the standard-library bytes.Buffer type.

| Pattern | View | Editable source |
| --- | --- | --- |
| Builder | [Class diagram](assets/builder.svg) | [Isomorph](assets/builder.isx) |
| Abstract Factory | [Class diagram](assets/abstract-factory.svg) | [Isomorph](assets/abstract-factory.isx) |
| Prototype | [Class diagram](assets/prototype.svg) | [Isomorph](assets/prototype.isx) |
| Singleton, optional | [Class diagram](assets/singleton.svg) | [Isomorph](assets/singleton.isx) |
| Factory Method, optional | [Class diagram](assets/factory-method.svg) | [Isomorph](assets/factory-method.isx) |
| Object Pooling, optional | [Class diagram](assets/object-pool.svg) | [Isomorph](assets/object-pool.isx) |

## UML sequence diagram

The sequence follows the core `--patterns` path through the Budget factory and creation of a Prototype copy. The factory uses Builder to produce its group input and supplies a compatible profile suggestion. `createPresetExamples` saves the group first, creates and saves the profile second, then joins them through the services. The demo copies the joined group's settings into a fresh draft and saves the copy, which receives a new ID. The Comfort factory follows the same family contract in the demonstration's other iteration. The optional mode's Singleton, Factory Method, and Object Pool flow is documented separately above.

![Creational pattern workflow](assets/sequence.svg)

[Editable Isomorph diagram](assets/sequence.isx)  |  [PNG export](assets/sequence.png)

The arrows distinguish preparing a draft from storing a record. The prototype returns an unsaved value with a blank ID and empty membership; the service and store perform final creation.

## Results

The `--patterns` demonstration creates four groups in one in-memory process: one constructed directly with Builder, one for each onboarding family, and one Prototype copy. The two preset profiles are saved and joined to their matching groups through the existing services. The copy carries the Budget group's cost and capacity but has a different name, a new ID, and zero members. The JSON output above exposes these results and the creation order.

The source test file covers invalid Builder capacity, normalization of valid input, isolation of the prototype snapshot and draft membership, assignment of a fresh ID when a clone is saved, and validity and affordability of each factory family's inputs. These checks address the edge cases implied by the patterns; the demonstration output shows their ordinary successful path.

The separate `--extra-patterns` demonstration also ran from the lab directory. Its JSON reports `sameInstance: true`, a catalog group saved as `group-1` and joined by `profile-1`, then student and quiet groups saved as `group-2` and `group-3`, each with one joined profile. Its buffer summary is `Catalog starter home has 1 member(s)`; `reused` and `resetOnReuse` are true, and `idleBuffers` is 1. These are observed results from one process with a new memory store, not performance measurements or persistent identifiers.

The optional test source checks singleton identity and returned-copy isolation, creator-hook behavior and joined membership, the separate demo-mode selection, and reset, reuse, and the idle-buffer bound. The HTTP routes and original `--demo` option remain available. Both pattern modes use the same services and store contracts, so neither adds another persistence or ID-assignment mechanism. Records disappear when each process stops because this store is in memory.

## Conclusion

Builder assembles and validates a group input through a staged API. Abstract Factory returns a related group and profile suggestion through one family interface, while a separate domain check enforces that the profile can afford the group. Prototype captures an isolated source snapshot and produces a renamed draft with the same cost and capacity but no inherited identity or membership. The optional examples show one shared source for fixed defaults, one creator hook for a single varying product, and bounded reuse of temporary buffers. The services and store remain responsible for saved records, membership, and ID assignment in both modes.
