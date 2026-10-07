# Lab 2 learning notes

## Go basics for a Python reader

The patterns use ordinary Go building blocks. These comparisons help translate the syntax without treating Go as Python with different punctuation.

| Python idea | Go code in this lab | What to notice |
| --- | --- | --- |
| A dictionary or data class holding fields | A `struct`, such as `GroupInput` | Declared field types let the compiler catch many mismatched values; domain rules still need validation. |
| An instance method using `self` | A method with a receiver, such as `(writer *AuditedGroupWriter)` | `writer` plays the role of `self`; `*` means the method receives a pointer to that writer value. |
| A protocol or base class | An `interface`, such as `GroupWriter` | Go checks whether a type has the required methods. It does not need to name or inherit from the interface. |
| A wrapper object holding another object | A field such as `next GroupWriter` | The decorator forwards a call to `next`, then adds its outcome record. |
| Exceptions raised by a call | A returned `error` value | Go code checks `if err != nil` and returns or handles the error explicitly. |
| `None` for a result that may be absent | A pointer that may be `nil` | `JoinedGroup` is nil until joining succeeds; `omitempty` then leaves it out of JSON. |

For example, a Python program might use `try` and `except` around a writer call. Go returns the result and possible error together, so the caller handles both in sequence:

```go
group, err := writer.next.CreateGroup(input)
if err != nil {
	return GroupView{}, err
}
return group, nil
```

The `:=` declares a local variable and lets Go infer its type from the returned value. The following `if` is an explicit error check; it is part of the function's normal control flow, not an exceptional jump. The same pattern appears in the adapter and facade.

## Match the lab to ELSE

The [course](https://else.fcim.utm.md/course/view.php?id=339) calls the structural assignment Lab 2. [Its brief](https://else.fcim.utm.md/mod/assign/view.php?id=27827) requires three structural implementations and knowledge of the remaining patterns. Our three are Adapter, Decorator, and Facade. Lab 1 is the creational assignment. Keeping these names separate matters when choosing which report and command to present.

## The three boundaries

The patterns solve different problems in the same Domi flow:

| Pattern | Input and output contract | Change it isolates |
| --- | --- | --- |
| Adapter | Legacy total and roommate count become `GroupInput`. | Data shape and cost conversion. |
| Decorator | `GroupWriter` goes in and `GroupWriter` comes out. | Outcome recording around the same operations. |
| Facade | Profile and group input become one onboarding result. | The order of several service calls. |

In Go, `GroupWriter` is a method contract. A type satisfies it by implementing `CreateGroup` and `JoinGroup`; no declaration is needed on the concrete type. The compiler checks the method set, while a caller still relies on the implementation to preserve useful behavior. Here, `MemoryStore` implements the write operations, `GroupService` delegates to a `GroupWriter`, and `AuditedGroupWriter` implements the same interface while holding another writer.

The important difference is whether the pattern changes a contract. The adapter translates between two data shapes. The decorator preserves the writer shape and adds work around calls. The facade offers one larger operation made from existing services. These are related through composition, but they are not interchangeable.

| Pattern | Strong point in Domi | Weak point in this implementation | Find it |
| --- | --- | --- | --- |
| Adapter | Integrates the legacy total-cost input without changing the core service's input format. | One whole-number threshold cannot express exact individual shares. | `LegacyOfferAdapter.CreateGroup`, `groupInputFromLegacyOffer`, [structural.go](structural.go). |
| Decorator | Adds outcome recording by changing composition rather than editing storage. | Logs can be lost; it adds another layer and does not affect ordinary HTTP composition. | `AuditedGroupWriter.CreateGroup`, `JoinGroup`, `record`, [structural.go](structural.go). |
| Facade | Gives onboarding a single entry point and one explicit sequence. | Completed writes remain after a later failure; unrelated use cases should not accumulate in it. | `OnboardingFacade.Onboard`, [structural.go](structural.go). |

The composition is in [structural_demo.go](structural_demo.go). Read it as `adapter → group service → audited writer → store` for importing a legacy group. For onboarding, read `facade → profile service → store`, followed by `facade → group service → audited writer → store` for group creation and joining. Arrows represent ordinary calls within one process, not separate servers.

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

```text
Keep the legacy offer's name.
Use the calculated per-person ceiling of 501 and capacity 3.
```

```go
GroupInput{Name: "Central apartment", MaxMonthlyCost: 501, Capacity: 3}
```

The usual ceiling formula, `(total + capacity - 1) / capacity`, can overflow when `total` is close to the largest `int`. Quotient plus a remainder check avoids that intermediate sum. For example, `math.MaxInt / 2` rounds up to `math.MaxInt/2 + 1` without overflowing. The adapter's tests encode this boundary.

For `1000 / 3`, the converted limit is 334. Three people whose budgets each cover 334 could amount to 1,002, two units above the source total. Rounding up avoids understating each equal share, but the current group model has one common ceiling and no field for uneven or exact rent shares. If individual amounts matter, the domain needs a richer model rather than a different rounding expression.

Only the legacy input checks happen in the adapter. It checks total and capacity, then calls `GroupService.CreateGroup`. The existing domain validation trims and checks the name and verifies that the final cost and capacity are valid. A blank name therefore returns an error from the normal group creation path. Invalid total or capacity returns an error wrapping `ErrInvalidInput` before the service is called.

## Decorator: wrap the same writer contract

The service can receive the decorator because both provide these methods:

```text
A writer must provide group creation and membership joining.
Both methods return their result and a possible error.
```

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

```text
Supply Alex with budget 600.
Supply a new group with cost 550 and capacity 2.
```

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

## The remaining structural patterns in ELSE

These four patterns are part of the theory list. They are possible extensions, not implemented features in this lab. Together with our existing three, they make the seven-item structural list in the brief.

| Pattern | Basic mechanism | Possible Domi use | Strong point | Weak point |
| --- | --- | --- | --- | --- |
| Bridge | An abstraction holds an implementation behind an interface so two dimensions can vary independently. | Different notification types could use email or SMS senders without creating a subclass for every combination. | Avoids a growing matrix of combinations. | Adds interfaces and forwarding that are unnecessary when only one dimension varies. |
| Composite | Leaves and containers implement the same interface; a container applies an operation to its children. | Treat a single roommate group and a collection of groups uniformly when calculating a listing summary. | Makes tree-shaped data recursive and uniform. | Rules meaningful only for containers or only for leaves may not fit one clean contract. |
| Flyweight | Many objects share immutable repeated state, while each retains its own contextual state. | Listings could refer to shared immutable amenity descriptions instead of copying them into every record. | Can reduce memory when repeated state is substantial. | Shared state must remain immutable; lookup and separation can cost more than they save for small data. |
| Proxy | An object controls access to another object through the same interface. | A group writer could reject calls after a configured write allowance is exhausted, then forward permitted calls to the real writer. | Introduces one controlled access boundary. | Hidden state or rejection behavior can surprise callers; limits need a precise lifetime and concurrent-access policy. |

Bridge is not simply another name for any interface field. Its useful distinction is independent variation on both sides. Flyweight is not the same as Object Pooling: flyweight users share immutable state concurrently, while a pool lends a mutable resource to one owner who returns it for reuse. A proxy and decorator can have the same wrapping shape; their primary purposes differ. Our decorator adds a record after a call. A proxy decides whether or how access to the wrapped object occurs.

### How another pattern could enter our actual logic

For a small Proxy extension, keep `GroupWriter` as the contract. A proposed `LimitedGroupWriter` would hold `next GroupWriter` and a remaining allowance. Both `CreateGroup` and `JoinGroup` would check that allowance before forwarding. Define whether the allowance counts attempts or successful operations; for this example it counts successful operations. No such limit currently exists.

```text
Receive a group mutation.
If no allowance remains, return a specific limit error without calling next.
Otherwise call next with the original arguments.
If it succeeds, decrease the remaining allowance by one.
Return the underlying result and error.
```

To integrate it, construct the proxy around the existing store at startup and supply it as the service's writer. The domain rules still execute in the store for permitted calls. A limit error would need an explicit HTTP mapping if the proxy were used in server mode; the current generic mapping would otherwise return 500. With overlapping requests, the allowance check, underlying call, and counter update would need a defined synchronization policy. This is why the change includes behavior and error decisions, not just a new type name.

The proposed UML would have the proxy realize `GroupWriter`, hold another `GroupWriter`, and receive calls from `GroupService`. The actual decorator already demonstrates this shape in the report; a proxy diagram would add a distinct type and a guard before its forwarded call. Useful focused checks would verify that rejected calls never reach the store, permitted calls preserve the underlying result, failed writes do not consume a success-based allowance, and the configured limit cannot be exceeded.

### Reading our implemented UML

In the report's class diagram, a dashed realization arrow means the type provides the methods required by an interface. It does not mean inheritance. The `AuditedGroupWriter → GroupWriter` dependency labeled `next` represents its wrapped collaborator; `MemoryStore` is the concrete collaborator supplied in the demo. The adapter's dependency on `GroupService` shows where translated values go. The facade's two service dependencies show which operations it coordinates.

In the Isomorph sequence diagram, time runs downward. A call arrow moves execution into another participant; its return brings a result back. The diagram focuses on the successful onboarding call, so it does not show the earlier legacy import or every failure branch. If group creation failed, the profile creation above it would remain completed, and the join arrows below it would never execute. That changed path is the reason the facade reports partial results.

## Oral prompts

Answer from the source before reading the checking answers.

1. Why does the adapter use quotient and remainder instead of adding capacity minus one before division?
2. What does rounding the per-person cost up guarantee, and what does it leave out?
3. Why is `LegacyOfferAdapter` an adapter rather than a helper that copies fields?
4. How can `AuditedGroupWriter` stand in for another `GroupWriter`?
5. What does the decorator record, and what does it fail to guarantee?
6. Why does the structural demo use the decorator while the ordinary server does not?
7. What does the facade validate before its first write?
8. What result and stored state remain if joining fails after the profile and group have been created?
9. Why is the facade's three-call sequence not atomic?
10. When is a facade useful, and when are direct service calls simpler?
11. Changed condition: change only Alex's budget from 600 to 500, leaving the onboarding group cost at 550 and the legacy offer unchanged. Where does onboarding stop, which write has already completed, and does the demo print its JSON result or audit log?

## Checking answers

1. The remainder shows whether the quotient must increase by one, without adding the capacity to the total. This avoids overflow when a valid total is near `math.MaxInt`.
2. Rounding up avoids understating a common per-person ceiling. It can make the sum of those ceilings exceed the original total, and the current model cannot express exact or uneven shares.
3. The legacy and current contracts represent cost differently, so conversion changes meaning as well as field names. A named adapter keeps that boundary reusable and out of request handlers. A one-off conversion could be a helper.
4. It provides the same `CreateGroup` and `JoinGroup` method signatures. Go's method set therefore satisfies `GroupWriter`; each method forwards its arguments and returns the wrapped call's value and error.
5. It records the operation name and `success` or `error`, without personal identifiers or raw error text. It ignores output write errors, so the record is best-effort diagnostics rather than a durable audit guarantee.
6. `runStructuralDemo` supplies `AuditedGroupWriter` when it constructs `GroupService`. The ordinary server supplies `MemoryStore` directly. The wrapper runs only on paths where composition passes it in.
7. It runs the profile and group domain constructors, then checks that the budget covers the group cost. This rejects known invalid input before writing records, but cannot guarantee later calls succeed.
8. It returns the profile and group views already received, leaves `JoinedGroup` nil, and returns the join error. Both records remain because the facade has no rollback.
9. The facade calls three separate service operations without one shared transaction. Atomic onboarding requires a storage boundary that commits or rejects the complete operation as one unit.
10. A facade helps when onboarding is a named operation that several callers need to perform consistently. A short sequence used once without its own policy may be clearer as direct service calls.
11. The legacy adapter has already created group-1 and buffered its successful `CreateGroup` record. The facade validates the onboarding inputs with temporary values and finds that 500 is below 550, so it returns `ErrBudgetTooLow` before creating Alex or the onboarding group. `runStructuralDemo` returns before it encodes the result, so it prints neither the JSON result nor the local audit buffer.

## Reconstruct the boundaries from a blank file

Use a disposable copy of this lab. Keep the original intact. Close `structural.go`, its tests and this guide after reading the task. Start the copy's `structural.go` blank; use `service.go`, `domain.go` and `store.go` only to inspect dependency contracts. Write your first design before opening the reference implementation.

Rebuild one structural flow with these requirements:

1. Define `LegacyRoommateOffer`, `LegacyOfferAdapter` and `NewLegacyOfferAdapter`. Implement `groupInputFromLegacyOffer` and `LegacyOfferAdapter.CreateGroup`: a positive whole-group total and capacity in `1..maxGroupCapacity` become a per-person ceiling, then follow normal group creation. Handle the largest valid `int` without overflow.
2. Define `AuditedGroupWriter` and `NewAuditedGroupWriter`. Its `CreateGroup` and `JoinGroup` must satisfy `GroupWriter`, forward each call once, attempt one operation/outcome record and preserve both returned values, including when output fails. Decide what nil output means.
3. Define `OnboardingInput`, `OnboardingResult`, `OnboardingFacade` and `NewOnboardingFacade`. Implement `Onboard`: reject known invalid input before writes; create a profile, create a group, join them; return completed views when a later step fails. Preserve the nil `JoinedGroup` distinction.
4. Compose the objects around one `MemoryStore` in the disposable copy. Trace an imported offer with total 1501 and capacity 3, then Alex's onboarding with budget 600, cost 550 and capacity 2. Explain which operations pass through the decorator and why its `next` field needs the `GroupWriter` contract while the adapter holds a `*GroupService`.
5. Defend the design before comparing code: which mismatch earns the adapter, which optional behavior earns the decorator, and which reusable use case earns the facade? Give one smaller situation where a direct conversion or service call would suffice.

### Check after the attempt

Compare behavior and dependency direction, rather than exact spelling, with [the adapter](structural.go#L30), [decorator](structural.go#L84), [facade](structural.go#L142) and [composition](structural_demo.go#L22). `GroupWriter` permits a store or wrapper with the same methods; the adapter uses the existing service entry point after translation. A different dependency can be valid if you explain its contract and scope.

In the disposable copy, run `gofmt -w structural.go` and `go test ./...`. The existing tests check conversion, exact result/error forwarding, validation before writes and partial failures. A successful run supports those cases; it does not demonstrate that you can reconstruct or defend the design. Note which steps you completed independently, which needed a reference and which remain unresolved. No attempt has been assessed here.

## Changed requirements to reason through

Answer these before opening the explanations. Each changes a different boundary; the proposed extensions are not implemented features.

1. **Changed input meaning.** A second partner supplies `MonthlyTotal = 600` and `RoommateLimit = 3`, but its documentation says 600 is already the price per roommate. The original feed still supplies whole-group totals. What does `LegacyOfferAdapter.CreateGroup` currently store for the second feed, why can validation accept the wrong result, and how would you support both contracts?
2. **Required audit durability.** A successful `JoinGroup` updates membership, but the audit output behaves like `failingAuditWriter`. The new rule requires the membership change and a durable audit event to be stored together, or neither to be stored. Trace today's result. Would returning the output error from `record` meet the new rule? Put the required guarantee at a concrete boundary.
3. **Retry after a partial write.** Start with a fresh store and no legacy import. Reproduce the test's group-creation failure, which occurs before a group is stored. Then reuse that store with `NewGroupService(store, store)` and retry the same valid `OnboardingInput` through a new facade. Predict the profile IDs, group ID, joined profile and remaining records. What additional contract would let repeated calls represent one logical onboarding?

### Reasoned answers

1. The current calculation stores `MaxMonthlyCost = 200` and `Capacity = 3`. Those values satisfy domain ranges, so validation cannot detect that the external price means something different. Give the second feed an explicitly different input type or version whose field states the per-person meaning. Convert its shape without dividing, then call `GroupService.CreateGroup`; retain the original adapter for total-based offers. If a caller already supplies valid `GroupInput`, it can use the service directly. The adapter is useful at a real contract mismatch, and a shared field name does not establish shared meaning. See [conversion](structural.go#L40) and [service delegation](service.go#L94).

2. The wrapped join returns its successful `GroupDetail` and nil error after updating membership. `record` attempts output and discards its error, so the decorator returns that same success even though no audit record was saved. Returning a logging error afterward would report failure while the membership remains changed; a retry could then repeat a committed operation. Writing a success record before the join also cannot make the two effects atomic. A redesigned storage operation must commit the membership and audit event within one transaction or equivalent atomic boundary. For an external sink, storing a pending audit event in that transaction can make later delivery retryable; it does not prove the sink has received it. This changes the storage guarantee beyond today's best-effort decorator. See [record](structural.go#L100), [membership write](store.go#L92) and [the failing output test](structural_test.go#L103).

3. The first call leaves `profile-1`, no group and a result containing only that profile view. The retry calls `CreateProfile` again, creating `profile-2`, then creates `group-1` and joins `profile-2`. Both profiles remain, and `profile-1` is unjoined. The facade has no request identity or resume operation; the same input is a fresh create request. If retries must denote one operation, introduce an idempotency key with stored progress/result and coordinate its recording with the writes, including concurrent calls. A resumable API could instead accept the completed IDs with an explicit contract. Comparing names is insufficient because different people can share a name. A transaction can remove partial writes but still needs request identity to prevent duplication after a committed response is lost. See [failure fixture](structural_test.go#L157), [facade sequence](structural.go#L159) and [profile ID allocation](store.go#L46).
