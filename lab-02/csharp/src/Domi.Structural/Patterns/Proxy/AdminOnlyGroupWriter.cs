using Domi.Structural.Domain;

namespace Domi.Structural.Patterns.Proxy;

// ===========================================================================
// PROXY (protection proxy)
// Same interface as the real writer, but it CONTROLS ACCESS:
// only admins may create groups; joining passes through unchanged.
// Decorator = always forwards and adds behaviour.
// Proxy     = decides whether the real object is reached at all.
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
}

/// <summary>Simple session user for the demo; a real app would read it from authentication.</summary>
public sealed class SessionUser(string name, bool isAdmin) : ICurrentUser
{
    public string Name { get; } = name;
    public bool IsAdmin { get; set; } = isAdmin;
}
