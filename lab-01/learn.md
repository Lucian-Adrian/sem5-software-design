# Lab 1 learning notes

These notes explain the Go mechanisms and design choices behind [the report](report.md) and [the implementation](patterns.go). The examples use the local Domi backend and its illustrative monthly EUR values.

The assignment calls for three of the six creational patterns and knowledge of the remaining three. The report identifies Builder, Abstract Factory, and Prototype as the required examples; `--extra-patterns` demonstrates Singleton, Factory Method, and Object Pooling for comparison.

For one walkthrough of the core implementation, start in [patterns.go](patterns.go) and trace Builder, Abstract Factory, then Prototype. [patterns_demo.go](patterns_demo.go) wires those types to the services and JSON output; [patterns_test.go](patterns_test.go) checks their behavior. This file is private oral coaching, while [report.md](report.md) is the professor-facing account of the design and results.

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

If Python is familiar, a Go method's receiver plays a role like `self`, but Go writes its name and type before the method. A value receiver gets a copy of the struct; a pointer receiver can change the original value. Go also checks field and method types when compiling the package.

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

This resembles Python's protocol or duck-typing idea: a value is usable when it has the required methods. In Go, the interface names that contract and the compiler checks it before the program runs.

The interface specifies which products can be requested, but it cannot prove that their values make sense together. `buildOnboardingPreset` performs that domain check by comparing the profile budget with the group's maximum monthly cost. An incompatible pair returns `ErrBudgetTooLow` instead of an `OnboardingPreset`.

### Slices and copying

A slice is a small descriptor containing a pointer to an array plus a length and capacity. Copying a slice copies that descriptor, not the array.

This is close to assigning a Python list to another variable: both can refer to the same underlying elements. Copying the `[]string` descriptor alone does not isolate its members, so the prototype appends them into a separate slice.

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

Python often reports failures by raising exceptions. Here the error is an explicit return value beside the result, and each caller decides whether to return it, handle it, or continue.

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

These steps are separate writes, not one transaction. If a later preset fails validation, the earlier groups and profiles remain in the in-memory store. The patterns prepare or validate inputs; the services and store create records and assign IDs.

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

## Oral prompts

Try answering from the source before reading the checking answers.

1. What does Builder build, and which component later creates the stored record?
2. Why does each Builder setter return a builder value?
3. What does `Build` validate before returning a draft?
4. When would a struct literal be simpler than Builder?
5. Why is `OnboardingPresetFactory` an Abstract Factory?
6. Which relationship between the group's cost and profile's budget does the interface itself fail to guarantee?
7. How does this Abstract Factory differ from the optional Factory Method example?
8. Which group settings does the prototype preserve?
9. Which group state does `CloneAsDraft` clear, and why?
10. Why does `NewGroupPrototype` copy the source member slice?
11. What could go wrong if the group struct and its slice were copied without copying the slice's elements?
12. At what point does a cloned draft receive its new ID?
13. How does Go check that a factory satisfies `OnboardingPresetFactory`?
14. If Domi had only one preset family, what simpler construction could replace the Abstract Factory interface?
15. What exactly does the optional Singleton share?
16. What does the optional Factory Method workflow vary, and what steps remain common?
17. How does its Go implementation differ from the classic subclass-and-override form?
18. When is Abstract Factory a better fit than Factory Method?
19. What happens to a buffer when `BufferPool.Release` is called?
20. What evidence would be needed before claiming the pool improves performance?
21. Can this `BufferPool` be shared by goroutines safely as written? Why?
22. Changed condition: if `ComfortPresetFactory` changed its suggested budget from 900 to 650 while the group still cost 700, where would the error occur, and which records would already have been created by then?

## Checking answers

1. Builder returns a validated `GroupInput`. `GroupService.CreateGroup` then delegates to the store, which constructs and saves the `Group` and assigns its ID.
2. Each setter has a value receiver, so it updates a copy and returns that copy. The next call receives the settings accumulated so far.
3. `Build` calls `normalizeGroupInput`, which trims the name and checks that the name is nonblank, the cost and capacity are positive, and capacity does not exceed `maxGroupCapacity`.
4. A struct literal is simpler for one small, obvious input when callers do not need staged setup or shared validation in a build method.
5. Each factory returns the same two kinds of related products, a group draft and a suggested profile, for one onboarding family.
6. The interface checks method names and signatures only. `buildOnboardingPreset` checks that the profile is valid and can afford the group's maximum monthly cost.
7. Abstract Factory creates two related product kinds. The optional Factory Method example changes one group-draft creation hook while the rest of its workflow stays common.
8. It preserves the source group's monthly cost and capacity, while the caller supplies a new name.
9. It clears the old ID and membership so a saved clone becomes a new listing with a fresh identity and no inherited members.
10. A slice value contains a reference to its backing array. Copying the slice descriptor alone would let later source changes affect the prototype snapshot.
11. The source and copy would share the backing array. Mutating a member ID through one slice could then change what the other reads.
12. The draft has no ID. The store assigns one when `GroupService.CreateGroup` saves its input.
13. Go checks whether the concrete type's method set contains both required methods with matching signatures. There is no `implements` declaration.
14. Ordinary functions could return the one group and profile suggestion if callers never need to choose among interchangeable families.
15. It shares one catalog pointer and fixed default inputs. Services still create separate profile and group records with their own IDs.
16. The creator varies the group draft. The workflow still saves the group and default profile through their services and joins them.
17. Go has no class inheritance or method overriding. The workflow accepts a `GroupDraftCreator` interface, and a concrete value supplies its creation method.
18. Abstract Factory fits when one choice must create several related product kinds as a family, such as the group and profile inputs in `OnboardingPresetFactory`.
19. Release ignores nil, resets a non-nil buffer, and retains it only if the idle pool has room. Reset clears the current length but keeps allocated capacity for possible reuse.
20. A benchmark using representative workloads would need to show lower cost or allocation than direct buffer creation. This demo checks reuse, not speed.
21. No. The idle slice and its limit are accessed without synchronization, so concurrent calls can race.
22. `buildOnboardingPreset` detects that 650 is below 700 before the Comfort group or profile is saved. The demo has already saved its standalone Builder group and the Budget family's group and profile, and joined that profile. Those earlier records remain in the in-memory store when the error propagates, and the failed run does not reach JSON encoding.

## Write the Builder yourself

This exercise checks whether you can construct and change the design, rather than recognize the completed source. Start after the Go value-receiver example and the comparison with a struct literal above. It is private practice; leave the submitted source unchanged.

First explain the problem: a creation draft is assembled in steps and validated before being handed to the service. Builder prepares a `GroupInput`; it does not assign an ID or save a group. In this small example a struct literal plus the same validation can also be sufficient. Fluent method chaining alone does not prove that a Builder is necessary.

Close `patterns.go`. In a scratch file outside the assessed package, reconstruct a minimal `GroupInput`, a `GroupBuilder` holding that draft, a constructor, `WithName`, `WithCapacity`, `WithMaxMonthlyCost`, and `Build`. Begin with fields and one method before adding the rest. Choose whether setters return updated values or mutate a pointer, explain your choice, and make the call sites consistent. Reuse the validation idea from `normalizeGroupInput`: trimmed nonempty name, positive monthly cost, and capacity from 1 through 20. A standalone scratch example must supply its own validation rather than pretending the lab helper is available.

Before running it, predict valid construction, missing name, zero cost and capacity 21. Explain the error path and why an invalid result must not reach persistence. Check your reconstruction against the original only after an attempt, and identify which parts needed help.

Then change the requirement: construction must support a second named preset without duplicating validation. Decide whether a helper returning the same Builder, a constructor or a factory is enough. Explain why adding an Abstract Factory is or is not justified by the actual family of products. This is a design question with defensible alternatives, not a request to insert more patterns.

Fresh copy-rule check: with the current value-receiver implementation, compare `builder.WithCapacity(4)` with `builder = builder.WithCapacity(4)`. The first discards the modified copy; the second retains it. Explain the effect before running. A pointer-receiver redesign would have different mutation behavior and method-set implications, so it is a separate choice rather than a cosmetic edit.

No independent attempt has been observed yet. Compilation verifies an implementation's syntax and types; the explanation and changed case supply separate evidence about understanding.

## Short practice prompts

Predict each result, then explain which check or copy rule determines it.

- Set the Builder capacity to 21.
- Change the Budget suggestion's monthly budget to 400 while its group costs 450.
- Change the source's first member ID after calling `NewGroupPrototype`.
- Save a `CloneAsDraft` result through the group service.

## Practice checking answers

- `Build` returns an error wrapping `ErrInvalidInput` because `normalizeGroupInput` enforces `maxGroupCapacity` before a service call.
- `buildOnboardingPreset` returns `ErrBudgetTooLow`; it does not return a usable preset because the suggested profile cannot afford the group.
- The prototype keeps the original ID in its snapshot because `NewGroupPrototype` copied the source member slice into a separate backing array.
- The saved view has a fresh ID and zero members because the clone cleared its ID and membership before the store created the new record.

```text
Run the patterns demonstration, the baseline demo, or the HTTP server.
Use this directory as the package source.
```

## Run the source examples

```powershell
go run . --patterns
go run . --demo
go run . --addr localhost:8080
```
