# Lab 1 defense: creational patterns in Domi (C#)

Run `dotnet run` for the three main patterns and `dotnet run -- --extra` for the other three. The code is in `Patterns.cs` and `ExtraPatterns.cs`.

## 1. What the professor is testing

He isn't checking whether you can name six patterns. He's checking whether you can answer four questions for each one:

1. **What problem** does it solve? What goes wrong without it?
2. **Where** in a real project does that problem appear? In your case, where in Domi?
3. **Why this pattern** and not a simpler one (plain `new`, a constructor, a static method)?
4. **What does it cost?** When is it the wrong choice?

He also wants you to write the skeleton on paper, so you need the mechanism, not a memorized file. Last time you understood the logic but explained badly. The fix is to always speak in this order: problem, then where in Domi, then mechanism, then why not simpler, then cost.

## 2. The context: why "creational" patterns exist at all

Lab 0 was SOLID. Lab 1 builds directly on it.

The simplest way to create an object is `new Group("Flat", 450, 4)`. That works until one of these happens:

- **Who creates it is fixed.** If `GroupService` writes `new BudgetGroup()`, it depends on a concrete class. Adding a `StudentGroup` means editing `GroupService`. That breaks Open/Closed (open for extension, closed for modification) and Dependency Inversion (depend on abstractions, not concrete classes).
- **Creating it is complicated.** Many parameters, validation, or rules that several objects must satisfy together. If every caller repeats that logic, it gets copied and drifts apart. That breaks Single Responsibility.
- **Creating it is expensive or must happen only once.** Example: a DB connection, or one shared configuration.

Creational patterns move the act of creation to one place, so the code that uses an object doesn't decide how it's made. Each of the six patterns does this for a different reason. That's your opening sentence.

The Gang of Four book (1994) lists five creational patterns: Singleton, Factory Method, Abstract Factory, Builder and Prototype. Object Pool isn't in that book. Courses often add it as a sixth, which is why your brief says six. If he asks, say this.

## 3. The project: Domi

Domi helps people who don't know each other form a group to rent a home together. It has two main objects:

- **Group:** name, maximum monthly cost per person, capacity (1 to 20), and a member list.
- **Profile:** name and a private monthly budget.

The rule: a profile can join a group only if its budget is at least the group's cost and the group isn't full.

All validation lives in one method, `Group.Normalize`. Every pattern goes through it, so no pattern can create an invalid group.

## 4. The three patterns implemented

### Builder (`GroupBuilder`)

**Problem.** A group needs a name, a cost and a capacity, and they must be valid together. A constructor call like `new GroupInput("Flat", 450, 4)` puts two ints side by side. Swap them and you silently get a group that costs 4 and holds 450 people. As more optional settings appear (city, pets, smoking), constructors multiply. That's called a telescoping constructor.

**Where in Domi.** A user fills in the "create group" form one field at a time. The Builder works the same way.

**Mechanism.**
```csharp
GroupInput input = new GroupBuilder()
    .WithName("Builder example home")
    .WithMaxMonthlyCost(650)
    .WithCapacity(3)
    .Build();
```
Each `With...` method stores one value and returns `this`, so the calls chain. Nothing is validated until `Build()`, which checks everything at once. A half-built object is never visible.

**Why not simpler.** Named arguments (`new GroupInput(Name: ..., Capacity: ...)`) solve readability. A Builder adds two things: it collects values step by step, possibly in different places, and it validates once at the end. Use it if both matter. If not, a record with named arguments is enough, and you can say so.

**Cost.** One more class to maintain. It's overkill for objects with two fields.

### Abstract Factory (`IOnboardingPresetFactory`)

**Problem.** Some objects only make sense in matching sets. In Domi, onboarding suggests a group and a profile together. A budget renter (500) shown a comfort group (700) can't afford it, and the join fails. If the code picks the group and the profile separately, nothing stops a mismatched pair.

**Where in Domi.** The onboarding screen: "Pick a style: Budget or Comfort." Choosing once must give a matching group and profile.

**Mechanism.**
```csharp
public interface IOnboardingPresetFactory
{
    GroupInput CreateGroupDraft();
    ProfileInput CreateSuggestedProfile();
}
// BudgetPresetFactory: group 450, renter 500
// ComfortPresetFactory: group 700, renter 900
OnboardingPreset preset = Presets.Build(new BudgetPresetFactory());
```
The client (`Presets.Build`) only knows the interface. Picking the factory picks the whole family. Mixing families is impossible because one object creates both parts.

**Why not simpler.** Two separate `if (style == "budget")` blocks, one for the group and one for the profile, would work until someone adds a third style and updates one block but not the other. With a factory, a new style is one new class, and `Presets.Build` doesn't change. That's Open/Closed.

**Cost.** Adding a new product to the family (for example, a suggested apartment) means changing the interface and every factory. Adding a new family is cheap. Adding a new product type is expensive. This is the classic trade-off, and a good answer if he asks about drawbacks.

### Prototype (`GroupPrototype`)

**Problem.** A user wants "a group like this one" with the same cost and capacity, a new name, and nobody in it. Copying field by field from outside means the caller must know every field. Copying the whole object by reference shares the member list.

**Where in Domi.** A "Duplicate group" button, or reusing last semester's group for a new one.

**Mechanism.**
```csharp
GroupPrototype prototype = new(source);                      // snapshot, member list copied
Group draft = prototype.CloneAsDraft("Budget shared home copy"); // same cost/capacity, no ID, no members
```

**The key concept: shallow vs deep copy.** A shallow copy copies field values. For a `List`, the value is a reference, so both groups point to the same list, and adding a member to the copy also adds them to the original. A deep copy creates a new list. The prototype copies the list in the snapshot and gives the clone an empty list. The demo proves it: the source has 1 member and the copy has 0.

**What is deliberately not copied.** The ID and the members. The ID is the identity of the original record, and two records with one ID is a bug. The members belong to the original group. Settings are copied.

**Why not simpler.** C# has `MemberwiseClone()`, but it's shallow, which is exactly the trap above. `ICloneable` doesn't say whether a copy is shallow or deep. That's why explicit copy logic is safer.

**Cost.** Every new field needs a decision: copy it, deep-copy it or reset it. Forget one and the copy is wrong.

## 5. The other three: explain and write

### Singleton (`CatalogDefaults`)

**Problem.** Some things must exist exactly once: global configuration, a logger, a hardware handle. Two instances would disagree.

**Where in Domi.** Fixed catalog defaults (default capacity 3, default cost 500) read by the creators.

**Mechanism.**
```csharp
public sealed class Config
{
    private static readonly Lazy<Config> _instance = new(() => new Config());
    public static Config Instance => _instance.Value;
    private Config() { }
}
```
The private constructor stops other code from calling `new`. The static `Instance` is the only way in. `Lazy<T>` creates it on first use and is thread-safe, so two threads can't create two instances at once. `sealed` stops a subclass from creating a second one.

**Why it's often criticized.** It's global state wearing a pattern name. Any class can reach it without declaring the dependency, which breaks Dependency Inversion. Tests can't swap it out, and mutable singleton state leaks between tests. Modern C# usually registers a service as a singleton in dependency injection (`services.AddSingleton<IConfig, Config>()`), so there's one instance but it's still passed through constructors. Say this and he'll know you understand it, not just the definition.

**Why not for groups or profiles.** Those are per-user data, and there are many of them. A Singleton is only for things that should exist once and don't change.

### Factory Method (`GroupCreator`)

**Problem.** A workflow is always the same (create the group, save the founder, join), but the type of group created differs. You don't want to copy the workflow for each type.

**Where in Domi.** `StudentGroupCreator` (cheap, 5 people) and `QuietGroupCreator` (2 people) share `CreateAndJoin`.

**Mechanism.**
```csharp
abstract class GroupCreator
{
    public GroupView CreateAndJoin(...)      // fixed workflow
    {
        GroupInput draft = CreateGroupDraft(); // the one step that varies
        ... save, join ...
    }
    protected abstract GroupInput CreateGroupDraft(); // the "factory method"
}
class StudentGroupCreator : GroupCreator
{
    protected override GroupInput CreateGroupDraft() => /* 300, capacity 5 */;
}
```
The base class calls an abstract method, and the subclass decides what it returns. That's the whole pattern.

**The most likely question: Factory Method vs Abstract Factory.**

| | Factory Method | Abstract Factory |
| --- | --- | --- |
| Creates | one product | a family of related products |
| How it varies | inheritance: a subclass overrides a method | composition: you pass in a factory object |
| In Domi | which group a creator makes | a matching group and profile |

One-sentence version: Factory Method is one method a subclass overrides. Abstract Factory is an object with several factory methods that belong together. Each method inside an Abstract Factory is often a factory method.

**Cost.** One subclass per variant. If the only difference is data, a parameter or a Builder would be simpler.

### Object Pool (`SummaryBuilderPool`)

**Problem.** Some objects are expensive to create or limited in number: DB connections, threads, big buffers. Creating and destroying one per request wastes time or hits limits.

**Where in Domi.** Building group summaries reuses `StringBuilder` buffers. The demo makes 2 summaries with only 1 builder created. In a real backend, the honest example is the database connection pool. ADO.NET already pools connections for you.

**Mechanism.**
```csharp
class Pool<T> where T : new()
{
    readonly Stack<T> idle = new();
    public T Acquire() => idle.Count > 0 ? idle.Pop() : new T();
    public void Release(T item) { /* reset item */ idle.Push(item); }
}
```
**Two rules he may test:**
1. Reset on release (`builder.Clear()`), or the next user sees the previous user's data.
2. Always release, even on an error. That's why `Summarize` uses `try/finally`. A forgotten release is a leak, and the pool keeps creating new objects.

`maxIdle` caps how many unused objects are kept, so the pool doesn't hold memory forever.

**Cost.** For cheap objects like a small `StringBuilder`, the garbage collector is already fast, so the pool is mainly a demonstration. Admit this if asked. That honesty scores well. Pools pay off for connections and large arrays (`ArrayPool<T>.Shared`).

## 6. How the patterns connect (say this at the end)

- The factories use the Builder internally. Patterns combine.
- All six send creation through one place, so validation (`Group.Normalize`) can't be skipped.
- Back to Lab 0: Abstract Factory and Factory Method give Open/Closed (a new type is a new class, no edits). The client depends on `IOnboardingPresetFactory`, which is Dependency Inversion. Builder and Prototype keep construction logic out of the services, which is Single Responsibility.

## 7. A 5-minute speaking route

1. (30 s) "Domi: people form groups to rent together. Creating groups has three different problems."
2. (1 min each) Builder, Abstract Factory, Prototype: problem, where, code, why not simpler.
3. (1 min) Run `dotnet run` and point at memberCount 1 for both families, and the copy with a new ID and 0 members.
4. (30 s) "I also know the other three", then run `--extra`.
5. Stop and let him ask.

## 8. Practice questions (cover the answers, say them out loud)

1. What problem do all creational patterns share?
2. Factory Method vs Abstract Factory, and give a Domi example of each.
3. Why is Singleton called an anti-pattern? What do we use instead in modern C#?
4. Shallow vs deep copy, and what breaks in Domi with a shallow copy?
5. Why doesn't the clone keep the ID?
6. Builder vs a constructor with named parameters, and when is a Builder worth it?
7. What happens if a pool user forgets to release? What if the pool doesn't reset?
8. Add a "Student" onboarding family. What files change? (Answer: one new factory class, nothing else.)
9. Add a "suggested apartment" to every family. What changes? (Answer: the interface and every factory. That's the Abstract Factory trade-off.)
10. Write a thread-safe Singleton on paper.

Answers to 1 to 7 are in sections 2 to 5. If you can say each one in two sentences without looking, you're ready.
