# Lab 1 study notes

Use these notes to trace the patterns in [report.md](report.md) and [source](patterns.go). The examples are local coursework code with illustrative values.

## Trace the demo

Run:

```powershell
go run . --patterns
```

The output shows four group records. The Builder creates a group at 650 per month with capacity 3. The Abstract Factory creates a Budget group at 450 for 4 people with a suggested 500 budget, and a Comfort group at 700 for 2 people with a suggested 900 budget. The demo saves the related profile and group through the services, then joins the profile. The Prototype copies the joined Budget group's settings and saves a differently named group. Its ID is new and its member count is zero.

## Questions for an oral explanation

1. **What does Builder add here?** It collects the three group settings and validates them together before the caller saves the draft.
2. **Why not always write a struct literal?** One fixed input could use a literal. Builder is useful here because the demo and two preset families assemble group drafts in multiple places and should share validation.
3. **Why is this not Factory Method?** No subclass overrides a creation method. Builder assembles one draft; the Abstract Factory interface creates two related product types.
4. **What makes the preset interface an Abstract Factory?** Each implementation creates both a group draft and a suggested profile for one onboarding family. The helper checks the relationship between the two products.
5. **What does the prototype preserve and clear?** It preserves the cost and capacity, applies a new name, and clears the old ID and all member IDs.
6. **When is the new ID assigned?** `CloneAsDraft` returns an unsaved value without an ID. The existing `CreateGroup` service path asks the store to assign a fresh ID.
7. **Why does the prototype snapshot its member slice?** A Go slice can refer to shared backing storage. Copying it prevents later source changes from changing the snapshot.
8. **How do the factories satisfy their interface?** Go checks method sets implicitly. Each concrete factory defines both required methods; there is no `implements` declaration.

## Small live changes

- Add a `StudentPresetFactory` with its own cost, capacity and profile budget. Add it to the demo and to the family-consistency test.
- Lower the Budget suggested profile's budget below its group's cost. Confirm that `buildOnboardingPreset` returns `ErrBudgetTooLow`, then restore an affordable value.
- Rename the prototype copy and check that the source name stays unchanged and the copied record still has zero members.
- Set the Builder's capacity to 21 and explain why `Build` returns an input error before the service is called.

After a change, run the focused checks:

```powershell
go test ./...
go run . --patterns
```
