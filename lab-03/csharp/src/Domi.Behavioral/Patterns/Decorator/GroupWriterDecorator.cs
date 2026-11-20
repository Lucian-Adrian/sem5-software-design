using Domi.Behavioral.Domain;

namespace Domi.Behavioral.Patterns.Decorator;

// ===========================================================================
// DECORATOR (reused from Lab 2, extended with LeaveGroup & DeleteGroup)
// Component:           IGroupWriter
// ConcreteComponent:   GroupService
// Base decorator:      GroupWriterDecorator
// Concrete decorators: AuditedGroupWriter, TimedGroupWriter
// ===========================================================================

/// <summary>BASE DECORATOR: same interface as the component, forwards by default.</summary>
public abstract class GroupWriterDecorator(IGroupWriter inner) : IGroupWriter
{
    protected IGroupWriter Inner { get; } = inner;

    public virtual Group CreateGroup(GroupInput input) => Inner.CreateGroup(input);

    public virtual Group JoinGroup(string groupId, string profileId) => Inner.JoinGroup(groupId, profileId);

    public virtual Group LeaveGroup(string groupId, string profileId) => Inner.LeaveGroup(groupId, profileId);

    public virtual Group DeleteGroup(string groupId) => Inner.DeleteGroup(groupId);
}

/// <summary>Writes one line per operation (ok / error). Never changes the result.</summary>
public sealed class AuditedGroupWriter(IGroupWriter inner, TextWriter log) : GroupWriterDecorator(inner)
{
    public override Group CreateGroup(GroupInput input) =>
        Record("CreateGroup", () => base.CreateGroup(input));

    public override Group JoinGroup(string groupId, string profileId) =>
        Record("JoinGroup", () => base.JoinGroup(groupId, profileId));

    public override Group LeaveGroup(string groupId, string profileId) =>
        Record("LeaveGroup", () => base.LeaveGroup(groupId, profileId));

    public override Group DeleteGroup(string groupId) =>
        Record("DeleteGroup", () => base.DeleteGroup(groupId));

    private Group Record(string operation, Func<Group> call)
    {
        try
        {
            var group = call();
            log.WriteLine($"[audit] {operation} ok -> {group.Id}");
            return group;
        }
        catch (DomainException e)
        {
            log.WriteLine($"[audit] {operation} error: {e.Message}");
            throw;
        }
    }
}

/// <summary>Measures how long the wrapped call took, including failed calls.</summary>
public sealed class TimedGroupWriter(IGroupWriter inner, TextWriter log) : GroupWriterDecorator(inner)
{
    public override Group CreateGroup(GroupInput input) =>
        Time("CreateGroup", () => base.CreateGroup(input));

    public override Group JoinGroup(string groupId, string profileId) =>
        Time("JoinGroup", () => base.JoinGroup(groupId, profileId));

    public override Group LeaveGroup(string groupId, string profileId) =>
        Time("LeaveGroup", () => base.LeaveGroup(groupId, profileId));

    public override Group DeleteGroup(string groupId) =>
        Time("DeleteGroup", () => base.DeleteGroup(groupId));

    private Group Time(string operation, Func<Group> call)
    {
        var watch = System.Diagnostics.Stopwatch.StartNew();
        try { return call(); }
        finally { log.WriteLine($"[timer] {operation} took {watch.Elapsed.TotalMilliseconds:F3} ms"); }
    }
}
