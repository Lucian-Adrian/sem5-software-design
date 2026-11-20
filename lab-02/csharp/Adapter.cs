namespace Domi.Structural;

// ===========================================================================
// ADAPTER
// Problem: an old housing site sends offers as text rows "name;total;people".
//          Our code wants GroupInput with a PER-PERSON cost.
//          We cannot change the old site, and we do not want parsing and
//          unit conversion leaking into GroupService.
// Idea:    put a translator between them that speaks both languages.
//
//   Client (ListingImporter) --> ITarget (IListingSource)
//                                    ^
//                                    | implements
//                               Adapter (LegacyFeedAdapter) --has-a--> Adaptee (LegacyHousingFeed)
// ===========================================================================

/// <summary>ADAPTEE: third-party code we cannot change. Its format is different.</summary>
public class LegacyHousingFeed
{
    // Format: "name;monthlyTotalForTheWholeFlat;howManyRoommates"
    public string[] DownloadRows() =>
    [
        "Riscani flat;1500;3",
        "Botanica studio;1501;3",   // does not divide evenly -> must round UP
        "Centru loft;0;2",          // broken row from the old site
    ];
}

/// <summary>TARGET: the interface our application already understands.</summary>
public interface IListingSource
{
    IEnumerable<GroupInput> GetListings();
}

/// <summary>ADAPTER: implements the target, holds the adaptee, translates.</summary>
public class LegacyFeedAdapter(LegacyHousingFeed feed) : IListingSource
{
    public List<string> RejectedRows { get; } = [];

    public IEnumerable<GroupInput> GetListings()
    {
        foreach (var row in feed.DownloadRows())
        {
            if (TryConvert(row, out var input))
                yield return input;
            else
                RejectedRows.Add(row);
        }
    }

    // The actual translation. Kept static and pure so it is easy to test.
    public static bool TryConvert(string row, out GroupInput input)
    {
        input = default!;
        var parts = row.Split(';');
        if (parts.Length != 3) return false;
        if (!int.TryParse(parts[1], out var total) || total <= 0) return false;
        if (!int.TryParse(parts[2], out var people) || people <= 0 || people > Group.MaxCapacity) return false;

        // Ceiling division without floating point:
        // 1500/3 = 500 exactly; 1501/3 = 500 remainder 1 -> 501.
        // Rounding DOWN would let three people "afford" 1500 and leave 1 unpaid.
        var perPerson = total / people;
        if (total % people != 0) perPerson++;

        input = new GroupInput(parts[0], perPerson, people);
        return true;
    }
}

/// <summary>CLIENT: only knows IListingSource. It never sees the legacy format.</summary>
public class ListingImporter(IListingSource source, IGroupWriter groups)
{
    public List<Group> ImportAll()
    {
        var created = new List<Group>();
        foreach (var listing in source.GetListings())
            created.Add(groups.CreateGroup(listing));
        return created;
    }
}
