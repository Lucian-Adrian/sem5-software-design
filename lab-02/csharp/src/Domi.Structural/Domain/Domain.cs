namespace Domi.Structural.Domain;

// The existing Domi domain. Nothing here knows about design patterns.
// The patterns are built AROUND these classes without editing them.

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

    /// <summary>The membership rule: not already inside, not full, affordable.</summary>
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
