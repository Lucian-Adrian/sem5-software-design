# Lab 1 (C#): creational patterns in Domi

Domi helps people form groups to rent a home together. This version of Lab 1 implements all six creational patterns on Domi groups and profiles in C# (.NET 8).

| Pattern | Class | File |
| --- | --- | --- |
| Builder | `GroupBuilder` | [Patterns.cs](Patterns.cs) |
| Abstract Factory | `IOnboardingPresetFactory`, `BudgetPresetFactory`, `ComfortPresetFactory` | [Patterns.cs](Patterns.cs) |
| Prototype | `GroupPrototype` | [Patterns.cs](Patterns.cs) |
| Singleton | `CatalogDefaults` | [ExtraPatterns.cs](ExtraPatterns.cs) |
| Factory Method | `GroupCreator`, `StudentGroupCreator`, `QuietGroupCreator` | [ExtraPatterns.cs](ExtraPatterns.cs) |
| Object Pool | `SummaryBuilderPool` | [ExtraPatterns.cs](ExtraPatterns.cs) |

## Run

```powershell
dotnet run              # Builder, Abstract Factory, Prototype (JSON output)
dotnet run -- --extra   # Singleton, Factory Method, Object Pool
```

## Report

- [report.pdf](report.pdf): UTM-format report with UML diagrams
- [report.md](report.md): the same report in Markdown, the content source
- [assets/diagrams.puml](assets/diagrams.puml): PlantUML source of all diagrams

`build-pdf.ps1` renders the diagrams, generates `report.tex` from `report.md` in the UTM template, validates it and compiles `report.pdf`.
