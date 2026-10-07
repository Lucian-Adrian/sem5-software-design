# Domi backend: TMPS coursework in Go

Lucian-Adrian Gavril, FAF-241.

Domi helps people find roommates to share rent. This project builds a Go backend around profiles, affordable groups, and membership rules. Each lab extends that same domain with a design idea that you can trace through working code.

| Lab | Topic | Read |
| --- | --- | --- |
| [Lab 0](lab-00/README.md) | All five SOLID principles | [Report and UML](lab-00/report.md) |
| [Lab 1](lab-01/README.md) | Three creational patterns | [Report and UML](lab-01/report.md) |
| [Lab 2](lab-02/README.md) | Three structural patterns | [Report and UML](lab-02/report.md) |
| [Lab 3](lab-03/README.md) | Strategy, Command, Observer | [Report and UML](lab-03/report.md) |

Start with [Go explained from Python in Lab 0](lab-00/learn.md#start-here-if-you-know-python). Each lab has a report for submission and separate learning notes for worked examples, oral questions and practice exercises. The 52-concept catalog is optional reference after the worked example.

## Which source to present

| Lab | Start with | Run its example |
| --- | --- | --- |
| 0: SOLID | `domain.go`, then `service.go` | `go run . --demo` |
| 1: creational | `patterns.go` | `go run . --patterns` |
| 2: structural | `structural.go` | `go run . --structural` |
| 3: behavioral | `behavioral.go` | `go run . --behavioral` |

Run each command inside its lab folder. The implementation file contains the rules and pattern objects. A demo file connects them and prints an example. A test file checks assertions, including failure cases. Go compiles ordinary source files together; `_demo.go` is our descriptive name, while `_test.go` is recognized by Go and excluded from normal builds. Keeping them separate means a presentation can follow the implementation without reading printing code and assertions at the same time.

The 17 October Explain and Defend revision adds source-based changed-condition cases to the Labs 0–2 guides: simultaneous joins in Lab 0, budget validation and completed writes in Lab 1, and facade validation versus a completed legacy import in Lab 2. Try the prompts before opening their checking answers. These are preparation materials; independent understanding is assessed through your explanation and a fresh case.

Lab 1 presents Builder, Abstract Factory, and Prototype with `--patterns`. Its optional `--extra-patterns` mode demonstrates Singleton, Factory Method, and Object Pooling for the remaining theory list. Object Pooling is optional in the ELSE brief. Lab 2 presents Adapter, Decorator, and Facade with `--structural`; its learning notes explain the other four structural patterns and a possible extension.

## Open and run locally

The project folder on this Windows machine is `D:\edu\faf\sem5\software-design`. Open that folder in your editor, or open PowerShell and run Lab 0:

```powershell
Set-Location D:\edu\faf\sem5\software-design\lab-00
go run . --demo
```

The demo prints JSON and exits. Running without a demo flag starts the HTTP server; stop it with `Ctrl+C`. This is a backend, so the server does not open a graphical interface.

Each lab also has a local executable, built from its source. For example:

```powershell
.\domi-lab-00.exe --demo
.\domi-lab-00.exe --help
```

Executables are local build outputs and are not committed to Git. Rebuild one after changing its source:

```powershell
go build -o domi-lab-00.exe .
```

## Get the project on your Devbook

With Git and a supported Go SDK installed, clone the repository once:

```sh
git clone https://github.com/Lucian-Adrian/sem5-software-design.git
cd sem5-software-design
```

If the repository is already cloned, run this inside that checkout to download the latest commits:

```sh
git pull --ff-only origin main
```

The repository includes the Go source, reports, learning notes, and diagram files. Local Windows executables are not included. `go run .` builds for the operating system on your Devbook, so use the commands below from the repository root.

## Run the checkpoints

Each lab is a self-contained Go module with a frozen copy of the preceding backend and its new features. This intentionally keeps earlier presentations runnable; it is one evolving product rather than four deployed services. Lab 3 is the latest source checkpoint. A production codebase would share the implementation and use version tags instead of copying source.

Use a supported Go SDK. The modules require the routing syntax available since Go 1.22. All dependencies are in the standard library.

```powershell
cd lab-00
go run . --demo
cd ../lab-01
go run . --patterns
cd ../lab-02
go run . --structural
cd ../lab-03
go run . --behavioral
```

From any lab directory, `go run . --addr localhost:8080` starts its local HTTP API. Run one server at a time or select different ports. These are in-memory coursework prototypes, separate from the [existing Domi frontend](https://trydomi.vercel.app). Authentication, durable database storage, member approval, and frontend integration remain future work.

## UML diagrams in Isomorph

Each lab keeps editable Isomorph sources and exports together in `assets`. Lab 0 includes a separate class diagram for each SOLID principle. Labs 1 and 2 include one for every implemented pattern, as well as their workflow sequence diagrams:

- `sequence.isx`: native Isomorph document.
- `sequence.svg`: vector export displayed in the report.
- `sequence.png`: image export for viewing or sharing.
- `srp.isx`, `builder.isx`, and other named triples: individual class diagrams, each with SVG and PNG exports.

Open a `.isx` file in [Isomorph](https://iso.meandivision.foo/app/) to edit the participants and messages. These exports use the parser, semantic checks, and renderer from the local Isomorph project. All four reports embed native Isomorph class diagrams beside the corresponding principle or pattern explanations.

## Course numbering and dates

**17 October readiness update:** Fresh tests and demonstrations passed for Labs 0–3. [Lab 3](lab-03/README.md) extends Lab 2 with Strategy, Command and Observer; its report, learning guide, built executable and four editable Isomorph diagrams are prepared. Read the [consolidated readiness matrix](../semester-info/defense-preparation/README.md) for requirements, evidence and oral rehearsal. These checks do not establish independent understanding or a completed defense.

The [ELSE course](https://else.fcim.utm.md/course/view.php?id=339) labels SOLID Lab 0, creational patterns Lab 1, structural patterns Lab 2, and behavioral patterns Lab 3. The [Lab 1 brief](https://else.fcim.utm.md/mod/assign/view.php?id=27825) requires three creational patterns and knowledge of the remaining listed patterns; Object Pooling is marked optional. The [Lab 2 brief](https://else.fcim.utm.md/mod/assign/view.php?id=27827) requires three structural patterns and knowledge of the rest. Its list is Adapter, Bridge, Composite, Decorator, Facade, Flyweight, and Proxy. Historical ELSE dates do not establish the 2026 deadlines.

The reports record their implementation revision date. The logged-in assignment pages expose submission controls, but none of the four labs has been submitted through ELSE.

The earlier C# task-board material was preserved locally in `D:/ops/uni/course-archive/software-design/csharp-study-project-2026-10-15` and remains recoverable through Git history. The active project is Domi in Go.

## Assessment preparation

**17 October access update:** The signed-in account is now enrolled. Structural Lab 2 and all four assignment briefs are accessible. Labs 0–3 show no submission and no grade; the gradebook supplies no completed-defense evidence. The 2025 due dates are historical, and Go acceptance/current quiz weighting remain unconfirmed. The seven-quiz snapshot below remains dated 15 October.

On 15 October 2026, the [TMPS course page](https://else.fcim.utm.md/course/view.php?id=339) was accessible, but the account was not enrolled. Seven quiz entries were read without attempting them. All require enrollment, allow one attempt, and show opening and closing dates in 2024. The first four also require a password.

| Quiz | Format | Time | Pass grade shown |
| --- | --- | ---: | ---: |
| [27823 Creational theory](https://else.fcim.utm.md/mod/quiz/view.php?id=27823) | Theory | 10 min | 5/10 |
| [27824 Creational blank UML](https://else.fcim.utm.md/mod/quiz/view.php?id=27824) | Blank UML | 5 min | 5/10 |
| [27826 Structural theory](https://else.fcim.utm.md/mod/quiz/view.php?id=27826) | Theory | 10 min | 5/10 |
| [30996 Structural blank UML](https://else.fcim.utm.md/mod/quiz/view.php?id=30996) | Blank UML | 8 min | 5/10 |
| [27828 Behavioral theory](https://else.fcim.utm.md/mod/quiz/view.php?id=27828) | Theory | 17 min | 5/10 |
| [30998 Behavioral blank UML](https://else.fcim.utm.md/mod/quiz/view.php?id=30998) | Blank UML | 10 min | 5/10 |
| [31000 General blank UML](https://else.fcim.utm.md/mod/quiz/view.php?id=31000) | Blank UML | 20 min | 5/11 |

These dates and quiz settings are historical preparation evidence. They do not confirm current lab allocation, the midterm-to-lab mapping, or current midterm settings. The visible briefs do not confirm Go acceptance for the current project. The local Go implementation and report are prepared work; they do not establish owner-completed work, submission, instructor acceptance, or a grade.
