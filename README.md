# TMPS laboratory work

Lucian-Adrian Gavril, FAF-241. C# implementations, source explanations and formatted reports for software design techniques.

| Laboratory | Implementation | Report |
|---|---|---|
| [Lab 0](lab-00/README.md) | Single Responsibility, Open/Closed and Dependency Inversion in a standalone task-list program | [PDF](lab-00/study-report.pdf) |
| [Lab 1](lab-01/README.md) | Builder, Factory Method and Abstract Factory in a task-reporting project | [PDF](lab-01/study-report.pdf) |
| [Lab 3](lab-03/README.md) | Command, Observer and Strategy, continuing the Lab 1 project | [PDF](lab-03/study-report.pdf) |

Labs 1 and 3 share one executable, one client and one `TaskItem` model. Lab 3 uses the creational patterns from Lab 1 as well as its behavioral patterns. Each laboratory README explains the brief, source, a small trace and oral questions. The reports include the source listings.

## Build and run

Install .NET 10. From the repository root in PowerShell:

```powershell
$build = Join-Path $env:TEMP 'tmps-build'
dotnet build ./lab-00/TMPSLab00.csproj -c Release --artifacts-path $build
dotnet (Join-Path $build 'bin/TMPSLab00/release/TMPSLab00.dll')
dotnet build ./lab-01/TMPSLab01.csproj -c Release --artifacts-path $build
dotnet (Join-Path $build 'bin/TMPSLab01/release/TMPSLab01.dll') lab-01
dotnet (Join-Path $build 'bin/TMPSLab01/release/TMPSLab01.dll') lab-03
```

The projects use the standard .NET libraries and have no third-party package references. `lab-01/TMPSLab01.csproj` includes `lab-03/BehavioralPatterns.cs` in the shared executable.

## Report sources

Each laboratory contains `report.tex`, the compiled PDF, and local copies of the report class and university logo. Compile from the laboratory directory with `tectonic report.tex`. The class comes from the [UTM academic paper template](https://github.com/Lucian-Adrian/utm-academic-paper-template).

The official briefs are available on ELSE: [Lab 0](https://else.fcim.utm.md/mod/assign/view.php?id=38256), [Lab 1](https://else.fcim.utm.md/mod/assign/view.php?id=27825), [Lab 3](https://else.fcim.utm.md/mod/assign/view.php?id=27829). The linked Lab 1 and Lab 3 instructions require a hosted repository, documentation and the repository URL on ELSE. Hosting the files does not establish an ELSE submission or grade.
