namespace Domi;

// In-memory storage, same as the Go lab. Records disappear when the program stops.
public sealed class ProfileService
{
    private readonly Dictionary<string, Profile> _profiles = new();
    private int _nextId = 1;

    public ProfileView CreateProfile(ProfileInput input)
    {
        Profile profile = new($"profile-{_nextId++}", input);
        _profiles[profile.Id] = profile;
        return profile.View();
    }

    public Profile Get(string id) =>
        _profiles.TryGetValue(id, out Profile? profile) ? profile : throw new DomainException("profile not found");
}

public sealed class GroupService(ProfileService profiles)
{
    private readonly Dictionary<string, Group> _groups = new();
    private int _nextId = 1;

    public GroupView CreateGroup(GroupInput input)
    {
        Group group = new($"group-{_nextId++}", input);
        _groups[group.Id] = group;
        return group.View();
    }

    public Group Get(string id) =>
        _groups.TryGetValue(id, out Group? group) ? group : throw new DomainException("group not found");

    public GroupView JoinGroup(string groupId, string profileId)
    {
        Group group = Get(groupId);
        group.AddMember(profiles.Get(profileId));
        return group.View();
    }
}
