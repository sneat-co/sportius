package facade4sportius

import (
	"context"
	"errors"
	"testing"

	sportius "github.com/sneat-co/sneat-ext-contracts/sportius"
)

func TestCreateSearchUpdateClubWithoutTeams(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	first, err := fixture.service.CreateClub(ctx, "owner", sportius.CreateClubRequest{
		RequestID:      "club-1",
		Name:           " Limerick   Celtics ",
		PrimarySportID: sportius.SportBasketball,
		SportIDs:       []sportius.SportID{sportius.SportBasketball, sportius.SportVolleyball},
		CreatorRoleIDs: []sportius.RoleID{sportius.RolePresident, sportius.RoleAdministrator},
		Location:       &sportius.LocationHint{Locality: "Limerick", CountryID: "ie"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Profile.Name != "Limerick Celtics" || len(first.Teams) != 0 || len(first.Staff) != 1 {
		t.Fatalf("created club = %#v", first)
	}
	second := createClub(t, fixture, "owner", "club-2", "Limerick Celtics")
	if first.Profile.SpaceID == second.Profile.SpaceID {
		t.Fatal("duplicate club names must be allowed")
	}
	results, err := fixture.service.SearchClubs(ctx, "member", sportius.SearchRequest{
		Name:     "LIMERICK---celtics",
		SportID:  sportius.SportVolleyball,
		Locality: "limerick",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].SpaceID != first.Profile.SpaceID {
		t.Fatalf("club search = %#v", results)
	}

	renamed := "Limerick Celtics Sports Club"
	primary := sportius.SportVolleyball
	updated, err := fixture.service.UpdateClub(ctx, "owner", first.Profile.SpaceID, sportius.UpdateClubRequest{
		RequestID:       "update-club-1",
		Name:            &renamed,
		PrimarySportID:  &primary,
		SportIDs:        []sportius.SportID{sportius.SportVolleyball},
		ReplaceSportIDs: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Profile.Name != renamed ||
		updated.Profile.PrimarySportID != sportius.SportVolleyball ||
		len(updated.Profile.SportIDs) != 1 {
		t.Fatalf("updated club = %#v", updated)
	}
}

func TestUpdateClubIsIdempotentAndRejectsRequestReuse(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	club := createClub(t, fixture, "owner", "create", "Original Club")
	renamed := "Renamed Club"
	request := sportius.UpdateClubRequest{RequestID: "same-update", Name: &renamed}
	first, err := fixture.service.UpdateClub(ctx, "owner", club.Profile.SpaceID, request)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := fixture.service.UpdateClub(ctx, "owner", club.Profile.SpaceID, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Profile.Name != renamed || replayed.Profile.Name != renamed || len(fixture.core.updatedNames) != 1 {
		t.Fatalf("first=%#v replay=%#v rename calls=%#v", first.Profile, replayed.Profile, fixture.core.updatedNames)
	}
	different := "Different Club"
	if _, err = fixture.service.UpdateClub(ctx, "owner", club.Profile.SpaceID, sportius.UpdateClubRequest{
		RequestID: "same-update", Name: &different,
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("request reuse error=%v", err)
	}
}

func TestCreateClubRequestIDRejectsDifferentOptionalPayload(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	_, err := fixture.service.CreateClub(ctx, "owner", sportius.CreateClubRequest{
		RequestID: "same-club", Name: "Fingerprint Club",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.service.CreateClub(ctx, "owner", sportius.CreateClubRequest{
		RequestID: "same-club", Name: "Fingerprint Club",
		Location: &sportius.LocationHint{Locality: "Cork", CountryID: "IE"},
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if len(fixture.core.createSpaces) != 1 {
		t.Fatalf("conflicting replay created another core space: %#v", fixture.core.createSpaces)
	}
}

func TestLinkTeamToClubAndAggregateMembers(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Celtics U14", sportius.JoinPolicyOpen)
	playerView, err := fixture.service.AddTeamPlayer(ctx, "owner", team.Profile.SpaceID, sportius.AddPlayerRequest{
		RequestID:   "player",
		DisplayName: "Taylor Player",
	})
	if err != nil {
		t.Fatal(err)
	}
	guardianView, err := fixture.service.LinkGuardian(ctx, "owner", team.Profile.SpaceID, playerView.Player.ContactID, sportius.LinkGuardianRequest{
		RequestID: "guardian", GuardianDisplayName: "Private Parent", RelationshipRoleID: "parent",
	})
	if err != nil || len(guardianView.Guardians) != 1 {
		t.Fatalf("guardian view=%#v err=%v", guardianView, err)
	}
	club := createClub(t, fixture, "owner", "club", "Limerick Celtics")
	linked, err := fixture.service.LinkTeamToClub(ctx, "owner", sportius.LinkTeamToClubRequest{
		RequestID:   "link",
		TeamSpaceID: team.Profile.SpaceID,
		ClubSpaceID: club.Profile.SpaceID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(linked.Teams) != 1 || linked.Teams[0].SpaceID != team.Profile.SpaceID {
		t.Fatalf("club teams = %#v", linked.Teams)
	}
	if len(linked.Staff) != 1 {
		t.Fatalf("club staff = %#v", linked.Staff)
	}
	if len(linked.Members) != 2 {
		t.Fatalf("club members = %#v; expected creator deduplicated plus player %s", linked.Members, playerView.Player.ContactID)
	}
	for _, member := range linked.Members {
		if member.DisplayName == "Private Parent" {
			t.Fatalf("guardian-only contact leaked into club members: %#v", linked.Members)
		}
	}
	if len(fixture.core.spaceLinks) != 1 {
		t.Fatalf("space links = %#v", fixture.core.spaceLinks)
	}
	invite, err := fixture.service.CreateInvitation(ctx, "owner", sportius.CreateInvitationRequest{
		RequestID:          "club-member-invite",
		SpaceID:            club.Profile.SpaceID,
		Kind:               sportius.SpaceKindClub,
		InviteeDisplayName: "Ordinary Club Member",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.service.AcceptInvitation(ctx, "member", invite.InvitationID, sportius.AcceptInvitationRequest{
		RequestID: "club-member-accept", ClaimToken: fixture.core.claimToken(invite.InvitationID),
	}); err != nil {
		t.Fatal(err)
	}
	memberView, err := fixture.service.GetClub(ctx, "member", club.Profile.SpaceID)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range memberView.Members {
		if member.ContactID == playerView.Player.ContactID || member.DisplayName == "Taylor Player" {
			t.Fatalf("ordinary club member received linked-team roster: %#v", memberView.Members)
		}
	}
	if len(memberView.Members) != 2 {
		t.Fatalf("ordinary member should see club contacts only: %#v", memberView.Members)
	}
	teamAfter, err := fixture.service.GetTeam(ctx, "owner", team.Profile.SpaceID)
	if err != nil {
		t.Fatal(err)
	}
	if teamAfter.Profile.Club == nil || teamAfter.Profile.Club.SpaceID != club.Profile.SpaceID {
		t.Fatalf("linked team profile = %#v", teamAfter.Profile)
	}

	repeated, err := fixture.service.LinkTeamToClub(ctx, "owner", sportius.LinkTeamToClubRequest{
		RequestID:   "link",
		TeamSpaceID: team.Profile.SpaceID,
		ClubSpaceID: club.Profile.SpaceID,
	})
	if err != nil || len(repeated.Teams) != 1 || len(fixture.core.spaceLinks) != 1 {
		t.Fatalf("idempotent link = %#v, err=%v, calls=%d", repeated, err, len(fixture.core.spaceLinks))
	}
}

func TestClubRosterAggregationRequiresExplicitAuthoritativePolicy(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Policy Team", sportius.JoinPolicyOpen)
	player, err := fixture.service.AddTeamPlayer(ctx, "owner", team.Profile.SpaceID, sportius.AddPlayerRequest{
		RequestID: "player", DisplayName: "Hidden Player",
	})
	if err != nil {
		t.Fatal(err)
	}
	club := createClub(t, fixture, "owner", "club", "Policy Club")
	fixture.core.mu.Lock()
	fixture.core.spaceLinks = append(fixture.core.spaceLinks, LinkSpacesInput{
		RequestID: "external-link", FromSpaceID: team.Profile.SpaceID,
		ToSpaceID: club.Profile.SpaceID, Role: "club",
		ActorUserID: "owner",
	})
	fixture.core.mu.Unlock()

	view, err := fixture.service.GetClub(ctx, "owner", club.Profile.SpaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Teams) != 1 || len(view.Members) != 1 {
		t.Fatalf("club view without roster policy = %#v", view)
	}
	for _, member := range view.Members {
		if member.ContactID == player.Player.ContactID {
			t.Fatalf("roster projected without explicit policy: %#v", view.Members)
		}
	}
}

func TestGenericTeamClubLinkageReconcilesProjectionCaches(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Reconcile Team", sportius.JoinPolicyOpen)
	club := createClub(t, fixture, "owner", "club", "Reconcile Club")

	// A stale positive cache must not suppress the authoritative generic write.
	if err := fixture.repository.Update(ctx, func(writer RepositoryWriter) error {
		storedTeam, _ := writer.GetTeam(team.Profile.SpaceID)
		storedClub, _ := writer.GetClub(club.Profile.SpaceID)
		brief := clubBrief(storedClub.Profile)
		storedTeam.Profile.Club = &brief
		storedClub.TeamSpaceIDs[storedTeam.Profile.SpaceID] = true
		writer.PutTeam(storedTeam)
		writer.PutClub(storedClub)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.LinkTeamToClub(ctx, "owner", sportius.LinkTeamToClubRequest{
		RequestID: "link", TeamSpaceID: team.Profile.SpaceID, ClubSpaceID: club.Profile.SpaceID,
	}); err != nil {
		t.Fatal(err)
	}
	if len(fixture.core.spaceLinks) != 1 {
		t.Fatalf("stale projection suppressed generic linkage write: %#v", fixture.core.spaceLinks)
	}

	// Removing the authoritative link clears both rebuildable caches.
	fixture.core.mu.Lock()
	fixture.core.spaceLinks = nil
	fixture.core.mu.Unlock()
	teamView, err := fixture.service.GetTeam(ctx, "owner", team.Profile.SpaceID)
	if err != nil {
		t.Fatal(err)
	}
	if teamView.Profile.Club != nil {
		t.Fatalf("stale team club cache survived reconciliation: %#v", teamView.Profile.Club)
	}
	clubView, err := fixture.service.GetClub(ctx, "owner", club.Profile.SpaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(clubView.Teams) != 0 {
		t.Fatalf("stale club team cache survived reconciliation: %#v", clubView.Teams)
	}

	// An externally-created generic link rebuilds missing caches on reads.
	if err = fixture.core.LinkSpaces(ctx, LinkSpacesInput{
		RequestID: "external", FromSpaceID: team.Profile.SpaceID, ToSpaceID: club.Profile.SpaceID,
		Role: "club", ActorUserID: "owner",
	}); err != nil {
		t.Fatal(err)
	}
	teamView, err = fixture.service.GetTeam(ctx, "owner", team.Profile.SpaceID)
	if err != nil || teamView.Profile.Club == nil || teamView.Profile.Club.SpaceID != club.Profile.SpaceID {
		t.Fatalf("team cache was not rebuilt: view=%#v err=%v", teamView, err)
	}
	clubView, err = fixture.service.GetClub(ctx, "owner", club.Profile.SpaceID)
	if err != nil || len(clubView.Teams) != 1 || clubView.Teams[0].SpaceID != team.Profile.SpaceID {
		t.Fatalf("club cache was not rebuilt: view=%#v err=%v", clubView, err)
	}
}

func TestTeamCannotLinkToSecondClubInMVP(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Independent Team", sportius.JoinPolicyOpen)
	first := createClub(t, fixture, "owner", "club-1", "First Club")
	second := createClub(t, fixture, "owner", "club-2", "Second Club")
	_, err := fixture.service.LinkTeamToClub(ctx, "owner", sportius.LinkTeamToClubRequest{
		RequestID: "link-1", TeamSpaceID: team.Profile.SpaceID, ClubSpaceID: first.Profile.SpaceID,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.service.LinkTeamToClub(ctx, "owner", sportius.LinkTeamToClubRequest{
		RequestID: "link-2", TeamSpaceID: team.Profile.SpaceID, ClubSpaceID: second.Profile.SpaceID,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
}

func TestLinkTeamToClubRequestIDRejectsDifferentCanonicalPair(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	firstTeam := createTeam(t, fixture, "owner", "team-1", "First Team", sportius.JoinPolicyOpen)
	secondTeam := createTeam(t, fixture, "owner", "team-2", "Second Team", sportius.JoinPolicyOpen)
	firstClub := createClub(t, fixture, "owner", "club-1", "First Club")
	secondClub := createClub(t, fixture, "owner", "club-2", "Second Club")
	firstRequest := sportius.LinkTeamToClubRequest{
		RequestID:   "shared-link-request",
		TeamSpaceID: firstTeam.Profile.SpaceID,
		ClubSpaceID: firstClub.Profile.SpaceID,
	}
	if _, err := fixture.service.LinkTeamToClub(ctx, "owner", firstRequest); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.LinkTeamToClub(ctx, "owner", firstRequest); err != nil {
		t.Fatalf("exact link replay failed: %v", err)
	}
	_, err := fixture.service.LinkTeamToClub(ctx, "owner", sportius.LinkTeamToClubRequest{
		RequestID:   firstRequest.RequestID,
		TeamSpaceID: secondTeam.Profile.SpaceID,
		ClubSpaceID: secondClub.Profile.SpaceID,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("different-pair replay error=%v, want ErrConflict", err)
	}
	if len(fixture.core.spaceLinks) != 1 {
		t.Fatalf("conflicting replay mutated generic linkages: %#v", fixture.core.spaceLinks)
	}
	secondView, viewErr := fixture.service.GetTeam(ctx, "owner", secondTeam.Profile.SpaceID)
	if viewErr != nil || secondView.Profile.Club != nil {
		t.Fatalf("conflicting replay linked second team: view=%#v err=%v", secondView, viewErr)
	}
}

func TestInvitationsSupportZeroAndMultipleSuggestedRoles(t *testing.T) {
	fixture := newServiceFixture()
	club := createClub(t, fixture, "owner", "club", "Invite Club")
	ctx := context.Background()
	empty, err := fixture.service.CreateInvitation(ctx, "owner", sportius.CreateInvitationRequest{
		RequestID:          "empty",
		SpaceID:            club.Profile.SpaceID,
		Kind:               sportius.SpaceKindClub,
		InviteeDisplayName: "First Invitee",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.SuggestedRoleIDs) != 0 || empty.DeepLink == "" {
		t.Fatalf("empty-role invite = %#v", empty)
	}
	multiple, err := fixture.service.CreateInvitation(ctx, "owner", sportius.CreateInvitationRequest{
		RequestID:          "multiple",
		SpaceID:            club.Profile.SpaceID,
		Kind:               sportius.SpaceKindClub,
		InviteeDisplayName: "Second Invitee",
		SuggestedRoleIDs: []sportius.RoleID{
			sportius.RoleCoach,
			sportius.RoleVolunteer,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(multiple.SuggestedRoleIDs) != 2 {
		t.Fatalf("multiple-role invite = %#v", multiple)
	}
	repeated, err := fixture.service.CreateInvitation(ctx, "owner", sportius.CreateInvitationRequest{
		RequestID:          "multiple",
		SpaceID:            club.Profile.SpaceID,
		Kind:               sportius.SpaceKindClub,
		InviteeDisplayName: "Second Invitee",
		SuggestedRoleIDs: []sportius.RoleID{
			sportius.RoleCoach,
			sportius.RoleVolunteer,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repeated.InvitationID != multiple.InvitationID || len(fixture.core.invitationOps) != 2 {
		t.Fatalf("idempotent invite = %#v; calls=%d", repeated, len(fixture.core.invitationOps))
	}
}
