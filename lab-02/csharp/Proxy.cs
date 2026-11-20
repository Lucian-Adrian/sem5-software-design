namespace Domi.Structural;

// ===========================================================================
// PROXY (bonus, 4th pattern)
// Problem: only admins may create groups, but GroupService should not know
//          about users and roles.
// Idea:    a stand-in with the SAME interface that controls ACCESS to the real
//          object. Same shape as Decorator; different intent:
//            Decorator = ADD behaviour, always forwards.
//            Proxy     = CONTROL access, may refuse and never forward.
// ===========================================================================

public class AdminOnlyGroupWriter(IGroupWriter real, Func<bool> currentUserIsAdmin) : IGroupWriter
{
    public Group CreateGroup(GroupInput input)
    {
        if (!currentUserIsAdmin())
            throw new DomainException("only admins can create groups");
        return real.CreateGroup(input);
    }

    // Joining is allowed for everyone, so the proxy just passes it through.
    public Group JoinGroup(string groupId, string profileId) => real.JoinGroup(groupId, profileId);
}
