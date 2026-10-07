# Lab 2 learning notes

## The three boundaries

The patterns solve different problems in the same Domi flow:

| Pattern | Input and output contract | Change it isolates |
| --- | --- | --- |
| Adapter | Legacy total and roommate count become `GroupInput`. | Data shape and cost conversion. |
| Decorator | `GroupWriter` goes in and `GroupWriter` comes out. | Outcome recording around the same operations. |
| Facade | Profile and group input become one onboarding result. | The order of several service calls. |

In Go, `GroupWriter` is a method contract. A type satisfies it by implementing `CreateGroup` and `JoinGroup`; no declaration is needed on the concrete type. The compiler checks the method set, while a caller still relies on the implementation to preserve useful behavior. Here, `MemoryStore` implements the write operations, `GroupService` delegates to a `GroupWriter`, and `AuditedGroupWriter` implements the same interface while holding another writer.

The important difference is whether the pattern changes a contract. The adapter translates between two data shapes. The decorator preserves the writer shape and adds work around calls. The facade offers one larger operation made from existing services. These are related through composition, but they are not interchangeable.

## Adapter: convert before crossing the service boundary

The legacy feed supplies a total for the whole group. `GroupInput.MaxMonthlyCost` is an affordability bound for each person. The adapter must convert both the cost and the capacity while preserving the intended meaning.

`groupInputFromLegacyOffer` follows this pseudocode:

```text
reject a total that is zero or negative
reject a roommate limit outside 1..20
divide the total by the limit
if a remainder exists, increase the quotient by one
return GroupInput with the converted cost and unchanged name and capacity
```

The order prevents division by zero and rejects capacities the group domain cannot store. For `MonthlyTotal = 1501` and `RoommateLimit = 3`, integer division gives quotient 500 and remainder 1. The adapter therefore sets `MaxMonthlyCost` to 501 and `Capacity` to 3. The full converted value is:

```go
GroupInput{Name: "Central apartment", MaxMonthlyCost: 501, Capacity: 3}
```

The usual ceiling formula, `(total + capacity - 1) / capacity`, can overflow when `total` is close to the largest `int`. Quotient plus a remainder check avoids that intermediate sum. For example, `math.MaxInt / 2` rounds up to `math.MaxInt/2 + 1` without overflowing. The adapter's tests encode this boundary.

For `1000 / 3`, the converted limit is 334. Three people whose budgets each cover 334 could amount to 1,002, two units above the source total. Rounding up avoids understating each equal share, but the current group model has one common ceiling and no field for uneven or exact rent shares. If individual amounts matter, the domain needs a richer model rather than a different rounding expression.

Only the legacy input checks happen in the adapter. It checks total and capacity, then calls `GroupService.CreateGroup`. The existing domain validation trims and checks the name and verifies that the final cost and capacity are valid. A blank name therefore returns an error from the normal group creation path. Invalid total or capacity returns an error wrapping `ErrInvalidInput` before the service is called.

## Decorator: wrap the same writer contract

The service can receive the decorator because both provide these methods:

```go
type GroupWriter interface {
	CreateGroup(input GroupInput) (GroupView, error)
	JoinGroup(groupID, profileID string) (GroupDetail, error)
}
```

For a call to `JoinGroup`, `AuditedGroupWriter` executes this sequence:

```text
call the wrapped writer with the original group and profile IDs
choose success when the returned error is nil, otherwise choose error
write one operation/result line
return the exact value and error from the wrapped writer
```

For example, the structural demo's successful calls produce these three lines because it creates one adapted group, creates another group during onboarding, and then joins that group:

```text
operation=CreateGroup result=success
operation=CreateGroup result=success
operation=JoinGroup result=success
```

If the wrapped call fails, the decorator still writes an `error` outcome and returns the wrapped error. The record intentionally leaves out names, budgets, IDs, and error text. If the audit output itself fails, `record` ignores that write error. This keeps logging from replacing the domain result, but it also means the log is best effort and cannot prove that every record was stored. A nil output becomes `io.Discard`, which deliberately disables those records.

The decorator's key promise is about the wrapped call's result and error. It does not promise durable logging. The caller chooses the `io.Writer`; if calls may happen concurrently, that writer must support the expected concurrent use because `AuditedGroupWriter` adds no synchronization around it. The demo uses a `bytes.Buffer` sequentially.

Compare the decorator with putting `fmt.Fprintf` inside `MemoryStore`. The store should own records and domain operations. The decorator lets composition add an outcome record without making storage depend on a logging policy. It also differs from the adapter: both sides of the decorator use `GroupWriter`, but the adapter accepts a legacy offer and returns a `GroupService` result.

## Facade: sequence calls without promising a transaction

`OnboardingFacade` exposes `Onboard(OnboardingInput)`. It uses the existing domain constructors to validate the profile and group with a temporary ID, then checks affordability. These temporary values are never stored. If those checks pass, it calls the existing services in this order:

```text
create profile
create group
join profile to group
return the views from the completed calls
```

The demo passes this input:

```go
OnboardingInput{
	Profile: ProfileInput{Name: "Alex", MonthlyBudget: 600},
	Group: GroupInput{
		Name: "Quiet shared flat", MaxMonthlyCost: 550, Capacity: 2,
	},
}
```

The pre-checks succeed because both names are nonblank, the group cost and capacity are valid, and `600 >= 550`. Then the services produce these results:

1. `CreateProfile` returns `ProfileView{ID: "profile-1", Name: "Alex"}`.
2. `CreateGroup` returns a group named `Quiet shared flat`, with cost 550, capacity 2, and no members. It receives ID `group-2` because the adapter created `group-1` earlier in this same demo store.
3. `JoinGroup("group-2", "profile-1")` returns a `GroupDetail` with one member, Alex. The profile view contains the name and ID but not the personal budget.

The facade returns the profile and group views plus a `JoinedGroup` pointer that is not nil. Since the demo's `GroupService` was given `AuditedGroupWriter`, both group writes and the join also create outcome lines. `ProfileService.CreateProfile` does not pass through that decorator because the decorator implements only `GroupWriter`.

Validation before writes and transaction atomicity are separate guarantees. The facade prevents known input and affordability errors from leaving records behind. A later storage or join error can still happen after an earlier call has written. In that case, the result reports completed steps:

| Failure point | Returned result | State already written |
| --- | --- | --- |
| Profile creation | Empty result and error. | No completed step. |
| Group creation | Profile view and error. | Profile remains created. |
| Join | Profile and group views, nil `JoinedGroup`, and error. | Profile and group remain created. |

The facade does not remove those records. To make onboarding atomic, one storage operation would need to own all three changes within a transaction or equivalent atomic boundary. Compensation could attempt to delete earlier records, but then deletion can fail too; that is a separate design with its own outcomes. For a single short call path, direct service calls can be clearer. A facade earns its place when onboarding is a meaningful operation with a reusable entry point.

## Read the source path

The implementation can be followed in this order:

1. [`structural_demo.go`](structural_demo.go) composes the store, the audited writer, services, adapter, and facade.
2. [`structural.go`](structural.go) converts the legacy offer, records write outcomes, and sequences onboarding.
3. [`service.go`](service.go) defines `GroupWriter` and delegates the service calls.
4. [`store.go`](store.go) assigns IDs, persists records in memory, and returns views.
5. [`domain.go`](domain.go) validates inputs and defines the behavior and data returned to callers.

Run the demo with `go run . --structural`. The regular `--demo`, `--patterns`, and HTTP server paths remain available in the integrated package.

## Oral questions and answers

### Why does the adapter use quotient and remainder instead of adding capacity minus one before division?

The remainder tells whether the quotient needs to increase by one, and the calculation does not add capacity to the total. That avoids an overflow when a valid total is near `math.MaxInt`.

### What does rounding up guarantee, and what does it lose?

It does not understate the equal ceiling for the total. A remainder is spread across the common ceiling, so multiplying that ceiling by capacity may exceed the total. The current model cannot represent exact shares or unequal shares.

### Why is `LegacyOfferAdapter` an adapter rather than a helper that copies fields?

The incoming and outgoing contracts express cost differently, so the conversion must change meaning as well as names. A named adapter makes that boundary reusable and keeps it out of request handlers. If there were one conversion at one call site, a helper could be enough.

### How can `AuditedGroupWriter` stand in for another `GroupWriter`?

It has the same `CreateGroup` and `JoinGroup` methods, so Go's method set satisfies the interface. Each method forwards its arguments and returns the wrapped call's value and error. The extra record does not change the declared contract.

### What exactly does the decorator record?

It records the operation name and `success` or `error`. It omits personal and identifying values. It ignores output write errors, so the record is best effort diagnostics rather than a durable audit guarantee.

### Why does the structural demo use the decorator while the ordinary server does not?

`runStructuralDemo` builds a `GroupService` with `AuditedGroupWriter` to show the pattern. The normal server constructs the service with `MemoryStore` directly. The wrapper affects a path only when composition supplies it.

### What does the facade validate before the first write?

It applies the profile and group domain constructors, then checks that the profile budget covers the group cost. This catches invalid input and known affordability failures before record creation. It cannot guarantee that later storage operations succeed.

### What does the facade return if joining fails after both records were created?

It returns the profile and group views it already received, leaves `JoinedGroup` nil, and returns the join error. The records remain in the store because the facade has no rollback.

### Why is that sequence not atomic?

The facade calls three service operations one after another. No shared transaction encloses those writes. Atomic behavior requires a storage boundary that can commit or reject the complete onboarding operation as one unit.

### When is a facade useful, and when are direct calls simpler?

A facade is useful when profile creation, group creation, and joining form one named use case that several callers need to invoke consistently. If the sequence has one short caller and no separate policy, direct calls may be easier to understand.
