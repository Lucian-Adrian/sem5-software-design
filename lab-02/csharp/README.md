# Lab 2 in C#: structural patterns

Adapter, Decorator and Facade (the required three) plus Proxy, on the Domi roommate domain, in C# / .NET 10. The Lab 1 Builder is reused to create the adapter's objects.

## Build and run

Requires the .NET 10 SDK. From this folder:

```bash
dotnet build
dotnet test                               # 12 xUnit tests, 2+ per pattern
dotnet run --project src/Domi.Structural  # demonstration of all four patterns
```

## Structure

| Path | Contents |
|---|---|
| `src/Domi.Structural/Domain/` | `Profile`, `Group`, `InMemoryStore`, services, `IGroupWriter`, `IProfileWriter` |
| `src/Domi.Structural/Creational/` | `GroupBuilder`, reused from Lab 1 |
| `src/Domi.Structural/Patterns/Adapter/` | `LegacyCsvListingAdapter` over the .NET library `TextFieldParser` (adaptee), `ListingImporter` client |
| `src/Domi.Structural/Patterns/Decorator/` | `GroupWriterDecorator` base, `AuditedGroupWriter`, `TimedGroupWriter` |
| `src/Domi.Structural/Patterns/Facade/` | `OnboardingFacade` |
| `src/Domi.Structural/Patterns/Proxy/` | `AdminOnlyGroupWriter`, `ICurrentUser` |
| `src/Domi.Structural/Client/Program.cs` | Composition root and demo: Client → Facade → Proxy → Decorators → GroupService |
| `src/Domi.Structural/data/legacy-listings.csv` | Legacy CSV the adapter imports |
| `tests/Domi.Structural.Tests/` | xUnit tests per pattern |
| [report.pdf](report.pdf) / [report.md](report.md) | Report in the UTM template with class diagrams, snippets, tests and conclusions |
| [learn.md](learn.md) | Study guide: pseudocode, traces, Socratic questions with answers, remaining patterns |
| `evidence/` | Recorded test log and demo output, 20 November 2026 |
| `assets/diagrams.puml` | PlantUML source; `build-pdf.ps1` renders diagrams and builds the PDF |

The earlier Go version of this lab remains one folder up; the handout requires a class-based OOP language, so the C# version is the submission.
