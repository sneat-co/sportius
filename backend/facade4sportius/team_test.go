package facade4sportius

import (
	"context"
	"errors"
	"testing"

	sportius "github.com/sneat-co/ext-sportius/backend"
)

func TestCreateSearchUpdateTeamAndAllowDuplicateNames(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	minAge, maxAge := 12, 14
	first, err := fixture.service.CreateTeam(ctx, "owner", sportius.CreateTeamRequest{
		RequestID:      "create-team-1",
		Name:           "  Limerick   Celtics  ",
		SportID:        sportius.SportBasketball,
		CreatorRoleIDs: []sportius.RoleID{sportius.RoleCoach},
		Gender:         sportius.GenderFemale,
		Age:            &sportius.AgeRange{MinAge: &minAge, MaxAge: &maxAge, Label: "U14"},
		Location:       &sportius.LocationHint{Locality: "Limerick", CountryID: "ie"},
		Media:          &sportius.MediaRef{FileID: "telegram-file", Kind: "logo"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Profile.Name != "Limerick Celtics" ||
		first.Profile.JoinPolicy != sportius.JoinPolicyOpen ||
		first.Profile.Location.CountryID != "IE" ||
		len(first.Staff) != 1 {
		t.Fatalf("created team = %#v", first)
	}
	if len(fixture.core.createSpaces) != 1 || fixture.core.createSpaces[0].OwnerUserID != "owner" {
		t.Fatalf("space creation calls = %#v", fixture.core.createSpaces)
	}
	if len(fixture.core.members) == 0 || !fixture.core.members[0].Owner {
		t.Fatalf("creator membership calls = %#v", fixture.core.members)
	}

	second := createTeam(t, fixture, "owner", "create-team-2", "Limerick Celtics", sportius.JoinPolicyOpen)
	if second.Profile.SpaceID == first.Profile.SpaceID {
		t.Fatal("duplicate names must still create distinct spaces")
	}

	results, err := fixture.service.SearchTeams(ctx, "member", sportius.SearchRequest{
		Name:     "limerick---CELTICS",
		SportID:  sportius.SportBasketball,
		Locality: "LIMERICK",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].SpaceID != first.Profile.SpaceID {
		t.Fatalf("search results = %#v", results)
	}

	renamed := "Limerick Celtics U14 Girls"
	mixed := sportius.GenderMixed
	updated, err := fixture.service.UpdateTeam(ctx, "owner", first.Profile.SpaceID, sportius.UpdateTeamRequest{
		Name:   &renamed,
		Gender: &mixed,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Profile.Name != renamed || updated.Profile.Gender != sportius.GenderMixed {
		t.Fatalf("updated team = %#v", updated)
	}
	if len(fixture.core.updatedNames) != 1 || fixture.core.updatedNames[0].Name != renamed {
		t.Fatalf("rename calls = %#v", fixture.core.updatedNames)
	}

	repeated, err := fixture.service.CreateTeam(ctx, "owner", sportius.CreateTeamRequest{
		RequestID:      "create-team-1",
		Name:           "Limerick Celtics",
		SportID:        sportius.SportBasketball,
		CreatorRoleIDs: []sportius.RoleID{sportius.RoleCoach},
		Gender:         sportius.GenderFemale,
		Age:            &sportius.AgeRange{MinAge: &minAge, MaxAge: &maxAge, Label: "U14"},
		Location:       &sportius.LocationHint{Locality: "Limerick", CountryID: "ie"},
		Media:          &sportius.MediaRef{FileID: "telegram-file", Kind: "logo"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Profile.SpaceID != first.Profile.SpaceID || len(fixture.core.createSpaces) != 2 {
		t.Fatalf("idempotent create = %#v; calls=%d", repeated, len(fixture.core.createSpaces))
	}
}

func TestCreateTeamSkipsOptionalFields(t *testing.T) {
	fixture := newServiceFixture()
	view, err := fixture.service.CreateTeam(context.Background(), "owner", sportius.CreateTeamRequest{
		RequestID: "minimal",
		Name:      "Park Court Basketball",
		SportID:   sportius.SportBasketball,
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Profile.Gender != sportius.GenderUnspecified ||
		view.Profile.Age != nil ||
		view.Profile.Location != nil ||
		view.Profile.Media != nil {
		t.Fatalf("minimal team = %#v", view.Profile)
	}
}

func TestCreateTeamRequestIDRejectsDifferentOptionalPayload(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	created, err := fixture.service.CreateTeam(ctx, "owner", sportius.CreateTeamRequest{
		RequestID: "same-create",
		Name:      "Fingerprint Team",
		SportID:   sportius.SportBasketball,
	})
	if err != nil {
		t.Fatal(err)
	}
	female := sportius.GenderFemale
	_, err = fixture.service.CreateTeam(ctx, "owner", sportius.CreateTeamRequest{
		RequestID: "same-create",
		Name:      "Fingerprint Team",
		SportID:   sportius.SportBasketball,
		Gender:    female,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if len(fixture.core.createSpaces) != 1 {
		t.Fatalf("conflicting replay created another core space: %#v", fixture.core.createSpaces)
	}
	view, err := fixture.service.GetTeam(ctx, "owner", created.Profile.SpaceID)
	if err != nil || view.Profile.Gender != sportius.GenderUnspecified {
		t.Fatalf("original team mutated: view=%#v err=%v", view, err)
	}
}

func TestJoinPoliciesAndRoleSelection(t *testing.T) {
	tests := []struct {
		name       string
		policy     sportius.JoinPolicy
		roles      []sportius.RoleID
		wantStatus sportius.JoinStatus
	}{
		{name: "open one role", policy: sportius.JoinPolicyOpen, roles: []sportius.RoleID{sportius.RolePlayer}, wantStatus: sportius.JoinStatusJoined},
		{name: "open multiple roles", policy: sportius.JoinPolicyOpen, roles: []sportius.RoleID{sportius.RolePlayer, sportius.RoleCoach}, wantStatus: sportius.JoinStatusJoined},
		{name: "open no role", policy: sportius.JoinPolicyOpen, roles: nil, wantStatus: sportius.JoinStatusJoined},
		{name: "open parent role", policy: sportius.JoinPolicyOpen, roles: []sportius.RoleID{sportius.RoleParentGuardian}, wantStatus: sportius.JoinStatusJoined},
		{name: "approval required", policy: sportius.JoinPolicyApprovalRequired, roles: []sportius.RoleID{sportius.RolePlayer}, wantStatus: sportius.JoinStatusRequested},
		{name: "invite only", policy: sportius.JoinPolicyInviteOnly, roles: []sportius.RoleID{sportius.RolePlayer}, wantStatus: sportius.JoinStatusInviteRequired},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newServiceFixture()
			team := createTeam(t, fixture, "owner", "create", "A Team", tc.policy)
			response, err := fixture.service.JoinTeam(context.Background(), "member", team.Profile.SpaceID, sportius.JoinTeamRequest{
				RequestID: "join",
				RoleIDs:   tc.roles,
			})
			if err != nil {
				t.Fatal(err)
			}
			if response.Status != tc.wantStatus || len(response.RoleIDs) != len(tc.roles) {
				t.Fatalf("response = %#v", response)
			}
			if tc.wantStatus == sportius.JoinStatusRequested && response.MembershipRequestID == "" {
				t.Fatalf("approval response has no membership request ID: %#v", response)
			}
			home, err := fixture.service.GetHome(context.Background(), "member")
			if err != nil {
				t.Fatal(err)
			}
			wantTeams := 0
			if tc.wantStatus == sportius.JoinStatusJoined {
				wantTeams = 1
			}
			if len(home.Teams) != wantTeams {
				t.Fatalf("home teams = %#v, want %d", home.Teams, wantTeams)
			}
		})
	}
}

func TestApprovalJoinIsPendingIdempotentAndDoesNotGrantMembership(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "create", "Approval Team", sportius.JoinPolicyApprovalRequired)
	membersBefore := len(fixture.core.members)
	request := sportius.JoinTeamRequest{
		RequestID: "approval-request",
		RoleIDs:   []sportius.RoleID{sportius.RolePlayer},
	}
	first, err := fixture.service.JoinTeam(ctx, "member", team.Profile.SpaceID, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := fixture.service.JoinTeam(ctx, "member", team.Profile.SpaceID, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != sportius.JoinStatusRequested ||
		second.MembershipRequestID != first.MembershipRequestID ||
		len(fixture.core.members) != membersBefore {
		t.Fatalf("approval replay first=%#v second=%#v members=%#v", first, second, fixture.core.members)
	}
	fixture.core.mu.Lock()
	isMember := fixture.core.spaceMembers[team.Profile.SpaceID]["member"]
	fixture.core.mu.Unlock()
	if isMember {
		t.Fatal("approval-required request granted generic membership")
	}
	view, err := fixture.service.GetTeam(ctx, "owner", team.Profile.SpaceID)
	if err != nil || len(view.Players) != 0 {
		t.Fatalf("pending join created a participant: view=%#v err=%v", view, err)
	}
	_, err = fixture.service.JoinTeam(ctx, "member", team.Profile.SpaceID, sportius.JoinTeamRequest{
		RequestID: "approval-request",
		RoleIDs:   []sportius.RoleID{sportius.RoleCoach},
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("different-role replay error = %v, want ErrConflict", err)
	}
}

func TestUpdateTeamRenameRequestIDsAreVersionedAcrossABA(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "create", "A", sportius.JoinPolicyOpen)
	b := "B"
	if _, err := fixture.service.UpdateTeam(ctx, "owner", team.Profile.SpaceID, sportius.UpdateTeamRequest{Name: &b}); err != nil {
		t.Fatal(err)
	}
	a := "A"
	if _, err := fixture.service.UpdateTeam(ctx, "owner", team.Profile.SpaceID, sportius.UpdateTeamRequest{Name: &a}); err != nil {
		t.Fatal(err)
	}
	if len(fixture.core.updatedNames) != 2 ||
		fixture.core.updatedNames[0].RequestID == fixture.core.updatedNames[1].RequestID {
		t.Fatalf("ABA rename keys = %#v", fixture.core.updatedNames)
	}
}

func TestInvitationBypassesInviteOnlyJoinPolicy(t *testing.T) {
	fixture := newServiceFixture()
	team := createTeam(t, fixture, "owner", "create", "Invite Team", sportius.JoinPolicyInviteOnly)
	invitation, err := fixture.service.CreateInvitation(context.Background(), "owner", sportius.CreateInvitationRequest{
		RequestID:          "invite",
		SpaceID:            team.Profile.SpaceID,
		Kind:               sportius.SpaceKindTeam,
		InviteeDisplayName: "Morgan Member",
		SuggestedRoleIDs:   []sportius.RoleID{sportius.RolePlayer, sportius.RoleCoach},
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := fixture.service.JoinTeam(context.Background(), "member", team.Profile.SpaceID, sportius.JoinTeamRequest{
		RequestID:    "join",
		RoleIDs:      []sportius.RoleID{sportius.RolePlayer},
		InvitationID: invitation.InvitationID,
		ClaimToken:   fixture.core.claimToken(invitation.InvitationID),
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != sportius.JoinStatusJoined {
		t.Fatalf("join response = %#v", response)
	}
}

func TestTeamManagementRequiresMembership(t *testing.T) {
	fixture := newServiceFixture()
	team := createTeam(t, fixture, "owner", "create", "Managed Team", sportius.JoinPolicyOpen)
	name := "Unauthorised rename"
	_, err := fixture.service.UpdateTeam(context.Background(), "outsider", team.Profile.SpaceID, sportius.UpdateTeamRequest{Name: &name})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestJoinTeamDoesNotTrustForgedMembershipProjection(t *testing.T) {
	fixture := newServiceFixture()
	team := createTeam(t, fixture, "owner", "create", "Invite-only Team", sportius.JoinPolicyInviteOnly)
	err := fixture.repository.Update(context.Background(), func(writer RepositoryWriter) error {
		record, _ := writer.GetTeam(team.Profile.SpaceID)
		record.MemberUserRoles["outsider"] = []sportius.RoleID{sportius.RolePlayer}
		writer.PutTeam(record)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := fixture.service.JoinTeam(context.Background(), "outsider", team.Profile.SpaceID, sportius.JoinTeamRequest{
		RequestID: "join", RoleIDs: []sportius.RoleID{sportius.RolePlayer},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != sportius.JoinStatusInviteRequired {
		t.Fatalf("forged projection bypassed policy: %#v", response)
	}
}

func TestJoinTeamRebuildsMissingProjectionForGenericMember(t *testing.T) {
	fixture := newServiceFixture()
	team := createTeam(t, fixture, "owner", "create", "Existing Member Team", sportius.JoinPolicyInviteOnly)
	fixture.core.mu.Lock()
	fixture.core.spaceMembers[team.Profile.SpaceID]["member"] = true
	fixture.core.mu.Unlock()

	response, err := fixture.service.JoinTeam(context.Background(), "member", team.Profile.SpaceID, sportius.JoinTeamRequest{
		RequestID: "rebuild", RoleIDs: []sportius.RoleID{sportius.RolePlayer},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != sportius.JoinStatusJoined {
		t.Fatalf("generic member was sent through join policy: %#v", response)
	}
	view, err := fixture.service.GetTeam(context.Background(), "member", team.Profile.SpaceID)
	if err != nil || len(view.Players) != 1 {
		t.Fatalf("rebuilt member view=%#v err=%v", view, err)
	}
}
