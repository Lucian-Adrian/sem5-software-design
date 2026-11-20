using Domi.Behavioral.Domain;
using Domi.Behavioral.Patterns.Observer;

namespace Domi.Behavioral.Patterns.Command;

/// <summary>
/// CONCRETE COMMAND: asks an IGroupWriter receiver to create a group,
/// and deletes that created group on Undo.
/// </summary>
public sealed class CreateGroupCommand(IGroupWriter receiver, GroupInput input) : IGroupCommand
{
    private string? _createdGroupId;

    public string Description => $"Create group '{input.Name}'";

    public GroupActivityEvent Execute()
    {
        if (_createdGroupId is not null)
            throw new InvalidOperationException("create command has already been executed");

        var group = receiver.CreateGroup(input);
        _createdGroupId = group.Id;
        return new GroupActivityEvent(
            "Created",
            group.Id,
            group.Name,
            group.MemberIds.Count,
            group.Capacity);
    }

    public GroupActivityEvent Undo()
    {
        if (_createdGroupId is null)
            throw new InvalidOperationException("cannot undo a create command that has not executed");

        var group = receiver.DeleteGroup(_createdGroupId);
        _createdGroupId = null;
        return new GroupActivityEvent(
            "UndidCreate",
            group.Id,
            group.Name,
            0,
            group.Capacity);
    }
}
