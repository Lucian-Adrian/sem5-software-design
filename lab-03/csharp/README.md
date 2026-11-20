# Lab 3 in C#: behavioral patterns

Strategy, Command and Observer on the Domi roommate domain in C# / .NET 10, building on Lab 1 (`GroupBuilder`) and Lab 2 (`Adapter`, `Decorator`, `Facade`, `Proxy`).

## Build and run

Requires the .NET 10 SDK. From this folder:

```bash
dotnet build
dotnet test                               # 25 xUnit tests (4+ per Lab 3 pattern + Lab 2 + interaction)
dotnet run --project src/Domi.Behavioral  # end-to-end demo across Lab 2 and Lab 3 patterns
dotnet run --project src/Domi.Behavioral -- --buffer 75   # optional runtime strategy config
```

## Structure

| Path | Contents |
|---|---|
| `src/Domi.Behavioral/Domain/` | `Profile`, `Group` (`Join` & `Leave`), `InMemoryStore`, services, `IGroupWriter`, `IProfileWriter` |
| `src/Domi.Behavioral/Creational/` | `GroupBuilder`, reused from Lab 1 |
| `src/Domi.Behavioral/Patterns/Adapter/` | `LegacyCsvListingAdapter` over `.NET` `TextFieldParser`, `ListingImporter` (Lab 2) |
| `src/Domi.Behavioral/Patterns/Decorator/` | `GroupWriterDecorator`, `AuditedGroupWriter`, `TimedGroupWriter` (Lab 2) |
| `src/Domi.Behavioral/Patterns/Facade/` | `OnboardingFacade` (Lab 2) |
| `src/Domi.Behavioral/Patterns/Proxy/` | `AdminOnlyGroupWriter`, `ICurrentUser` (Lab 2) |
| `src/Domi.Behavioral/Patterns/Strategy/` | `IGroupMatchingStrategy`, `GroupMatcher`, `AffordableWithSpaceStrategy`, `BudgetBufferStrategy`, `SmallGroupStrategy` |
| `src/Domi.Behavioral/Patterns/Command/` | `IGroupCommand`, `JoinGroupCommand`, `CreateGroupCommand`, `GroupCommandInvoker` (queue + LIFO undo) |
| `src/Domi.Behavioral/Patterns/Observer/` | `IGroupActivitySubject`, `GroupActivityNotifier`, `IGroupActivityObserver`, `ActivityLogObserver`, `FullGroupAlertObserver`, `ActivityHistoryObserver` |
| `src/Domi.Behavioral/Client/` | `GroupMatchingWorkflow` and `Program.cs` composition root |
| `tests/Domi.Behavioral.Tests/` | `BehavioralPatternTests.cs` (Strategy, Command, Observer, Interaction) and `StructuralPatternTests.cs` |
| [report.pdf](report.pdf) / [report.md](report.md) | Report in the UTM template with UML class diagrams, code snippets, tests and conclusions |
| [learn.md](learn.md) | Study guide: behavioral idea, pseudocode, traces, Socratic Q&A, remaining patterns |
| [PRESENT.md](PRESENT.md) | Defense sheet: 4-question cards, live change requests with exact C#, presentation script |
| `evidence/` | Recorded `test-output.txt` and `demo-output.txt` (20 November 2026) |
| `assets/diagrams.puml` | PlantUML source; `build-pdf.ps1` renders SVG/PNG diagrams and compiles `report.pdf` |
