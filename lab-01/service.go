package main

// GroupReader and GroupWriter are separate so a service only asks for the
// operations it uses. A database-backed store can satisfy the same contracts.
type GroupReader interface {
	ListGroups() ([]GroupView, error)
	GetGroup(id string) (GroupDetail, error)
}

type GroupWriter interface {
	CreateGroup(input GroupInput) (GroupView, error)
	JoinGroup(groupID, profileID string) (GroupDetail, error)
}

type ProfileWriter interface {
	CreateProfile(input ProfileInput) (ProfileView, error)
}

type GroupService struct {
	reader GroupReader
	writer GroupWriter
}

type ProfileService struct {
	writer ProfileWriter
}

// GroupFilter lets callers add a new listing rule without changing the service.
type GroupFilter interface {
	Matches(group GroupView) bool
}

type AllGroups struct{}

// AllGroups accepts every group returned by storage.
// It gives the listing method a default rule without a special filtering branch.
func (AllGroups) Matches(GroupView) bool {
	return true
}

type AffordableGroups struct {
	MonthlyBudget int
}

// AffordableGroups compares the group's cost with the caller's budget.
// Equality is allowed: someone with exactly 500 can afford a cost of 500.
func (filter AffordableGroups) Matches(group GroupView) bool {
	return group.MaxMonthlyCost <= filter.MonthlyBudget
}

// NewGroupService receives the objects that will read and change groups.
// The caller chooses storage; the service only needs the interface methods.
func NewGroupService(reader GroupReader, writer GroupWriter) *GroupService {
	return &GroupService{reader: reader, writer: writer}
}

// NewProfileService receives the object that can save profiles.
// Keeping this dependency as an interface allows another implementation later.
func NewProfileService(writer ProfileWriter) *ProfileService {
	return &ProfileService{writer: writer}
}

// ListGroups gets the saved groups and keeps those accepted by the supplied rule.
// Any type with Matches(GroupView) bool can provide that rule.
func (service *GroupService) ListGroups(filter GroupFilter) ([]GroupView, error) {
	// No explicit selection rule means every group is eligible for listing.
	if filter == nil {
		filter = AllGroups{}
	}

	// Fetch the views once; each filter decides which views to keep.
	groups, err := service.reader.ListGroups()
	if err != nil {
		return nil, err
	}

	matchingGroups := []GroupView{}
	for _, group := range groups {
		if filter.Matches(group) {
			matchingGroups = append(matchingGroups, group)
		}
	}
	return matchingGroups, nil
}

// GetGroup asks the reader for one group and its members.
// Return its value and error unchanged so storage keeps its own responsibility.
func (service *GroupService) GetGroup(id string) (GroupDetail, error) {
	return service.reader.GetGroup(id)
}

// CreateGroup passes creation data to the writer.
// This small method is the application's entry point for creating a group.
func (service *GroupService) CreateGroup(input GroupInput) (GroupView, error) {
	return service.writer.CreateGroup(input)
}

// JoinGroup passes the two identities to the writer's membership operation.
// The writer must check and save together so concurrent requests stay safe.
func (service *GroupService) JoinGroup(groupID, profileID string) (GroupDetail, error) {
	return service.writer.JoinGroup(groupID, profileID)
}

// CreateProfile sends a name and budget to the profile writer.
// Its result contains the public profile fields, without exposing the budget.
func (service *ProfileService) CreateProfile(input ProfileInput) (ProfileView, error) {
	return service.writer.CreateProfile(input)
}
