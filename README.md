# Domi backend: TMPS coursework in Go

Lucian-Adrian Gavril, FAF-241.

Domi helps people find roommates to share rent. This course project builds its backend in Go. Lab 1 lets a person create a profile, browse affordable roommate groups, and join a group when its budget and capacity rules allow it.

Start with the [Lab 1 README](lab-01/README.md), then read the [report and UML diagrams](lab-01/report.md). The [learning guide](lab-01/learn.md) explains the four OOP pillars, all five SOLID principles, and the 52 concepts from the group chat, with examples and presentation questions.

```powershell
cd lab-01
go run . --demo
go run . --addr localhost:8080
```

The implementation uses the Go standard library and in-memory storage. It is a working coursework prototype. Authentication, durable database storage, member approval, and integration with the [existing Domi frontend](https://trydomi.vercel.app) remain future work.

## Four labs, one project

| Lab | Purpose | Status |
|---|---|---|
| 1 | Explain and demonstrate all five SOLID principles | Implemented, report prepared |
| 2 | Add a useful creational pattern to Domi | Planned |
| 3 | Add a useful structural pattern to the same backend | Planned |
| 4 | Add a useful behavioral pattern to the same backend | Planned |

Later pattern choices will follow concrete features and the current professor's brief. Each later report will have its own lab folder. The learning guide covers the wider vocabulary without forcing every pattern into Lab 1.

The public ELSE course calls the first SOLID assignment “Laboratory Work #0”; this repository uses the student's Lab 1 numbering. Its displayed dates are from 2025 and do not establish the 2026 deadline. Submission is pending course enrollment because ELSE requires an enrollment key.

The report carries the requested **7 October 2026 lab reference date**. Preparation and revision occurred on **15 October 2026**. Git commit and submission timestamps retain their actual dates.

The earlier C# task-board material was preserved locally in `D:/ops/uni/course-archive/software-design/csharp-study-project-2026-10-15` and remains recoverable through Git history. The active project is Domi in Go.
