# Lab 2: Structural patterns

This lab adapts a legacy group price, records the results of group writes, and combines profile creation, group creation, and joining into one onboarding call. It demonstrates Adapter, Decorator, and Facade using the existing Domi services and in-memory store.

## Start here

Run the focused demonstration from this folder:

```powershell
D:/apps/go/bin/go.exe run . --structural
```

It prints the converted legacy group, the decorator's outcome lines, and the facade's onboarding result as JSON. To check the lab's behavior:

```powershell
D:/apps/go/bin/go.exe test ./...
```

The store is in memory, so records reset when the process exits.

## Read and run

- [learn.md](learn.md) connects the Go code to Python basics, traces the three patterns, and includes oral prompts with checking answers.
- [report.md](report.md) explains the design, source excerpts, UML diagrams, command, and demonstrated output.
- [structural.go](structural.go) implements the adapter, decorator, and facade; [structural_demo.go](structural_demo.go) assembles and runs them.

The same executable keeps Lab 1's creational examples and the HTTP API:

```powershell
D:/apps/go/bin/go.exe run . --patterns
D:/apps/go/bin/go.exe run . --demo
D:/apps/go/bin/go.exe run . --addr localhost:8080
```

![Lab 2 overview](assets/summary.png)

## Editable UML diagrams

Open an `.isx` file in [Isomorph](https://iso.meandivision.foo/app/) to edit its diagram. The report displays the sequence [SVG](assets/sequence.svg), and a [PNG](assets/sequence.png) is included.

| Pattern | Class diagram | Editable source |
| --- | --- | --- |
| Adapter | [SVG](assets/adapter.svg) | [Isomorph](assets/adapter.isx) |
| Decorator | [SVG](assets/decorator.svg) | [Isomorph](assets/decorator.isx) |
| Facade | [SVG](assets/facade.svg) | [Isomorph](assets/facade.isx) |
