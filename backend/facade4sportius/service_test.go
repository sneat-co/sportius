package facade4sportius

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	sportius "github.com/sneat-co/ext-sportius/backend"
)

type fakeCorePort struct {
	mu sync.Mutex

	nextSpace      int
	nextContact    int
	nextInvitation int
	nextExpiry     string

	spaces            map[string]string
	contacts          map[string]string
	spaceContacts     map[string]map[string]CoreContact
	invitations       map[string]CoreInvitation
	invitationTargets map[string]fakeInvitationTarget
	displayNames      map[string]string
	spaceOwners       map[string]string
	spaceMembers      map[string]map[string]bool
	spaceKinds        map[string]sportius.SpaceKind

	createSpaces  []CreateSpaceInput
	updatedNames  []UpdateSpaceNameInput
	members       []EnsureSpaceMemberInput
	contactLinks  []LinkContactsInput
	spaceLinks    []LinkSpacesInput
	invitationOps []CoreInvitationInput
	acceptanceOps []CoreAcceptInvitationInput

	acceptClaimOverride *CoreInvitationClaim
}

type fakeInvitationTarget struct {
	spaceID    string
	contactID  string
	claimToken string
	status     sportius.InvitationStatus
	claim      CoreInvitationClaim
}

func newFakeCorePort() *fakeCorePort {
	return &fakeCorePort{
		spaces:            make(map[string]string),
		contacts:          make(map[string]string),
		spaceContacts:     make(map[string]map[string]CoreContact),
		invitations:       make(map[string]CoreInvitation),
		invitationTargets: make(map[string]fakeInvitationTarget),
		displayNames:      map[string]string{"owner": "Alex Owner", "member": "Morgan Member", "coach": "Casey Coach"},
		spaceOwners:       make(map[string]string),
		spaceMembers:      make(map[string]map[string]bool),
		spaceKinds:        make(map[string]sportius.SpaceKind),
	}
}

func (f *fakeCorePort) CreateSpace(_ context.Context, input CreateSpaceInput) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := string(input.Kind) + "\x00" + input.OwnerUserID + "\x00" + input.RequestID
	if id := f.spaces[key]; id != "" {
		return id, nil
	}
	f.nextSpace++
	id := fmt.Sprintf("%s-%d", input.Kind, f.nextSpace)
	f.spaces[key] = id
	f.spaceOwners[id] = input.OwnerUserID
	f.spaceMembers[id] = map[string]bool{input.OwnerUserID: true}
	f.spaceKinds[id] = input.Kind
	f.createSpaces = append(f.createSpaces, input)
	return id, nil
}

func (f *fakeCorePort) UpdateSpaceName(_ context.Context, input UpdateSpaceNameInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updatedNames = append(f.updatedNames, input)
	return nil
}

func (f *fakeCorePort) EnsureSpaceMember(_ context.Context, input EnsureSpaceMemberInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.members = append(f.members, input)
	if f.spaceMembers[input.SpaceID] == nil {
		f.spaceMembers[input.SpaceID] = make(map[string]bool)
	}
	if input.UserID != "" {
		f.spaceMembers[input.SpaceID][input.UserID] = true
	}
	return nil
}

func (f *fakeCorePort) CreateContact(_ context.Context, input CreateContactInput) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := input.SpaceID + "\x00" + input.RequestID
	if id := f.contacts[key]; id != "" {
		return id, nil
	}
	f.nextContact++
	id := fmt.Sprintf("contact-%d", f.nextContact)
	f.contacts[key] = id
	if f.spaceContacts[input.SpaceID] == nil {
		f.spaceContacts[input.SpaceID] = make(map[string]CoreContact)
	}
	f.spaceContacts[input.SpaceID][id] = CoreContact{
		ContactID:   id,
		UserID:      input.UserID,
		DisplayName: input.DisplayName,
	}
	return id, nil
}

func (f *fakeCorePort) GetSpaceContact(_ context.Context, input GetSpaceContactInput) (CoreContact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	contact, ok := f.spaceContacts[input.SpaceID][input.ContactID]
	if !ok {
		return CoreContact{}, ErrNotFound
	}
	if f.spaceOwners[input.SpaceID] != input.ActorUserID {
		return CoreContact{}, ErrForbidden
	}
	return contact, nil
}

func (f *fakeCorePort) LinkContacts(_ context.Context, input LinkContactsInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.contactLinks = append(f.contactLinks, input)
	return nil
}

func (f *fakeCorePort) LinkSpaces(_ context.Context, input LinkSpacesInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.spaceLinks {
		if existing.RequestID == input.RequestID &&
			existing.FromSpaceID == input.FromSpaceID &&
			existing.ToSpaceID == input.ToSpaceID &&
			existing.Role == input.Role {
			return nil
		}
	}
	f.spaceLinks = append(f.spaceLinks, input)
	return nil
}

func (f *fakeCorePort) ResolveTeamClubLinks(_ context.Context, input ResolveTeamClubLinksInput) ([]CoreTeamClubLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]CoreTeamClubLink, 0)
	for _, link := range f.spaceLinks {
		if link.Role != "club" {
			continue
		}
		if input.TeamSpaceID != "" && link.FromSpaceID != input.TeamSpaceID {
			continue
		}
		if input.ClubSpaceID != "" && link.ToSpaceID != input.ClubSpaceID {
			continue
		}
		result = append(result, CoreTeamClubLink{
			TeamSpaceID: link.FromSpaceID,
			ClubSpaceID: link.ToSpaceID,
		})
	}
	return result, nil
}

func (f *fakeCorePort) CreateInvitation(_ context.Context, input CoreInvitationInput) (CoreInvitation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := input.ActorUserID + "\x00" + input.RequestID
	if invite, ok := f.invitations[key]; ok {
		return invite, nil
	}
	f.nextInvitation++
	invite := CoreInvitation{
		InvitationID: fmt.Sprintf("invite-%d", f.nextInvitation),
		DeepLink:     fmt.Sprintf("https://t.me/sneat_bot?start=invite-%d_token-%d", f.nextInvitation, f.nextInvitation),
		ExpiresAt:    f.nextExpiry,
	}
	f.invitations[key] = invite
	f.invitationTargets[invite.InvitationID] = fakeInvitationTarget{
		spaceID: input.SpaceID, contactID: input.ContactID,
		claimToken: fmt.Sprintf("token-%d", f.nextInvitation),
		status:     sportius.InvitationStatusPending,
	}
	f.invitationOps = append(f.invitationOps, input)
	return invite, nil
}

func (f *fakeCorePort) ResolveInvitation(_ context.Context, input CoreResolveInvitationInput) (CoreInvitationResolution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	target, ok := f.invitationTargets[input.InvitationID]
	if !ok {
		return CoreInvitationResolution{}, ErrNotFound
	}
	if input.ClaimToken == "" || input.ClaimToken != target.claimToken {
		return CoreInvitationResolution{}, ErrForbidden
	}
	return CoreInvitationResolution{
		InvitationID: input.InvitationID,
		SpaceID:      target.spaceID,
		ContactID:    target.contactID,
		Status:       target.status,
		Claim:        target.claim,
	}, nil
}

func (f *fakeCorePort) AcceptInvitation(_ context.Context, input CoreAcceptInvitationInput) (CoreInvitationClaim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	target, ok := f.invitationTargets[input.InvitationID]
	if !ok {
		return CoreInvitationClaim{}, ErrNotFound
	}
	if input.ClaimToken == "" || input.ClaimToken != target.claimToken {
		return CoreInvitationClaim{}, ErrForbidden
	}
	f.acceptanceOps = append(f.acceptanceOps, input)
	if target.status == sportius.InvitationStatusRevoked {
		return CoreInvitationClaim{}, ErrConflict
	}
	if target.status == sportius.InvitationStatusAccepted {
		if target.claim.UserID != input.ActorUserID {
			return CoreInvitationClaim{}, ErrConflict
		}
		return target.claim, nil
	}
	contact, ok := f.spaceContacts[target.spaceID][target.contactID]
	if !ok {
		return CoreInvitationClaim{}, ErrNotFound
	}
	contact.UserID = input.ActorUserID
	if f.spaceContacts[target.spaceID] == nil {
		f.spaceContacts[target.spaceID] = make(map[string]CoreContact)
	}
	f.spaceContacts[target.spaceID][target.contactID] = contact
	if f.spaceMembers[target.spaceID] == nil {
		f.spaceMembers[target.spaceID] = make(map[string]bool)
	}
	f.spaceMembers[target.spaceID][input.ActorUserID] = true
	claim := CoreInvitationClaim(contact)
	if f.acceptClaimOverride != nil {
		claim = *f.acceptClaimOverride
	}
	target.status = sportius.InvitationStatusAccepted
	target.claim = claim
	f.invitationTargets[input.InvitationID] = target
	return claim, nil
}

func (f *fakeCorePort) claimToken(invitationID string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.invitationTargets[invitationID].claimToken
}

func (f *fakeCorePort) UserDisplayName(_ context.Context, userID string) (string, error) {
	return f.displayNames[userID], nil
}

func (f *fakeCorePort) GetPersonalSpaceID(_ context.Context, actorUserID string) (string, error) {
	return "personal-" + actorUserID, nil
}

func (f *fakeCorePort) GetSpaceAccess(_ context.Context, actorUserID, spaceID string) (SpaceAccess, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return SpaceAccess{
		IsMember:  f.spaceMembers[spaceID][actorUserID],
		CanManage: f.spaceOwners[spaceID] == actorUserID,
	}, nil
}

func (f *fakeCorePort) ListUserSportSpaces(_ context.Context, actorUserID string) ([]UserSportSpaceAccess, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]UserSportSpaceAccess, 0)
	for spaceID, kind := range f.spaceKinds {
		isMember := f.spaceMembers[spaceID][actorUserID]
		canManage := f.spaceOwners[spaceID] == actorUserID
		if !isMember && !canManage {
			continue
		}
		result = append(result, UserSportSpaceAccess{
			SpaceID: spaceID, Kind: kind, IsMember: isMember, CanManage: canManage,
		})
	}
	return result, nil
}

func (f *fakeCorePort) ListSpaceMembers(_ context.Context, spaceID string) ([]CoreSpaceMember, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]CoreSpaceMember, 0, len(f.spaceMembers[spaceID]))
	for userID, isMember := range f.spaceMembers[spaceID] {
		if isMember {
			result = append(result, CoreSpaceMember{UserID: userID})
		}
	}
	return result, nil
}

type serviceFixture struct {
	service    *Service
	repository *MemoryRepository
	core       *fakeCorePort
}

// mutateBeforeUpdateRepository injects a concurrent committed write immediately
// before the service transaction. It makes lost-update regressions
// deterministic without relying on goroutine scheduling.
type mutateBeforeUpdateRepository struct {
	delegate     Repository
	nextMutation func(RepositoryWriter) error
}

func (r *mutateBeforeUpdateRepository) View(ctx context.Context, fn func(RepositoryReader) error) error {
	return r.delegate.View(ctx, fn)
}

func (r *mutateBeforeUpdateRepository) Update(ctx context.Context, fn func(RepositoryWriter) error) error {
	if r.nextMutation != nil {
		mutation := r.nextMutation
		r.nextMutation = nil
		if err := r.delegate.Update(ctx, mutation); err != nil {
			return err
		}
	}
	return r.delegate.Update(ctx, fn)
}

func newServiceFixture() serviceFixture {
	repository := NewMemoryRepository()
	core := newFakeCorePort()
	return serviceFixture{
		service:    NewService(repository, core),
		repository: repository,
		core:       core,
	}
}

func createTeam(t *testing.T, fixture serviceFixture, owner, requestID, name string, policy sportius.JoinPolicy) sportius.TeamView {
	t.Helper()
	view, err := fixture.service.CreateTeam(context.Background(), owner, sportius.CreateTeamRequest{
		RequestID:      requestID,
		Name:           name,
		SportID:        sportius.SportBasketball,
		CreatorRoleIDs: []sportius.RoleID{sportius.RoleCoach},
		JoinPolicy:     policy,
	})
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	return view
}

func createClub(t *testing.T, fixture serviceFixture, owner, requestID, name string) sportius.ClubView {
	t.Helper()
	view, err := fixture.service.CreateClub(context.Background(), owner, sportius.CreateClubRequest{
		RequestID:      requestID,
		Name:           name,
		PrimarySportID: sportius.SportBasketball,
		CreatorRoleIDs: []sportius.RoleID{sportius.RolePresident},
	})
	if err != nil {
		t.Fatalf("CreateClub: %v", err)
	}
	return view
}

func TestPersonalSportsProfileLifecycleAndHome(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()

	empty, err := fixture.service.GetPersonalProfile(ctx, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if empty.UserID != "owner" || len(empty.Sports) != 0 {
		t.Fatalf("empty profile = %#v", empty)
	}

	profile, err := fixture.service.PutPersonalSport(ctx, "owner", sportius.SportBasketball, sportius.PutPersonalSportRequest{
		RoleIDs: []sportius.RoleID{sportius.RolePlayer, sportius.RoleCoach, sportius.RolePlayer},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := profile.Sports[0]; got.Visibility != sportius.VisibilityPrivate || len(got.RoleIDs) != 2 {
		t.Fatalf("saved sport = %#v", got)
	}

	createTeam(t, fixture, "owner", "team-1", "Limerick Celtics", sportius.JoinPolicyOpen)
	createClub(t, fixture, "owner", "club-1", "Limerick Celtics")
	home, err := fixture.service.GetHome(ctx, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if len(home.Sports) != 1 || len(home.Teams) != 1 || len(home.Clubs) != 1 {
		t.Fatalf("home = %#v", home)
	}

	profile, err = fixture.service.DeletePersonalSport(ctx, "owner", sportius.SportBasketball)
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Sports) != 0 {
		t.Fatalf("profile after delete = %#v", profile)
	}
}

func TestPersonalSportRejectsClubOnlyRole(t *testing.T) {
	fixture := newServiceFixture()
	_, err := fixture.service.PutPersonalSport(context.Background(), "owner", sportius.SportBasketball, sportius.PutPersonalSportRequest{
		RoleIDs: []sportius.RoleID{sportius.RolePresident},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}

func TestSportsHomeUsesAuthoritativeGenericSpaceMembership(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Authoritative Team", sportius.JoinPolicyInviteOnly)
	club := createClub(t, fixture, "owner", "club", "Authoritative Club")

	// Generic membership added outside Sportius is enough to discover the
	// corresponding Sportius profile; extension member-role maps are not.
	fixture.core.mu.Lock()
	fixture.core.spaceMembers[team.Profile.SpaceID]["member"] = true
	fixture.core.spaceMembers[club.Profile.SpaceID]["member"] = true
	fixture.core.mu.Unlock()
	memberHome, err := fixture.service.GetHome(ctx, "member")
	if err != nil {
		t.Fatal(err)
	}
	if len(memberHome.Teams) != 1 || len(memberHome.Clubs) != 1 {
		t.Fatalf("generic memberships missing from home: %#v", memberHome)
	}

	// A removed generic member/owner must disappear even while stale Sportius
	// owner and member projections remain.
	fixture.core.mu.Lock()
	fixture.core.spaceMembers[team.Profile.SpaceID]["owner"] = false
	fixture.core.spaceMembers[club.Profile.SpaceID]["owner"] = false
	delete(fixture.core.spaceOwners, team.Profile.SpaceID)
	delete(fixture.core.spaceOwners, club.Profile.SpaceID)
	fixture.core.mu.Unlock()
	ownerHome, err := fixture.service.GetHome(ctx, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if len(ownerHome.Teams) != 0 || len(ownerHome.Clubs) != 0 {
		t.Fatalf("stale Sportius projection granted home membership: %#v", ownerHome)
	}
}

func TestMemoryRepositoryRollsBackFailedUpdateAndCopiesValues(t *testing.T) {
	repository := NewMemoryRepository()
	ctx := context.Background()
	expected := errors.New("fail")
	err := repository.Update(ctx, func(writer RepositoryWriter) error {
		writer.PutPersonalProfile(profileRecord("owner", sportius.SportBasketball))
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("Update error = %v", err)
	}
	if err = repository.View(ctx, func(reader RepositoryReader) error {
		if _, ok := reader.GetPersonalProfile(personalProfileRef("owner")); ok {
			t.Fatal("failed transaction was committed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err = repository.Update(ctx, func(writer RepositoryWriter) error {
		writer.PutPersonalProfile(profileRecord("owner", sportius.SportBasketball))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err = repository.View(ctx, func(reader RepositoryReader) error {
		record, _ := reader.GetPersonalProfile(personalProfileRef("owner"))
		delete(record.Sports, sportius.SportBasketball)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	_ = repository.View(ctx, func(reader RepositoryReader) error {
		record, _ := reader.GetPersonalProfile(personalProfileRef("owner"))
		if len(record.Sports) != 1 {
			t.Fatal("read result mutated repository state")
		}
		return nil
	})
}

func TestProfileUpdatesPatchLatestRecordWithoutLosingConcurrentFields(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Original Team", sportius.JoinPolicyOpen)
	club := createClub(t, fixture, "owner", "club", "Original Club")
	wrapper := &mutateBeforeUpdateRepository{delegate: fixture.repository}
	service := NewService(wrapper, fixture.core)

	wrapper.nextMutation = func(writer RepositoryWriter) error {
		record, ok := writer.GetTeam(team.Profile.SpaceID)
		if !ok {
			return ErrNotFound
		}
		record.Profile.Location = &sportius.LocationHint{Locality: "Cork", CountryID: "IE"}
		writer.PutTeam(record)
		return nil
	}
	teamName := "Renamed Team"
	updatedTeam, err := service.UpdateTeam(ctx, "owner", team.Profile.SpaceID, sportius.UpdateTeamRequest{
		RequestID: "rename-team",
		Name:      &teamName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updatedTeam.Profile.Name != teamName ||
		updatedTeam.Profile.Location == nil ||
		updatedTeam.Profile.Location.Locality != "Cork" {
		t.Fatalf("team update lost concurrent location: %#v", updatedTeam.Profile)
	}

	wrapper.nextMutation = func(writer RepositoryWriter) error {
		record, ok := writer.GetClub(club.Profile.SpaceID)
		if !ok {
			return ErrNotFound
		}
		record.Profile.SportIDs = []sportius.SportID{sportius.SportRugby}
		writer.PutClub(record)
		return nil
	}
	clubName := "Renamed Club"
	updatedClub, err := service.UpdateClub(ctx, "owner", club.Profile.SpaceID, sportius.UpdateClubRequest{
		RequestID: "rename-club",
		Name:      &clubName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updatedClub.Profile.Name != clubName || !hasSport(updatedClub.Profile.SportIDs, sportius.SportRugby) {
		t.Fatalf("club update lost concurrent sports: %#v", updatedClub.Profile)
	}
}
