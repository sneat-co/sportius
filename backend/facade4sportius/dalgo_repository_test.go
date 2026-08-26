package facade4sportius

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dal-go/dalgo/dal"
	dalrecord "github.com/dal-go/record"
	sportius "github.com/sneat-co/sneat-ext-contracts/sportius"
	"github.com/sneat-co/sneat-go-core/sneatcoretesting"
	"github.com/sneat-co/sportius/backend/models4sportius"
)

func TestDalgoRepositoryPersistsCanonicalDocumentsAndIndexes(t *testing.T) {
	ctx := context.Background()
	db := sneatcoretesting.NewInMemoryTestDB()
	repository := NewDalgoRepository(db)
	core := newFakeCorePort()
	service := NewService(repository, core)

	if _, err := service.PutPersonalSport(ctx, "owner", sportius.SportBasketball, sportius.PutPersonalSportRequest{
		RoleIDs: []sportius.RoleID{sportius.RoleCoach},
	}); err != nil {
		t.Fatal(err)
	}
	team, err := service.CreateTeam(ctx, "owner", sportius.CreateTeamRequest{
		RequestID: "team", Name: "Persisted Team", SportID: sportius.SportBasketball,
		CreatorRoleIDs: []sportius.RoleID{sportius.RoleCoach},
	})
	if err != nil {
		t.Fatal(err)
	}
	club, err := service.CreateClub(ctx, "owner", sportius.CreateClubRequest{
		RequestID: "club", Name: "Persisted Club", PrimarySportID: sportius.SportBasketball,
		CreatorRoleIDs: []sportius.RoleID{sportius.RolePresident},
	})
	if err != nil {
		t.Fatal(err)
	}
	player, err := service.AddTeamPlayer(ctx, "owner", team.Profile.SpaceID, sportius.AddPlayerRequest{
		RequestID: "private-player", DisplayName: "Private Player",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.LinkGuardian(ctx, "owner", team.Profile.SpaceID, player.Player.ContactID, sportius.LinkGuardianRequest{
		RequestID: "private-guardian", GuardianDisplayName: "Private Guardian", RelationshipRoleID: "parent",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.AddClubStaff(ctx, "owner", club.Profile.SpaceID, sportius.AddStaffRequest{
		RequestID: "private-staff", DisplayName: "Private Treasurer",
		RoleIDs: []sportius.RoleID{sportius.RoleTreasurer},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.LinkTeamToClub(ctx, "owner", sportius.LinkTeamToClubRequest{
		RequestID: "persisted-link", TeamSpaceID: team.Profile.SpaceID, ClubSpaceID: club.Profile.SpaceID,
	}); err != nil {
		t.Fatal(err)
	}
	invitation, err := service.CreateInvitation(ctx, "owner", sportius.CreateInvitationRequest{
		RequestID:          "invite",
		SpaceID:            team.Profile.SpaceID,
		Kind:               sportius.SpaceKindTeam,
		InviteeDisplayName: "Invitee",
	})
	if err != nil {
		t.Fatal(err)
	}

	storedPersonalExtension := new(spaceExtensionDBO)
	assertDalgoRecordExists(t, db, spaceExtensionKey("personal-owner"), storedPersonalExtension)
	if storedPersonalExtension.Personal == nil ||
		storedPersonalExtension.Personal.SpaceID != "personal-owner" ||
		storedPersonalExtension.Personal.UserID != "owner" {
		t.Fatalf("personal profile is not owned by the personal Space: %#v", storedPersonalExtension.Personal)
	}
	storedTeamExtension := new(spaceExtensionDBO)
	assertDalgoRecordExists(t, db, spaceExtensionKey(team.Profile.SpaceID), storedTeamExtension)
	if storedTeamExtension.Team == nil || len(storedTeamExtension.Team.LinkRequestFingerprints) != 1 {
		t.Fatalf("team link request fingerprint not persisted: %#v", storedTeamExtension.Team)
	}
	assertDalgoRecordExists(t, db, spaceExtensionKey(club.Profile.SpaceID), new(spaceExtensionDBO))
	teamSearch := new(models4sportius.TeamSearchRecord)
	clubSearch := new(models4sportius.ClubSearchRecord)
	assertDalgoRecordExists(t, db, extensionItemKey(teamsIndexCollection, team.Profile.SpaceID), teamSearch)
	assertDalgoRecordExists(t, db, extensionItemKey(clubsIndexCollection, club.Profile.SpaceID), clubSearch)
	assertPublicSearchDocument(t, teamSearch)
	assertPublicSearchDocument(t, clubSearch)
	storedInvitation := new(models4sportius.InvitationRecord)
	assertDalgoRecordExists(t, db, invitationKey(invitation.InvitationID), storedInvitation)
	storedData, err := json.Marshal(storedInvitation)
	if err != nil {
		t.Fatal(err)
	}
	if storedInvitation.Invitation.DeepLink != "" ||
		strings.Contains(string(storedData), core.claimToken(invitation.InvitationID)) ||
		strings.Contains(string(storedData), invitation.DeepLink) {
		t.Fatalf("invitation proof was persisted: %s", storedData)
	}

	// A new repository/service instance sees the same canonical records and
	// discovery projections, proving this is not an in-process cache.
	reloaded := NewService(NewDalgoRepository(db), core)
	profile, err := reloaded.GetPersonalProfile(ctx, "owner")
	if err != nil || len(profile.Sports) != 1 {
		t.Fatalf("profile=%#v err=%v", profile, err)
	}
	teams, err := reloaded.SearchTeams(ctx, "owner", sportius.SearchRequest{Name: "Persisted Team"})
	if err != nil || len(teams) != 1 {
		t.Fatalf("teams=%#v err=%v", teams, err)
	}
	clubs, err := reloaded.SearchClubs(ctx, "owner", sportius.SearchRequest{Name: "Persisted Club"})
	if err != nil || len(clubs) != 1 {
		t.Fatalf("clubs=%#v err=%v", clubs, err)
	}
	inviteView, err := reloaded.GetInvitation(
		ctx, "member", invitation.InvitationID, core.claimToken(invitation.InvitationID),
	)
	if err != nil || inviteView.Status != sportius.InvitationStatusPending {
		t.Fatalf("invite=%#v err=%v", inviteView, err)
	}
}

func TestDalgoRepositoryLazilyMigratesLegacyUserProfileToPersonalSpace(t *testing.T) {
	ctx := context.Background()
	db := sneatcoretesting.NewInMemoryTestDB()
	legacy := profileRecord("owner", sportius.SportBasketball)
	legacy.SpaceID = ""
	if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, dalrecord.NewRecordWithData(legacyProfileKey("owner"), &legacy))
	}); err != nil {
		t.Fatal(err)
	}

	service := NewService(NewDalgoRepository(db), newFakeCorePort())
	profile, err := service.GetPersonalProfile(ctx, "owner")
	if err != nil || len(profile.Sports) != 1 {
		t.Fatalf("legacy profile=%#v err=%v", profile, err)
	}
	profile, err = service.PutPersonalSport(ctx, "owner", sportius.SportFootball, sportius.PutPersonalSportRequest{})
	if err != nil || len(profile.Sports) != 2 {
		t.Fatalf("migrated profile=%#v err=%v", profile, err)
	}

	stored := new(spaceExtensionDBO)
	assertDalgoRecordExists(t, db, spaceExtensionKey("personal-owner"), stored)
	if stored.Personal == nil ||
		stored.Personal.SpaceID != "personal-owner" ||
		stored.Personal.UserID != "owner" ||
		len(stored.Personal.Sports) != 2 {
		t.Fatalf("personal-Space profile after lazy migration = %#v", stored.Personal)
	}
}

func assertPublicSearchDocument(t *testing.T, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	payload := string(data)
	for _, privateField := range []string{
		"ownerUserIDs",
		"memberUserRoles",
		"memberRoles",
		"participants",
		"participantRequests",
		"guardianRequests",
		"guardianLinks",
		"joinRequests",
		"linkRequestFingerprints",
		"createdByUserID",
		"clubManagerRosterTeamIDs",
	} {
		if strings.Contains(payload, `"`+privateField+`"`) {
			t.Fatalf("public search document contains private field %q: %s", privateField, payload)
		}
	}
}

func assertDalgoRecordExists(t *testing.T, db dal.DB, key *dalrecord.Key, destination any) {
	t.Helper()
	err := db.RunReadonlyTransaction(context.Background(), func(ctx context.Context, tx dal.ReadTransaction) error {
		return tx.Get(ctx, dalrecord.NewRecordWithData(key, destination))
	})
	if err != nil {
		t.Fatalf("record %v not found: %v", key, err)
	}
}
