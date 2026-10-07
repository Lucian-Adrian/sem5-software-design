package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type extraPatternsDemoResult struct {
	Singleton     singletonDemoResult     `json:"singleton"`
	FactoryMethod factoryMethodDemoResult `json:"factoryMethod"`
	ObjectPool    objectPoolDemoResult    `json:"objectPool"`
}

type singletonDemoResult struct {
	SameInstance   bool        `json:"sameInstance"`
	DefaultGroup   GroupDetail `json:"defaultGroup"`
	DefaultProfile ProfileView `json:"defaultProfile"`
}

type factoryMethodDemoResult struct {
	Creations []groupOnboardingDemoResult `json:"creations"`
}

type groupOnboardingDemoResult struct {
	Creator string      `json:"creator"`
	Group   GroupDetail `json:"group"`
	Profile ProfileView `json:"profile"`
}

type objectPoolDemoResult struct {
	Summary      string `json:"summary"`
	Reused       bool   `json:"reused"`
	ResetOnReuse bool   `json:"resetOnReuse"`
	IdleBuffers  int    `json:"idleBuffers"`
}

// runExtraPatternsDemo runs the optional Singleton, Factory Method, and Pool examples.
// It encodes their results as indented JSON on standard output.
func runExtraPatternsDemo(groups *GroupService, profiles *ProfileService) error {
	catalogResult, err := createCatalogExample(groups, profiles)
	if err != nil {
		return err
	}

	workflow := NewGroupOnboardingWorkflow(groups, profiles)
	creations := make([]groupOnboardingDemoResult, 0, 2)
	for _, example := range []struct {
		name    string
		creator GroupDraftCreator
	}{
		{name: "student", creator: StudentGroupCreator{}},
		{name: "quiet", creator: QuietGroupCreator{}},
	} {
		result, err := workflow.Run(example.creator)
		if err != nil {
			return err
		}
		creations = append(creations, groupOnboardingDemoResult{
			Creator: example.name,
			Group:   result.Group,
			Profile: result.Profile,
		})
	}

	poolResult := createPooledSummary(catalogResult, creations[len(creations)-1])
	result := extraPatternsDemoResult{
		Singleton:     catalogResult,
		FactoryMethod: factoryMethodDemoResult{Creations: creations},
		ObjectPool:    poolResult,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// createCatalogExample reads fixed inputs from the shared catalog and saves them.
// It joins the saved profile to the saved group through the services.
func createCatalogExample(groups *GroupService, profiles *ProfileService) (singletonDemoResult, error) {
	catalog := DefaultOnboardingCatalog()
	group, err := groups.CreateGroup(catalog.DefaultGroup())
	if err != nil {
		return singletonDemoResult{}, err
	}

	profile, err := profiles.CreateProfile(catalog.DefaultProfile())
	if err != nil {
		return singletonDemoResult{}, err
	}

	joined, err := groups.JoinGroup(group.ID, profile.ID)
	if err != nil {
		return singletonDemoResult{}, err
	}
	return singletonDemoResult{
		SameInstance:   catalog == DefaultOnboardingCatalog(),
		DefaultGroup:   joined,
		DefaultProfile: profile,
	}, nil
}

// createPooledSummary formats one group, releases its buffer, then reuses it.
// The result records whether reuse occurred and whether the buffer was reset.
func createPooledSummary(catalog singletonDemoResult, factory groupOnboardingDemoResult) objectPoolDemoResult {
	pool := NewBufferPool(1)
	first := pool.Acquire()
	fmt.Fprintf(first, "%s has %d member(s)", factory.Group.Group.Name, factory.Group.Group.MemberCount)
	pool.Release(first)

	reused := pool.Acquire()
	resetOnReuse := reused == first && reused.Len() == 0
	fmt.Fprintf(reused, "%s has %d member(s)", catalog.DefaultGroup.Group.Name, catalog.DefaultGroup.Group.MemberCount)
	summary := reused.String()
	pool.Release(reused)

	return objectPoolDemoResult{
		Summary:      summary,
		Reused:       reused == first,
		ResetOnReuse: resetOnReuse,
		IdleBuffers:  pool.IdleCount(),
	}
}
