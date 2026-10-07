# Domi backend: TMPS coursework in Go

Lucian-Adrian Gavril, FAF-241.

Domi helps people find roommates to share rent. This project builds a Go backend around profiles, affordable groups, and membership rules. Each lab extends that same domain with a design idea that you can trace through working code.

| Lab                       | Topic                     | Read                               |
| ------------------------- | ------------------------- | ---------------------------------- |
| [Lab 0](lab-00/README.md) | All five SOLID principles | [Report and UML](lab-00/report.md) |
| [Lab 1](lab-01/README.md) | Three creational patterns | [Report and UML](lab-01/report.md) |
| [Lab 2](lab-02/README.md) | Three structural patterns | [Report and UML](lab-02/report.md) |

Start with the four OOP pillars and SOLID in the [52-concept learning guide](lab-00/learn.md). Each lab has a report for submission and separate learning notes for worked examples, oral questions, and practice exercises.

## Run the checkpoints

Each lab is a self-contained Go module with a frozen copy of the preceding backend and its new features. This intentionally keeps earlier presentations runnable; it is one evolving product rather than three deployed services. Lab 2 is the latest source checkpoint. A production codebase would share the implementation and use version tags instead of copying source.

Use a supported Go SDK. The modules require the routing syntax available since Go 1.22. All dependencies are in the standard library.

```powershell
cd lab-00
go run . --demo
cd ../lab-01
go run . --patterns
cd ../lab-02
go run . --structural
```

From any lab directory, `go run . --addr localhost:8080` starts its local HTTP API. Run one server at a time or select different ports. These are in-memory coursework prototypes, separate from the [existing Domi frontend](https://trydomi.vercel.app). Authentication, durable database storage, member approval, and frontend integration remain future work.

## Course numbering and dates

The [ELSE course](https://else.fcim.utm.md/course/view.php?id=339) labels SOLID Lab 0, creational patterns Lab 1, structural patterns Lab 2, and behavioral patterns Lab 3. The [visible Lab 1 brief](https://else.fcim.utm.md/mod/assign/view.php?id=27825) requires three creational patterns. The Lab 2 detailed brief is group-restricted, so its three-pattern scope is prepared coursework rather than a confirmed current rubric. Historical ELSE dates do not establish the 2026 deadlines.

The reports record their implementation revision date. ELSE submission remains pending the enrollment key.

The earlier C# task-board material was preserved locally in `D:/ops/uni/course-archive/software-design/csharp-study-project-2026-10-15` and remains recoverable through Git history. The active project is Domi in Go.

## Assessment preparation

On 15 October 2026, the [TMPS course page](https://else.fcim.utm.md/course/view.php?id=339) was accessible, but the account was not enrolled. Seven quiz entries were read without attempting them. All require enrollment, allow one attempt, and show opening and closing dates in 2024. The first four also require a password.

| Quiz                                                                              | Format    | Time   | Pass grade shown |
| --------------------------------------------------------------------------------- | --------- | -----: | ---------------: |
| [27823 Creational theory](https://else.fcim.utm.md/mod/quiz/view.php?id=27823)    | Theory    | 10 min | 5/10             |
| [27824 Creational blank UML](https://else.fcim.utm.md/mod/quiz/view.php?id=27824) | Blank UML | 5 min  | 5/10             |
| [27826 Structural theory](https://else.fcim.utm.md/mod/quiz/view.php?id=27826)    | Theory    | 10 min | 5/10             |
| [30996 Structural blank UML](https://else.fcim.utm.md/mod/quiz/view.php?id=30996) | Blank UML | 8 min  | 5/10             |
| [27828 Behavioral theory](https://else.fcim.utm.md/mod/quiz/view.php?id=27828)    | Theory    | 17 min | 5/10             |
| [30998 Behavioral blank UML](https://else.fcim.utm.md/mod/quiz/view.php?id=30998) | Blank UML | 10 min | 5/10             |
| [31000 General blank UML](https://else.fcim.utm.md/mod/quiz/view.php?id=31000)    | Blank UML | 20 min | 5/11             |

These dates and quiz settings are historical preparation evidence. They do not confirm current lab allocation, the midterm-to-lab mapping, or current midterm settings. The visible briefs do not confirm Go acceptance for the current project. The local Go implementation and report are prepared work; they do not establish owner-completed work, submission, instructor acceptance, or a grade.
