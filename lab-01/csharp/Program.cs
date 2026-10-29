using System.Text.Json;
using Domi;

ProfileService profiles = new();
GroupService groups = new(profiles);

if (args.Length > 0 && args[0] == "--extra")
{
    // Singleton: both reads return the same object.
    Console.WriteLine($"Singleton same instance: {ReferenceEquals(CatalogDefaults.Instance, CatalogDefaults.Instance)}");

    // Factory Method: same workflow, each creator picks its own group.
    GroupCreator[] creators = [new StudentGroupCreator(), new QuietGroupCreator()];
    SummaryBuilderPool pool = new(maxIdle: 1);
    foreach (GroupCreator creator in creators)
    {
        GroupView group = creator.CreateAndJoin(groups, profiles, new ProfileInput("Founder", 600));
        Console.WriteLine(pool.Summarize(group));
    }

    // Object Pool: two summaries, but only one StringBuilder was ever created.
    Console.WriteLine($"Pool created {pool.Created} StringBuilder(s) for 2 summaries");
    return;
}

// 1. Builder: one group, built step by step and validated in Build().
GroupInput builderInput = new GroupBuilder()
    .WithName("Builder example home")
    .WithMaxMonthlyCost(650)
    .WithCapacity(3)
    .Build();
GroupView builderGroup = groups.CreateGroup(builderInput);

// 2. Abstract Factory: each family gives a matching group + profile; the profile joins.
var families = new (string Name, IOnboardingPresetFactory Factory)[]
{
    ("budget", new BudgetPresetFactory()),
    ("comfort", new ComfortPresetFactory()),
};

var presetResults = new List<object>();
string budgetGroupId = "";
foreach (var (name, factory) in families)
{
    OnboardingPreset preset = Presets.Build(factory);
    GroupView group = groups.CreateGroup(preset.Group);
    ProfileView profile = profiles.CreateProfile(preset.SuggestedProfile);
    GroupView joined = groups.JoinGroup(group.Id, profile.Id);

    presetResults.Add(new { family = name, group = joined, suggestedProfile = profile });
    if (name == "budget") budgetGroupId = joined.Id;
}

// 3. Prototype: copy the Budget group's settings into a new group with no members.
Group source = groups.Get(budgetGroupId);
GroupPrototype prototype = new(source);
Group draft = prototype.CloneAsDraft("Budget shared home copy");
GroupView copy = groups.CreateGroup(draft.ToInput());

var result = new
{
    builderGroup,
    presetGroups = presetResults,
    prototype = new { sourceGroupId = source.Id, sourceMemberCount = source.MemberProfileIds.Count, copy },
};

Console.WriteLine(JsonSerializer.Serialize(result, new JsonSerializerOptions
{
    WriteIndented = true,
    PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
}));
