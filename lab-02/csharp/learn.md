# Learn and defend Lab 2 (C#)

Order: the one-sentence idea, then pseudocode, then the real C#, then questions. If you can draw the box diagram and write the pseudocode from memory, you understand the pattern. The C# syntax is the last 10%.

## The one idea behind all structural patterns

> Structural patterns change **how objects are connected**, not what each object does.

Every pattern in this lab is "put an object between two other objects". The difference is **why** it stands there:

| Pattern | It stands in the middle to... | Same interface as the thing it wraps? |
|---|---|---|
| Adapter | **translate** one interface into another | No, that is the whole point |
| Decorator | **add** something, then forward | Yes |
| Proxy | **control** whether the real one is reached | Yes |
| Facade | **simplify** many calls into one | No, it offers a new, smaller one |

If you remember this table you can answer most questions.

## C# you need, compared with Python

| C# | Python equivalent | Where |
|---|---|---|
| `interface IGroupWriter { Group CreateGroup(GroupInput i); }` | a `Protocol` / abstract class with only method signatures | Domain.cs |
| `class GroupService(InMemoryStore store) : IGroupWriter` | `class GroupService(IGroupWriter): def __init__(self, store): self.store = store` (primary constructor) | Domain.cs |
| `record GroupInput(string Name, int MaxMonthlyCost, int Capacity)` | `@dataclass(frozen=True)` | Domain.cs |
| `throw new DomainException("...")` / `catch` | `raise` / `except` | everywhere |
| `throw;` inside catch | bare `raise` (rethrow the same exception) | Decorator.cs |
| `Func<Group>` | a lambda / callable returning a Group | Decorator.cs |
| `yield return x` | `yield x` (generator) | Adapter.cs |
| `out var input` | returning a tuple `(ok, value)` | Adapter.cs |

## 1. Adapter

**Story.** The old site says "flat costs 1500, for 3 people". We say "500 per person, capacity 3". Different language. A translator sits between.

**Pseudocode**

```
interface ListingSource:
    get_listings() -> list of GroupInput

class LegacyFeedAdapter implements ListingSource:
    feed                               # the old object, kept inside
    get_listings():
        for row in feed.download_rows():
            name, total, people = split(row, ";")
            if bad numbers: reject row; continue
            per_person = total // people
            if total % people != 0: per_person += 1     # round UP
            yield GroupInput(name, per_person, people)

importer = ListingImporter(LegacyFeedAdapter(old_feed), groups)
importer.import_all()                  # importer never sees ";" rows
```

**Trace "Botanica studio;1501;3"**: split → `1501`, `3` → `1501 / 3 = 500`, `1501 % 3 = 1` → not zero → `501` → `GroupInput("Botanica studio", 501, 3)` → `GroupService.CreateGroup` validates and stores it.

**Socratic questions** (answer aloud first, then read the answer)

1. *Why not just change `GroupService` to accept `total` and `people`?*
   Then every user of `GroupService` must know about the old site's format. Tomorrow a second legacy site arrives with another format and the service grows again. The adapter keeps foreign formats at the edge.
2. *Why round up and not down?*
   3 × 500 = 1500 < 1501. Rounding down would mean the group can never pay the full rent. With 501, 3 × 501 = 1503 ≥ 1501.
3. *Why `total / people` and then `+1`, not `(total + people - 1) / people`?*
   Both are correct ceiling division. The quotient-plus-remainder form cannot overflow when `total` is near `int.MaxValue`.
4. *Which class is the Target, which is the Adaptee? How do you know?*
   Target = the interface the **client** wants (`IListingSource`). Adaptee = the existing thing with the wrong interface (`LegacyHousingFeed`). The adapter implements the first and contains the second.
5. *This is an object adapter. What is a class adapter?*
   The adapter inherits from the adaptee instead of holding it. C# allows only one base class, so object adapters (composition) are the normal choice.
6. *Changed condition: the old site starts sending prices in euros. Where does the change go?*
   Only in `TryConvert`: convert currency before dividing. Nothing else changes. That is the proof the boundary is in the right place.

## 2. Decorator

**Story.** A gift (the real service) wrapped in paper (audit), wrapped again in more paper (timer). Each layer looks like a gift from the outside.

**Pseudocode**

```
class AuditedWriter implements GroupWriter:
    inner: GroupWriter
    create_group(input):
        try:
            g = inner.create_group(input)
            log("CreateGroup ok", g.id)
            return g                       # unchanged
        except DomainError as e:
            log("CreateGroup error", e)
            raise                          # unchanged

groups = Timed( Audited( GroupService(store) ) )
```

**Call order for `groups.JoinGroup(...)`**: Timer starts clock → Audit → GroupService → Group.Join (throws or not) → Audit logs → Timer logs time (in `finally`) → caller.

**Socratic questions**

1. *Why not inherit: `class AuditedGroupService : GroupService`?*
   Inheritance fixes the combination at compile time. Audit + timing + retry would need a subclass for every combination (2ⁿ). Decorators combine at runtime: `new A(new B(new Real()))`.
2. *What would break if the audit decorator returned `null` on error instead of rethrowing?*
   The caller would think the join "succeeded with nothing". A decorator must not change the meaning of the call. That is why it uses `throw;`.
3. *`throw;` vs `throw e;`?*
   `throw;` keeps the original stack trace. `throw e;` resets it to the decorator line, hiding where the error happened.
4. *If you swap the order, `Audited(Timed(real))`, what changes in the output?*
   The timer line is printed before the audit line, and the timed duration no longer includes the audit's own logging.
5. *Why do both decorators take `IGroupWriter` and not `GroupService`?*
   So they can wrap each other. If the constructor took `GroupService`, you could only wrap the real service once.
6. *Changed condition: add a decorator that rejects group names with bad words. Is it still a decorator?*
   It forwards only when allowed, so it is closer to a Proxy (access control) or a validator. Be ready to argue that a decorator **adds** and a proxy **guards**.

## 3. Facade

**Story.** A hotel reception desk. You say "check me in". Behind the desk: room system, payment, key cards. You can still go to payments directly; the desk just makes the common case easy.

**Pseudocode**

```
class OnboardingFacade:
    profiles, groups
    onboard(person, group_data):
        draft_p = Profile(person)            # validates, writes nothing
        draft_g = Group(group_data)          # validates, writes nothing
        if draft_p.budget < draft_g.cost: error
        p = profiles.create(person)          # step 1
        g = groups.create(group_data)        # step 2
        groups.join(g.id, p.id)              # step 3
        return (p, g)
```

**Socratic questions**

1. *Is the facade adding business rules?*
   No. Budget, capacity and name rules still live in `Profile` and `Group`. The facade only reuses them early and fixes the order.
2. *Why validate with draft objects first?*
   So "Dan can't afford it" is detected before a profile is written. Otherwise Dan's profile and an empty group would remain.
3. *Is it a transaction? What if step 3 fails anyway?*
   No. If `JoinGroup` failed after steps 1 and 2, those records stay. A real database would wrap the three steps in a transaction. Saying this honestly scores well.
4. *Facade vs Adapter: both put a class in front of others. Difference?*
   Adapter makes **one** object fit an interface the client already expects. Facade creates a **new, simpler** interface over **several** objects.
5. *Can the frontend still call `ProfileService` directly?*
   Yes. A facade does not lock the subsystem; it is a convenience.

## 4. Proxy (bonus)

```
class AdminOnlyWriter implements GroupWriter:
    real, is_admin
    create_group(input):
        if not is_admin(): error "only admins"
        return real.create_group(input)
    join_group(...): return real.join_group(...)
```

*Decorator and Proxy have the same class diagram. How do you tell them apart?* By intent. Decorator always forwards and adds behaviour. Proxy controls access to the real object: protection proxy (this one), virtual proxy (create the expensive object only when first needed), remote proxy (the real object lives on another machine), caching proxy.

*Why `Func<bool>` and not `bool`?* The admin check is re-evaluated on every call. If the user logs out, the proxy sees it immediately.

## The three patterns not implemented

Know the problem, the idea and one example.

**Bridge.** Problem: two independent dimensions multiply classes. `EmailInvite`, `SmsInvite`, `EmailReminder`, `SmsReminder`... Idea: split into two hierarchies and connect them by a field.
```
abstract class Notification { channel: Channel; send() }
class Invite : Notification { send() { channel.deliver("You are invited") } }
interface Channel { deliver(text) }   // EmailChannel, SmsChannel
new Invite(new SmsChannel()).send()
```
n + m classes instead of n × m. *Bridge vs Adapter?* Bridge is designed up front so two sides can vary. Adapter is added afterwards to make two existing things fit.

**Composite.** Problem: a tree where leaves and branches should be treated the same. Idea: both implement one interface; the branch loops over children.
```
interface RentPart { total_rent() }
class Room : RentPart { rent; total_rent() = rent }
class Flat : RentPart { parts; total_rent() = sum(p.total_rent() for p in parts) }
```
A building of flats of rooms answers `total_rent()` with the same call.

**Flyweight.** Problem: millions of small objects repeat the same heavy data. Idea: split state into **intrinsic** (shared, immutable, e.g. neighbourhood name and map polygon) and **extrinsic** (per object, e.g. the listing's price). A factory returns the same shared object for the same key.
```
class NeighbourhoodFactory { cache; get(name) = cache[name] ??= new Neighbourhood(name) }
```
*Why must the shared part be immutable?* If one listing changed it, all listings sharing it would change.

## Rebuild from a blank file (the real test)

Do this without looking, in a new console project, in 20 minutes:

1. Write `interface IGreeter { string Greet(string name); }` and `class Greeter : IGreeter` returning `"Hello " + name`.
2. Decorator: `LoudGreeter(IGreeter inner)` that returns `inner.Greet(name).ToUpper() + "!"`. Stack it twice.
3. Adapter: an `OldGreeter` with `string SayHi(string first, string last)`; write `OldGreeterAdapter : IGreeter` that splits the name.
4. Facade: `WelcomeDesk.Welcome(name)` that greets, then prints the date, then logs, in one call.
5. Proxy: `OnlyWeekdaysGreeter` that refuses on Sunday.

If all five compile and you can explain each in one sentence, you are ready.

## Three-minute presentation script

1. "Domi helps people find roommates. The domain has `Profile`, `Group` and one rule in `Group.Join`. I did not modify the domain; I added four structural patterns around it."
2. Run `dotnet run`. Point at section 1: "The old feed sends whole-flat totals. The adapter converts to per-person cost, rounding 1501/3 up to 501, and rejects the broken row."
3. Section 2: "The writer is wrapped twice: timer, audit, real service. The failed join is logged and the caller still gets the same error."
4. Section 3: "One call onboards Ana. Dan is refused before anything is written, because the facade validates drafts first."
5. Section 4: "Proxy has the same interface as decorator but controls access instead of adding behaviour."
6. Run `dotnet run -- --check`: 13 passed.
7. Close with the table from the top of this file.
