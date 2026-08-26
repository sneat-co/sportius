package facade4sportius

import (
	"context"
	"errors"
	"testing"

	sportius "github.com/sneat-co/sneat-ext-contracts/sportius"
)

func TestResolveTwoPlayerRosterReturnsStableAuthoritativeSnapshot(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team, err := fixture.service.CreateTeam(ctx, "owner", sportius.CreateTeamRequest{
		RequestID: "create", Name: "Roster Team", SportID: sportius.SportBasketball,
		CreatorRoleIDs: []sportius.RoleID{sportius.RolePlayer}, JoinPolicy: sportius.JoinPolicyOpen,
	})
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if _, err = fixture.service.JoinTeam(ctx, "member", team.Profile.SpaceID, sportius.JoinTeamRequest{
		RequestID: "join", RoleIDs: []sportius.RoleID{sportius.RolePlayer},
	}); err != nil {
		t.Fatalf("JoinTeam: %v", err)
	}
	// A non-player member must neither expose themselves in the roster nor
	// invalidate the two-player entry.
	if _, err = fixture.service.AddTeamStaff(ctx, "owner", team.Profile.SpaceID, sportius.AddStaffRequest{
		RequestID: "coach", DisplayName: "Casey Coach", UserID: "coach", RoleIDs: []sportius.RoleID{sportius.RoleCoach},
	}); err != nil {
		t.Fatalf("AddTeamStaff: %v", err)
	}

	snapshot, err := fixture.service.ResolveTwoPlayerRoster(ctx, sportius.TwoPlayerRosterRequest{TeamSpaceID: team.Profile.SpaceID})
	if err != nil {
		t.Fatalf("ResolveTwoPlayerRoster: %v", err)
	}
	if snapshot.SchemaVersion != sportius.TwoPlayerRosterSchemaVersion || snapshot.TeamSpaceID != team.Profile.SpaceID || snapshot.Version == "" {
		t.Fatalf("snapshot identity = %#v", snapshot)
	}
	if len(snapshot.Players) != 2 || snapshot.Players[0].UserID != "member" || snapshot.Players[1].UserID != "owner" {
		t.Fatalf("snapshot players = %#v", snapshot.Players)
	}
	// The response owns its slice: a consumer cannot mutate retained service
	// state by changing its accepted snapshot.
	snapshot.Players[0].UserID = "mutated"
	resolvedAgain, err := fixture.service.ResolveTwoPlayerRoster(ctx, sportius.TwoPlayerRosterRequest{TeamSpaceID: team.Profile.SpaceID})
	if err != nil || resolvedAgain.Players[0].UserID != "member" {
		t.Fatalf("immutable snapshot check = %#v, %v", resolvedAgain, err)
	}
}

func TestResolveTwoPlayerRosterRejectsStaleOrInvalidMembership(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team, err := fixture.service.CreateTeam(ctx, "owner", sportius.CreateTeamRequest{
		RequestID: "create", Name: "Roster Team", SportID: sportius.SportBasketball,
		CreatorRoleIDs: []sportius.RoleID{sportius.RolePlayer}, JoinPolicy: sportius.JoinPolicyOpen,
	})
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	for _, userID := range []string{"member", "replacement"} {
		if _, err = fixture.service.JoinTeam(ctx, userID, team.Profile.SpaceID, sportius.JoinTeamRequest{
			RequestID: "join-" + userID, RoleIDs: []sportius.RoleID{sportius.RolePlayer},
		}); err != nil {
			t.Fatalf("JoinTeam(%s): %v", userID, err)
		}
	}
	// Three active player members never becomes an arbitrary two-player entry.
	if _, err = fixture.service.ResolveTwoPlayerRoster(ctx, sportius.TwoPlayerRosterRequest{TeamSpaceID: team.Profile.SpaceID}); !errors.Is(err, ErrConflict) {
		t.Fatalf("three-player roster error = %v", err)
	}

	delete(fixture.core.spaceMembers[team.Profile.SpaceID], "replacement")
	accepted, err := fixture.service.ResolveTwoPlayerRoster(ctx, sportius.TwoPlayerRosterRequest{TeamSpaceID: team.Profile.SpaceID})
	if err != nil {
		t.Fatalf("Resolve accepted roster: %v", err)
	}
	delete(fixture.core.spaceMembers[team.Profile.SpaceID], "member")
	fixture.core.spaceMembers[team.Profile.SpaceID]["replacement"] = true
	if _, err = fixture.service.ResolveTwoPlayerRoster(ctx, sportius.TwoPlayerRosterRequest{
		TeamSpaceID: team.Profile.SpaceID, ExpectedVersion: accepted.Version,
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale roster error = %v", err)
	}
}

type noRosterCore struct{ CorePort }

func TestResolveTwoPlayerRosterFailsClosedWithoutHostRosterPort(t *testing.T) {
	fixture := newServiceFixture()
	service := NewService(fixture.repository, noRosterCore{fixture.core})
	_, err := service.ResolveTwoPlayerRoster(context.Background(), sportius.TwoPlayerRosterRequest{TeamSpaceID: "team-1"})
	var contractErr *sportius.Error
	if !errors.As(err, &contractErr) || contractErr.Code != sportius.ErrorCodeRetryable {
		t.Fatalf("missing roster port error = %#v", err)
	}
}
