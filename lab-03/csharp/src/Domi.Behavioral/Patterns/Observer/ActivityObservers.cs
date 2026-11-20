namespace Domi.Behavioral.Patterns.Observer;

/// <summary>CONCRETE OBSERVER: writes one readable line per group activity event.</summary>
public sealed class ActivityLogObserver(TextWriter output) : IGroupActivityObserver
{
    public void OnGroupChanged(GroupActivityEvent activity)
    {
        var who = activity.ProfileId is null ? "" : $" profile={activity.ProfileId}";
        output.WriteLine(
            $"[observer:log] {activity.Action} {activity.GroupName} [{activity.GroupId}]" +
            $" members={activity.MemberCount}/{activity.Capacity}{who}");
    }
}

/// <summary>
/// CONCRETE OBSERVER: announces when a group becomes full, and when an undo
/// opens a spot again.
/// </summary>
public sealed class FullGroupAlertObserver(TextWriter output) : IGroupActivityObserver
{
    private readonly HashSet<string> _fullGroups = [];

    public void OnGroupChanged(GroupActivityEvent activity)
    {
        if (activity.Action == "UndidCreate")
        {
            _fullGroups.Remove(activity.GroupId);
            return;
        }

        if (activity.MemberCount >= activity.Capacity && _fullGroups.Add(activity.GroupId))
        {
            output.WriteLine(
                $"[observer:alert] {activity.GroupName} [{activity.GroupId}] is now FULL" +
                $" ({activity.MemberCount}/{activity.Capacity})");
        }
        else if (activity.MemberCount < activity.Capacity && _fullGroups.Remove(activity.GroupId))
        {
            output.WriteLine(
                $"[observer:alert] {activity.GroupName} [{activity.GroupId}] has a free spot again" +
                $" ({activity.MemberCount}/{activity.Capacity})");
        }
    }
}

/// <summary>CONCRETE OBSERVER: keeps an in-memory log of every received event.</summary>
public sealed class ActivityHistoryObserver : IGroupActivityObserver
{
    private readonly List<GroupActivityEvent> _events = [];

    public IReadOnlyList<GroupActivityEvent> Events => _events;

    public void OnGroupChanged(GroupActivityEvent activity) => _events.Add(activity);
}
