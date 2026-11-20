namespace Domi.Behavioral.Domain;

/// <summary>In-memory storage. Records disappear when the program exits.</summary>
public class InMemoryStore
{
    private readonly Dictionary<string, Profile> _profiles = [];
    private readonly Dictionary<string, Group> _groups = [];
    private int _nextId = 1;

    public string NextId(string prefix) => $"{prefix}-{_nextId++}";

    public void Save(Profile profile) => _profiles[profile.Id] = profile;
    public void Save(Group group) => _groups[group.Id] = group;

    public void RemoveGroup(string id)
    {
        if (!_groups.Remove(id))
            throw new DomainException($"group {id} not found");
    }

    public Profile GetProfile(string id) =>
        _profiles.TryGetValue(id, out var p) ? p : throw new DomainException($"profile {id} not found");

    public Group GetGroup(string id) =>
        _groups.TryGetValue(id, out var g) ? g : throw new DomainException($"group {id} not found");

    public IReadOnlyCollection<Group> AllGroups => _groups.Values;
    public IReadOnlyCollection<Profile> AllProfiles => _profiles.Values;
}

/// <summary>
/// The abstraction every group-writing client and Command receiver depends on.
/// GroupService, the decorators and the proxy all implement it.
/// </summary>
public interface IGroupWriter
{
    Group CreateGroup(GroupInput input);
    Group JoinGroup(string groupId, string profileId);
    Group LeaveGroup(string groupId, string profileId);
    Group DeleteGroup(string groupId);
}

/// <summary>The abstraction for creating profiles, so the facade does not depend on a concrete service.</summary>
public interface IProfileWriter
{
    Profile CreateProfile(ProfileInput input);
}

/// <summary>The real group service: validate, store, return.</summary>
public class GroupService(InMemoryStore store) : IGroupWriter
{
    public Group CreateGroup(GroupInput input)
    {
        var group = new Group(store.NextId("group"), input); // constructor validates
        store.Save(group);
        return group;
    }

    public Group JoinGroup(string groupId, string profileId)
    {
        var group = store.GetGroup(groupId);
        var profile = store.GetProfile(profileId);
        group.Join(profile); // the domain decides
        return group;
    }

    public Group LeaveGroup(string groupId, string profileId)
    {
        var group = store.GetGroup(groupId);
        var profile = store.GetProfile(profileId);
        group.Leave(profile); // the domain decides
        return group;
    }

    public Group DeleteGroup(string groupId)
    {
        var group = store.GetGroup(groupId);
        if (group.MemberIds.Count > 0)
            throw new DomainException($"cannot delete {group.Name} while it still has members");
        store.RemoveGroup(groupId);
        return group;
    }
}

/// <summary>The real profile service.</summary>
public class ProfileService(InMemoryStore store) : IProfileWriter
{
    public Profile CreateProfile(ProfileInput input)
    {
        var profile = new Profile(store.NextId("profile"), input);
        store.Save(profile);
        return profile;
    }
}
