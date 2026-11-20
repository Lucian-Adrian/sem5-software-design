using Domi.Behavioral.Creational;
using Domi.Behavioral.Domain;
using Microsoft.VisualBasic.FileIO;

namespace Domi.Behavioral.Patterns.Adapter;

// ===========================================================================
// ADAPTER (reused from Lab 2)
// Target:  IListingSource
// Adaptee: Microsoft.VisualBasic.FileIO.TextFieldParser (.NET library)
// Adapter: LegacyCsvListingAdapter
// ===========================================================================

/// <summary>TARGET: what Domi's client code wants.</summary>
public interface IListingSource
{
    IEnumerable<GroupInput> GetListings();
}

/// <summary>ADAPTER over the library's TextFieldParser.</summary>
public sealed class LegacyCsvListingAdapter : IListingSource, IDisposable
{
    private readonly TextFieldParser _parser;

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

    public static bool TryTranslate(string[] fields, out GroupInput input)
    {
        input = default!;
        if (fields.Length != 3) return false;
        if (!int.TryParse(fields[1], out var total) || total <= 0) return false;
        if (!int.TryParse(fields[2], out var people) || people <= 0) return false;

        var perPerson = total / people;
        if (total % people != 0) perPerson++;

        try
        {
            input = new GroupBuilder()
                .WithName(fields[0])
                .WithMaxMonthlyCost(perPerson)
                .WithCapacity(people)
                .Build();
            return true;
        }
        catch (DomainException)
        {
            return false;
        }
    }

    public void Dispose() => _parser.Dispose();
}

/// <summary>CLIENT: knows only IListingSource and IGroupWriter (abstractions).</summary>
public sealed class ListingImporter(IListingSource source, IGroupWriter groups)
{
    public List<Group> ImportAll() => source.GetListings().Select(groups.CreateGroup).ToList();
}
