# Lab 2 in C#: structural patterns

Adapter, Decorator and Facade (required three) plus Proxy (extra), on the Domi roommate domain, in C# / .NET 10. No external packages.

```bash
dotnet run              # demo of all four patterns
dotnet run -- --check   # 13 self-checks
```

| File | Contents |
|---|---|
| [Domain.cs](Domain.cs) | Profile, Group, store, services, `IGroupWriter` (unchanged by the patterns) |
| [Adapter.cs](Adapter.cs) | Legacy `name;total;people` feed → per-person `GroupInput` |
| [Decorator.cs](Decorator.cs) | `AuditedGroupWriter`, `TimedGroupWriter` |
| [Facade.cs](Facade.cs) | `OnboardingFacade.Onboard` |
| [Proxy.cs](Proxy.cs) | `AdminOnlyGroupWriter` |
| [Program.cs](Program.cs) | Wiring, demo and self-checks |
| [report.pdf](report.pdf) | Submission report in the UTM template, 23 pages, 7 UML diagrams |
| [report.md](report.md) | Report source; `build-pdf.ps1` renders diagrams and builds the PDF (same pipeline as Lab 1) |
| [assets/diagrams.puml](assets/diagrams.puml) | PlantUML source of the diagrams |
| [learn.md](learn.md) | Pseudocode, traces, Socratic questions with answers, the other three patterns |
| [evidence/demo-output.txt](evidence/demo-output.txt) | Recorded run on 20 November 2026 |

The earlier Go version of this lab remains one folder up.
