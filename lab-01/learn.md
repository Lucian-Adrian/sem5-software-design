# Lab 1 learning notes

These notes explain the Go mechanisms and design choices behind [the report](report.md) and [the implementation](patterns.go). The examples use the local Domi backend and its illustrative monthly EUR values.

## Start with the application data

`GroupInput` is a request to create a group. It has a name, maximum monthly cost, and capacity. It does not have an ID or members because it has not been saved. `Group` is the domain record, with an ID and member profile IDs. Its fields are unexported, which means callers in other Go packages cannot change its state directly. `GroupView` is the public summary returned by the service.

This distinction explains why the patterns return different types:

- `GroupBuilder.Build` returns a validated `GroupInput`.
- `OnboardingPresetFactory` returns `GroupInput` and `ProfileInput` together as an onboarding suggestion.
- `GroupPrototype.CloneAsDraft` returns a `Group` value with no ID or members. The demo converts it back to input with `group.input()` before saving.

The service owns the normal creation path. `GroupService.CreateGroup` delegates to the store, and the store assigns the ID. A pattern that prepares an object does not automatically persist it.

## Go concepts used by the code

### Structs and methods

A struct groups fields into a value. `GroupBuilder` holds one `GroupInput` in its `input` field. A method is a function with a receiver, such as `WithCapacity` on `GroupBuilder`.

```text
Copy the builder.
Set its capacity and return the updated value.
```

```go
func (builder GroupBuilder) WithCapacity(people int) GroupBuilder {
    builder.input.Capacity = people
    return builder
}
```

The receiver is a value, so the method receives a copy of the builder, changes the copy, and returns it. This is why the following chain carries all three settings forward.

```text
Start with an empty builder.
Apply each setting to the returned value, then build the completed input.
```

```go
input, err := NewGroupBuilder().
    WithName("Budget shared home").
    WithMaxMonthlyCost(450).
    WithCapacity(4).
    Build()
```

Each method receives the builder returned by the previous method. If the caller writes `builder.WithCapacity(4)` and ignores the returned builder, the caller's original value is unchanged. That behavior is intentional for this value-style builder.

`GroupBuilder.Build` calls `normalizeGroupInput`. The function trims whitespace from the name and rejects an empty name, a non-positive cost, a non-positive capacity, or a capacity above `maxGroupCapacity`. It returns a new normalized input and an `error`; an invalid input is not silently turned into a usable draft.

### Interfaces and implicit implementation

`OnboardingPresetFactory` requires two methods. `BudgetPresetFactory` and `ComfortPresetFactory` each define both methods, so Go allows either value where the interface is expected.

```text
Create one concrete factory.
Store it in a variable typed by the shared interface.
```

```go
var factory OnboardingPresetFactory = BudgetPresetFactory{}
```

There is no `implements` keyword. The compiler checks whether the method set matches. The parameter type of `buildOnboardingPreset` is the interface, so that helper can use either family without an `if` or `switch` on a concrete type.

The interface specifies which products can be requested, but it cannot prove that their values make sense together. `buildOnboardingPreset` performs that domain check by comparing the profile budget with the group's maximum monthly cost. An incompatible pair returns `ErrBudgetTooLow` instead of an `OnboardingPreset`.

### Slices and copying

A slice is a small descriptor containing a pointer to an array plus a length and capacity. Copying a slice copies that descriptor, not the array.

```text
Copy the group record.
Copy each member ID into a new slice for the prototype snapshot.
```

```go
snapshot := source
snapshot.memberProfileIDs = append([]string(nil), source.memberProfileIDs...)
```

The first line copies the `Group` fields. The source and snapshot now have separate struct values, but their member slices initially refer to the same array. The second line appends every member ID into a newly allocated slice, so changing one slice element cannot change the other.

When the prototype creates a draft, it copies the template struct again and then replaces the copied slice.

```text
Copy the stored template.
Clear its ID and replace its member list with an empty slice.
```

```go
clone := prototype.template
clone.id = ""
clone.memberProfileIDs = make([]string, 0)
```

The clone therefore has no source identity and starts with an empty membership. The prototype's own snapshot remains intact.

### Errors as returned values

Go functions commonly return a result and an `error`. Callers check the error before using the result. For example, `NewGroupPrototype` returns an empty prototype plus an error when its source has invalid group settings. The demo propagates errors to `main`, which prints the failure and exits. This keeps invalid values from moving unnoticed through the creation flow.

## Trace the Budget family through the code

The clearest end-to-end example is the Budget onboarding family.

1. `BudgetPresetFactory.CreateGroupDraft` uses `GroupBuilder` to set the group name, cost 450, and capacity 4. `Build` normalizes and validates the input.
2. `BudgetPresetFactory.CreateSuggestedProfile` returns a profile suggestion named `Budget renter` with monthly budget 500.
3. `buildOnboardingPreset` validates the group and profile and confirms that 500 covers 450. It returns both inputs as one preset.
4. `runCreationalDemo` sends the group input to `GroupService.CreateGroup` and the profile input to `ProfileService.CreateProfile`. The store assigns IDs.
5. The demo calls `GroupService.JoinGroup`. The service and store apply the existing budget and capacity rules.
6. The joined group's response is converted back to a `Group` value. `NewGroupPrototype` takes an independent snapshot.
7. `CloneAsDraft` copies the cost and capacity, sets the requested copy name, clears the ID, and removes all member IDs.
8. The demo passes the draft's input through `GroupService.CreateGroup`. The store assigns a fresh ID, and the copy has zero members.

The same `runCreationalDemo` also creates one standalone Builder example and a Comfort family. Its JSON result keeps the existing field names: `builderGroup`, `presetGroups`, and `prototype`. The prototype result includes `sourceGroupId`, `sourceMemberCount`, and `copy` so the source and copy can be compared.

## Why use each pattern here?

### Builder compared with a struct literal

A struct literal directly names the fields and is the simplest choice for a one-off value.

```text
Create one group input.
Assign its name, cost, and capacity directly.
```

```go
GroupInput{Name: "Shared home", MaxMonthlyCost: 450, Capacity: 4}
```

Builder is useful when several callers set the same group properties through a readable sequence and should all call the same whole-input validator. It can also make optional configuration easier to follow when there are many fields. For a small input built in one place, a builder can add more methods and state than the problem needs.

### Abstract Factory compared with a function or Factory Method

A plain function can return one preset when the application has only one. The `OnboardingPresetFactory` contract is useful when the caller chooses among families that each provide the same two related product types. The caller can iterate over families and use the same validation and saving logic for all of them.

Factory Method is a different creational pattern: creation of one product is deferred to a method that implementations can vary. The code here does not create a subclass hierarchy or override one creation method. Each factory provides two product-creation methods, so the family of products is the central idea.

### Prototype compared with copying fields at the call site

Copying a few settings manually is easy once. A prototype centralizes the rules that make this a new listing: choose a new name, preserve cost and capacity, clear the old identity, and remove all members. The snapshot copy also shows how to isolate a slice from later source changes.

The prototype is worthwhile while the application has a meaningful template-copy operation or expects group settings to grow. For a one-off clone with only two fields, a small helper that takes those fields could be simpler. The important rule is that a new persisted record must not retain the old record's identity or membership.

### Singleton: one process-wide catalog

The optional `--extra-patterns` demonstration adds a singleton catalog for fixed default group and profile inputs. `DefaultOnboardingCatalog` uses `sync.Once` to initialize one pointer, then returns that same pointer on every call. The accessors return `GroupInput` and `ProfileInput` values, so a caller receives copies of the simple fields. The catalog contains defaults only: it does not own services, stored groups, profiles, or IDs. Because all lab source files belong to `package main`, the fields are unexported outside the package but any file in the package could still change them. Their immutability is a convention the package follows, not a Go-enforced rule.

```text
Get the catalog twice and compare the returned references.
Read a copy of the default group input.
```

```go
catalog := DefaultOnboardingCatalog()
sameInstance := catalog == DefaultOnboardingCatalog()
groupInput := catalog.DefaultGroup()
```

This is a useful demonstration of identity and shared defaults, but it does not make the in-memory store a singleton. Putting mutable storage or application services behind a process-wide accessor would hide dependencies and make isolated tests harder. A singleton is appropriate only when the shared lifetime and identity are part of the requirement.

### Factory Method: vary one product behind a hook

The optional Factory Method example varies one product, the group draft. `GroupDraftCreator` defines the creation hook. `StudentGroupCreator` and `QuietGroupCreator` satisfy that interface implicitly, and `GroupOnboardingWorkflow.Run` accepts any creator. The workflow keeps the common operations in one place: request the draft, save it through `GroupService`, save the catalog's default profile through `ProfileService`, then join the profile to the group.

```text
Ask the creator hook for one group draft.
Save it and join the catalog's default profile through the same workflow.
```

```go
type GroupDraftCreator interface {
    CreateGroupDraft() (GroupInput, error)
}

type StudentGroupCreator struct{}

func (StudentGroupCreator) CreateGroupDraft() (GroupInput, error) {
    return NewGroupBuilder().
        WithName("Student shared home").
        WithMaxMonthlyCost(650).
        WithCapacity(3).
        Build()
}
```

```text
Accept the creator through its interface.
Save the draft and catalog profile, then join them.
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

Both excerpts are from the implementation. The hook returns a validated `GroupInput`; `Run` controls all steps after that choice, so it can reuse one workflow for either creator. The saves are sequential rather than transactional: if profile creation or joining fails, an earlier group may already be stored even though `Run` returns a zero result and an error. The in-memory store does not roll that group back.

In the classic Factory Method structure, a creator operation calls an overridable factory method, often with subclasses supplying different products. Go has no class inheritance or overriding. This implementation uses composition instead: `Run` receives the `GroupDraftCreator` interface, then calls its method; concrete types provide different hook results. It keeps the Factory Method relationship between a creator hook and one product without pretending that the Go types inherit from a base creator.

Abstract Factory remains the better fit when a caller must select a coordinated family of multiple product kinds. Here the Factory Method workflow varies only the group. Its profile suggestion is the same catalog default for both creators. This separation is useful, though introducing a workflow and interface for only two fixed options adds more structure than a direct function call would need.

### Object Pooling: reuse bounded temporary buffers

`BufferPool` retains up to `maxIdle` `bytes.Buffer` values. `Acquire` removes one retained buffer or allocates a new one. `Release` ignores `nil`, calls `Reset`, and retains the cleared buffer only when there is room. `Reset` clears the current contents while keeping the buffer's allocated storage available for reuse.

```text
Clear the released buffer.
Discard it if the pool is full; otherwise retain it for the next acquisition.
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

The demonstration sets the limit to one, writes a summary using a buffer, releases it, and acquires the same buffer again. The second acquisition checks both pointer identity and zero length before writing another summary; the JSON key is `resetOnReuse`. This shows the pool's reset-and-reuse behavior for that run, but it does not show a speedup. `Reset` clears the logical contents but does not securely erase the old bytes or free the backing capacity. The configured limit counts only idle buffers retained by the pool, not buffers currently checked out or total retained bytes. For these short strings, allocation is unlikely to justify the extra API. This custom pool is also not synchronized, so it is intended for a single goroutine. A concurrent pool would need synchronization, and production use should follow a measured allocation need.

## Add another Abstract Factory family

The following is a proposed extension, not an implemented family in this lab. It demonstrates the shortest path to adding a third onboarding family while reusing the existing contract and shared flow.

1. Choose compatible values. The proposed student group costs 650 EUR per person per month, so the suggested monthly budget is 800.
2. Implement both `OnboardingPresetFactory` methods. Use `GroupBuilder` for the group and the existing profile-suggestion validator for the profile.
3. Add one family value to the `families` slice in `createPresetExamples`. The loop already handles each item through the interface.
4. The shared `buildOnboardingPreset` function checks affordability, and the existing services save and join the products.

```text
Build the student group and compatible profile input.
Return both from the new factory's methods.
```

```go
type StudentPresetFactory struct{}

func (StudentPresetFactory) CreateGroupDraft() (GroupInput, error) {
    return NewGroupBuilder().
        WithName("Student shared home").
        WithMaxMonthlyCost(650).
        WithCapacity(3).
        Build()
}

func (StudentPresetFactory) CreateSuggestedProfile() (ProfileInput, error) {
    return buildProfileSuggestion("Student renter", 800)
}
```

```text
Add the new concrete factory to the interface-typed family list.
Let the existing loop process it.
```

```go
{name: "student", factory: StudentPresetFactory{}},
```

The proposed factory meets the interface because its method names and signatures match. The 800 budget covers the 650 cost, and `buildOnboardingPreset` repeats the domain check rather than relying on the example's arithmetic. Registering it in the family list lets the same save-and-join path create the records. No production use or successful run is claimed for this proposal.

## Questions for an oral explanation

1. **What does Builder build?** It builds a validated `GroupInput` draft. The service and store later create and save the `Group` record.
2. **Why does each setter return a builder?** The value receiver changes a copy and returns it, so the next method receives all settings chosen so far.
3. **Why validate in `Build`?** Validation sees the complete set of group fields in one place and returns a normalized input or an error before the service is called.
4. **When would a struct literal be better?** For a one-off input with a few obvious fields and no need for staged construction or shared build behavior.
5. **Why is the onboarding interface an Abstract Factory?** Each implementation creates the same two product kinds, a group draft and a suggested profile, for one coherent family.
6. **What does the interface fail to guarantee?** Matching method signatures does not guarantee that the returned profile can afford the returned group. `buildOnboardingPreset` checks that relationship.
7. **Why not call this Factory Method?** The code creates two related product types from one family interface rather than varying one product-creation method.
8. **What does the prototype preserve?** It preserves the source group's monthly cost and capacity and applies the new name.
9. **What does the prototype clear?** It clears the old ID and all member IDs so the result represents a fresh listing.
10. **Why copy the member slice when the draft removes it?** The prototype stores a snapshot of the source. Copying the slice makes that stored snapshot independent before a draft is created.
11. **What happens if a struct with a slice is copied directly?** The slice header is copied, so both structs may still point at the same backing array. Mutating an element through either slice can affect the other.
12. **When does a cloned group receive its ID?** It has no ID while it is a draft. `GroupService.CreateGroup` delegates to the store, which assigns the new ID.
13. **How does Go know a factory implements the interface?** It checks that the type's method set contains the interface's required methods. There is no explicit implementation declaration.
14. **What is the simplest pattern to remove if Domi has only one preset family forever?** The family interface may be replaced by ordinary functions if no caller needs to choose among interchangeable families.
15. **What does the optional Singleton share?** One catalog pointer and fixed default inputs. The group and profile returned by the services are separately created records with IDs.
16. **Why does the Factory Method workflow take an interface?** The common workflow needs one group-draft creation hook, while concrete creators supply different group settings.
17. **How is this different from the classic inheritance structure?** Go has no class inheritance or method override; the workflow accepts an interface value whose concrete type provides the creation method.
18. **When would Abstract Factory be a better fit than this Factory Method?** When a choice must produce several related product types as a family, as with the group and profile inputs in `OnboardingPresetFactory`.
19. **What does releasing a buffer do?** It calls `Reset`, which clears its current contents, then retains it only if the bounded pool has space.
20. **Does this pool prove improved performance?** No. It demonstrates reuse and reset only. A performance claim would need a benchmark for representative allocations.
21. **Can this `BufferPool` be shared by goroutines?** Not safely as written, because its idle slice and limit are accessed without synchronization.

## Short practice cases

For each case, predict the result first, then explain the rule that determines it.

- Set the Builder capacity to 21. `Build` returns an `ErrInvalidInput`-wrapped error because `normalizeGroupInput` enforces `maxGroupCapacity` before a service call.
- Change the Budget suggestion to a monthly budget of 400 while the group costs 450. `buildOnboardingPreset` returns `ErrBudgetTooLow` and does not return a usable preset.
- Change the source's first member ID after calling `NewGroupPrototype`. The stored snapshot keeps the original ID because its member slice has a separate backing array.
- Save a `CloneAsDraft` result through the group service. The returned view has a fresh ID and a member count of zero because the clone cleared both fields before the store created it.

```text
Run the patterns demonstration, the baseline demo, or the HTTP server.
Use this directory as the package source.
```

```powershell
go run . --patterns
go run . --demo
go run . --addr localhost:8080
```
