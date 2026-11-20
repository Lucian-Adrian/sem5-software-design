using Domi.Behavioral.Domain;
using Domi.Behavioral.Patterns.Command;
using Domi.Behavioral.Patterns.Observer;
using Domi.Behavioral.Patterns.Strategy;

namespace Domi.Behavioral.Client;

/// <summary>
/// Connects Strategy, Command, the Lab 2 IGroupWriter chain, and Observer on one path:
///   GroupMatcher (Strategy) -> JoinGroupCommand -> GroupCommandInvoker
///   -> IGroupWriter (Proxy -> Decorators -> GroupService) -> IGroupActivitySubject (Observers).
/// </summary>
public sealed class GroupMatchingWorkflow(
    GroupMatcher matcher,
    GroupCommandInvoker invoker,
    IGroupWriter groups)
{
    public GroupActivityEvent? JoinBestMatch(IEnumerable<Group> catalog, Profile seeker)
    {
        var matches = matcher.FindMatches(catalog, seeker);
        if (matches.Count == 0)
            return null;

        var chosen = matches[0];
        var command = new JoinGroupCommand(groups, chosen.Id, seeker.Id);
        return invoker.Run(command);
    }
}
