namespace Domi.Structural;

// ---------------------------------------------------------------------------
// The existing Domi domain. Nothing here knows about design patterns.
// The patterns in Adapter.cs, Decorator.cs, Facade.cs and Proxy.cs are built
// AROUND these classes without editing them. That is the point of a
// structural pattern: change how parts connect, not what the parts are.
// ---------------------------------------------------------------------------

/// <summary>Thrown when input breaks a domain rule (empty name, full group...).</summary>
public class DomainException(string message) : Exception(message);

/// <summary>Data a caller sends to create a profile.</summary>
public record ProfileInput(string Name, int MonthlyBudget);

/// <summary>Data a caller sends to create a group. Cost is PER PERSON.</summary>
public record GroupInput(string Name, int MaxMonthlyCost, int Capacity);

/// <summary>A person looking for roommates.</summary>
public class Profile
{
    public string Id { get; }
    public string Name { get; }
    public int MonthlyBudget { get; }

    public Profile(string id, ProfileInput input)
    {
        if (string.IsNullOrWhiteSpace(input.Name))
            throw new DomainException("profile name is required");
        if (input.MonthlyBudget < 0)
            throw new DomainException("monthly budget cannot be negative");

        Id = id;
        Name = input.Name.Trim();
        MonthlyBudget = input.MonthlyBudget;
    }
}

/// <summary>A roommate group. It owns its membership rule.</summary>
public class Group
{
    public const int MaxCapacity = 20;

    private readonly List<string> _memberIds = [];

    public string Id { get; }
    public string Name { get; }
    public int MaxMonthlyCost { get; }
    public int Capacity { get; }
    public IReadOnlyList<string> MemberIds => _memberIds;

    public Group(string id, GroupInput input)
    {
        if (string.IsNullOrWhiteSpace(input.Name))
            throw new DomainException("group name is required");
        if (input.MaxMonthlyCost <= 0)
            throw new DomainException("monthly cost must be greater than zero");
        if (input.Capacity <= 0 || input.Capacity > MaxCapacity)
            throw new DomainException($"capacity must be between 1 and {MaxCapacity}");

        Id = id;
        Name = input.Name.Trim();
        MaxMonthlyCost = input.MaxMonthlyCost;
        Capacity = input.Capacity;
    }

    /// <summary>The membership rule: not full, affordable, not already inside.</summary>
    public void Join(Profile profile)
    {
        if (_memberIds.Contains(profile.Id))
            throw new DomainException($"{profile.Name} is already in {Name}");
        if (_memberIds.Count >= Capacity)
            throw new DomainException($"{Name} is full");
        if (profile.MonthlyBudget < MaxMonthlyCost)
            throw new DomainException(
                $"{profile.Name}'s budget {profile.MonthlyBudget} is below {Name}'s cost {MaxMonthlyCost}");

        _memberIds.Add(profile.Id);
    }

    public override string ToString() =>
        $"{Name} [{Id}] cost={MaxMonthlyCost}/person members={_memberIds.Count}/{Capacity}";
}

/// <summary>In-memory storage. Records disappear when the program exits.</summary>
public class InMemoryStore
{
    private readonly Dictionary<string, Profile> _profiles = [];
    private readonly Dictionary<string, Group> _groups = [];
    private int _nextId = 1;

    public string NextId(string prefix) => $"{prefix}-{_nextId++}";

    public void Save(Profile profile) => _profiles[profile.Id] = profile;
    public void Save(Group group) => _groups[group.Id] = group;

    public Profile GetProfile(string id) =>
        _profiles.TryGetValue(id, out var p) ? p : throw new DomainException($"profile {id} not found");

    public Group GetGroup(string id) =>
        _groups.TryGetValue(id, out var g) ? g : throw new DomainException($"group {id} not found");

    public IReadOnlyCollection<Group> AllGroups => _groups.Values;
}

// ---------------------------------------------------------------------------
// The interface that Decorator and Proxy will both implement.
// Anyone who writes groups depends on THIS, not on GroupService directly.
// ---------------------------------------------------------------------------
public interface IGroupWriter
{
    Group CreateGroup(GroupInput input);
    Group JoinGroup(string groupId, string profileId);
}

/// <summary>The real group service: validate, store, return.</summary>
public class GroupService(InMemoryStore store) : IGroupWriter
{
    public Group CreateGroup(GroupInput input)
    {
        var group = new Group(store.NextId("group"), input); // constructor validates
        store.Save(group);
        return group;
    }

    public Group JoinGroup(string groupId, string profileId)
    {
        var group = store.GetGroup(groupId);
        var profile = store.GetProfile(profileId);
        group.Join(profile); // the domain decides
        return group;
    }
}

/// <summary>The real profile service.</summary>
public class ProfileService(InMemoryStore store)
{
    public Profile CreateProfile(ProfileInput input)
    {
        var profile = new Profile(store.NextId("profile"), input);
        store.Save(profile);
        return profile;
    }
}
