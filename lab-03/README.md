# Lab 3: Behavioral patterns in Domi

This self-contained checkpoint extends Lab 2 with **Strategy, Command, and Observer**. One `main.go` client selects a demonstration or serves the inherited HTTP API. No services are deployed.

From this folder, with Go 1.22 or newer:

```powershell
go test -count=1 ./...
go run . --behavioral
go run . --structural
go run . --patterns
go run . --demo
go run . --addr localhost:8080
```

Select only one demo flag at a time. The behavioral demonstration calls the application directly; it does not introduce new HTTP routes. Its store is new for each invocation.

- [Report](report.md): assignment, design, diagrams, demonstrated results and limitations.
- [Learning and oral defense](learn.md): worked trace, adjacent concepts, script, prompts and checking answers.
- [Behavioral implementation](behavioral.go), [demo](behavioral_demo.go), [tests](behavioral_test.go).
- [Verification record](evidence/verification.json): commands, exit codes and source hashes; the build log and built-executable demo JSON are in the same folder.
- `domain.go`, `service.go`, `store.go`, `http.go`: inherited domain, contracts, persistence and transport responsibilities. The creational and structural files retain their earlier responsibilities.

The [ELSE brief](https://else.fcim.utm.md/mod/assign/view.php?id=27829) requires three behavioral patterns and knowledge of the remaining patterns. This lab uses Strategy, Command and Observer. Current deadline and Go acceptance remain unconfirmed. Prepared work and passing tests do not establish submission, a completed defense or independent understanding.

The implementation, demo and tests separate reusable behavior, sample setup and behavior checks. The demo prints JSON; the tests check behavior without constructing presentation output.

## Diagram sources and exports

The three class diagrams and successful-call sequence have native `.isx`, portable `.svg` and `.png` versions in `assets/`. The report and learning guide embed SVGs. The sources passed the local Isomorph parser and semantic analyzer, and exports were visually inspected.

To regenerate with this machine's existing Isomorph checkout and Bun, run from this lab folder:

```powershell
D:/apps/bun/bun.exe assets/render-diagrams.ts
python -B assets/portable-svg.py
```

The optional diagram export uses `D:/dev/isomorph` by default (`ISOMORPH_ROOT` can select another existing checkout). The PNG step requires PyMuPDF. Neither is required to build or run the Go lab. The portability step resolves SVG dashes and UML markers into geometry so image importers preserve their meaning.
