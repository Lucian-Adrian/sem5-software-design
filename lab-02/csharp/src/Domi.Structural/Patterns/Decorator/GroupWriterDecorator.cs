using Domi.Structural.Domain;

namespace Domi.Structural.Patterns.Decorator;

// ===========================================================================
// DECORATOR
// Component:          IGroupWriter
// ConcreteComponent:  GroupService (Domain/Services.cs)
// Base decorator:     GroupWriterDecorator - holds the inner writer and forwards
// Concrete decorators: AuditedGroupWriter, TimedGroupWriter
// A new concern (retry, caching...) is one new subclass; nothing else changes.
// ===========================================================================

/// <summary>BASE DECORATOR: same interface as the component, forwards by default.</summary>
public abstract class GroupWriterDecorator(IGroupWriter inner) : IGroupWriter
{
    protected IGroupWriter Inner { get; } = inner;

    public virtual Group CreateGroup(GroupInput input) => Inner.CreateGroup(input);

    public virtual Group JoinGroup(string groupId, string profileId) => Inner.JoinGroup(groupId, profileId);
}

/// <summary>Writes one line per operation (ok / error). Never changes the result.</summary>
public sealed class AuditedGroupWriter(IGroupWriter inner, TextWriter log) : GroupWriterDecorator(inner)
{
    public override Group CreateGroup(GroupInput input) =>
        Record("CreateGroup", () => base.CreateGroup(input));

    public override Group JoinGroup(string groupId, string profileId) =>
        Record("JoinGroup", () => base.JoinGroup(groupId, profileId));

    private Group Record(string operation, Func<Group> call)
    {
        try
        {
            var group = call();
            log.WriteLine($"[audit] {operation} ok -> {group.Id}");
            return group;   // the same object the inner writer returned
        }
        catch (DomainException e)
        {
            log.WriteLine($"[audit] {operation} error: {e.Message}");
            throw;          // the same exception, stack trace kept
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

    private Group Time(string operation, Func<Group> call)
    {
        var watch = System.Diagnostics.Stopwatch.StartNew();
        try { return call(); }
        finally { log.WriteLine($"[timer] {operation} took {watch.Elapsed.TotalMilliseconds:F3} ms"); }
    }
}
