using Domi.Behavioral.Domain;

namespace Domi.Behavioral.Creational;

// Reused from Lab 1 (creational patterns): Builder.
// The Adapter uses it to create the GroupInput objects that the structural
// and behavioral patterns then pass along. Build() validates with the domain's rules.
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

    public GroupInput Build()
    {
        var input = new GroupInput(_name.Trim(), _maxMonthlyCost, _capacity);
        _ = new Group("draft", input); // throws DomainException if any rule is broken
        return input;
    }
}
