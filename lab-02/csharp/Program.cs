using Domi.Structural;

// Usage:
//   dotnet run            -> demo of all four patterns
//   dotnet run -- --check -> self-checks (exit code 0 = all passed)

if (args.Contains("--check"))
    return SelfCheck.Run();

Demo.Run();
return 0;

static class Demo
{
    public static void Run()
    {
        var store = new InMemoryStore();
        var profiles = new ProfileService(store);
        var log = Console.Out;

        // Composition root: the ONLY place that decides how objects are wired.
        // Timer outside, audit inside, real service in the middle.
        IGroupWriter groups = new TimedGroupWriter(new AuditedGroupWriter(new GroupService(store), log), log);

        Title("1. ADAPTER: import the legacy feed");
        var adapter = new LegacyFeedAdapter(new LegacyHousingFeed());
        var imported = new ListingImporter(adapter, groups).ImportAll();
        foreach (var g in imported) Console.WriteLine($"  imported {g}");
        foreach (var row in adapter.RejectedRows) Console.WriteLine($"  rejected \"{row}\"");

        Title("2. DECORATOR: a failing join is logged and still fails");
        var poor = profiles.CreateProfile(new ProfileInput("Ion", 300));
        try { groups.JoinGroup(imported[0].Id, poor.Id); }
        catch (DomainException e) { Console.WriteLine($"  caller still gets the error: {e.Message}"); }

        Title("3. FACADE: onboard Ana in one call");
        var facade = new OnboardingFacade(profiles, groups);
        var result = facade.Onboard(new ProfileInput("Ana", 700), new GroupInput("Ana's flat", 650, 2));
        Console.WriteLine($"  {result.Profile.Name} [{result.Profile.Id}] is in {result.Group}");
        try { facade.Onboard(new ProfileInput("Dan", 100), new GroupInput("Too pricey", 900, 2)); }
        catch (DomainException e) { Console.WriteLine($"  refused before any write: {e.Message}"); }

        Title("4. PROXY (bonus): non-admin cannot create, can still join");
        var isAdmin = false;
        IGroupWriter guarded = new AdminOnlyGroupWriter(groups, () => isAdmin);
        try { guarded.CreateGroup(new GroupInput("Sneaky", 100, 2)); }
        catch (DomainException e) { Console.WriteLine($"  blocked: {e.Message}"); }
        var rich = profiles.CreateProfile(new ProfileInput("Maria", 900));
        guarded.JoinGroup(imported[1].Id, rich.Id);
        Console.WriteLine($"  Maria joined {store.GetGroup(imported[1].Id)}");

        Title("Store at the end");
        foreach (var g in store.AllGroups) Console.WriteLine($"  {g}");
    }

    static void Title(string text) => Console.WriteLine($"\n=== {text} ===");
}

static class SelfCheck
{
    static int _passed, _failed;

    public static int Run()
    {
        // Adapter
        Check("1500/3 -> 500", Convert("A;1500;3").MaxMonthlyCost == 500);
        Check("1501/3 rounds up to 501", Convert("A;1501;3").MaxMonthlyCost == 501);
        Check("total 0 rejected", !LegacyFeedAdapter.TryConvert("A;0;3", out _));
        Check("21 people rejected", !LegacyFeedAdapter.TryConvert("A;100;21", out _));
        Check("garbage rejected", !LegacyFeedAdapter.TryConvert("nonsense", out _));

        // Decorator: same result, same error, one log line each
        var store = new InMemoryStore();
        var log = new StringWriter();
        IGroupWriter audited = new AuditedGroupWriter(new GroupService(store), log);
        var g = audited.CreateGroup(new GroupInput("G", 500, 1));
        Check("decorator returns the stored group", ReferenceEquals(g, store.GetGroup(g.Id)));
        var p1 = new ProfileService(store).CreateProfile(new ProfileInput("P1", 600));
        var p2 = new ProfileService(store).CreateProfile(new ProfileInput("P2", 600));
        audited.JoinGroup(g.Id, p1.Id);
        Check("decorator rethrows (group full)", Throws(() => audited.JoinGroup(g.Id, p2.Id)));
        Check("three audit lines", log.ToString().Split('\n', StringSplitOptions.RemoveEmptyEntries).Length == 3);

        // Facade: validates before writing
        store = new InMemoryStore();
        var facade = new OnboardingFacade(new ProfileService(store), new GroupService(store));
        Check("facade refuses unaffordable", Throws(() => facade.Onboard(new("Dan", 100), new("X", 900, 2))));
        Check("nothing written after refusal", store.AllGroups.Count == 0);
        var ok = facade.Onboard(new("Ana", 700), new("Flat", 650, 2));
        Check("facade joined the member", ok.Group.MemberIds.Contains(ok.Profile.Id));

        // Proxy
        var admin = false;
        var proxy = new AdminOnlyGroupWriter(new GroupService(store), () => admin);
        Check("proxy blocks non-admin", Throws(() => proxy.CreateGroup(new("Y", 100, 2))));
        admin = true;
        Check("proxy allows admin", proxy.CreateGroup(new("Y", 100, 2)).Name == "Y");

        Console.WriteLine($"\n{_passed} passed, {_failed} failed");
        return _failed == 0 ? 0 : 1;
    }

    static GroupInput Convert(string row) =>
        LegacyFeedAdapter.TryConvert(row, out var input) ? input : throw new Exception($"bad row {row}");

    static bool Throws(Action action)
    {
        try { action(); return false; } catch (DomainException) { return true; }
    }

    static void Check(string name, bool ok)
    {
        if (ok) _passed++; else _failed++;
        Console.WriteLine($"{(ok ? "PASS" : "FAIL")}  {name}");
    }
}
