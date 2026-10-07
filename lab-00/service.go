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

func (AllGroups) Matches(GroupView) bool {
	return true
}

type AffordableGroups struct {
	MonthlyBudget int
}

func (filter AffordableGroups) Matches(group GroupView) bool {
	return group.MaxMonthlyCost <= filter.MonthlyBudget
}

func NewGroupService(reader GroupReader, writer GroupWriter) *GroupService {
	return &GroupService{reader: reader, writer: writer}
}

func NewProfileService(writer ProfileWriter) *ProfileService {
	return &ProfileService{writer: writer}
}

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

	matchingGroups := make([]GroupView, 0, len(groups))
	for _, group := range groups {
		if filter.Matches(group) {
			matchingGroups = append(matchingGroups, group)
		}
	}
	return matchingGroups, nil
}

func (service *GroupService) GetGroup(id string) (GroupDetail, error) {
	return service.reader.GetGroup(id)
}

func (service *GroupService) CreateGroup(input GroupInput) (GroupView, error) {
	return service.writer.CreateGroup(input)
}

func (service *GroupService) JoinGroup(groupID, profileID string) (GroupDetail, error) {
	return service.writer.JoinGroup(groupID, profileID)
}

func (service *ProfileService) CreateProfile(input ProfileInput) (ProfileView, error) {
	return service.writer.CreateProfile(input)
}
