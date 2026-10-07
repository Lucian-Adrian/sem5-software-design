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

A struct groups fields into a value. `GroupBuilder` holds one `GroupInput` in its `input` field. A method is a function with a receiver, such as `WithCapacity` on `GroupBuilder`:

```go
func (builder GroupBuilder) WithCapacity(people int) GroupBuilder {
    builder.input.Capacity = people
    return builder
}
```

The receiver is a value, so the method receives a copy of the builder, changes the copy, and returns it. This is why the following chain carries all three settings forward:

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

`OnboardingPresetFactory` requires two methods. `BudgetPresetFactory` and `ComfortPresetFactory` each define both methods, so Go allows either value where the interface is expected:

```go
var factory OnboardingPresetFactory = BudgetPresetFactory{}
```

There is no `implements` keyword. The compiler checks whether the method set matches. The parameter type of `buildOnboardingPreset` is the interface, so that helper can use either family without an `if` or `switch` on a concrete type.

The interface specifies which products can be requested, but it cannot prove that their values make sense together. `buildOnboardingPreset` performs that domain check by comparing the profile budget with the group's maximum monthly cost. An incompatible pair returns `ErrBudgetTooLow` instead of an `OnboardingPreset`.

### Slices and copying

A slice is a small descriptor containing a pointer to an array plus a length and capacity. Copying a slice copies that descriptor, not the array. In the prototype constructor:

```go
snapshot := source
snapshot.memberProfileIDs = append([]string(nil), source.memberProfileIDs...)
```

The first line copies the `Group` fields. The source and snapshot now have separate struct values, but their member slices initially refer to the same array. The second line appends every member ID into a newly allocated slice, so changing one slice element cannot change the other.

When the prototype creates a draft, it copies the template struct again and then replaces the copied slice:

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

A struct literal directly names the fields and is the simplest choice for a one-off value:

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

### The other creational patterns in the brief

- **Singleton** restricts a type to one shared instance. The Domi store is constructed once in `main` and passed to services, but the code does not enforce a process-wide singleton. Passing the store explicitly keeps dependencies visible and allows tests to construct their own store.
- **Object Pooling** reuses a bounded set of expensive objects. The in-memory application creates small values and does not have an expensive resource that needs pooling.
- **Factory Method** delegates creation of one product to a method that can vary by implementation. The factory interface in this lab creates a related pair, which is why Abstract Factory is the closer fit.

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

## Short practice cases

For each case, predict the result first, then explain the rule that determines it.

- Set the Builder capacity to 21. `Build` returns an `ErrInvalidInput`-wrapped error because `normalizeGroupInput` enforces `maxGroupCapacity` before a service call.
- Change the Budget suggestion to a monthly budget of 400 while the group costs 450. `buildOnboardingPreset` returns `ErrBudgetTooLow` and does not return a usable preset.
- Change the source's first member ID after calling `NewGroupPrototype`. The stored snapshot keeps the original ID because its member slice has a separate backing array.
- Save a `CloneAsDraft` result through the group service. The returned view has a fresh ID and a member count of zero because the clone cleared both fields before the store created it.

The CLI examples can be run from this folder:

```powershell
go run . --patterns
go run . --demo
go run . --addr localhost:8080
```
