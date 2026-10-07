package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestGroupFiltersCanBeUsedThroughOneContract(t *testing.T) {
	store := NewMemoryStore()
	service := NewGroupService(store, store)
	service.CreateGroup(GroupInput{Name: "Low cost", MaxMonthlyCost: 400, Capacity: 2})
	service.CreateGroup(GroupInput{Name: "Higher cost", MaxMonthlyCost: 700, Capacity: 2})

	filterCases := []struct {
		name   string
		filter GroupFilter
		want   []string
	}{
		{name: "all groups", filter: AllGroups{}, want: []string{"group-1", "group-2"}},
		{name: "affordable groups", filter: AffordableGroups{MonthlyBudget: 500}, want: []string{"group-1"}},
	}

	for _, testCase := range filterCases {
		t.Run(testCase.name, func(t *testing.T) {
			groups, err := service.ListGroups(testCase.filter)
			if err != nil {
				t.Fatalf("ListGroups() error = %v", err)
			}
			if len(groups) != len(testCase.want) {
				t.Fatalf("ListGroups() returned %d groups, want %d", len(groups), len(testCase.want))
			}
			for index, group := range groups {
				if group.ID != testCase.want[index] {
					t.Errorf("group %d ID = %q, want %q", index, group.ID, testCase.want[index])
				}
			}
		})
	}
}

func TestConcurrentJoinsDoNotExceedCapacityAndDuplicatesAreIdempotent(t *testing.T) {
	store := NewMemoryStore()
	group, err := store.CreateGroup(GroupInput{Name: "Shared flat", MaxMonthlyCost: 400, Capacity: 3})
	if err != nil {
		t.Fatal(err)
	}

	const profileCount = 30
	profileIDs := make([]string, profileCount)
	for index := range profileIDs {
		profile, createErr := store.CreateProfile(ProfileInput{Name: "Roommate", MonthlyBudget: 500})
		if createErr != nil {
			t.Fatal(createErr)
		}
		profileIDs[index] = profile.ID
	}

	var waitGroup sync.WaitGroup
	var resultLock sync.Mutex
	joinedIDs := make([]string, 0, group.Capacity)
	fullCount := 0
	for _, profileID := range profileIDs {
		waitGroup.Add(1)
		go func(id string) {
			defer waitGroup.Done()
			_, joinErr := store.JoinGroup(group.ID, id)
			resultLock.Lock()
			defer resultLock.Unlock()
			if joinErr == nil {
				joinedIDs = append(joinedIDs, id)
			} else if errors.Is(joinErr, ErrGroupFull) {
				fullCount++
			} else {
				t.Errorf("JoinGroup() error = %v", joinErr)
			}
		}(profileID)
	}
	waitGroup.Wait()

	if len(joinedIDs) != group.Capacity {
		t.Fatalf("successful joins = %d, want capacity %d", len(joinedIDs), group.Capacity)
	}
	if fullCount != profileCount-group.Capacity {
		t.Fatalf("full-group errors = %d, want %d", fullCount, profileCount-group.Capacity)
	}

	_, err = store.JoinGroup(group.ID, joinedIDs[0])
	if err != nil {
		t.Fatalf("joining again should be idempotent, got %v", err)
	}
	detail, err := store.GetGroup(group.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Group.MemberCount != group.Capacity || len(detail.Members) != group.Capacity {
		t.Fatalf("member count = %d and returned members = %d, want %d", detail.Group.MemberCount, len(detail.Members), group.Capacity)
	}
}

func TestHTTPWorkflowValidatesAndOmitsPrivateBudget(t *testing.T) {
	store := NewMemoryStore()
	handler := NewHTTPHandler(NewGroupService(store, store), NewProfileService(store))
	server := httptest.NewServer(handler)
	defer server.Close()

	profile := requestJSON[ProfileView](t, server.Client(), http.MethodPost, server.URL+"/profiles", `{"name":"Alex","monthlyBudget":550}`, http.StatusCreated)
	group := requestJSON[GroupView](t, server.Client(), http.MethodPost, server.URL+"/groups", `{"name":"Shared flat","maxMonthlyCost":500,"capacity":1}`, http.StatusCreated)

	response := request(t, server.Client(), http.MethodGet, server.URL+"/groups?budget=500", "", http.StatusOK)
	var groups []GroupView
	decodeResponse(t, response, &groups)
	if len(groups) != 1 || groups[0].ID != group.ID {
		t.Fatalf("affordable groups = %v, want only %q", groups, group.ID)
	}

	joined := requestJSON[GroupDetail](t, server.Client(), http.MethodPost, server.URL+"/groups/"+group.ID+"/join", `{"profileId":"`+profile.ID+`"}`, http.StatusOK)
	if len(joined.Members) != 1 || joined.Members[0].ID != profile.ID {
		t.Fatalf("joined members = %v, want profile %q", joined.Members, profile.ID)
	}
	if strings.Contains(string(encodeForTest(t, joined)), "monthlyBudget") {
		t.Fatal("group response exposed a member's personal monthly budget")
	}

	requestJSON[GroupDetail](t, server.Client(), http.MethodPost, server.URL+"/groups/"+group.ID+"/join", `{"profileId":"`+profile.ID+`"}`, http.StatusOK)
	fullProfile := requestJSON[ProfileView](t, server.Client(), http.MethodPost, server.URL+"/profiles", `{"name":"Sam","monthlyBudget":500}`, http.StatusCreated)
	requestError(t, server.Client(), http.MethodPost, server.URL+"/groups/"+group.ID+"/join", `{"profileId":"`+fullProfile.ID+`"}`, http.StatusConflict, "group_full")
	requestError(t, server.Client(), http.MethodPost, server.URL+"/groups", `{"name":"  ","maxMonthlyCost":500,"capacity":1}`, http.StatusBadRequest, "invalid_request")
	requestError(t, server.Client(), http.MethodPost, server.URL+"/groups", `{"name":"Too many roommates","maxMonthlyCost":500,"capacity":21}`, http.StatusBadRequest, "invalid_request")
}

func requestJSON[T any](t *testing.T, client *http.Client, method, url, body string, wantStatus int) T {
	t.Helper()
	response := request(t, client, method, url, body, wantStatus)
	var value T
	decodeResponse(t, response, &value)
	return value
}

func request(t *testing.T, client *http.Client, method, url, body string, wantStatus int) *http.Response {
	t.Helper()
	requestBody := bytes.NewBufferString(body)
	request, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != wantStatus {
		defer response.Body.Close()
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("%s %s status = %d, want %d; body: %s", method, url, response.StatusCode, wantStatus, payload)
	}
	return response
}

func requestError(t *testing.T, client *http.Client, method, url, body string, wantStatus int, wantCode string) {
	t.Helper()
	response := request(t, client, method, url, body, wantStatus)
	var value errorResponse
	decodeResponse(t, response, &value)
	if value.Error != wantCode {
		t.Fatalf("error code = %q, want %q", value.Error, wantCode)
	}
}

func decodeResponse(t *testing.T, response *http.Response, destination any) {
	t.Helper()
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func encodeForTest(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
