namespace Domi.Structural;

// ===========================================================================
// FACADE
// Problem: to onboard a new user the frontend must call 3 services in the
//          right order: create profile -> create group -> join. If it gets
//          the order wrong, or stops halfway, data is left half-made.
// Idea:    one simple front door that knows the order. The subsystems still
//          exist and can still be used directly; the facade only COORDINATES.
//
//   Frontend --> OnboardingFacade.Onboard(...)
//                     |--> ProfileService.CreateProfile
//                     |--> IGroupWriter.CreateGroup
//                     '--> IGroupWriter.JoinGroup
// ===========================================================================

public record OnboardingResult(Profile Profile, Group Group);

public class OnboardingFacade(ProfileService profiles, IGroupWriter groups)
{
    public OnboardingResult Onboard(ProfileInput person, GroupInput newGroup)
    {
        // Step 0: check everything we can BEFORE writing anything.
        // Building throw-away domain objects reuses the same rules.
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
