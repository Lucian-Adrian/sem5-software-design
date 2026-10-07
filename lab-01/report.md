# Lab 1: Creational patterns in the Domi backend

**Student:** Lucian-Adrian Gavril, FAF-241 | **Language:** Go | **Revision:** 7 October 2026

## Introduction

Domi helps people find roommates and form groups to rent a home together. The backend from Lab 0 already creates profiles and groups, checks whether a profile can afford a group's monthly cost, and stores records through application services. This laboratory asks how the application can prepare valid new groups, create compatible onboarding suggestions, and copy an existing group's settings without carrying over its identity or members.

The [ELSE Laboratory work #1 brief](https://else.fcim.utm.md/mod/assign/view.php?id=27825), titled “Creational Design Patterns,” asks for three patterns from Singleton, Builder, Prototype, Object Pooling, Factory Method, and Abstract Factory. This implementation uses Builder, Prototype, and Abstract Factory. Each handles a different creation decision: assembling a group draft, producing a related set of onboarding products, and deriving a new draft from an existing group.

The examples use whole EUR per person per month. A group has a name, a maximum monthly cost, and a capacity from 1 to 20. A profile has a name and a monthly budget. These values illustrate application behavior; they are not measurements of a deployed service. The application is a local prototype with an in-memory store.

## Go types used by the patterns

Go represents the application's records and drafts as structs. `Group` is the domain record with private fields, while `GroupInput` is the data needed to request a new group. A `GroupInput` is not saved and has no ID. `GroupService.CreateGroup` passes it to the store, which creates the domain record and assigns an ID. The same distinction applies to `ProfileInput` and `Profile`.

A method associates behavior with a type. For example, `WithName` is a method on `GroupBuilder`. Its value receiver gets a copy of the builder, changes that copy, and returns it. This lets the caller chain settings while keeping each builder value independent. `Build` passes all fields together to `normalizeGroupInput`, the same validator used by the domain constructor.

An interface lists the operations that a caller needs. A Go type implements an interface implicitly by defining its methods with matching signatures. `BudgetPresetFactory` and `ComfortPresetFactory` both satisfy `OnboardingPresetFactory`; there is no separate declaration. The caller can ask either factory for products through the same interface.

Go slices need special care when a struct is copied. Copying a slice copies its pointer, length, and capacity, not the underlying elements. Two copied structs can therefore refer to the same backing array. `NewGroupPrototype` copies the member-ID slice to keep its snapshot independent of later changes to the source group.

## Builder: assemble and validate a group draft

Builder separates the steps for assembling a product from the code that uses the finished product. It is useful when several fields must be set before the result can be checked as a whole. Here the builder creates a `GroupInput`, not a stored `Group`.

The builder keeps the draft in its `input` field. Each `With...` method changes one field and returns the updated builder. `Build` normalizes the completed input and returns the validation error if the name is blank, the cost is not positive, or the capacity is outside the domain limit.

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

`BudgetPresetFactory.CreateGroupDraft` uses this builder to make a group costing 450 per person per month with capacity 4. `Build` checks the three settings together. The caller then passes that input to `GroupService.CreateGroup`, where the normal store path assigns an ID. Validation is not duplicated in a second creation path.

A struct literal would be shorter for one fixed group written in one place. For example, `GroupInput{Name: "Budget shared home", MaxMonthlyCost: 450, Capacity: 4}` is appropriate when the caller has no staged assembly to express. The builder adds value in this implementation because the standalone demonstration and both preset families assemble group inputs through the same named steps and validator. It would become unnecessary ceremony if group inputs stayed small and had only one construction site.

## Abstract Factory: create a compatible onboarding family

Abstract Factory provides an interface for creating related products without requiring the caller to select each concrete product independently. Domi's onboarding family consists of a group draft and a suggested profile. The products are related because the profile's monthly budget must cover the group's maximum monthly cost.

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
```

`ComfortPresetFactory` implements the same two methods with a 700 cost, capacity 2, and a suggested budget of 900. `buildOnboardingPreset` asks one factory for both products, validates the profile and group, then checks that the budget is at least the group cost. The resulting `OnboardingPreset` cannot be returned successfully when the pair is unaffordable.

The shared interface allows the demonstration to iterate over a list of factories instead of branching on each family inside the creation loop. The interface alone cannot guarantee that products are compatible; that rule is enforced by `buildOnboardingPreset`. If Domi had only one preset and no caller needed to switch families, two ordinary functions returning the inputs could be simpler. The interface is useful when a caller chooses among complete families through one contract.

## Prototype: derive a clean draft from an existing group

Prototype creates a new object by starting from an existing object's state. In this application, an existing group can provide the settings for a new listing. Reusing the whole record would also reuse its ID and members, so `CloneAsDraft` preserves only the desired configuration and requires a new name.

`NewGroupPrototype` validates the source settings and takes a snapshot. The assignment `snapshot := source` copies the struct fields, but its slice field would still point at the source's member-ID array. The `append` expression copies those IDs into a separate slice. Later source changes therefore cannot rewrite the prototype's snapshot.

`CloneAsDraft` validates the requested name and copied settings. It then clears the old ID and replaces the member slice with an empty one. The returned value is not saved. Passing its input to `GroupService.CreateGroup` follows the regular storage path and gives the new group its own ID.

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

The snapshot retains a copy of the members so the prototype itself is isolated. A new draft deliberately starts with no members. Directly copying the source group at the call site could perform the same field assignments, but it would spread the reset rules among callers and risk keeping identity or membership by accident. A prototype method gives this specific copy operation one place to enforce those rules.

## One creation path from preset to copy

The Budget family shows how the three patterns fit the existing backend. `BudgetPresetFactory` uses `GroupBuilder` to prepare the group input and creates the suggested profile input. `buildOnboardingPreset` checks the two together. `runCreationalDemo` then calls `GroupService.CreateGroup` and `ProfileService.CreateProfile`, and joins the profile to the group through `GroupService.JoinGroup`.

The demo reconstructs a domain `Group` from the joined group's response before creating a prototype. This conversion copies the member IDs from the response and checks that their count matches `MemberCount`. The prototype captures the source state. `CloneAsDraft("Budget shared home copy")` preserves the cost and capacity, clears the source identity and membership, and returns a new draft. The service stores that draft and assigns a fresh ID.

This path keeps pattern-specific construction separate from persistence. Builder, Abstract Factory, and Prototype prepare values; the existing services and store remain responsible for saving records, assigning IDs, and enforcing membership rules.

## Source responsibilities

| Source | Responsibility |
| --- | --- |
| [domain.go](domain.go) | Domain records, input types, validation, and response views |
| [patterns.go](patterns.go) | Builder, onboarding factory interface and families, and group prototype |
| [patterns_demo.go](patterns_demo.go) | Pattern examples, service calls, result assembly, and JSON output |
| [main.go](main.go) | HTTP server, baseline demo, and the `--patterns` command-line option |
| [service.go](service.go) | Existing application services and storage interfaces |
| [store.go](store.go) | In-memory persistence, synchronization, and ID assignment |
| [patterns_test.go](patterns_test.go) | Builder validation, prototype isolation, and factory-family consistency checks |

## UML class diagram

The dashed realization arrows show implicit Go interface implementation. `GroupPrototype` holds a snapshot of a `Group`; its draft copy is saved later through the existing service and store.

```mermaid
classDiagram
    class GroupInput {
        +string Name
        +int MaxMonthlyCost
        +int Capacity
    }
    class ProfileInput {
        +string Name
        +int MonthlyBudget
    }
    class GroupBuilder {
        -GroupInput input
        +WithName(string) GroupBuilder
        +WithMaxMonthlyCost(int) GroupBuilder
        +WithCapacity(int) GroupBuilder
        +Build() GroupInput
    }
    class OnboardingPresetFactory {
        <<interface>>
        +CreateGroupDraft() GroupInput
        +CreateSuggestedProfile() ProfileInput
    }
    class BudgetPresetFactory
    class ComfortPresetFactory
    class GroupPrototype {
        -Group template
        +CloneAsDraft(string) Group
    }
    class Group
    class GroupService
    class ProfileService
    class MemoryStore
    class GroupReader {
        <<interface>>
    }
    class GroupWriter {
        <<interface>>
    }
    class ProfileWriter {
        <<interface>>
    }
    GroupBuilder ..> GroupInput : builds
    BudgetPresetFactory ..|> OnboardingPresetFactory
    ComfortPresetFactory ..|> OnboardingPresetFactory
    BudgetPresetFactory ..> GroupBuilder : uses
    ComfortPresetFactory ..> GroupBuilder : uses
    OnboardingPresetFactory ..> GroupInput : creates
    OnboardingPresetFactory ..> ProfileInput : creates
    GroupPrototype o-- Group : snapshots
    GroupPrototype ..> Group : creates draft copy
    GroupService --> GroupReader : reads through
    GroupService --> GroupWriter : writes through
    ProfileService --> ProfileWriter : writes through
    MemoryStore ..|> GroupReader
    MemoryStore ..|> GroupWriter
    MemoryStore ..|> ProfileWriter
```

## UML sequence diagram

The sequence follows construction through the Budget factory and the creation of a Prototype copy. The factory uses Builder to produce its group draft and supplies a compatible profile suggestion. The application saves and joins those records through the services, then copies the joined group's settings into a fresh draft. Saving the copy assigns its new ID. The Comfort factory follows the same family contract in the demonstration's other iteration.

![Creational pattern workflow](assets/sequence.svg)

[Editable Isomorph diagram](assets/sequence.isx) · [PNG export](assets/sequence.png)

The arrows distinguish preparing a draft from storing a record. The prototype returns an unsaved value with a blank ID and empty membership; the service and store perform the final creation.

## Results

The focused `go test ./...` run passes. The pattern tests check that Builder rejects an invalid capacity and normalizes valid input, that a prototype snapshot and its draft do not share membership, that saving a clone assigns a fresh ID, and that each factory produces valid related inputs with sufficient profile budget.

The `go run . --patterns` result contains four groups: one built directly with Builder, one for each onboarding family, and the Prototype copy. The Budget and Comfort groups each have one suggested profile joined through the service. The copy has a new ID and `memberCount: 0`; its source has one member. The output is the JSON result of the in-memory demonstration.

The HTTP routes and the original `--demo` option remain available. The `--patterns` path uses the same services and store, so it does not introduce another persistence or ID-assignment mechanism.

## Conclusion

The three patterns address different creation needs in Domi. Builder assembles and validates a group draft. Abstract Factory returns a compatible group and profile suggestion through one family interface. Prototype snapshots an existing group and creates a renamed draft with a new identity and no members. Each result passes through the existing application services before it is stored, so these patterns add construction options without replacing the backend's validation, membership, or persistence path.
