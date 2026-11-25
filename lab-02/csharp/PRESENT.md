# Lab 2 defense: structural patterns in Domi (C#)

Read this in the 10 minutes before your turn. The deeper version, with pseudocode, traces and more questions, is [learn.md](learn.md).

## 1. What Istrati is testing

The handout gives 1.5 points per pattern, 1 for justification, 1 for interaction, 1 for SOLID and 1 for tests. Another **1.5 is for the defense: explaining the code and making a change he asks for, live**. AI code you cannot explain counts as not done.

For each pattern he wants four answers:

1. **What problem** does it solve? What goes wrong without it?
2. **Where** in Domi?
3. **Why this pattern**, and not something simpler (edit the class, inheritance, a plain method)?
4. **What does it cost?** When is it the wrong choice?

**What went wrong last time (Lab 1):** you understood the logic but explained it in a messy order. Fix: always speak in this order: **problem → where in Domi → mechanism → why not simpler → cost**. Stop after the cost. Let him ask the next question.

He may ask you to draw the skeleton on paper. That's why each card below has a 4-line skeleton.

## 2. Opening (30 seconds)

> "Domi helps strangers form a group to rent a flat together. The domain is `Profile`, `Group`, and one rule in `Group.Join`: budget ≥ cost, not full, not a duplicate. Structural patterns change **how objects are connected**, not what they do. Every pattern here puts an object between two others. The difference is *why* it stands there."

| Pattern | Stands in the middle to… | Same interface as what it wraps? |
|---|---|---|
| Adapter | **translate** | No, that's the point |
| Decorator | **add** something, then forward | Yes |
| Proxy | **control** whether the real one is reached | Yes |
| Facade | **simplify** many calls into one | No, it offers a new, smaller one |

Then show the single execution path that uses every pattern (from `Program.cs:16-19`):

```
Client → OnboardingFacade → AdminOnlyGroupWriter (proxy) → TimedGroupWriter → AuditedGroupWriter (decorators) → GroupService
Client → ListingImporter → LegacyCsvListingAdapter → TextFieldParser (.NET library) + GroupBuilder (Lab 1)
```

## 2b. "What changed in your app when you implemented these?"

Istrati asks this. Do not define the pattern. Show **the code before, the code after, and the line in the demo output that proves it**. Each block below is real code from the repo (Lab 1 `lab-01/csharp`, Lab 2 `src/Domi.Structural`).

### Step 0: the only edit to old code is one interface

**Before (Lab 1, `Program.cs` and `Services.cs`):** the client creates the concrete service and calls it directly. Nothing can sit between the client and the service.

```csharp
GroupService groups = new(profiles);              // concrete class
groups.JoinGroup(group.Id, profile.Id);           // straight into the service
```

**After (Lab 2, `Domain/Services.cs`):** I pulled the two write methods into an interface, and `GroupService` implements it. The method bodies did not change.

```csharp
public interface IGroupWriter
{
    Group CreateGroup(GroupInput input);
    Group JoinGroup(string groupId, string profileId);
}
public class GroupService(InMemoryStore store) : IGroupWriter { ... same bodies ... }
```

**Why it matters:** now anything that implements `IGroupWriter` can stand in front of `GroupService`. The decorators and the proxy are exactly that. Clients (`ListingImporter`, `OnboardingFacade`) take an `IGroupWriter` and never learn what is behind it.

**What the client builds now (`Client/Program.cs`):**

```csharp
IGroupWriter decorated = new TimedGroupWriter(new AuditedGroupWriter(new GroupService(store), log), log);
IGroupWriter guarded   = new AdminOnlyGroupWriter(decorated, admin);
var facade             = new OnboardingFacade(profiles, guarded);
```

A call to `facade.Onboard(...)` therefore goes: facade → proxy (admin check) → timer → audit → `GroupService` → `Group.Join` (the rules).

### Adapter: Domi can now import the old site's CSV

- **Before:** groups existed only if someone typed them into `CreateGroup`. The old site exports `name;monthly_total;roommates`, which Domi could not read. The library `TextFieldParser` can read it, but it returns `string[]`, and Domi wants `GroupInput`.
- **After:** one new class translates between the two.

```csharp
public sealed class LegacyCsvListingAdapter : IListingSource      // Domi's interface
{
    private readonly TextFieldParser _parser;                     // the library class (adaptee)
    public IEnumerable<GroupInput> GetListings() { ... _parser.ReadFields() ... TryTranslate ... }
}
// TryTranslate: "Riscani flat;1500;3" -> total 1500 / 3 people = 500 per person (rounded up)
//               -> new GroupBuilder().WithName(..).WithMaxMonthlyCost(500).WithCapacity(3).Build()
```

`ListingImporter` only knows `IListingSource`, so a JSON feed later is one more adapter, and the importer does not change.

- **Proof in the demo (section 1):**
  ```text
  imported Riscani flat [group-1] cost=500/person members=0/3
  imported Botanica studio; near park [group-2] cost=501/person members=0/3   <- 1501/3 rounded up; the ";" inside quotes is kept
  rejected "Centru loft;0;2"                                                <- total 0: reported, not a crash
  ```

### Decorator: every write is now logged and timed, and the service was not touched

- **Before:** `GroupService.JoinGroup` either worked or threw. Nothing recorded who tried what, whether it failed, or how long it took. Adding that inside `GroupService` would mix logging into the business service.
- **After:** two wrappers with the same interface. Each one does its own job and then calls the next one (`base.JoinGroup` → `Inner.JoinGroup`).

```csharp
public sealed class AuditedGroupWriter(IGroupWriter inner, TextWriter log) : GroupWriterDecorator(inner)
{
    public override Group JoinGroup(string g, string p) => Record("JoinGroup", () => base.JoinGroup(g, p));
    // Record: on success write "[audit] ... ok", on DomainException write "[audit] ... error" and rethrow
}
public sealed class TimedGroupWriter(...) : GroupWriterDecorator(inner)   // Stopwatch, "[timer] ... took x ms"
```

Turning them on or off, or changing their order, is only the `new` line in `Program.cs`.

- **Proof in the demo (section 2):** Ion (budget 300) tries to join Riscani flat (500).
  ```text
  [audit] JoinGroup error: Ion's budget 300 is below Riscani flat's cost 500
  [timer] JoinGroup took 1.795 ms
    caller still gets the error: Ion's budget 300 is below Riscani flat's cost 500
  ```
  The failure is logged and timed, and the caller still gets the same exception. The decorator added behaviour and changed nothing in the result.

### Facade: onboarding is now one call that refuses bad input before writing

- **Before:** to put a new person into a new group, each caller had to make three calls, in the right order, with the right ids:

```csharp
var profile = profiles.CreateProfile(person);
var group   = groups.CreateGroup(newGroup);
groups.JoinGroup(group.Id, profile.Id);     // if this fails, the first two writes stay behind
```

- **After:** one method hides the three calls and checks the budget first:

```csharp
public OnboardingResult Onboard(ProfileInput person, GroupInput newGroup)
{
    var draftProfile = new Profile("draft", person);       // validate without saving
    var draftGroup   = new Group("draft", newGroup);
    if (draftProfile.MonthlyBudget < draftGroup.MaxMonthlyCost) throw new DomainException(...);
    var profile = profiles.CreateProfile(person);
    var group   = groups.CreateGroup(newGroup);
    groups.JoinGroup(group.Id, profile.Id);
    return new OnboardingResult(profile, group);
}
```

- **Proof in the demo (section 3):**
  ```text
  Ana [profile-4] is in Ana's flat [group-5] cost=650/person members=1/2     <- one call, 3 writes
  refused before any write: Dan cannot afford Too pricey (100 < 900)          <- no profile, no group created
  ```
  The facade's writes go through the proxy and the decorators too: you can see `[audit] CreateGroup ok -> group-5` right above Ana's line.

### Proxy (bonus): only admins can create groups

- **Before:** anyone who could reach `GroupService` could create a group.
- **After:** a stand-in with the same interface checks the current user and then forwards the call:

```csharp
public sealed class AdminOnlyGroupWriter(IGroupWriter real, ICurrentUser user) : IGroupWriter
{
    public Group CreateGroup(GroupInput input)
    {
        if (!user.IsAdmin) throw new UnauthorizedAccessException($"{user.Name} is not an admin ...");
        return real.CreateGroup(input);
    }
    public Group JoinGroup(string groupId, string profileId) => real.JoinGroup(groupId, profileId);  // joining stays open
}
```

- **Proof in the demo (section 4):** the same facade, with the session switched to non-admin:
  ```text
  blocked: Lucian is not an admin and cannot create groups
  but Maria can still join: Botanica studio; near park [group-2] ... members=1/3
  ```
  Honest limit, if asked: Maria's profile was created before the proxy blocked the group, so the facade is not a transaction.

### What did NOT change

`Group.Join` still holds every rule (budget ≥ cost, group not full, no duplicate member). `Profile`, `Group` and the `GroupService` method bodies were not edited. The patterns added abilities **around** the domain and did not move rules out of it.

### 30-second spoken version

> "In Lab 1 the client did `new GroupService` and called it directly. In Lab 2 I changed old code in one place: I extracted `IGroupWriter` with `CreateGroup` and `JoinGroup`, and `GroupService` implements it with the same bodies. Everything else is new classes that implement or use that interface. The **Adapter** wraps the library `TextFieldParser`, so Domi now imports the old CSV: in the demo, Riscani at 1500 for 3 becomes 500 per person, and the row with total 0 is rejected. The **Decorators** wrap the service, so every write is logged and timed: Ion's failed join shows `[audit] error` and `[timer]`, and he still gets the exception. The **Facade** turns create, create, join into one `Onboard` call that refuses Dan before writing anything. The **Proxy** blocks non-admins from creating groups but lets Maria join. The business rules never moved: they are still in `Group.Join`."

## 3. The four cards

### Adapter: `Patterns/Adapter/LegacyCsvListingAdapter.cs`

- **Problem.** An old site exports `name;monthly_total;roommates`. We think in *per-person* cost and typed `GroupInput`. The format and the meaning are both different.
- **Where.** Importing listings. The adaptee is `TextFieldParser`, a .NET library class we can't change. All it offers is `ReadFields()` → `string[]`.
- **Mechanism.** The adapter implements the target `IListingSource` and *holds* the parser (line 28). `GetListings` skips the header, reads rows and calls `TryTranslate`. That method does ceiling division (`1501/3` → 500 remainder 1 → **501**, so 3×501 ≥ 1501), then builds the input with the **Lab 1 `GroupBuilder`**. Bad rows go to `RejectedRows`. The client `ListingImporter` sees only `IListingSource` (line 84).
- **Why not simpler.** If `GroupService` parsed CSV itself, every foreign format would grow the core service. The adapter keeps foreign formats at the edge.
- **Cost.** One extra class per format. Bad rows are skipped, not fatal. That's a choice, so say it.

```
class Adapter : Target { adaptee; Request() { x = adaptee.SpecificRequest(); return translate(x); } }
```

### Decorator: `Patterns/Decorator/GroupWriterDecorator.cs`

- **Problem.** We want audit logging and timing on writes without touching `GroupService`, and in any combination.
- **Where.** Around every `CreateGroup` and `JoinGroup`.
- **Mechanism.** The abstract base `GroupWriterDecorator(IGroupWriter inner)` implements the same interface and forwards by default (lines 102-109). `AuditedGroupWriter` logs ok or error and **rethrows with `throw;`** (line 131). `TimedGroupWriter` logs in `finally`, so failed calls are timed too. Stacked at runtime: `new Timed(new Audited(new GroupService(store)))`.
- **Why not simpler.** Inheritance (`AuditedGroupService : GroupService`) fixes the combination at compile time: 2ⁿ subclasses for n concerns. Decorators compose at runtime. The base class means a new decorator overrides only what it changes.
- **Cost.** Many small objects. Debugging goes through layers. **Order matters** (`Audited(Timed(x))` prints the lines in the other order and times different work).

```
abstract class Decorator : Component { inner; Op() => inner.Op(); }
class Concrete : Decorator { override Op() { before(); r = base.Op(); after(); return r; } }
```

### Facade: `Patterns/Facade/OnboardingFacade.cs`

- **Problem.** Onboarding is 3 calls in a fixed order: create profile → create group → join. Every caller would repeat that order and get it wrong.
- **Where.** `OnboardingFacade.Onboard(person, group)`.
- **Mechanism.** Step 0 builds **draft** `Profile` and `Group` objects. They validate with the domain's own rules and write nothing. Then it checks budget ≥ cost (lines 208-212). Only then does it do the 3 writes. It depends only on `IProfileWriter` and `IGroupWriter`.
- **Why not simpler.** Without it, the order and pre-check logic is copied into every client. It adds **no new business rules**; those stay in `Profile` and `Group`.
- **Cost.** **Not a transaction.** If step 3 failed, steps 1–2 would stay. The demo shows it: Maria's profile exists after the proxy blocked her group. A real DB would wrap the three steps. Say this before he does.

```
class Facade { a; b; c; DoIt() { a.Step1(); b.Step2(); c.Step3(); } }
```

### Proxy (bonus): `Patterns/Proxy/AdminOnlyGroupWriter.cs`

- **Problem.** Only admins may create groups, but anyone may join.
- **Mechanism.** Same interface `IGroupWriter`. `CreateGroup` checks `user.IsAdmin` **on every call** (line 174) and throws `UnauthorizedAccessException`, so the real object is never reached. `JoinGroup` passes straight through.
- **Decorator vs Proxy?** Same class diagram, different **intent**. A decorator always forwards and adds. A proxy decides *whether* the real object is reached. Other kinds of proxy: virtual (lazy creation), remote, caching.

## 4. Live change requests: what to type

**"Add a retry decorator."** Write one new class, wire it in `Program.cs:16`, and touch nothing else (Open/Closed):

```csharp
public sealed class RetryGroupWriter(IGroupWriter inner, int attempts) : GroupWriterDecorator(inner)
{
    public override Group CreateGroup(GroupInput input)
    {
        for (var i = 1; ; i++)
        {
            try { return base.CreateGroup(input); }
            catch (IOException) when (i < attempts) { }   // retry only transient errors
        }
    }
}
// Program.cs: new TimedGroupWriter(new RetryGroupWriter(new AuditedGroupWriter(...), 3), log)
```

Don't retry `DomainException`: "group full" won't fix itself.

**"Import from JSON too."** Write one new `IListingSource` class. `ListingImporter` doesn't change:

```csharp
// needs: using System.Text.Json;
public sealed class JsonListingSource(string json) : IListingSource
{
    public IEnumerable<GroupInput> GetListings() =>
        JsonSerializer.Deserialize<List<GroupInput>>(json) ?? [];
}
```

**"Prices now come in euros."** Change only `TryTranslate`, before the division.

**"Non-admins may create at most 1 group."** Change `AdminOnlyGroupWriter`, or add a second proxy in the chain.

**"Send a welcome message after onboarding."** Inject an `INotifier` into the facade and call it after step 3. That's a new dependency, not new rules.

After any change, run `dotnet test`. Then add one test that copies the shape of an existing one in `tests/Domi.Structural.Tests/PatternTests.cs`.

## 5. Hard questions, short answers

1. **Target vs adaptee?** Target = what the client wants (`IListingSource`). Adaptee = the existing thing with the wrong interface (`TextFieldParser`).
2. **Object adapter vs class adapter?** We hold the adaptee (composition). A class adapter inherits from it. C# has single inheritance, so composition is normal.
3. **Why round up?** 3×500 = 1500 < 1501; rounding down means the rent can never be covered.
4. **`throw;` vs `throw e;`?** `throw;` keeps the original stack trace.
5. **Why do the decorators take `IGroupWriter`, not `GroupService`?** So they can wrap each other.
6. **Why does the timer use `finally`?** So failed calls are timed too, and the exception still reaches the caller.
7. **Facade vs Adapter?** An adapter makes **one** object fit an interface the client already expects. A facade invents a **new, simpler** interface over **several** objects.
8. **Can clients still bypass the facade?** Yes. It's a convenience, not a lock.
9. **Where is SOLID?**
   - S: each class has one job.
   - O: a new decorator or source is a new class.
   - L: every wrapper can replace `IGroupWriter`.
   - I: `IGroupWriter` and `IProfileWriter` are separate.
   - D: the facade and importer depend on interfaces, and only `Program.cs` uses `new` on concrete classes (composition root).
10. **Where is Lab 1?** `GroupBuilder` in `Creational/`, used by the adapter to build every imported `GroupInput`.
11. **The 3 you didn't implement?**
    - **Bridge:** two dimensions (notification type × channel) → n+m classes instead of n×m.
    - **Composite:** a tree where leaf and branch share one interface (room/flat/building `TotalRent()`).
    - **Flyweight:** share immutable intrinsic state (the neighbourhood) across many objects; the extrinsic state (price) stays per object.

## 6. What to show on screen, in order

1. `dotnet run --project src/Domi.Structural`, then walk sections 1→4 of the output.
2. `report.pdf` → the overview diagram (d01) to show the interaction path.
3. Open one pattern file when he asks; the role names are in the header comment.
4. `dotnet test` → 12 passed.
