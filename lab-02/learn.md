# Lab 2 learning notes

## Worked trace

The legacy housing feed gives a total monthly cost of 1,501 and a roommate limit of three.

1. Divide 1,501 by 3. The quotient is 500 and the remainder is 1.
2. Since there is a remainder, add one to the quotient. The adapter uses 501 as the per-person maximum.
3. The adapter creates GroupInput with capacity 3 and sends it through GroupService. The service and domain constructors remain responsible for validating the group name and domain constraints.
4. For onboarding, the facade checks that a 600 monthly budget covers a group limit of 550. It performs this check before creating either record.
5. The facade creates the profile, creates the group through the decorated writer, and joins the profile. The decorator writes an outcome record after each group mutation.
6. If the group is created but the join fails, the facade returns the profile and group views together with the error. It does not remove either created record.

## Run and observe

~~~powershell
go test ./...
go run . --structural
go run . --patterns
go run . --demo
~~~

The structural command prints JSON containing the adapted group, the audit log, and the onboarding result. The audit lines include operation names and outcomes. They do not include names, budgets, IDs, or error text.

## Oral questions

1. Why does the adapter use quotient and remainder instead of adding capacity minus one before division?
2. Why is rounding the per-person limit up conservative? What information is lost by storing one group-wide amount?
3. Which contract does the decorator preserve? Why should an audit output failure not replace the mutation result?
4. Which fields are deliberately absent from the audit line? What durability does this writer provide?
5. Which onboarding inputs can be checked before writing? What state can remain after a later service call fails?
6. What storage contract would be needed to make the three-step onboarding atomic?

## Limits to explain

- The cost conversion assumes the total should be split across the full roommate limit and records the ceiling as one common affordability limit.
- The audit output is best-effort logging. It is not a durable audit trail.
- The facade coordinates calls but has no transaction or rollback.
- The in-memory store loses data when the process exits.
- The course page is gated. This exercise records prepared work, not confirmed rubric compliance or instructor acceptance.
