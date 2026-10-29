namespace Domi;

// Thrown when a rule rejects input. Go returns an error value; C# throws.
public sealed class DomainException(string message) : Exception(message);

// Creation data: what a caller sends before the store assigns an ID.
public sealed record GroupInput(string Name, int MaxMonthlyCost, int Capacity);

public sealed record ProfileInput(string Name, int MonthlyBudget);

// Public views: what the API shows. The profile budget stays private.
public sealed record GroupView(string Id, string Name, int MaxMonthlyCost, int Capacity, int MemberCount);

public sealed record ProfileView(string Id, string Name);

public sealed class Group
{
    public const int MaxCapacity = 20;

    private readonly List<string> _memberProfileIds = new();

    public Group(string id, GroupInput input)
    {
        GroupInput clean = Normalize(input);
        Id = id;
        Name = clean.Name;
        MaxMonthlyCost = clean.MaxMonthlyCost;
        Capacity = clean.Capacity;
    }

    public string Id { get; internal set; }
    public string Name { get; internal set; }
    public int MaxMonthlyCost { get; }
    public int Capacity { get; }
    public IReadOnlyList<string> MemberProfileIds => _memberProfileIds;

    // Builder, store and Prototype all use these rules, so they reject the same input.
    public static GroupInput Normalize(GroupInput input)
    {
        string name = (input.Name ?? "").Trim();
        if (name == "") throw new DomainException("group name is required");
        if (input.MaxMonthlyCost <= 0) throw new DomainException("maxMonthlyCost must be greater than zero");
        if (input.Capacity <= 0) throw new DomainException("capacity must be greater than zero");
        if (input.Capacity > MaxCapacity) throw new DomainException($"capacity cannot exceed {MaxCapacity} people");
        return input with { Name = name };
    }

    public void AddMember(Profile profile)
    {
        // Repeating a finished join succeeds even if the group is now full.
        if (_memberProfileIds.Contains(profile.Id)) return;

        if (profile.MonthlyBudget < MaxMonthlyCost) throw new DomainException("monthly budget is below the group's cost");
        if (_memberProfileIds.Count >= Capacity) throw new DomainException("group is full");

        _memberProfileIds.Add(profile.Id);
    }

    // Used by Prototype. Copies the member list so the copy never shares it with the source.
    internal Group Copy(string id, string name, IEnumerable<string> memberIds)
    {
        Group copy = new(id, new GroupInput(name, MaxMonthlyCost, Capacity));
        copy._memberProfileIds.AddRange(memberIds);
        return copy;
    }

    public GroupInput ToInput() => new(Name, MaxMonthlyCost, Capacity);

    public GroupView View() => new(Id, Name, MaxMonthlyCost, Capacity, _memberProfileIds.Count);
}

public sealed class Profile
{
    public Profile(string id, ProfileInput input)
    {
        string name = (input.Name ?? "").Trim();
        if (name == "") throw new DomainException("profile name is required");
        if (input.MonthlyBudget < 0) throw new DomainException("monthlyBudget cannot be negative");

        Id = id;
        Name = name;
        MonthlyBudget = input.MonthlyBudget;
    }

    public string Id { get; }
    public string Name { get; }
    public int MonthlyBudget { get; }

    public ProfileView View() => new(Id, Name);
}
