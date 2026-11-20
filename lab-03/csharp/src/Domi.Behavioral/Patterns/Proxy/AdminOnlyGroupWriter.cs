using Domi.Behavioral.Domain;

namespace Domi.Behavioral.Patterns.Proxy;

// ===========================================================================
// PROXY (protection proxy, reused from Lab 2)
// Only admins may create or delete groups; joining and leaving pass through.
// ===========================================================================

public interface ICurrentUser
{
    string Name { get; }
    bool IsAdmin { get; }
}

public sealed class AdminOnlyGroupWriter(IGroupWriter real, ICurrentUser user) : IGroupWriter
{
    public Group CreateGroup(GroupInput input)
    {
        if (!user.IsAdmin)
            throw new UnauthorizedAccessException($"{user.Name} is not an admin and cannot create groups");
        return real.CreateGroup(input);
    }

    public Group JoinGroup(string groupId, string profileId) => real.JoinGroup(groupId, profileId);

    public Group LeaveGroup(string groupId, string profileId) => real.LeaveGroup(groupId, profileId);

    public Group DeleteGroup(string groupId)
    {
        if (!user.IsAdmin)
            throw new UnauthorizedAccessException($"{user.Name} is not an admin and cannot delete groups");
        return real.DeleteGroup(groupId);
    }
}

/// <summary>Simple session user for the demo; a real app would read it from authentication.</summary>
public sealed class SessionUser(string name, bool isAdmin) : ICurrentUser
{
    public string Name { get; } = name;
    public bool IsAdmin { get; set; } = isAdmin;
}
