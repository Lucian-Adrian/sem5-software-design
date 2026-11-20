using Domi.Structural.Creational;
using Domi.Structural.Domain;
using Microsoft.VisualBasic.FileIO;

namespace Domi.Structural.Patterns.Adapter;

// ===========================================================================
// ADAPTER
// Adaptee: Microsoft.VisualBasic.FileIO.TextFieldParser, a .NET library class
//          we did not write and cannot change. It reads a legacy CSV export
//          ("name;monthly_total;roommates") and only offers string[] rows
//          through ReadFields() / EndOfData.
// Target:  IListingSource, which our importer understands: typed GroupInput
//          with a PER-PERSON cost.
// Adapter: LegacyCsvListingAdapter implements the target, holds the adaptee,
//          and translates both the call shape and the data meaning.
// ===========================================================================

/// <summary>TARGET: what Domi's client code wants.</summary>
public interface IListingSource
{
    IEnumerable<GroupInput> GetListings();
}

/// <summary>ADAPTER over the library's TextFieldParser.</summary>
public sealed class LegacyCsvListingAdapter : IListingSource, IDisposable
{
    private readonly TextFieldParser _parser;   // the adaptee

    public List<string> RejectedRows { get; } = [];

    public LegacyCsvListingAdapter(TextReader csv)
    {
        _parser = new TextFieldParser(csv);
        _parser.TextFieldType = FieldType.Delimited;
        _parser.SetDelimiters(";");
        _parser.HasFieldsEnclosedInQuotes = true;
    }

    public IEnumerable<GroupInput> GetListings()
    {
        _parser.ReadFields(); // skip the header row
        while (!_parser.EndOfData)
        {
            var fields = _parser.ReadFields() ?? [];
            if (TryTranslate(fields, out var input))
                yield return input;
            else
                RejectedRows.Add(string.Join(";", fields));
        }
    }

    // Translates the legacy meaning (whole-flat total) into ours (per person).
    public static bool TryTranslate(string[] fields, out GroupInput input)
    {
        input = default!;
        if (fields.Length != 3) return false;
        if (!int.TryParse(fields[1], out var total) || total <= 0) return false;
        if (!int.TryParse(fields[2], out var people) || people <= 0) return false;

        // Ceiling division: 1501 / 3 = 500 remainder 1 -> 501, so the rent is covered.
        var perPerson = total / people;
        if (total % people != 0) perPerson++;

        try
        {
            input = new GroupBuilder()          // creational pattern from Lab 1
                .WithName(fields[0])
                .WithMaxMonthlyCost(perPerson)
                .WithCapacity(people)
                .Build();
            return true;
        }
        catch (DomainException)
        {
            return false;                        // e.g. more than 20 people
        }
    }

    public void Dispose() => _parser.Dispose();
}

/// <summary>CLIENT: knows only IListingSource and IGroupWriter (abstractions).</summary>
public sealed class ListingImporter(IListingSource source, IGroupWriter groups)
{
    public List<Group> ImportAll() => source.GetListings().Select(groups.CreateGroup).ToList();
}
