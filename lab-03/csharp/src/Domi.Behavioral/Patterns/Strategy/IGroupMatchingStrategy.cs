using Domi.Behavioral.Domain;

namespace Domi.Behavioral.Patterns.Strategy;

// ===========================================================================
// STRATEGY
// Strategy:          IGroupMatchingStrategy
// ConcreteStrategy:  AffordableWithSpaceStrategy, BudgetBufferStrategy,
//                    SmallGroupStrategy
// Context:           GroupMatcher (GroupMatcher.cs)
// No if/else in the context; strategies share one interface and swap at runtime.
// ===========================================================================

/// <summary>STRATEGY: interchangeable rule for deciding whether a group fits a profile.</summary>
public interface IGroupMatchingStrategy
{
    string Name { get; }
    bool Matches(Group group, Profile profile);
}

/// <summary>Matches any group with free space whose per-person cost is within budget.</summary>
public sealed class AffordableWithSpaceStrategy : IGroupMatchingStrategy
{
    public string Name => "Affordable with space";

    public bool Matches(Group group, Profile profile) =>
        group.MaxMonthlyCost <= profile.MonthlyBudget &&
        group.MemberIds.Count < group.Capacity;
}

/// <summary>Matches groups that leave at least a fixed monthly reserve unspent.</summary>
public sealed class BudgetBufferStrategy : IGroupMatchingStrategy
{
    public int ReserveAmount { get; }

    public BudgetBufferStrategy(int reserveAmount)
    {
        if (reserveAmount < 0)
            throw new ArgumentOutOfRangeException(nameof(reserveAmount), "reserve cannot be negative");
        ReserveAmount = reserveAmount;
    }

    public string Name => $"Budget buffer (>= {ReserveAmount} left)";

    public bool Matches(Group group, Profile profile) =>
        group.MaxMonthlyCost <= profile.MonthlyBudget - ReserveAmount &&
        group.MemberIds.Count < group.Capacity;
}

/// <summary>Matches affordable groups whose total capacity stays at or below a limit.</summary>
public sealed class SmallGroupStrategy : IGroupMatchingStrategy
{
    public int MaxCapacity { get; }

    public SmallGroupStrategy(int maxCapacity)
    {
        if (maxCapacity <= 0)
            throw new ArgumentOutOfRangeException(nameof(maxCapacity), "capacity limit must be positive");
        MaxCapacity = maxCapacity;
    }

    public string Name => $"Small group (<= {MaxCapacity} people)";

    public bool Matches(Group group, Profile profile) =>
        group.Capacity <= MaxCapacity &&
        group.MaxMonthlyCost <= profile.MonthlyBudget &&
        group.MemberIds.Count < group.Capacity;
}
