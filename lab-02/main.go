package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type demoResult struct {
	AffordableGroups []GroupView `json:"affordableGroups"`
	JoinedGroup      GroupDetail `json:"joinedGroup"`
}

// main selects one demonstration or starts the HTTP server.
// The flags choose the example; all modes reuse this lab's domain rules.
func main() {
	address := flag.String("addr", "localhost:8080", "HTTP server address")
	demo := flag.Bool("demo", false, "create sample profiles and groups, then print the result")
	patterns := flag.Bool("patterns", false, "demonstrate the creational patterns, then print the result")
	structural := flag.Bool("structural", false, "run the Adapter, Decorator, and Facade demonstration")
	flag.Parse()

	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		flag.Usage()
		os.Exit(2)
	}
	if (*demo && *patterns) || (*demo && *structural) || (*patterns && *structural) {
		fmt.Fprintln(os.Stderr, "choose one of --demo, --patterns, or --structural")
		flag.Usage()
		os.Exit(2)
	}

	// Choose concrete storage here; services receive only their interfaces.
	store := NewMemoryStore()
	groups := NewGroupService(store, store)
	profiles := NewProfileService(store)

	if *patterns {
		if err := runCreationalDemo(groups, profiles); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *structural {
		if err := runStructuralDemo(os.Stdout); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *demo {
		if err := runDemo(groups, profiles); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Demo modes return above; the normal mode serves HTTP until stopped.
	server := &http.Server{
		Addr:              *address,
		Handler:           NewHTTPHandler(groups, profiles),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Domi roommate groups listening at http://%s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// runDemo creates Alex and two groups, then joins the affordable group.
// It prints the service results directly, without making HTTP requests.
func runDemo(groups *GroupService, profiles *ProfileService) error {
	// Create one profile and two groups with different affordability outcomes.
	profile, err := profiles.CreateProfile(ProfileInput{Name: "Alex", MonthlyBudget: 550})
	if err != nil {
		return err
	}
	if _, err := groups.CreateGroup(GroupInput{Name: "Sunny apartment", MaxMonthlyCost: 700, Capacity: 3}); err != nil {
		return err
	}
	affordable, err := groups.CreateGroup(GroupInput{Name: "Shared flat", MaxMonthlyCost: 500, Capacity: 2})
	if err != nil {
		return err
	}

	// Join the affordable group, then list groups within the same budget.
	joined, err := groups.JoinGroup(affordable.ID, profile.ID)
	if err != nil {
		return err
	}
	affordableGroups, err := groups.ListGroups(AffordableGroups{MonthlyBudget: 550})
	if err != nil {
		return err
	}

	result := demoResult{AffordableGroups: affordableGroups, JoinedGroup: joined}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
