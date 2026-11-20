using Domi.Behavioral.Domain;
using Domi.Behavioral.Patterns.Adapter;
using Domi.Behavioral.Patterns.Decorator;
using Domi.Behavioral.Patterns.Facade;
using Domi.Behavioral.Patterns.Proxy;

namespace Domi.Behavioral.Tests;

/// <summary>A test double that records every call, in order.</summary>
internal sealed class RecordingWriter(List<string> calls) : IGroupWriter, IProfileWriter
{
    private readonly InMemoryStore _store = new();

    public Profile CreateProfile(ProfileInput input)
    {
        calls.Add("CreateProfile");
        var p = new Profile(_store.NextId("profile"), input);
        _store.Save(p);
        return p;
    }

    public Group CreateGroup(GroupInput input)
    {
        calls.Add("CreateGroup");
        var g = new Group(_store.NextId("group"), input);
        _store.Save(g);
        return g;
    }

    public Group JoinGroup(string groupId, string profileId)
    {
        calls.Add("JoinGroup");
        var g = _store.GetGroup(groupId);
        g.Join(_store.GetProfile(profileId));
        return g;
    }

    public Group LeaveGroup(string groupId, string profileId)
    {
        calls.Add("LeaveGroup");
        var g = _store.GetGroup(groupId);
        g.Leave(_store.GetProfile(profileId));
        return g;
    }

    public Group DeleteGroup(string groupId)
    {
        calls.Add("DeleteGroup");
        var g = _store.GetGroup(groupId);
        _store.RemoveGroup(groupId);
        return g;
    }
}

public class AdapterTests
{
    private static List<GroupInput> Read(string csv, out List<string> rejected)
    {
        using var adapter = new LegacyCsvListingAdapter(new StringReader("name;total;people\n" + csv));
        var result = adapter.GetListings().ToList();
        rejected = adapter.RejectedRows;
        return result;
    }

    [Fact]
    public void Translates_whole_flat_total_into_per_person_cost()
    {
        var listings = Read("Riscani flat;1500;3", out _);
        Assert.Equal(new GroupInput("Riscani flat", 500, 3), Assert.Single(listings));
    }

    [Fact]
    public void Rounds_up_so_the_rent_is_always_covered()
    {
        var listing = Assert.Single(Read("Botanica;1501;3", out _));
        Assert.Equal(501, listing.MaxMonthlyCost);
        Assert.True(listing.MaxMonthlyCost * listing.Capacity >= 1501);
    }

    [Fact]
    public void Handles_quoted_fields_from_the_library_parser()
    {
        var listing = Assert.Single(Read("\"Studio; near park\";900;2", out _));
        Assert.Equal("Studio; near park", listing.Name);
    }

    [Fact]
    public void Rejects_invalid_rows_without_stopping_the_import()
    {
        var listings = Read("Bad;0;2\nToo many;1000;21\nGood;800;2", out var rejected);
        Assert.Single(listings);
        Assert.Equal(2, rejected.Count);
    }
}

public class DecoratorTests
{
    private static (IGroupWriter writer, StringWriter log, InMemoryStore store) Stack(bool auditOutside)
    {
        var store = new InMemoryStore();
        var log = new StringWriter();
        IGroupWriter real = new GroupService(store);
        IGroupWriter writer = auditOutside
            ? new AuditedGroupWriter(new TimedGroupWriter(real, log), log)
            : new TimedGroupWriter(new AuditedGroupWriter(real, log), log);
        return (writer, log, store);
    }

    [Theory]
    [InlineData(true)]
    [InlineData(false)]
    public void Stacked_in_any_order_returns_the_same_stored_group(bool auditOutside)
    {
        var (writer, log, store) = Stack(auditOutside);
        var group = writer.CreateGroup(new GroupInput("Flat", 500, 2));
        Assert.Same(store.GetGroup(group.Id), group);
        Assert.Contains("[audit] CreateGroup ok", log.ToString());
        Assert.Contains("[timer] CreateGroup", log.ToString());
    }

    [Fact]
    public void Errors_are_logged_and_rethrown_unchanged()
    {
        var (writer, log, store) = Stack(auditOutside: false);
        var group = writer.CreateGroup(new GroupInput("Flat", 500, 1));
        var poor = new ProfileService(store).CreateProfile(new ProfileInput("Ion", 100));
        var error = Assert.Throws<DomainException>(() => writer.JoinGroup(group.Id, poor.Id));
        Assert.Contains("below", error.Message);
        Assert.Contains("[audit] JoinGroup error", log.ToString());
        Assert.Contains("[timer] JoinGroup", log.ToString());
    }
}

public class FacadeTests
{
    [Fact]
    public void Calls_the_subsystem_in_the_right_order()
    {
        var calls = new List<string>();
        var stub = new RecordingWriter(calls);
        new OnboardingFacade(stub, stub).Onboard(new("Ana", 700), new("Flat", 650, 2));
        Assert.Equal(["CreateProfile", "CreateGroup", "JoinGroup"], calls);
    }

    [Fact]
    public void Refuses_unaffordable_onboarding_before_any_write()
    {
        var calls = new List<string>();
        var stub = new RecordingWriter(calls);
        Assert.Throws<DomainException>(() =>
            new OnboardingFacade(stub, stub).Onboard(new("Dan", 100), new("Pricey", 900, 2)));
        Assert.Empty(calls);
    }
}

public class ProxyTests
{
    [Fact]
    public void Blocks_unauthorized_create_without_reaching_the_real_writer()
    {
        var calls = new List<string>();
        var proxy = new AdminOnlyGroupWriter(new RecordingWriter(calls), new SessionUser("Guest", false));
        Assert.Throws<UnauthorizedAccessException>(() => proxy.CreateGroup(new("X", 100, 2)));
        Assert.Empty(calls);
    }

    [Fact]
    public void Passes_authorized_calls_through_unchanged()
    {
        var calls = new List<string>();
        var real = new RecordingWriter(calls);
        var proxy = new AdminOnlyGroupWriter(real, new SessionUser("Admin", true));
        var group = proxy.CreateGroup(new("X", 100, 2));
        Assert.Equal("X", group.Name);
        Assert.Equal(["CreateGroup"], calls);
    }

    [Fact]
    public void Non_admins_can_still_join()
    {
        var calls = new List<string>();
        var real = new RecordingWriter(calls);
        var group = real.CreateGroup(new("X", 100, 2));
        var p = real.CreateProfile(new("Ana", 200));
        var proxy = new AdminOnlyGroupWriter(real, new SessionUser("Ana", false));
        Assert.Single(proxy.JoinGroup(group.Id, p.Id).MemberIds);
    }
}
