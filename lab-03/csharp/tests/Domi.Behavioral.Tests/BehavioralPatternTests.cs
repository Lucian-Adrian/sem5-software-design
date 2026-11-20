using Domi.Behavioral.Client;
using Domi.Behavioral.Domain;
using Domi.Behavioral.Patterns.Adapter;
using Domi.Behavioral.Patterns.Command;
using Domi.Behavioral.Patterns.Decorator;
using Domi.Behavioral.Patterns.Facade;
using Domi.Behavioral.Patterns.Observer;
using Domi.Behavioral.Patterns.Proxy;
using Domi.Behavioral.Patterns.Strategy;

namespace Domi.Behavioral.Tests;

public class StrategyTests
{
    private static (List<Group> groups, Profile seeker) SampleData()
    {
        var store = new InMemoryStore();
        var groups = new GroupService(store);
        var profiles = new ProfileService(store);

        var cheap = groups.CreateGroup(new GroupInput("Riscani", 500, 3));
        var edge = groups.CreateGroup(new GroupInput("Botanica", 501, 3));
        var small = groups.CreateGroup(new GroupInput("Center", 450, 2));
        var full = groups.CreateGroup(new GroupInput("Full studio", 400, 1));

        var filler = profiles.CreateProfile(new ProfileInput("Filler", 800));
        groups.JoinGroup(full.Id, filler.Id);

        var seeker = profiles.CreateProfile(new ProfileInput("Vlad", 550));
        return ([cheap, edge, small, full], seeker);
    }

    [Fact]
    public void AffordableWithSpaceStrategy_matches_only_affordable_groups_with_free_places()
    {
        var (groups, seeker) = SampleData();
        var matcher = new GroupMatcher(new AffordableWithSpaceStrategy());

        var matches = matcher.FindMatches(groups, seeker);

        Assert.Equal(["Riscani", "Botanica", "Center"], matches.Select(g => g.Name).ToArray());
    }

    [Fact]
    public void BudgetBufferStrategy_requires_extra_margin_above_group_cost()
    {
        var (groups, seeker) = SampleData();
        var matcher = new GroupMatcher(new BudgetBufferStrategy(50));

        var matches = matcher.FindMatches(groups, seeker);

        Assert.Equal(["Riscani", "Center"], matches.Select(g => g.Name).ToArray());
    }

    [Fact]
    public void SmallGroupStrategy_filters_by_maximum_capacity()
    {
        var (groups, seeker) = SampleData();
        var matcher = new GroupMatcher(new SmallGroupStrategy(maxCapacity: 2));

        var matches = matcher.FindMatches(groups, seeker);

        Assert.Equal(["Center"], matches.Select(g => g.Name).ToArray());
    }

    [Fact]
    public void GroupMatcher_changes_results_when_strategy_is_swapped_at_runtime()
    {
        var (groups, seeker) = SampleData();
        var matcher = new GroupMatcher(new AffordableWithSpaceStrategy());

        Assert.Equal(3, matcher.FindMatches(groups, seeker).Count);

        matcher.SetStrategy(new BudgetBufferStrategy(100));
        var afterSwap = matcher.FindMatches(groups, seeker);

        Assert.Single(afterSwap);
        Assert.Equal("Center", afterSwap[0].Name);
    }
}

public class CommandTests
{
    private static (InMemoryStore store, IGroupWriter writer, ProfileService profiles, GroupCommandInvoker invoker, ActivityHistoryObserver history) Setup()
    {
        var store = new InMemoryStore();
        IGroupWriter writer = new GroupService(store);
        var profiles = new ProfileService(store);
        var notifier = new GroupActivityNotifier();
        var history = new ActivityHistoryObserver();
        notifier.Subscribe(history);
        var invoker = new GroupCommandInvoker(notifier);
        return (store, writer, profiles, invoker, history);
    }

    [Fact]
    public void JoinGroupCommand_execute_and_undo_restores_original_membership()
    {
        var (store, writer, profiles, invoker, _) = Setup();
        var group = writer.CreateGroup(new GroupInput("Riscani", 500, 2));
        var ana = profiles.CreateProfile(new ProfileInput("Ana", 700));

        invoker.Run(new JoinGroupCommand(writer, group.Id, ana.Id));
        Assert.Equal([ana.Id], store.GetGroup(group.Id).MemberIds);

        var undone = invoker.UndoLast();
        Assert.Equal("UndidJoin", undone.Action);
        Assert.Empty(store.GetGroup(group.Id).MemberIds);
    }

    [Fact]
    public void CreateGroupCommand_execute_and_undo_removes_created_group()
    {
        var (store, writer, _, invoker, _) = Setup();

        var created = invoker.Run(new CreateGroupCommand(writer, new GroupInput("Temporary flat", 450, 2)));
        Assert.Single(store.AllGroups);

        var undone = invoker.UndoLast();
        Assert.Equal("UndidCreate", undone.Action);
        Assert.Equal(created.GroupId, undone.GroupId);
        Assert.Empty(store.AllGroups);
    }

    [Fact]
    public void CommandInvoker_does_not_record_failed_command_in_undo_history()
    {
        var (_, writer, profiles, invoker, history) = Setup();
        var group = writer.CreateGroup(new GroupInput("Pricey", 900, 2));
        var poor = profiles.CreateProfile(new ProfileInput("Dan", 100));

        Assert.Throws<DomainException>(() => invoker.Run(new JoinGroupCommand(writer, group.Id, poor.Id)));
        Assert.Equal(0, invoker.HistoryCount);
        Assert.Empty(history.Events);
    }

    [Fact]
    public void CommandInvoker_runs_queued_commands_in_fifo_order_and_undoes_in_lifo_order()
    {
        var (store, writer, profiles, invoker, history) = Setup();
        var group = writer.CreateGroup(new GroupInput("Flat", 400, 3));
        var ana = profiles.CreateProfile(new ProfileInput("Ana", 700));
        var vlad = profiles.CreateProfile(new ProfileInput("Vlad", 600));

        invoker.Enqueue(new JoinGroupCommand(writer, group.Id, ana.Id));
        invoker.Enqueue(new JoinGroupCommand(writer, group.Id, vlad.Id));
        Assert.Equal(2, invoker.QueuedCount);

        var results = invoker.RunQueued();
        Assert.Equal(0, invoker.QueuedCount);
        Assert.Equal(new string?[] { ana.Id, vlad.Id }, results.Select(r => r.ProfileId).ToArray());
        Assert.Equal([ana.Id, vlad.Id], store.GetGroup(group.Id).MemberIds);

        var firstUndo = invoker.UndoLast();
        Assert.Equal(vlad.Id, firstUndo.ProfileId);
        Assert.Equal([ana.Id], store.GetGroup(group.Id).MemberIds);
        Assert.Equal(3, history.Events.Count);
    }
}

public class ObserverTests
{
    [Fact]
    public void All_subscribed_observers_receive_notifications_after_successful_command()
    {
        var store = new InMemoryStore();
        IGroupWriter writer = new GroupService(store);
        var profiles = new ProfileService(store);
        var group = writer.CreateGroup(new GroupInput("Flat", 500, 2));
        var ana = profiles.CreateProfile(new ProfileInput("Ana", 600));

        var notifier = new GroupActivityNotifier();
        var first = new ActivityHistoryObserver();
        var second = new ActivityHistoryObserver();
        notifier.Subscribe(first);
        notifier.Subscribe(second);

        var invoker = new GroupCommandInvoker(notifier);
        invoker.Run(new JoinGroupCommand(writer, group.Id, ana.Id));

        Assert.Single(first.Events);
        Assert.Single(second.Events);
        Assert.Equal("Joined", first.Events[0].Action);
        Assert.Equal(second.Events[0], first.Events[0]);
    }

    [Fact]
    public void Unsubscribed_observer_receives_no_further_notifications()
    {
        var store = new InMemoryStore();
        IGroupWriter writer = new GroupService(store);
        var profiles = new ProfileService(store);
        var group = writer.CreateGroup(new GroupInput("Flat", 500, 2));
        var ana = profiles.CreateProfile(new ProfileInput("Ana", 600));
        var vlad = profiles.CreateProfile(new ProfileInput("Vlad", 600));

        var notifier = new GroupActivityNotifier();
        var staying = new ActivityHistoryObserver();
        var leaving = new ActivityHistoryObserver();
        notifier.Subscribe(staying);
        notifier.Subscribe(leaving);

        var invoker = new GroupCommandInvoker(notifier);
        invoker.Run(new JoinGroupCommand(writer, group.Id, ana.Id));

        notifier.Unsubscribe(leaving);
        invoker.Run(new JoinGroupCommand(writer, group.Id, vlad.Id));

        Assert.Equal(2, staying.Events.Count);
        Assert.Single(leaving.Events);
    }

    [Fact]
    public void FullGroupAlertObserver_announces_when_group_becomes_full_and_reopens_on_undo()
    {
        var store = new InMemoryStore();
        IGroupWriter writer = new GroupService(store);
        var profiles = new ProfileService(store);
        var group = writer.CreateGroup(new GroupInput("Studio", 500, 1));
        var ana = profiles.CreateProfile(new ProfileInput("Ana", 600));

        var alertText = new StringWriter();
        var notifier = new GroupActivityNotifier();
        notifier.Subscribe(new FullGroupAlertObserver(alertText));

        var invoker = new GroupCommandInvoker(notifier);
        invoker.Run(new JoinGroupCommand(writer, group.Id, ana.Id));
        Assert.Contains("is now FULL", alertText.ToString());

        invoker.UndoLast();
        Assert.Contains("has a free spot again", alertText.ToString());
    }

    [Fact]
    public void Subscribing_same_observer_twice_does_not_duplicate_notifications()
    {
        var store = new InMemoryStore();
        IGroupWriter writer = new GroupService(store);
        var profiles = new ProfileService(store);
        var group = writer.CreateGroup(new GroupInput("Flat", 500, 2));
        var ana = profiles.CreateProfile(new ProfileInput("Ana", 600));

        var notifier = new GroupActivityNotifier();
        var observer = new ActivityHistoryObserver();
        notifier.Subscribe(observer);
        notifier.Subscribe(observer);

        var invoker = new GroupCommandInvoker(notifier);
        invoker.Run(new JoinGroupCommand(writer, group.Id, ana.Id));

        Assert.Equal(1, notifier.SubscriberCount);
        Assert.Single(observer.Events);
    }
}

public class InteractionTests
{
    [Fact]
    public void End_to_end_path_crosses_structural_and_behavioral_patterns()
    {
        var store = new InMemoryStore();
        var log = new StringWriter();
        IProfileWriter profiles = new ProfileService(store);
        IGroupWriter decorated = new TimedGroupWriter(new AuditedGroupWriter(new GroupService(store), log), log);
        IGroupWriter guarded = new AdminOnlyGroupWriter(decorated, new SessionUser("Lucian", isAdmin: true));

        // Lab 2 Adapter + Lab 1 GroupBuilder seed the catalog through Proxy + Decorators.
        using var adapter = new LegacyCsvListingAdapter(
            new StringReader("name;total;people\nRiscani flat;1500;3\nBotanica;1501;3"));
        new ListingImporter(adapter, guarded).ImportAll();

        // Lab 2 Facade onboards Ana through the same guarded writer.
        new OnboardingFacade(profiles, guarded).Onboard(new("Ana", 700), new("Ana's flat", 650, 2));

        // Lab 3 Strategy + Command + Observer run on the same catalog and writer.
        var vlad = profiles.CreateProfile(new ProfileInput("Vlad", 550));
        var matcher = new GroupMatcher(new BudgetBufferStrategy(50));
        var notifier = new GroupActivityNotifier();
        var history = new ActivityHistoryObserver();
        notifier.Subscribe(history);

        var invoker = new GroupCommandInvoker(notifier);
        var workflow = new GroupMatchingWorkflow(matcher, invoker, guarded);
        var joined = workflow.JoinBestMatch(store.AllGroups, vlad);

        Assert.NotNull(joined);
        Assert.Equal("Riscani flat", joined.GroupName);
        Assert.Single(history.Events);
        Assert.Contains("[audit] JoinGroup ok", log.ToString());
        Assert.Contains("[timer] JoinGroup", log.ToString());

        // Undoing reverses the domain state through Proxy + Decorators and notifies Observers.
        invoker.UndoLast();
        Assert.Equal(2, history.Events.Count);
        Assert.Equal("UndidJoin", history.Events[1].Action);
        Assert.Empty(store.GetGroup(joined.GroupId).MemberIds);
    }
}
