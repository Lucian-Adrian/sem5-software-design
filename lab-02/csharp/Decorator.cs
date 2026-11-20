using System.Diagnostics;

namespace Domi.Structural;

// ===========================================================================
// DECORATOR
// Problem: we want every group write to be logged, and also timed,
//          but GroupService should keep doing ONE job (domain writes).
//          Subclassing GroupService for each combination
//          (Logged, Timed, LoggedAndTimed...) explodes.
// Idea:    wrap the object in another object with the SAME interface.
//          The wrapper does its extra step and forwards to the inner one.
//          Wrappers stack in any order at runtime.
//
//   IGroupWriter <---- GroupService            (the real work)
//        ^      <---- AuditedGroupWriter  --has-a--> IGroupWriter (inner)
//        |      <---- TimedGroupWriter    --has-a--> IGroupWriter (inner)
// ===========================================================================

/// <summary>Writes one line per operation: name + success/error. Never changes the result.</summary>
public class AuditedGroupWriter(IGroupWriter inner, TextWriter log) : IGroupWriter
{
    public Group CreateGroup(GroupInput input) =>
        Record("CreateGroup", () => inner.CreateGroup(input));

    public Group JoinGroup(string groupId, string profileId) =>
        Record("JoinGroup", () => inner.JoinGroup(groupId, profileId));

    private Group Record(string operation, Func<Group> call)
    {
        try
        {
            var group = call();
            log.WriteLine($"[audit] {operation} ok -> {group.Id}");
            return group;               // same object the inner writer returned
        }
        catch (DomainException e)
        {
            log.WriteLine($"[audit] {operation} error: {e.Message}");
            throw;                      // same exception, the caller still sees it
        }
    }
}

/// <summary>Measures how long the inner call took. Second decorator to show stacking.</summary>
public class TimedGroupWriter(IGroupWriter inner, TextWriter log) : IGroupWriter
{
    public Group CreateGroup(GroupInput input) => Time("CreateGroup", () => inner.CreateGroup(input));

    public Group JoinGroup(string groupId, string profileId) =>
        Time("JoinGroup", () => inner.JoinGroup(groupId, profileId));

    private Group Time(string operation, Func<Group> call)
    {
        var watch = Stopwatch.StartNew();
        try { return call(); }
        finally { log.WriteLine($"[timer] {operation} took {watch.Elapsed.TotalMilliseconds:F3} ms"); }
    }
}
