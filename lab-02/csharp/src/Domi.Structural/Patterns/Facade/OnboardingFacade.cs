using Domi.Structural.Domain;

namespace Domi.Structural.Patterns.Facade;

// ===========================================================================
// FACADE
// Onboarding needs three subsystem calls in a fixed order:
//   create profile -> create group -> join.
// The facade is one simple entry point that knows the order. It holds only
// abstractions (IProfileWriter, IGroupWriter) and adds no business rules:
// the rules stay in Profile and Group.
// ===========================================================================

public sealed record OnboardingResult(Profile Profile, Group Group);

public sealed class OnboardingFacade(IProfileWriter profiles, IGroupWriter groups)
{
    public OnboardingResult Onboard(ProfileInput person, GroupInput newGroup)
    {
        // Step 0: validate everything BEFORE writing, using the domain's own rules.
        var draftProfile = new Profile("draft", person);
        var draftGroup = new Group("draft", newGroup);
        if (draftProfile.MonthlyBudget < draftGroup.MaxMonthlyCost)
            throw new DomainException(
                $"{draftProfile.Name} cannot afford {draftGroup.Name} ({draftProfile.MonthlyBudget} < {draftGroup.MaxMonthlyCost})");

        // Steps 1-3 in the only correct order.
        var profile = profiles.CreateProfile(person);
        var group = groups.CreateGroup(newGroup);
        groups.JoinGroup(group.Id, profile.Id);

        return new OnboardingResult(profile, group);
    }
}
