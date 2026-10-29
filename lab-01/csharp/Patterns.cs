namespace Domi;

// ---------- Builder ----------
// Collects name, cost and capacity step by step, then validates them together in Build().
public sealed class GroupBuilder
{
    private string _name = "";
    private int _maxMonthlyCost;
    private int _capacity;

    public GroupBuilder WithName(string name)
    {
        _name = name;
        return this;
    }

    public GroupBuilder WithMaxMonthlyCost(int amount)
    {
        _maxMonthlyCost = amount;
        return this;
    }

    public GroupBuilder WithCapacity(int people)
    {
        _capacity = people;
        return this;
    }

    // Returns validated input only. GroupService saves it and assigns the ID.
    public GroupInput Build() => Group.Normalize(new GroupInput(_name, _maxMonthlyCost, _capacity));
}

// ---------- Abstract Factory ----------
// One factory makes a matching pair: a group draft and a profile that can afford it.
public sealed record OnboardingPreset(GroupInput Group, ProfileInput SuggestedProfile);

public interface IOnboardingPresetFactory
{
    GroupInput CreateGroupDraft();
    ProfileInput CreateSuggestedProfile();
}

public sealed class BudgetPresetFactory : IOnboardingPresetFactory
{
    public GroupInput CreateGroupDraft() => new GroupBuilder()
        .WithName("Budget shared home")
        .WithMaxMonthlyCost(450)
        .WithCapacity(4)
        .Build();

    public ProfileInput CreateSuggestedProfile() => Presets.ProfileSuggestion("Budget renter", 500);
}

public sealed class ComfortPresetFactory : IOnboardingPresetFactory
{
    public GroupInput CreateGroupDraft() => new GroupBuilder()
        .WithName("Comfort shared home")
        .WithMaxMonthlyCost(700)
        .WithCapacity(2)
        .Build();

    public ProfileInput CreateSuggestedProfile() => Presets.ProfileSuggestion("Comfort renter", 900);
}

public static class Presets
{
    // The client only knows the interface. It checks that the pair really fits together.
    public static OnboardingPreset Build(IOnboardingPresetFactory factory)
    {
        GroupInput group = factory.CreateGroupDraft();
        ProfileInput profile = factory.CreateSuggestedProfile();

        if (profile.MonthlyBudget < group.MaxMonthlyCost)
            throw new DomainException("monthly budget is below the group's cost");

        return new OnboardingPreset(group, profile);
    }

    public static ProfileInput ProfileSuggestion(string name, int monthlyBudget)
    {
        Profile checkedProfile = new("suggested-profile", new ProfileInput(name, monthlyBudget));
        return new ProfileInput(checkedProfile.Name, checkedProfile.MonthlyBudget);
    }
}

// ---------- Prototype ----------
// Takes a snapshot of an existing group, then clones its settings into new drafts.
// A clone keeps cost and capacity but gets no ID and no members.
public sealed class GroupPrototype
{
    private readonly Group _template;

    public GroupPrototype(Group source)
    {
        // Copy the member list so later changes to the source do not change the snapshot.
        _template = source.Copy(source.Id, source.Name, source.MemberProfileIds);
    }

    public Group CloneAsDraft(string name) => _template.Copy(id: "", name: name, memberIds: []);
}
