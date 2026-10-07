# Domi structural patterns

This folder contains a small Go backend exercise for Domi roommate groups. It applies Adapter, Decorator, and Facade around the existing group and profile services while retaining the HTTP API and basic demo.

The course's Lab 2 page is [gated](https://else.fcim.utm.md/course/view.php?id=339), so this folder records the structural-pattern scope prepared on 15 October 2026. It does not claim that the current handout or rubric was confirmed.

## Run

Use a supported Go SDK; the module uses routing syntax available since Go 1.22.

```powershell
go test ./...
go run . --structural
go run . --patterns
go run . --demo
go run . -addr localhost:8080
```

The HTTP server keeps the existing routes: GET /groups, GET /groups/{id}, POST /groups, POST /groups/{id}/join, and POST /profiles. The in-memory store resets when the process exits.

The Lab 2 package also retains the three creational pattern examples from [Lab 1](../lab-01/report.md) through [patterns.go](patterns.go) and [patterns_demo.go](patterns_demo.go). The creational demonstration uses --patterns.

See [report.md](report.md) for the professor-facing problem, design diagrams, results, and limits. See [learn.md](learn.md) for the worked trace and oral questions.

![Lab 2 overview](assets/summary.png)
