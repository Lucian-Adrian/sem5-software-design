using Domi.Behavioral.Client;
using Domi.Behavioral.Domain;
using Domi.Behavioral.Patterns.Adapter;
using Domi.Behavioral.Patterns.Command;
using Domi.Behavioral.Patterns.Decorator;
using Domi.Behavioral.Patterns.Facade;
using Domi.Behavioral.Patterns.Observer;
using Domi.Behavioral.Patterns.Proxy;
using Domi.Behavioral.Patterns.Strategy;

// Composition root: the ONLY place that constructs concrete classes.
// Interaction path across Lab 2 structural and Lab 3 behavioral patterns:
//   GroupMatchingWorkflow -> GroupMatcher (Strategy)
//                         -> JoinGroupCommand -> GroupCommandInvoker (Command)
//                         -> AdminOnlyGroupWriter (Proxy)
//                         -> TimedGroupWriter -> AuditedGroupWriter (Decorators)
//                         -> GroupService -> GroupActivityNotifier -> Observers

var store = new InMemoryStore();
var log = Console.Out;
IProfileWriter profiles = new ProfileService(store);
IGroupWriter decorated = new TimedGroupWriter(new AuditedGroupWriter(new GroupService(store), log), log);

var admin = new SessionUser("Lucian", isAdmin: true);
IGroupWriter guarded = new AdminOnlyGroupWriter(decorated, admin);

Title("1. LAB 1 + LAB 2 SETUP: Adapter + Builder + Facade through Proxy and Decorators");
var csvPath = Path.Combine(AppContext.BaseDirectory, "data", "legacy-listings.csv");
using (var adapter = new LegacyCsvListingAdapter(new StreamReader(csvPath)))
{
    var imported = new ListingImporter(adapter, guarded).ImportAll();
    foreach (var g in imported) Console.WriteLine($"  imported {g}");
    foreach (var row in adapter.RejectedRows) Console.WriteLine($"  rejected \"{row}\"");
}

var facade = new OnboardingFacade(profiles, guarded);
var anaResult = facade.Onboard(new ProfileInput("Ana", 700), new GroupInput("Ana's flat", 650, 2));
Console.WriteLine($"  onboarded {anaResult.Profile.Name} into {anaResult.Group}");

var vlad = profiles.CreateProfile(new ProfileInput("Vlad", 550));
var ion = profiles.CreateProfile(new ProfileInput("Ion", 300));
var elena = profiles.CreateProfile(new ProfileInput("Elena", 520));

Title("2. STRATEGY: filter groups and swap matching rules at runtime");
IGroupMatchingStrategy initialStrategy = ParseStrategyArg(args);
var matcher = new GroupMatcher(initialStrategy);
PrintMatches(matcher, store.AllGroups, vlad);

matcher.SetStrategy(new BudgetBufferStrategy(50));
PrintMatches(matcher, store.AllGroups, vlad);

matcher.SetStrategy(new SmallGroupStrategy(maxCapacity: 2));
PrintMatches(matcher, store.AllGroups, anaResult.Profile);

// Switch back to AffordableWithSpaceStrategy for the join workflow.
matcher.SetStrategy(new AffordableWithSpaceStrategy());

Title("3. INTERACTION PATH: Strategy -> Command -> Proxy/Decorators -> Observer");
var notifier = new GroupActivityNotifier();
var logObserver = new ActivityLogObserver(log);
var alertObserver = new FullGroupAlertObserver(log);
var historyObserver = new ActivityHistoryObserver();

notifier.Subscribe(logObserver);
notifier.Subscribe(alertObserver);
notifier.Subscribe(historyObserver);

var invoker = new GroupCommandInvoker(notifier);
var workflow = new GroupMatchingWorkflow(matcher, invoker, guarded);

var matchedEvent = workflow.JoinBestMatch(store.AllGroups, vlad);
Console.WriteLine($"  workflow matched and executed: {matchedEvent?.Action} -> {matchedEvent?.GroupName}");

Title("4. COMMAND QUEUE: enqueue CreateGroupCommand + JoinGroupCommand until full");
var studioInput = new GroupInput("Center studio", 500, 1);
invoker.Enqueue(new CreateGroupCommand(guarded, studioInput));
var queuedEvents = invoker.RunQueued();
var studioId = queuedEvents[0].GroupId;
invoker.Run(new JoinGroupCommand(guarded, studioId, elena.Id));

Title("5. FAILED COMMAND: error passes through decorators, no undo entry, no notification");
try
{
    invoker.Run(new JoinGroupCommand(guarded, studioId, ion.Id));
}
catch (DomainException e)
{
    Console.WriteLine($"  rejected command: {e.Message}");
}
Console.WriteLine($"  undo stack size stays {invoker.HistoryCount}, observer events={historyObserver.Events.Count}");

Title("6. RUNTIME UNSUBSCRIBE + UNDO: detach logObserver, then undo Elena's join");
notifier.Unsubscribe(logObserver);
Console.WriteLine($"  subscribers after unsubscribe: {notifier.SubscriberCount}");
var undone = invoker.UndoLast();
Console.WriteLine($"  undone action={undone.Action} on {undone.GroupName}, members={undone.MemberCount}/{undone.Capacity}");

Title("7. SUMMARY: store and observer history");
foreach (var g in store.AllGroups) Console.WriteLine($"  {g}");
Console.WriteLine($"  observer history count: {historyObserver.Events.Count}");
foreach (var ev in historyObserver.Events)
    Console.WriteLine($"    event: {ev.Action} {ev.GroupName} [{ev.GroupId}] {ev.MemberCount}/{ev.Capacity}");

static IGroupMatchingStrategy ParseStrategyArg(string[] cliArgs)
{
    for (var i = 0; i < cliArgs.Length - 1; i++)
    {
        if (cliArgs[i] == "--buffer" && int.TryParse(cliArgs[i + 1], out var reserve))
            return new BudgetBufferStrategy(reserve);
        if (cliArgs[i] == "--max-size" && int.TryParse(cliArgs[i + 1], out var maxCap))
            return new SmallGroupStrategy(maxCap);
    }
    return new AffordableWithSpaceStrategy();
}

static void PrintMatches(GroupMatcher matcher, IEnumerable<Group> groups, Profile profile)
{
    var found = matcher.FindMatches(groups, profile);
    Console.WriteLine($"  [{matcher.Strategy.Name}] for {profile.Name} (budget={profile.MonthlyBudget}): {found.Count} match(es)");
    foreach (var g in found) Console.WriteLine($"    - {g}");
}

static void Title(string text) => Console.WriteLine($"\n=== {text} ===");
