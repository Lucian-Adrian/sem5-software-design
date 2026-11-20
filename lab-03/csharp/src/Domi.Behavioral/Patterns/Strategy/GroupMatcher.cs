using Domi.Behavioral.Domain;

namespace Domi.Behavioral.Patterns.Strategy;

/// <summary>
/// CONTEXT: filters candidate groups by delegating to the current IGroupMatchingStrategy.
/// The strategy can be swapped at runtime with SetStrategy without editing this class.
/// </summary>
public sealed class GroupMatcher(IGroupMatchingStrategy initialStrategy)
{
    public IGroupMatchingStrategy Strategy { get; private set; } =
        initialStrategy ?? throw new ArgumentNullException(nameof(initialStrategy));

    public void SetStrategy(IGroupMatchingStrategy strategy)
    {
        Strategy = strategy ?? throw new ArgumentNullException(nameof(strategy));
    }

    public IReadOnlyList<Group> FindMatches(IEnumerable<Group> groups, Profile profile) =>
        groups.Where(group => Strategy.Matches(group, profile)).ToList();
}
