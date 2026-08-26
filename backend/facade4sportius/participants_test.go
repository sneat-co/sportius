package facade4sportius

import (
	"context"
	"errors"
	"testing"

	sportius "github.com/sneat-co/sneat-ext-contracts/sportius"
)

func TestAddPlayerWithDisplayNameOnlyMakesSpaceMember(t *testing.T) {
	fixture := newServiceFixture()
	team := createTeam(t, fixture, "owner", "create", "Players Team", sportius.JoinPolicyOpen)

	playerView, err := fixture.service.AddTeamPlayer(context.Background(), "owner", team.Profile.SpaceID, sportius.AddPlayerRequest{
		RequestID: "player-1", DisplayName: "  Magic   Mike ",
	})
	if err != nil {
		t.Fatal(err)
	}
	player := playerView.Player
	if player.DisplayName != "Magic Mike" || !player.SpaceMember || player.UserID != "" || !hasRole(player.RoleIDs, sportius.RolePlayer) {
		t.Fatalf("player = %#v", player)
	}
	if len(fixture.core.members) != 2 {
		t.Fatalf("membership calls = %#v", fixture.core.members)
	}
	repeated, err := fixture.service.AddTeamPlayer(context.Background(), "owner", team.Profile.SpaceID, sportius.AddPlayerRequest{
		RequestID: "player-1", DisplayName: "Magic Mike",
	})
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Player.ContactID != player.ContactID || len(fixture.core.members) != 2 {
		t.Fatalf("idempotent player = %#v; calls=%d", repeated, len(fixture.core.members))
	}
	view, err := fixture.service.GetTeam(context.Background(), "owner", team.Profile.SpaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Players) != 1 || view.Players[0].ContactID != player.ContactID {
		t.Fatalf("team players = %#v", view.Players)
	}
}

func TestParticipantAndRoleRequestIDsRejectDifferentPayloads(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "create", "Idempotent Team", sportius.JoinPolicyOpen)
	player, err := fixture.service.AddTeamPlayer(ctx, "owner", team.Profile.SpaceID, sportius.AddPlayerRequest{
		RequestID: "player", DisplayName: "First Name",
	})
	if err != nil {
		t.Fatal(err)
	}
	contactsBefore, membersBefore := fixture.core.nextContact, len(fixture.core.members)
	_, err = fixture.service.AddTeamPlayer(ctx, "owner", team.Profile.SpaceID, sportius.AddPlayerRequest{
		RequestID: "player", DisplayName: "Different Name",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("participant conflict error = %v", err)
	}
	if fixture.core.nextContact != contactsBefore || len(fixture.core.members) != membersBefore {
		t.Fatalf("conflicting participant replay mutated core: contacts=%d members=%d", fixture.core.nextContact, len(fixture.core.members))
	}

	_, err = fixture.service.SetParticipantRoles(
		ctx, "owner", sportius.SpaceKindTeam, team.Profile.SpaceID, player.Player.ContactID,
		sportius.SetParticipantRolesRequest{
			RequestID: "roles", RoleIDs: []sportius.RoleID{sportius.RolePlayer},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.service.SetParticipantRoles(
		ctx, "owner", sportius.SpaceKindTeam, team.Profile.SpaceID, player.Player.ContactID,
		sportius.SetParticipantRolesRequest{
			RequestID: "roles", RoleIDs: []sportius.RoleID{sportius.RoleCoach},
		},
	)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("role conflict error = %v", err)
	}
}

func TestGuardianRequestIDRejectsDifferentRelationship(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "create", "Guardian Team", sportius.JoinPolicyOpen)
	player, err := fixture.service.AddTeamPlayer(ctx, "owner", team.Profile.SpaceID, sportius.AddPlayerRequest{
		RequestID: "player", DisplayName: "Player",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := sportius.LinkGuardianRequest{
		RequestID: "guardian", GuardianDisplayName: "Guardian", RelationshipRoleID: "parent",
	}
	if _, err = fixture.service.LinkGuardian(ctx, "owner", team.Profile.SpaceID, player.Player.ContactID, request); err != nil {
		t.Fatal(err)
	}
	contactsBefore, linksBefore := fixture.core.nextContact, len(fixture.core.contactLinks)
	request.RelationshipRoleID = "grandparent"
	_, err = fixture.service.LinkGuardian(ctx, "owner", team.Profile.SpaceID, player.Player.ContactID, request)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("guardian conflict error = %v", err)
	}
	if fixture.core.nextContact != contactsBefore || len(fixture.core.contactLinks) != linksBefore {
		t.Fatal("conflicting guardian replay mutated core")
	}
}

func TestAddStaffLinkGuardianAndReadPlayerView(t *testing.T) {
	fixture := newServiceFixture()
	team := createTeam(t, fixture, "owner", "create", "Family Team", sportius.JoinPolicyOpen)
	playerView, err := fixture.service.AddTeamPlayer(context.Background(), "owner", team.Profile.SpaceID, sportius.AddPlayerRequest{
		RequestID: "player", DisplayName: "Jamie",
	})
	if err != nil {
		t.Fatal(err)
	}
	staff, err := fixture.service.AddTeamStaff(context.Background(), "owner", team.Profile.SpaceID, sportius.AddStaffRequest{
		RequestID: "staff", DisplayName: "Coach Pat",
		RoleIDs: []sportius.RoleID{sportius.RoleCoach, sportius.RoleAssistantCoach},
	})
	if err != nil || !staff.SpaceMember {
		t.Fatalf("staff=%#v err=%v", staff, err)
	}
	membersBeforeGuardian := len(fixture.core.members)
	linked, err := fixture.service.LinkGuardian(context.Background(), "owner", team.Profile.SpaceID, playerView.Player.ContactID, sportius.LinkGuardianRequest{
		RequestID: "guardian", GuardianDisplayName: "Sam Parent", RelationshipRoleID: "parent",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(linked.Guardians) != 1 || linked.Guardians[0].Contact.DisplayName != "Sam Parent" {
		t.Fatalf("player view = %#v", linked)
	}
	if len(fixture.core.members) != membersBeforeGuardian {
		t.Fatal("guardian unexpectedly became a member")
	}
	guardianContactID := linked.Guardians[0].Contact.ContactID
	guardian, err := fixture.service.SetParticipantRoles(
		context.Background(),
		"owner",
		sportius.SpaceKindTeam,
		team.Profile.SpaceID,
		guardianContactID,
		sportius.SetParticipantRolesRequest{
			RequestID: "guardian-role", RoleIDs: []sportius.RoleID{sportius.RoleParentGuardian},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if guardian.SpaceMember || len(fixture.core.members) != membersBeforeGuardian {
		t.Fatalf("guardian role incorrectly implied membership: %#v", guardian)
	}
	read, err := fixture.service.GetTeamPlayer(context.Background(), "owner", team.Profile.SpaceID, playerView.Player.ContactID)
	if err != nil || len(read.Guardians) != 1 {
		t.Fatalf("read player=%#v err=%v", read, err)
	}
	guardians, err := fixture.service.ListTeamGuardians(context.Background(), "owner", team.Profile.SpaceID)
	if err != nil || len(guardians) != 1 || guardians[0].ContactID != guardianContactID {
		t.Fatalf("guardians=%#v err=%v", guardians, err)
	}
	if _, err = fixture.service.ListTeamGuardians(context.Background(), "outsider", team.Profile.SpaceID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("outsider list guardians error=%v", err)
	}
	secondPlayer, err := fixture.service.AddTeamPlayer(context.Background(), "owner", team.Profile.SpaceID, sportius.AddPlayerRequest{
		RequestID: "player-two", DisplayName: "Alex",
	})
	if err != nil {
		t.Fatal(err)
	}
	secondLinked, err := fixture.service.LinkGuardian(context.Background(), "owner", team.Profile.SpaceID, secondPlayer.Player.ContactID, sportius.LinkGuardianRequest{
		RequestID: "guardian-reuse", GuardianContactID: guardianContactID,
		GuardianDisplayName: guardians[0].DisplayName, RelationshipRoleID: "parent",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(secondLinked.Guardians) != 1 || secondLinked.Guardians[0].Contact.ContactID != guardianContactID {
		t.Fatalf("reused guardian link=%#v", secondLinked)
	}
	view, _ := fixture.service.GetTeam(context.Background(), "owner", team.Profile.SpaceID)
	if len(view.Players) != 2 || len(view.Staff) != 2 {
		t.Fatalf("team view = %#v", view)
	}
}

func TestAddClubStaffAndSetParticipantRoles(t *testing.T) {
	fixture := newServiceFixture()
	club := createClub(t, fixture, "owner", "club", "Staff Club")
	staff, err := fixture.service.AddClubStaff(context.Background(), "owner", club.Profile.SpaceID, sportius.AddStaffRequest{
		RequestID: "staff", DisplayName: "Terry Treasurer", UserID: "member",
		RoleIDs: []sportius.RoleID{sportius.RoleTreasurer},
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := fixture.service.SetParticipantRoles(context.Background(), "owner", sportius.SpaceKindClub, club.Profile.SpaceID, staff.ContactID, sportius.SetParticipantRolesRequest{
		RequestID: "roles", RoleIDs: []sportius.RoleID{sportius.RoleSecretary, sportius.RoleVolunteer},
	})
	if err != nil || len(updated.RoleIDs) != 2 {
		t.Fatalf("updated=%#v err=%v", updated, err)
	}
	view, _ := fixture.service.GetClub(context.Background(), "owner", club.Profile.SpaceID)
	if len(view.Staff) != 2 || len(view.Members) != 2 {
		t.Fatalf("club view = %#v", view)
	}
}
