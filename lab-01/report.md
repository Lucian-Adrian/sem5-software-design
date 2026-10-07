# Lab 1: SOLID in the Domi roommate-group backend

Student: Lucian-Adrian Gavril, FAF-241. Language: Go. Lab reference date: 7 October 2026, selected by the student. Implementation/report revision: 15 October 2026.

## Introduction and condition

Domi helps people find roommates and form groups to rent a home together. This lab implements a small HTTP backend for creating profiles and groups, browsing affordable groups and joining one. It gives the design principles a visible user purpose and supplies JSON input/output for demonstration.

The student's current four-lab brief asks for one project that grows across the labs, at least three SOLID principles in the first lab, GitHub source, a Markdown report, UML and an oral defense with possible live changes. The accessible [ELSE SOLID brief](https://else.fcim.utm.md/mod/assign/view.php?id=38256) says to implement three SOLID principles and labels that slot Lab 0. This implementation explains all five. The site's historical dates are not adopted as current deadlines. The next labs will introduce appropriate creational, structural and behavioral patterns after their current briefs are established.

Go supports object-oriented design through structs, methods, interfaces and composition. It has no class-based inheritance. The following explanation uses the actual Go mechanisms rather than inventing a superclass/subclass hierarchy. [Go FAQ](https://go.dev/doc/faq#Is_Go_an_object-oriented_language).

## Tasks and work completed

- Define profiles with a name and monthly budget, and groups with a name, maximum per-person monthly cost, capacity and member IDs.
- Create valid records and expose safe response views.
- List all groups or only affordable groups through interchangeable filters.
- Join a group only when the profile exists, its budget covers the advertised cost and a place remains.
- Keep repeated joins idempotent and prevent simultaneous joins from exceeding capacity.
- Provide HTTP routes, a deterministic CLI demo, tests and source-level explanations.

Amounts are illustrative whole EUR per person per month. Groups have capacity 1 through 20. This stage joins immediately; pending requests and owner approval are planned behavior for a later lab. The API is local, in-memory and separate from the deployed [Domi frontend](https://trydomi.vercel.app).

## One small example

Ana has a monthly budget of 550. Two illustrative groups cost at most 500 and 700. `AffordableGroups{MonthlyBudget: 550}` accepts only the 500 group. Joining it checks the stored profile and group, changes membership once and returns a response with member count 1. A repeated request returns the same membership. Ana's personal budget is absent from the member response.

Actual demo values use the profile name Alex and groups `Sunny apartment` and `Shared flat` in [main.go](main.go). The 700 group receives `group-1`; the 500 group receives `group-2`. The demo prints only `group-2` in `affordableGroups`, followed by the joined group containing `profile-1` and member count 1. These are synthetic example values, not real housing data.

## The five SOLID principles

### SRP: single responsibility

**Each component has a focused reason to change.** [domain.go](domain.go) changes when membership rules change; [store.go](store.go) changes when storage changes; [http.go](http.go) changes when transport or JSON presentation changes. The application service connects the use case through contracts, while [main.go](main.go) constructs concrete collaborators once.

For example, a new capacity rule belongs in `Group.addMember` or validated construction. It should not be copied into every HTTP handler. A new JSON field should not require rewriting the group-filter algorithm.

### OCP: open for extension, closed for modification

**A new group selection rule can be added without editing the selection loop.** [service.go](service.go) defines the required behavior:

```go
type GroupFilter interface {
    Matches(group GroupView) bool
}
```

`GroupService.ListGroups` calls `filter.Matches(group)`. `AllGroups` accepts every group. `AffordableGroups` compares its budget with the group's maximum cost. Adding an `OpenPlaces` filter implements the same contract; the existing service loop remains unchanged. The client still needs to choose the new filter, so OCP does not mean no file can ever be edited.

### LSP: Liskov substitution

**Each implementation must preserve the expectations of the interface's caller.** Here the filter contract is: accept a `GroupView`, decide whether to include it, and return a boolean for any valid group. It does not require every filter to select identical groups. Both existing filters can be supplied to the same `ListGroups` call without a type check or special case.

```go
func (AllGroups) Matches(GroupView) bool { return true }

func (filter AffordableGroups) Matches(group GroupView) bool {
    return group.MaxMonthlyCost <= filter.MonthlyBudget
}
```

A new filter that panics for a valid zero-member group would violate the stated contract. Matching the method signature alone is insufficient. `TestGroupFiltersCanBeUsedThroughOneContract` exercises both current implementations through one service. A future PostgreSQL repository must also preserve the reader/writer contracts, including error meanings and atomic capacity enforcement; its existence is not claimed here.

### ISP: interface segregation

**Clients should request only the capabilities they use.** `GroupReader` has group listing and lookup; `GroupWriter` has creation and membership mutation; `ProfileWriter` has profile creation. `ProfileService` receives only a profile writer, rather than a large interface containing unrelated group operations.

```go
type ProfileWriter interface {
    CreateProfile(input ProfileInput) (ProfileView, error)
}

type ProfileService struct {
    writer ProfileWriter
}
```

The group service uses both reading and writing, so its constructor requests both. Small interfaces are useful boundaries; adding an interface for every trivial function would create needless indirection.

### DIP: dependency inversion

**The use-case service depends on storage behavior rather than constructing a concrete memory map.** `NewGroupService` receives `GroupReader` and `GroupWriter`. The composition point chooses `MemoryStore`:

```go
store := NewMemoryStore()
groups := NewGroupService(store, store)
profiles := NewProfileService(store)
```

Supplying those values through constructors is dependency injection. Depending on the small interfaces is the dependency-inversion decision. A future database implementation can satisfy those contracts while the service retains its workflow. The database implementation still needs validation and integration tests; changing a constructor argument alone does not make storage production-ready.

## OOP in the actual source

| Idea | Source-level example |
|---|---|
| Encapsulation | `Group` has unexported fields; `addMember` guards changes. Go's privacy boundary is the package, not the individual struct. |
| Abstraction | Services ask for reader/writer operations without knowing map layout or mutex details. |
| Polymorphism | `AllGroups` and `AffordableGroups` are values used through `GroupFilter`. |
| Composition | Services hold collaborators as fields; a store is passed into the service. There is no artificial inheritance tree. |

Response views contain public data rather than the mutable stored group. Profile views omit personal budgets. This is a limited exposure decision; authenticated access control is still absent.

## UML class diagram

These UML classes represent Go structs; the realization arrows represent implicit interface implementation. There are no superclass arrows. `MemoryStore` contains stored groups and profiles, while services depend on its contracts.

```mermaid
classDiagram
    class Group {
        -string id
        -string name
        -int maxMonthlyCost
        -int capacity
        -string[] memberProfileIDs
        -addMember(Profile) error
        -view() GroupView
    }
    class Profile {
        -string id
        -string name
        -int monthlyBudget
        -view() ProfileView
    }
    class GroupReader {
        <<interface>>
        +ListGroups() GroupView[]
        +GetGroup(string) GroupDetail
    }
    class GroupWriter {
        <<interface>>
        +CreateGroup(GroupInput) GroupView
        +JoinGroup(string, string) GroupDetail
    }
    class ProfileWriter {
        <<interface>>
        +CreateProfile(ProfileInput) ProfileView
    }
    class GroupFilter {
        <<interface>>
        +Matches(GroupView) bool
    }
    class AllGroups
    class AffordableGroups
    class MemoryStore
    class GroupService
    class ProfileService
    class API
    MemoryStore ..|> GroupReader
    MemoryStore ..|> GroupWriter
    MemoryStore ..|> ProfileWriter
    MemoryStore *-- Group
    MemoryStore *-- Profile
    GroupService --> GroupReader
    GroupService --> GroupWriter
    GroupService --> GroupFilter
    ProfileService --> ProfileWriter
    AllGroups ..|> GroupFilter
    AffordableGroups ..|> GroupFilter
    API --> GroupService
    API --> ProfileService
```

For readability, repository method labels omit their additional Go `error` result in the diagram. The exact signatures are in `service.go`.

## UML sequence diagram: joining a group

The sequence traces [http.go](http.go), [service.go](service.go), [store.go](store.go) and [domain.go](domain.go). One memory-store lock covers lookup, validation and mutation. The lock is released on every return through `defer`.

```mermaid
sequenceDiagram
    actor Student
    participant API as HTTP API
    participant Service as GroupService
    participant Store as MemoryStore
    participant Group
    Student->>API: POST /groups/group-2/join {profileId}
    API->>API: Decode one bounded JSON body
    API->>Service: JoinGroup(groupID, profileID)
    Service->>Store: JoinGroup(groupID, profileID)
    Store->>Store: Lock, find group and profile
    Store->>Group: addMember(profile)
    alt Already a member
        Group-->>Store: nil, no change
        Store-->>Service: existing GroupDetail
        Service-->>API: GroupDetail
        API-->>Student: 200, unchanged membership
    else Cost too high or capacity full
        Group-->>Store: domain error
        Store-->>Service: error
        Service-->>API: error
        API-->>Student: 422 or 409 JSON error
    else Eligible and a place remains
        Group->>Group: Append profile ID
        Group-->>Store: nil
        Store->>Store: Save changed group, prepare safe view
        Store-->>Service: GroupDetail
        Service-->>API: GroupDetail
        API-->>Student: 200, updated membership
    end
    Note over Store: Unlock before returning to caller
```

Unknown group/profile IDs fail during lookup with 404, before `addMember`. All branches return the same explicit JSON transport format. The prototype checks an illustrative numeric budget rule; it does not calculate a compatibility percentage or establish that a real lease can be signed.

## Run, input/output and verification

The full reproducible request sequence is in [README.md](README.md). From this folder:

```sh
go run . --demo
go run . --addr localhost:8080
go test ./...
```

`POST /profiles` and `POST /groups` return 201; list/details and successful or repeated joins return 200. Invalid data returns 400, missing records 404, full groups 409 and insufficient budgets 422. Example error output is:

```json
{"error":"group_full","message":"group is full"}
```

On 15 October 2026, the tests, CLI demo and live HTTP scenario passed. The concurrent test runs 30 eligible profiles against a capacity of 3 and checks exactly three successful memberships plus 27 full-group errors. Rejoining a successful profile leaves that count unchanged. Another test checks that returned member views omit the private budget. The focused capacity-limit test rejects capacity 21. These are measured test results for synthetic input; there are no throughput benchmarks or production-user claims.

## Oral questions and live changes

1. Explain the project without naming a pattern. People create profiles, find affordable roommate groups and join one.
2. Why split `http.go`, `service.go` and `store.go`? Transport, use-case coordination and storage change for different reasons.
3. Why does `AffordableGroups` implement `GroupFilter` without an `implements` keyword? Go interfaces are satisfied implicitly by method sets.
4. Does OCP prevent editing `main.go` to choose a new filter? No. The existing filtering algorithm remains unchanged; composition still chooses a collaborator.
5. Does LSP mean all filters must return the same groups? No. They preserve the inclusion-predicate contract while varying the rule.
6. What does the interface buy over passing `MemoryStore` directly? The service depends on the required operations and can accept another conforming implementation.
7. What prevents concurrent requests from taking the last place twice? One store lock covers both checking and appending membership.
8. Why is a duplicate join checked before capacity? Existing membership should remain idempotent even when the group has since become full.
9. Does private budget omission prove the backend is secure? No. It limits response data, while authentication and authorization remain future work.
10. What changes to use PostgreSQL? Implement the contracts with transactions, migrations and comparable error semantics; wire that store in the composition point.

Practice these small changes:

- Add `OpenPlaces` with `Matches(group GroupView) bool { return group.MemberCount < group.Capacity }`, then pass it to `ListGroups`.
- Add a `NamedGroups` filter using `strings.Contains` and explain why the selection loop stays unchanged.
- Add a maximum profile-name length at validated construction and explain its error response.
- Extend the repeated-join test to confirm membership remains unchanged after the group becomes full.

## Paper-exam outline

Practice writing this from memory: product purpose; one input/output example; the five SOLID names and one actual source role for each; a small class diagram showing the service, its interface and concrete store; a sequence diagram for joining; one limitation; one justified future extension. Use the compact source examples above rather than attempting to reproduce the whole server.

## Conclusion

Lab 1 provides one understandable Domi workflow and shows all five SOLID principles using Go's structs, methods and interfaces. The design keeps transport, storage and domain rules separate, makes listing rules replaceable, and protects the membership invariant in one place. Later labs can extend this same backend as requirements appear.

This is an in-memory learning prototype. PostgreSQL, authenticated identity, owner approval, external notifications and frontend integration are not implemented. The report and code were prepared with AI assistance; the student should review and explain them, and confirm the instructor's assessed-use requirements. The requested 7 October lab label does not claim that this implementation was committed or submitted on that date.
