package facade4sportius

import (
	"context"
	"errors"
	"testing"

	sportius "github.com/sneat-co/ext-sportius/backend"
)

func TestOutsiderCanBrowseProfilesButNotContactsOrCapabilities(t *testing.T) {
	fixture := newServiceFixture()
	team := createTeam(t, fixture, "owner", "team", "Private Roster", sportius.JoinPolicyOpen)
	_, err := fixture.service.AddTeamPlayer(context.Background(), "owner", team.Profile.SpaceID, sportius.AddPlayerRequest{
		RequestID: "player", DisplayName: "Private Player",
	})
	if err != nil {
		t.Fatal(err)
	}
	club := createClub(t, fixture, "owner", "club", "Private Club")
	_, err = fixture.service.LinkTeamToClub(context.Background(), "owner", sportius.LinkTeamToClubRequest{
		RequestID: "link", TeamSpaceID: team.Profile.SpaceID, ClubSpaceID: club.Profile.SpaceID,
	})
	if err != nil {
		t.Fatal(err)
	}

	teamView, err := fixture.service.GetTeam(context.Background(), "outsider", team.Profile.SpaceID)
	if err != nil {
		t.Fatal(err)
	}
	if teamView.Profile.Name == "" || len(teamView.Players) != 0 || len(teamView.Staff) != 0 || teamView.Capabilities.CanEdit {
		t.Fatalf("outsider team view leaked contacts/capability: %#v", teamView)
	}
	if _, err = fixture.service.GetTeamPlayer(context.Background(), "outsider", team.Profile.SpaceID, "contact-unknown"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("outsider player access error = %v, want forbidden", err)
	}
	clubView, err := fixture.service.GetClub(context.Background(), "outsider", club.Profile.SpaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(clubView.Teams) != 1 || len(clubView.Staff) != 0 || len(clubView.Members) != 0 || clubView.Capabilities.CanInvite {
		t.Fatalf("outsider club view = %#v", clubView)
	}
}

func TestSportiusProjectionCannotGrantManagementAccess(t *testing.T) {
	fixture := newServiceFixture()
	team := createTeam(t, fixture, "owner", "team", "Authoritative Access", sportius.JoinPolicyOpen)
	err := fixture.repository.Update(context.Background(), func(writer RepositoryWriter) error {
		record, _ := writer.GetTeam(team.Profile.SpaceID)
		record.OwnerUserIDs["outsider"] = true
		record.MemberUserRoles["outsider"] = []sportius.RoleID{sportius.RoleAdministrator}
		writer.PutTeam(record)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	name := "Cross-space rename"
	_, err = fixture.service.UpdateTeam(context.Background(), "outsider", team.Profile.SpaceID, sportius.UpdateTeamRequest{RequestID: "outsider-team", Name: &name})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want forbidden despite projection", err)
	}
	var contractError *sportius.Error
	if !errors.As(err, &contractError) || contractError.Code != sportius.ErrorCodeForbidden {
		t.Fatalf("contract error = %#v", contractError)
	}

	club := createClub(t, fixture, "owner", "club", "Authoritative Club")
	err = fixture.repository.Update(context.Background(), func(writer RepositoryWriter) error {
		record, _ := writer.GetClub(club.Profile.SpaceID)
		record.OwnerUserIDs["outsider"] = true
		record.MemberRoles["outsider"] = []sportius.RoleID{sportius.RolePresident}
		writer.PutClub(record)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	clubName := "Cross-space club rename"
	_, err = fixture.service.UpdateClub(context.Background(), "outsider", club.Profile.SpaceID, sportius.UpdateClubRequest{RequestID: "outsider-club", Name: &clubName})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("club error = %v, want forbidden despite projection", err)
	}
}

func TestExplicitClearFlagsAndRichTeamBrief(t *testing.T) {
	fixture := newServiceFixture()
	minAge := 14
	team, err := fixture.service.CreateTeam(context.Background(), "owner", sportius.CreateTeamRequest{
		RequestID: "team", Name: "U14 Girls", SportID: sportius.SportBasketball,
		Gender:   sportius.GenderFemale,
		Age:      &sportius.AgeRange{MinAge: &minAge, Label: "U14"},
		Location: &sportius.LocationHint{Locality: "Limerick"},
		Media:    &sportius.MediaRef{FileID: "logo", Kind: "logo"},
	})
	if err != nil {
		t.Fatal(err)
	}
	briefs, err := fixture.service.SearchTeams(context.Background(), "outsider", sportius.SearchRequest{Name: "U14 Girls"})
	if err != nil {
		t.Fatal(err)
	}
	if len(briefs) != 1 || briefs[0].Gender != sportius.GenderFemale || briefs[0].Age == nil ||
		briefs[0].Locality != "Limerick" || briefs[0].JoinPolicy != sportius.JoinPolicyOpen {
		t.Fatalf("rich brief = %#v", briefs)
	}
	cleared, err := fixture.service.UpdateTeam(context.Background(), "owner", team.Profile.SpaceID, sportius.UpdateTeamRequest{
		RequestID: "clear-team", ClearAge: true, ClearLocation: true, ClearMedia: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Profile.Age != nil || cleared.Profile.Location != nil || cleared.Profile.Media != nil {
		t.Fatalf("clear result = %#v", cleared.Profile)
	}

	club, err := fixture.service.CreateClub(context.Background(), "owner", sportius.CreateClubRequest{
		RequestID: "club", Name: "Multi Sport", PrimarySportID: sportius.SportBasketball,
		SportIDs: []sportius.SportID{sportius.SportBasketball}, Location: &sportius.LocationHint{Locality: "Cork"},
		Media: &sportius.MediaRef{FileID: "club-logo", Kind: "logo"},
	})
	if err != nil {
		t.Fatal(err)
	}
	clubCleared, err := fixture.service.UpdateClub(context.Background(), "owner", club.Profile.SpaceID, sportius.UpdateClubRequest{
		RequestID:         "clear-club",
		ClearPrimarySport: true, ReplaceSportIDs: true, SportIDs: []sportius.SportID{},
		ClearLocation: true, ClearMedia: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if clubCleared.Profile.PrimarySportID != "" || len(clubCleared.Profile.SportIDs) != 0 ||
		clubCleared.Profile.Location != nil || clubCleared.Profile.Media != nil {
		t.Fatalf("club clear result = %#v", clubCleared.Profile)
	}
}

func TestValidationUsesStableContractError(t *testing.T) {
	fixture := newServiceFixture()
	_, err := fixture.service.CreateTeam(context.Background(), "owner", sportius.CreateTeamRequest{})
	var contractError *sportius.Error
	if !errors.As(err, &contractError) {
		t.Fatalf("error %v is not contract Error", err)
	}
	if contractError.Code != sportius.ErrorCodeValidation || contractError.MessageKey == "" {
		t.Fatalf("contract error = %#v", contractError)
	}
}
