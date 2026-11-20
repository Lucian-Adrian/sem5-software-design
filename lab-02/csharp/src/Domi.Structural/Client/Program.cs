using Domi.Structural.Domain;
using Domi.Structural.Patterns.Adapter;
using Domi.Structural.Patterns.Decorator;
using Domi.Structural.Patterns.Facade;
using Domi.Structural.Patterns.Proxy;

// Composition root: the ONLY place that knows concrete classes.
// One execution path goes through four patterns:
//   client -> OnboardingFacade -> AdminOnlyGroupWriter (proxy)
//          -> TimedGroupWriter -> AuditedGroupWriter (decorators) -> GroupService
// and the importer goes: client -> ListingImporter -> LegacyCsvListingAdapter -> TextFieldParser.

var store = new InMemoryStore();
var log = Console.Out;
IProfileWriter profiles = new ProfileService(store);
IGroupWriter decorated = new TimedGroupWriter(new AuditedGroupWriter(new GroupService(store), log), log);

var admin = new SessionUser("Lucian", isAdmin: true);
IGroupWriter guarded = new AdminOnlyGroupWriter(decorated, admin);

Title("1. ADAPTER: import a legacy CSV through a library class we did not write");
var csvPath = Path.Combine(AppContext.BaseDirectory, "data", "legacy-listings.csv");
using (var adapter = new LegacyCsvListingAdapter(new StreamReader(csvPath)))
{
    var imported = new ListingImporter(adapter, guarded).ImportAll();
    foreach (var g in imported) Console.WriteLine($"  imported {g}");
    foreach (var row in adapter.RejectedRows) Console.WriteLine($"  rejected \"{row}\"");
}

Title("2. DECORATOR: a failing join is logged, timed, and still fails");
var ion = profiles.CreateProfile(new ProfileInput("Ion", 300));
var firstGroup = store.AllGroups.First();
try { guarded.JoinGroup(firstGroup.Id, ion.Id); }
catch (DomainException e) { Console.WriteLine($"  caller still gets the error: {e.Message}"); }

Title("3. FACADE (through PROXY and DECORATORS): onboard Ana in one call");
var facade = new OnboardingFacade(profiles, guarded);
var result = facade.Onboard(new ProfileInput("Ana", 700), new GroupInput("Ana's flat", 650, 2));
Console.WriteLine($"  {result.Profile.Name} [{result.Profile.Id}] is in {result.Group}");
try { facade.Onboard(new ProfileInput("Dan", 100), new GroupInput("Too pricey", 900, 2)); }
catch (DomainException e) { Console.WriteLine($"  refused before any write: {e.Message}"); }

Title("4. PROXY: the same facade for a non-admin");
admin.IsAdmin = false;
try { facade.Onboard(new ProfileInput("Maria", 900), new GroupInput("Maria's flat", 400, 2)); }
catch (UnauthorizedAccessException e) { Console.WriteLine($"  blocked: {e.Message}"); }
var maria = store.AllProfiles.Last();
guarded.JoinGroup(store.AllGroups.ElementAt(1).Id, maria.Id);
Console.WriteLine($"  note: {maria.Name}'s profile was already created in step 1 - the facade is not a transaction");
Console.WriteLine($"  but {maria.Name} can still join: {store.AllGroups.ElementAt(1)}");

Title("Store at the end");
foreach (var g in store.AllGroups) Console.WriteLine($"  {g}");

static void Title(string text) => Console.WriteLine($"\n=== {text} ===");
