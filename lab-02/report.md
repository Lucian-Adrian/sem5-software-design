# Lab 2: Structural patterns in a Domi roommate backend

Student: Lucian-Adrian Gavril, FAF-241. Language: Go. Report revision: 15 October 2026.

## Introduction

Domi's backend stores roommate profiles and groups. The current service accepts a group name, a maximum monthly cost per person, and a capacity. A legacy housing offer instead provides one total monthly cost and a target roommate count. This lab applies three structural patterns to connect the legacy input, decorate group writes with audit outcomes, and coordinate onboarding through the existing services.

The official [Lab 2 course page](https://else.fcim.utm.md/course/view.php?id=339) requires course access. The current handout and rubric were not confirmed for this preparation. This report describes the implemented scope and does not claim instructor approval or submission.

## Conditions

- Implementation language: Go 1.22, using the standard library.
- Persistence: process-local in-memory store implementing the existing reader and writer contracts.
- Group capacity: one to twenty members, as enforced by the existing domain.
- Audit output: injected writer; outcome records are best-effort and not durable storage.
- External systems: no database, authentication, or frontend is included.
- The current course brief remains unverified because the linked page is gated.

## Tasks

1. Adapt a legacy offer containing total monthly cost and roommate capacity into the current group service input.
2. Decorate group creation and joining with outcome-only audit records while preserving the wrapped result and error.
3. Provide a facade that validates known onboarding constraints before writes, then creates a profile and group and joins them.
4. Keep the existing HTTP API and basic demonstration available.
5. Retain the three creational pattern examples from Lab 1 in the integrated Lab 2 package. See [Lab 1 report](../lab-01/report.md) for the corresponding creational scope.

## Mechanism

### Adapter

LegacyRoommateOffer carries a whole monthly cost and a target roommate count. LegacyOfferAdapter converts it to GroupInput and calls GroupService. The per-person limit is calculated as total divided by capacity, plus one when a remainder exists. This ceiling calculation avoids the potentially overflowing intermediate addition used by the common total plus capacity minus one formula. Non-positive totals and capacities outside the domain's one to twenty range are rejected. GroupService still validates the name and final domain input.

MaxMonthlyCost represents a conservative whole-unit per-person amount. For a total of 1,501 and three roommates, it becomes 501. Because the current model stores one uniform affordability limit instead of individual rent shares, the sum of three such limits can exceed the original total by up to two units.

### Decorator

AuditedGroupWriter implements the same GroupWriter methods and forwards each call to its wrapped writer. It writes one outcome-only record after each mutation, for example operation=JoinGroup result=success. Records omit names, budgets, IDs, and raw errors. The decorator returns the wrapped result and error unchanged. Audit output failure does not replace a domain result. This is best-effort diagnostic logging, not durable audit storage.

### Facade

OnboardingFacade coordinates profile creation, group creation, and joining. It checks profile, group, capacity, and affordability constraints before the first write. It then calls the existing services in sequence. If a later call fails, the returned result includes views for completed steps together with the error. The workflow has no rollback because these service calls do not share a transaction.

## Source code

- domain.go defines validated profile and group models and the public views.
- service.go defines GroupReader, GroupWriter, ProfileWriter, and the existing services.
- store.go provides the in-memory implementation of those contracts.
- structural.go defines LegacyRoommateOffer, LegacyOfferAdapter, AuditedGroupWriter, and OnboardingFacade.
- structural_demo.go composes the patterns and prints a JSON result for the --structural command.
- main.go retains the HTTP server and --demo entry point, with --structural added independently.
- structural_test.go checks rounding, invalid adapter input, result and error preservation, private-data omission, pre-write validation, and partial failure reporting.
- The integrated folder also includes the three creational pattern implementations carried from Lab 1 and exposed through --patterns.

## Implementation excerpts

The adapter rounds up without adding the total to capacity:

```go
perPersonCost := offer.MonthlyTotal / offer.RoommateLimit
if offer.MonthlyTotal%offer.RoommateLimit != 0 {
    perPersonCost++
}
```

The decorator forwards the original outcome and writes its own diagnostic record:

```go
func (writer *AuditedGroupWriter) JoinGroup(groupID, profileID string) (GroupDetail, error) {
    group, err := writer.next.JoinGroup(groupID, profileID)
    writer.record("JoinGroup", err)
    return group, err
}
```

The facade validates before writes, then delegates each step to its existing service:

```go
result.Profile, err = facade.profiles.CreateProfile(input.Profile)
if err != nil {
    return result, err
}
result.Group, err = facade.groups.CreateGroup(input.Group)
if err != nil {
    return result, err
}
joined, err := facade.groups.JoinGroup(result.Group.ID, result.Profile.ID)
```

The complete functions and their error handling are in [structural.go](structural.go).

## Run and input/output

```powershell
go run . --structural
go run . --patterns
go run . --addr localhost:8080
```

The structural demo supplies `LegacyRoommateOffer{Name: "Central apartment", MonthlyTotal: 1501, RoommateLimit: 3}`. Its JSON output contains `adaptedGroup.maxMonthlyCost: 501`, an `auditLog` with operation outcomes, and `onboarding.joinedGroup.group.memberCount: 1`. The onboarding input is Alex with budget 600 and a group costing 550 with capacity 2. The HTTP routes inherited from Lab 0 still accept JSON profiles, groups, and join requests.

## UML

### Class diagram

~~~mermaid
classDiagram
    class LegacyRoommateOffer {
        +string Name
        +int MonthlyTotal
        +int RoommateLimit
    }
    class LegacyOfferAdapter {
        -GroupService groups
        +CreateGroup(offer) GroupView, error
    }
    class AuditedGroupWriter {
        -GroupWriter next
        -Writer output
        +CreateGroup(input) GroupView, error
        +JoinGroup(groupID, profileID) GroupDetail, error
    }
    class OnboardingFacade {
        -ProfileService profiles
        -GroupService groups
        +Onboard(input) OnboardingResult, error
    }
    class GroupService {
        +CreateGroup(input) GroupView, error
        +JoinGroup(groupID, profileID) GroupDetail, error
    }
    class ProfileService {
        +CreateProfile(input) ProfileView, error
    }
    class GroupWriter {
        <<interface>>
        +CreateGroup(input) GroupView, error
        +JoinGroup(groupID, profileID) GroupDetail, error
    }

    LegacyOfferAdapter --> GroupService : adapts legacy input
    AuditedGroupWriter ..|> GroupWriter : decorates
    GroupService --> GroupWriter : delegates writes
    OnboardingFacade --> ProfileService : creates profile
    OnboardingFacade --> GroupService : creates and joins group
~~~

### Sequence diagram for onboarding

~~~mermaid
sequenceDiagram
    actor Caller
    participant Facade as OnboardingFacade
    participant Profiles as ProfileService
    participant Groups as GroupService
    participant Writer as AuditedGroupWriter
    participant Store as MemoryStore
    participant Audit as Audit output

    Caller->>Facade: Onboard(profile, group)
    Facade->>Facade: validate profile, group, capacity, affordability
    Facade->>Profiles: CreateProfile(profile)
    Profiles->>Store: CreateProfile(profile)
    Store-->>Profiles: ProfileView
    Profiles-->>Facade: ProfileView
    Facade->>Groups: CreateGroup(group)
    Groups->>Writer: CreateGroup(group)
    Writer->>Store: CreateGroup(group)
    Store-->>Writer: GroupView
    Writer->>Audit: operation=CreateGroup result=success
    Writer-->>Groups: GroupView
    Groups-->>Facade: GroupView
    Facade->>Groups: JoinGroup(groupID, profileID)
    Groups->>Writer: JoinGroup(groupID, profileID)
    Writer->>Store: JoinGroup(groupID, profileID)
    Store-->>Writer: GroupDetail
    Writer->>Audit: operation=JoinGroup result=success
    Writer-->>Groups: GroupDetail
    Groups-->>Facade: GroupDetail
    Facade-->>Caller: profile, group, joined group
~~~

If group creation or joining fails after an earlier write, the facade returns the completed views and the error. It cannot undo the earlier write.

## Results

The local structural implementation covers all three requested patterns and keeps the previous HTTP and demo entry points. The adapter turns a legacy cost of 1,501 for three roommates into a per-person limit of 501. The decorator records operation outcomes without writing personal budgets and preserves the wrapped call's result and error. The facade rejects an unaffordable onboarding input before it writes, and its result identifies completed steps after a later failure.

On 15 October 2026, `go test ./...` passed after integrating the Lab 1 extensions. Both `go run . --structural` and `go run . --patterns` completed with the outputs described above. The creational examples are carried into this Lab 2 package from Lab 1 and remain independently demonstrable through --patterns.

## Conclusion

The implementation uses the current reader, writer, and service contracts to demonstrate Adapter, Decorator, and Facade in a small roommate backend. It also retains all three creational examples from Lab 1 in the integrated package. Verification is local to the Go module. The store is process-local, audit output is best-effort, and onboarding is not transactional.
