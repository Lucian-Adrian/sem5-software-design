using System.Text;

namespace Domi;

// ---------- Singleton ----------
// One shared, read-only set of catalog defaults for the whole program.
// Fits fixed settings only. Per-user data (groups, profiles) must never live here.
public sealed class CatalogDefaults
{
    // Lazy<T> creates the instance once, on first use, and is thread-safe.
    private static readonly Lazy<CatalogDefaults> _instance = new(() => new CatalogDefaults());

    public static CatalogDefaults Instance => _instance.Value;

    // Private constructor: nobody else can call `new CatalogDefaults()`.
    private CatalogDefaults() { }

    public int DefaultCapacity { get; } = 3;
    public int DefaultMaxMonthlyCost { get; } = 500;
}

// ---------- Factory Method ----------
// The base class owns the workflow (create group, save profile, join).
// Each subclass overrides one step: which group draft to create.
public abstract class GroupCreator
{
    public GroupView CreateAndJoin(GroupService groups, ProfileService profiles, ProfileInput founder)
    {
        GroupInput draft = CreateGroupDraft();          // the factory method
        GroupView group = groups.CreateGroup(draft);
        ProfileView profile = profiles.CreateProfile(founder);
        return groups.JoinGroup(group.Id, profile.Id);
    }

    protected abstract GroupInput CreateGroupDraft();
}

public sealed class StudentGroupCreator : GroupCreator
{
    protected override GroupInput CreateGroupDraft() => new GroupBuilder()
        .WithName("Student flat")
        .WithMaxMonthlyCost(300)
        .WithCapacity(CatalogDefaults.Instance.DefaultCapacity + 2)
        .Build();
}

public sealed class QuietGroupCreator : GroupCreator
{
    protected override GroupInput CreateGroupDraft() => new GroupBuilder()
        .WithName("Quiet home")
        .WithMaxMonthlyCost(CatalogDefaults.Instance.DefaultMaxMonthlyCost)
        .WithCapacity(2)
        .Build();
}

// ---------- Object Pool ----------
// Reuses StringBuilders for group summaries instead of allocating a new one each time.
// Release() clears the builder before it goes back, so old text never leaks into the next use.
public sealed class SummaryBuilderPool(int maxIdle)
{
    private readonly Stack<StringBuilder> _idle = new();

    public int Created { get; private set; }

    public StringBuilder Acquire()
    {
        if (_idle.Count > 0) return _idle.Pop();
        Created++;
        return new StringBuilder();
    }

    public void Release(StringBuilder builder)
    {
        builder.Clear();
        if (_idle.Count < maxIdle) _idle.Push(builder);  // past the limit, let GC take it
    }

    public string Summarize(GroupView group)
    {
        StringBuilder sb = Acquire();
        try
        {
            return sb.Append(group.Name).Append(": ")
                .Append(group.MemberCount).Append('/').Append(group.Capacity)
                .Append(" members, ").Append(group.MaxMonthlyCost).Append(" per month")
                .ToString();
        }
        finally
        {
            Release(sb);
        }
    }
}
