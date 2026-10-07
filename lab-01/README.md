# Lab 1: Creational patterns in the Domi backend

Domi helps people form groups to rent a home together. This checkpoint adds three creational patterns to the profile and group backend copied from Lab 0: Builder, Prototype and Abstract Factory.

Implementation and report revision: **15 October 2026**.

## Read and run

Read the professor-facing [report](report.md) for the condition, design, exact source examples, UML and observed results. Use [learn.md](learn.md) for oral questions and small live-change exercises.

Use a supported Go SDK. The module uses routing syntax available since Go 1.22 and the standard library only.

```powershell
go test ./...
go run . --patterns
go run . --demo
go run . --addr localhost:8080
```

`--patterns` prints JSON for a Builder-created group, two Abstract Factory onboarding families, and a Prototype copy. The budget and comfort families each create a group, a suggested profile and a successful service-level join. The prototype copies the budget group's settings, then saves a fresh group with a new ID and zero members.

## HTTP example

While the server runs, create a profile and group, then join them:

```powershell
$api = 'http://localhost:8080'
$profile = Invoke-RestMethod -Method Post -Uri "$api/profiles" -ContentType 'application/json' -Body '{"name":"Ana","monthlyBudget":550}'
$group = Invoke-RestMethod -Method Post -Uri "$api/groups" -ContentType 'application/json' -Body '{"name":"Shared flat","maxMonthlyCost":500,"capacity":2}'
Invoke-RestMethod -Uri "$api/groups?budget=550"
$joinBody = @{ profileId = $profile.id } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$api/groups/$($group.id)/join" -ContentType 'application/json' -Body $joinBody
Invoke-RestMethod -Uri "$api/groups/$($group.id)"
```

The HTTP contract remains the Lab 0 contract: `POST /profiles`, `POST /groups`, `GET /groups`, `GET /groups/{id}` and `POST /groups/{id}/join`. Data is held in memory and disappears when the process stops.

## Files

| File | Responsibility |
|---|---|
| [domain.go](domain.go) | Domain validation, membership rules and safe response views |
| [patterns.go](patterns.go) | Builder, Prototype and Abstract Factory types |
| [patterns_demo.go](patterns_demo.go) | JSON demo through the existing services |
| [main.go](main.go) | HTTP server, baseline demo and `--patterns` option |
| [patterns_test.go](patterns_test.go) | Pattern validation, clone isolation and family consistency checks |
| [service_test.go](service_test.go) | Existing service and HTTP behavior checks |

Authentication, persistent storage and frontend integration are outside this checkpoint.

![Lab 1 overview](assets/summary.png)
