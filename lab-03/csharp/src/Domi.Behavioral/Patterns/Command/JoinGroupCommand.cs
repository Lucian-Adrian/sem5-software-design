using Domi.Behavioral.Domain;
using Domi.Behavioral.Patterns.Observer;

namespace Domi.Behavioral.Patterns.Command;

/// <summary>
/// CONCRETE COMMAND: asks an IGroupWriter receiver to add a profile to a group,
/// and removes that profile on Undo.
/// </summary>
public sealed class JoinGroupCommand(IGroupWriter receiver, string groupId, string profileId) : IGroupCommand
{
    private bool _executed;

    public string Description => $"Join {profileId} -> {groupId}";

    public GroupActivityEvent Execute()
    {
        if (_executed)
            throw new InvalidOperationException("join command has already been executed");

        var group = receiver.JoinGroup(groupId, profileId);
        _executed = true;
        return new GroupActivityEvent(
            "Joined",
            group.Id,
            group.Name,
            group.MemberIds.Count,
            group.Capacity,
            profileId);
    }

    public GroupActivityEvent Undo()
    {
        if (!_executed)
            throw new InvalidOperationException("cannot undo a join command that has not executed");

        var group = receiver.LeaveGroup(groupId, profileId);
        _executed = false;
        return new GroupActivityEvent(
            "UndidJoin",
            group.Id,
            group.Name,
            group.MemberIds.Count,
            group.Capacity,
            profileId);
    }
}
