using Domi.Behavioral.Domain;

namespace Domi.Behavioral.Patterns.Facade;

// ===========================================================================
// FACADE (reused from Lab 2)
// Onboarding needs three subsystem calls in a fixed order:
//   create profile -> create group -> join.
// ===========================================================================

public sealed record OnboardingResult(Profile Profile, Group Group);

public sealed class OnboardingFacade(IProfileWriter profiles, IGroupWriter groups)
{
    public OnboardingResult Onboard(ProfileInput person, GroupInput newGroup)
    {
        var draftProfile = new Profile("draft", person);
        var draftGroup = new Group("draft", newGroup);
        if (draftProfile.MonthlyBudget < draftGroup.MaxMonthlyCost)
            throw new DomainException(
                $"{draftProfile.Name} cannot afford {draftGroup.Name} ({draftProfile.MonthlyBudget} < {draftGroup.MaxMonthlyCost})");

        var profile = profiles.CreateProfile(person);
        var group = groups.CreateGroup(newGroup);
        groups.JoinGroup(group.Id, profile.Id);

        return new OnboardingResult(profile, group);
    }
}
