package facade4sportius

import (
	"context"
	"errors"
	"testing"

	sportius "github.com/sneat-co/ext-sportius/backend"
	"github.com/sneat-co/sportius/backend/models4sportius"
)

type failOnceRepository struct {
	delegate Repository
	failNext bool
}

func (r *failOnceRepository) View(ctx context.Context, fn func(RepositoryReader) error) error {
	return r.delegate.View(ctx, fn)
}

func (r *failOnceRepository) Update(ctx context.Context, fn func(RepositoryWriter) error) error {
	if r.failNext {
		r.failNext = false
		return errors.New("injected projection failure")
	}
	return r.delegate.Update(ctx, fn)
}

func TestInvitationGetAcceptAndEditableRoles(t *testing.T) {
	fixture := newServiceFixture()
	team := createTeam(t, fixture, "owner", "team", "Invite Team", sportius.JoinPolicyInviteOnly)
	contactsBeforeInvite := len(fixture.core.contacts)
	invite, err := fixture.service.CreateInvitation(context.Background(), "owner", sportius.CreateInvitationRequest{
		RequestID:          "invite",
		SpaceID:            team.Profile.SpaceID,
		Kind:               sportius.SpaceKindTeam,
		InviteeDisplayName: "Morgan Member",
		SuggestedRoleIDs:   []sportius.RoleID{sportius.RoleCoach},
	})
	if err != nil {
		t.Fatal(err)
	}
	if invite.ContactID == "" ||
		invite.InviteeDisplayName != "Morgan Member" ||
		len(fixture.core.contacts) != contactsBeforeInvite+1 {
		t.Fatalf("invite target=%#v contacts=%#v", invite, fixture.core.contacts)
	}
	claimToken := fixture.core.claimToken(invite.InvitationID)
	view, err := fixture.service.GetInvitation(context.Background(), "member", invite.InvitationID, claimToken)
	if err != nil || view.Status != sportius.InvitationStatusPending || view.SpaceName != "Invite Team" {
		t.Fatalf("invite view=%#v err=%v", view, err)
	}
	if view.Invitation.DeepLink != "" {
		t.Fatalf("inspection leaked creation-only deep link: %#v", view.Invitation)
	}
	_ = fixture.repository.View(context.Background(), func(reader RepositoryReader) error {
		stored, _ := reader.GetInvitation(invite.InvitationID)
		if stored.Invitation.DeepLink != "" {
			t.Fatalf("deep link persisted in Sportius projection: %#v", stored.Invitation)
		}
		return nil
	})
	accepted, err := fixture.service.AcceptInvitation(context.Background(), "member", invite.InvitationID, sportius.AcceptInvitationRequest{
		RequestID: "accept", ClaimToken: claimToken,
		RoleIDs: []sportius.RoleID{sportius.RolePlayer, sportius.RoleCoach},
	})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.ContactID != invite.ContactID ||
		len(accepted.RoleIDs) != 2 ||
		len(fixture.core.acceptanceOps) != 1 ||
		len(fixture.core.contacts) != contactsBeforeInvite+1 {
		t.Fatalf("acceptance=%#v ops=%#v", accepted, fixture.core.acceptanceOps)
	}
	view, _ = fixture.service.GetInvitation(context.Background(), "member", invite.InvitationID, claimToken)
	if view.Status != sportius.InvitationStatusAccepted {
		t.Fatalf("accepted view = %#v", view)
	}
	teamView, _ := fixture.service.GetTeam(context.Background(), "member", team.Profile.SpaceID)
	if len(teamView.Players) != 1 ||
		teamView.Players[0].ContactID != invite.ContactID ||
		len(teamView.Staff) != 2 ||
		teamView.Capabilities.CanEdit {
		t.Fatalf("member team view = %#v", teamView)
	}
	repeated, err := fixture.service.AcceptInvitation(context.Background(), "member", invite.InvitationID, sportius.AcceptInvitationRequest{
		RequestID: "retry", ClaimToken: claimToken, RoleIDs: nil,
	})
	if err != nil || len(repeated.RoleIDs) != 2 || len(fixture.core.acceptanceOps) != 1 {
		t.Fatalf("idempotent acceptance=%#v err=%v", repeated, err)
	}
}

func TestInvitationClaimTokenIsRequiredAndWrongProofHasNoSideEffects(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Proof Team", sportius.JoinPolicyInviteOnly)
	invite, err := fixture.service.CreateInvitation(ctx, "owner", sportius.CreateInvitationRequest{
		RequestID:          "invite",
		SpaceID:            team.Profile.SpaceID,
		Kind:               sportius.SpaceKindTeam,
		InviteeDisplayName: "Protected Invitee",
	})
	if err != nil {
		t.Fatal(err)
	}
	membersBefore := len(fixture.core.members)
	contactsBefore := len(fixture.core.contacts)

	if _, err = fixture.service.GetInvitation(ctx, "member", invite.InvitationID, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing inspection proof error=%v", err)
	}
	if _, err = fixture.service.GetInvitation(ctx, "member", invite.InvitationID, "wrong-proof"); err == nil {
		t.Fatal("wrong inspection proof unexpectedly succeeded")
	}
	for _, claimToken := range []string{"", "wrong-proof"} {
		_, acceptErr := fixture.service.AcceptInvitation(ctx, "member", invite.InvitationID, sportius.AcceptInvitationRequest{
			RequestID: "accept-" + claimToken, ClaimToken: claimToken,
			RoleIDs: []sportius.RoleID{sportius.RolePlayer},
		})
		if acceptErr == nil {
			t.Fatalf("acceptance succeeded with proof %q", claimToken)
		}
		_, joinErr := fixture.service.JoinTeam(ctx, "member", team.Profile.SpaceID, sportius.JoinTeamRequest{
			RequestID: "join-" + claimToken, InvitationID: invite.InvitationID, ClaimToken: claimToken,
			RoleIDs: []sportius.RoleID{sportius.RolePlayer},
		})
		if joinErr == nil {
			t.Fatalf("join succeeded with proof %q", claimToken)
		}
	}
	if len(fixture.core.acceptanceOps) != 0 ||
		len(fixture.core.members) != membersBefore ||
		len(fixture.core.contacts) != contactsBefore {
		t.Fatalf(
			"invalid proof caused generic side effects: accepts=%#v members=%d contacts=%d",
			fixture.core.acceptanceOps, len(fixture.core.members), len(fixture.core.contacts),
		)
	}
	teamView, err := fixture.service.GetTeam(ctx, "owner", team.Profile.SpaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(teamView.Players) != 0 || len(teamView.Staff) != 1 {
		t.Fatalf("invalid proof changed roster: %#v", teamView)
	}
}

func TestInvitationAcceptanceRejectsClaimWithoutAuthenticatedUserIdentity(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Strict Claim Team", sportius.JoinPolicyInviteOnly)
	invite, err := fixture.service.CreateInvitation(ctx, "owner", sportius.CreateInvitationRequest{
		RequestID:          "invite",
		SpaceID:            team.Profile.SpaceID,
		Kind:               sportius.SpaceKindTeam,
		InviteeDisplayName: "Unbound Claim",
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture.core.mu.Lock()
	fixture.core.acceptClaimOverride = &CoreInvitationClaim{
		ContactID: invite.ContactID, UserID: "", DisplayName: "Unbound Claim",
	}
	fixture.core.mu.Unlock()

	_, err = fixture.service.AcceptInvitation(ctx, "member", invite.InvitationID, sportius.AcceptInvitationRequest{
		RequestID: "accept", ClaimToken: fixture.core.claimToken(invite.InvitationID),
		RoleIDs: []sportius.RoleID{sportius.RolePlayer},
	})
	if err == nil {
		t.Fatal("claim without an authenticated user ID was accepted")
	}
	var stored models4sportius.InvitationRecord
	if readErr := fixture.repository.View(ctx, func(reader RepositoryReader) error {
		stored, _ = reader.GetInvitation(invite.InvitationID)
		return nil
	}); readErr != nil {
		t.Fatal(readErr)
	}
	if stored.Status != sportius.InvitationStatusPending {
		t.Fatalf("invalid claim changed invitation projection: %#v", stored)
	}
	view, viewErr := fixture.service.GetTeam(ctx, "owner", team.Profile.SpaceID)
	if viewErr != nil || len(view.Players) != 0 {
		t.Fatalf("invalid claim changed team roster: view=%#v err=%v", view, viewErr)
	}
}

func TestCreateInvitationRetryRejectsChangedGenericInvitationID(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Stable Invite Team", sportius.JoinPolicyInviteOnly)
	request := sportius.CreateInvitationRequest{
		RequestID:          "stable-invite",
		SpaceID:            team.Profile.SpaceID,
		Kind:               sportius.SpaceKindTeam,
		InviteeDisplayName: "Stable Invitee",
	}
	invite, err := fixture.service.CreateInvitation(ctx, "owner", request)
	if err != nil {
		t.Fatal(err)
	}
	fixture.core.mu.Lock()
	coreRequestID := fixture.core.invitationOps[len(fixture.core.invitationOps)-1].RequestID
	fixture.core.invitations["owner\x00"+coreRequestID] = CoreInvitation{
		InvitationID: "different-generic-invite",
		DeepLink:     "https://t.me/sneat_bot?start=different",
	}
	fixture.core.mu.Unlock()

	if _, err = fixture.service.CreateInvitation(ctx, "owner", request); !errors.Is(err, ErrConflict) {
		t.Fatalf("mismatched generic retry error=%v, want ErrConflict", err)
	}
	var stored models4sportius.InvitationRecord
	if readErr := fixture.repository.View(ctx, func(reader RepositoryReader) error {
		stored, _ = reader.FindInvitationByRequest("owner", request.RequestID)
		return nil
	}); readErr != nil {
		t.Fatal(readErr)
	}
	if stored.Invitation.InvitationID != invite.InvitationID {
		t.Fatalf("persisted invitation identity changed: %#v", stored.Invitation)
	}
}

func TestCreateInvitationConcurrentRecoveryRejectsDifferentPersistedID(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Concurrent Invite Team", sportius.JoinPolicyInviteOnly)
	contactID, err := fixture.core.CreateContact(ctx, CreateContactInput{
		RequestID: "target", SpaceID: team.Profile.SpaceID,
		DisplayName: "Concurrent Invitee", ActorUserID: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := sportius.CreateInvitationRequest{
		RequestID: "concurrent-invite",
		SpaceID:   team.Profile.SpaceID,
		Kind:      sportius.SpaceKindTeam,
		ContactID: contactID,
	}
	wrapper := &mutateBeforeUpdateRepository{delegate: fixture.repository}
	wrapper.nextMutation = func(writer RepositoryWriter) error {
		writer.PutInvitation(models4sportius.InvitationRecord{
			Invitation: sportius.Invitation{
				InvitationID:       "concurrently-persisted-invite",
				SpaceID:            team.Profile.SpaceID,
				Kind:               sportius.SpaceKindTeam,
				ContactID:          contactID,
				InviteeDisplayName: "Concurrent Invitee",
			},
			CreatedBy: "owner",
			RequestID: request.RequestID,
			Status:    sportius.InvitationStatusPending,
		})
		return nil
	}
	service := NewService(wrapper, fixture.core)
	if _, err = service.CreateInvitation(ctx, "owner", request); !errors.Is(err, ErrConflict) {
		t.Fatalf("concurrent identity mismatch error=%v, want ErrConflict", err)
	}
	var stored models4sportius.InvitationRecord
	if readErr := fixture.repository.View(ctx, func(reader RepositoryReader) error {
		stored, _ = reader.FindInvitationByRequest("owner", request.RequestID)
		return nil
	}); readErr != nil {
		t.Fatal(readErr)
	}
	if stored.Invitation.InvitationID != "concurrently-persisted-invite" {
		t.Fatalf("concurrent record was overwritten: %#v", stored.Invitation)
	}
}

func TestInvitationReusesExistingContactAndGenericClaimPreservesIt(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Existing Contact Team", sportius.JoinPolicyInviteOnly)
	contactID, err := fixture.core.CreateContact(ctx, CreateContactInput{
		RequestID:   "existing-contact",
		SpaceID:     team.Profile.SpaceID,
		DisplayName: "Existing Player",
		ActorUserID: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	contactsBeforeInvite := len(fixture.core.contacts)
	membersBeforeInvite := len(fixture.core.members)

	invite, err := fixture.service.CreateInvitation(ctx, "owner", sportius.CreateInvitationRequest{
		RequestID: "invite-existing",
		SpaceID:   team.Profile.SpaceID,
		Kind:      sportius.SpaceKindTeam,
		ContactID: contactID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if invite.ContactID != contactID ||
		invite.InviteeDisplayName != "Existing Player" ||
		len(fixture.core.contacts) != contactsBeforeInvite ||
		len(fixture.core.members) != membersBeforeInvite {
		t.Fatalf("invite=%#v contacts=%d members=%d", invite, len(fixture.core.contacts), len(fixture.core.members))
	}
	if got := fixture.core.invitationOps[len(fixture.core.invitationOps)-1].ContactID; got != contactID {
		t.Fatalf("generic invite target = %q, want %q", got, contactID)
	}

	accepted, err := fixture.service.AcceptInvitation(ctx, "member", invite.InvitationID, sportius.AcceptInvitationRequest{
		RequestID:  "accept-existing",
		ClaimToken: fixture.core.claimToken(invite.InvitationID),
		RoleIDs:    []sportius.RoleID{sportius.RolePlayer},
	})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.ContactID != contactID || len(fixture.core.contacts) != contactsBeforeInvite {
		t.Fatalf("acceptance=%#v contacts=%d", accepted, len(fixture.core.contacts))
	}
	view, err := fixture.service.GetTeam(ctx, "member", team.Profile.SpaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Players) != 1 || view.Players[0].ContactID != contactID || view.Players[0].UserID != "member" {
		t.Fatalf("players = %#v", view.Players)
	}
}

func TestAcceptedInvitationReplayRecoversProjectionWithoutSecondGenericClaim(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Recovery Team", sportius.JoinPolicyInviteOnly)
	invite, err := fixture.service.CreateInvitation(ctx, "owner", sportius.CreateInvitationRequest{
		RequestID:          "invite",
		SpaceID:            team.Profile.SpaceID,
		Kind:               sportius.SpaceKindTeam,
		InviteeDisplayName: "Recovering Member",
	})
	if err != nil {
		t.Fatal(err)
	}
	failingRepository := &failOnceRepository{delegate: fixture.repository, failNext: true}
	service := NewService(failingRepository, fixture.core)
	_, err = service.AcceptInvitation(ctx, "member", invite.InvitationID, sportius.AcceptInvitationRequest{
		RequestID: "accept", ClaimToken: fixture.core.claimToken(invite.InvitationID),
		RoleIDs: []sportius.RoleID{sportius.RolePlayer},
	})
	if err == nil {
		t.Fatal("expected injected projection failure")
	}
	if len(fixture.core.acceptanceOps) != 1 {
		t.Fatalf("generic acceptance calls = %#v", fixture.core.acceptanceOps)
	}
	var storedStatus sportius.InvitationStatus
	if err = fixture.repository.View(ctx, func(reader RepositoryReader) error {
		record, _ := reader.GetInvitation(invite.InvitationID)
		storedStatus = record.Status
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if storedStatus != sportius.InvitationStatusPending {
		t.Fatalf("projection unexpectedly committed: %q", storedStatus)
	}

	recovered, err := service.AcceptInvitation(ctx, "member", invite.InvitationID, sportius.AcceptInvitationRequest{
		RequestID: "accept", ClaimToken: fixture.core.claimToken(invite.InvitationID),
		RoleIDs: []sportius.RoleID{sportius.RolePlayer},
	})
	if err != nil {
		t.Fatal(err)
	}
	if recovered.ContactID != invite.ContactID || len(fixture.core.acceptanceOps) != 1 {
		t.Fatalf("recovered=%#v generic calls=%#v", recovered, fixture.core.acceptanceOps)
	}
	teamView, err := service.GetTeam(ctx, "member", team.Profile.SpaceID)
	if err != nil || len(teamView.Players) != 1 || teamView.Players[0].ContactID != invite.ContactID {
		t.Fatalf("recovered roster=%#v err=%v", teamView.Players, err)
	}
}

func TestGenericInvitationRevocationIsAuthoritative(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Revoked Team", sportius.JoinPolicyInviteOnly)
	invite, err := fixture.service.CreateInvitation(ctx, "owner", sportius.CreateInvitationRequest{
		RequestID:          "invite",
		SpaceID:            team.Profile.SpaceID,
		Kind:               sportius.SpaceKindTeam,
		InviteeDisplayName: "Revoked Member",
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture.core.mu.Lock()
	target := fixture.core.invitationTargets[invite.InvitationID]
	target.status = sportius.InvitationStatusRevoked
	fixture.core.invitationTargets[invite.InvitationID] = target
	fixture.core.mu.Unlock()

	claimToken := fixture.core.claimToken(invite.InvitationID)
	view, err := fixture.service.GetInvitation(ctx, "member", invite.InvitationID, claimToken)
	if err != nil || view.Status != sportius.InvitationStatusRevoked {
		t.Fatalf("view=%#v err=%v", view, err)
	}
	_, err = fixture.service.AcceptInvitation(ctx, "member", invite.InvitationID, sportius.AcceptInvitationRequest{
		RequestID: "accept", ClaimToken: claimToken, RoleIDs: []sportius.RoleID{sportius.RolePlayer},
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error=%v, want conflict", err)
	}
	if len(fixture.core.acceptanceOps) != 0 {
		t.Fatalf("revoked invite reached generic acceptance: %#v", fixture.core.acceptanceOps)
	}
	var storedStatus sportius.InvitationStatus
	_ = fixture.repository.View(ctx, func(reader RepositoryReader) error {
		record, _ := reader.GetInvitation(invite.InvitationID)
		storedStatus = record.Status
		return nil
	})
	if storedStatus != sportius.InvitationStatusRevoked {
		t.Fatalf("revocation was not reconciled: %q", storedStatus)
	}
}

func TestInvitationContactReconciliationRemapsGuardianAndRequestReferences(t *testing.T) {
	fixture := newServiceFixture()
	ctx := context.Background()
	team := createTeam(t, fixture, "owner", "team", "Remap Team", sportius.JoinPolicyInviteOnly)
	oldPlayer, err := fixture.service.AddTeamPlayer(ctx, "owner", team.Profile.SpaceID, sportius.AddPlayerRequest{
		RequestID: "old-player", DisplayName: "Old Player Contact", UserID: "member",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.service.LinkGuardian(ctx, "owner", team.Profile.SpaceID, oldPlayer.Player.ContactID, sportius.LinkGuardianRequest{
		RequestID: "guardian", GuardianDisplayName: "Parent Contact", RelationshipRoleID: "parent",
	}); err != nil {
		t.Fatal(err)
	}
	invite, err := fixture.service.CreateInvitation(ctx, "owner", sportius.CreateInvitationRequest{
		RequestID:          "invite",
		SpaceID:            team.Profile.SpaceID,
		Kind:               sportius.SpaceKindTeam,
		InviteeDisplayName: "Claimed Player Contact",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.service.AcceptInvitation(ctx, "member", invite.InvitationID, sportius.AcceptInvitationRequest{
		RequestID: "accept", ClaimToken: fixture.core.claimToken(invite.InvitationID),
		RoleIDs: []sportius.RoleID{sportius.RolePlayer},
	}); err != nil {
		t.Fatal(err)
	}

	if err = fixture.repository.View(ctx, func(reader RepositoryReader) error {
		record, _ := reader.GetTeam(team.Profile.SpaceID)
		if _, exists := record.Participants[oldPlayer.Player.ContactID]; exists {
			t.Fatalf("old participant projection remains: %#v", record.Participants)
		}
		if _, exists := record.Participants[invite.ContactID]; !exists {
			t.Fatalf("claimed contact missing: %#v", record.Participants)
		}
		for requestID, contactID := range record.ParticipantRequests {
			if contactID == oldPlayer.Player.ContactID {
				t.Fatalf("participant request %q still points to old contact", requestID)
			}
		}
		links := record.GuardianLinks[invite.ContactID]
		if len(links) != 1 ||
			links[0].PlayerContactID != invite.ContactID ||
			links[0].GuardianContactID == "" {
			t.Fatalf("guardian references were not remapped: %#v", record.GuardianLinks)
		}
		if _, exists := record.GuardianLinks[oldPlayer.Player.ContactID]; exists {
			t.Fatalf("old guardian-link key remains: %#v", record.GuardianLinks)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestInvitationRequiresExactlyOneContactTarget(t *testing.T) {
	fixture := newServiceFixture()
	team := createTeam(t, fixture, "owner", "team", "Target Validation Team", sportius.JoinPolicyInviteOnly)
	base := sportius.CreateInvitationRequest{
		RequestID: "invite",
		SpaceID:   team.Profile.SpaceID,
		Kind:      sportius.SpaceKindTeam,
	}
	for _, request := range []sportius.CreateInvitationRequest{
		base,
		{
			RequestID:          base.RequestID,
			SpaceID:            base.SpaceID,
			Kind:               base.Kind,
			ContactID:          "contact-unknown",
			InviteeDisplayName: "Both supplied",
		},
	} {
		_, err := fixture.service.CreateInvitation(context.Background(), "owner", request)
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("request=%#v error=%v, want validation error", request, err)
		}
	}
	if len(fixture.core.invitationOps) != 0 {
		t.Fatalf("generic invitation calls = %#v", fixture.core.invitationOps)
	}
}

func TestClubInvitationCanBeAcceptedWithNoRoles(t *testing.T) {
	fixture := newServiceFixture()
	club := createClub(t, fixture, "owner", "club", "Invite Club")
	invite, err := fixture.service.CreateInvitation(context.Background(), "owner", sportius.CreateInvitationRequest{
		RequestID:          "invite",
		SpaceID:            club.Profile.SpaceID,
		Kind:               sportius.SpaceKindClub,
		InviteeDisplayName: "Morgan Member",
		SuggestedRoleIDs:   []sportius.RoleID{sportius.RoleVolunteer},
	})
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := fixture.service.AcceptInvitation(context.Background(), "member", invite.InvitationID, sportius.AcceptInvitationRequest{
		RequestID: "accept", ClaimToken: fixture.core.claimToken(invite.InvitationID), RoleIDs: nil,
	})
	if err != nil || len(accepted.RoleIDs) != 0 {
		t.Fatalf("acceptance=%#v err=%v", accepted, err)
	}
	view, _ := fixture.service.GetClub(context.Background(), "member", club.Profile.SpaceID)
	if len(view.Members) != 2 {
		t.Fatalf("club member view = %#v", view)
	}
}

func TestExpiredInvitationReturnsStableError(t *testing.T) {
	fixture := newServiceFixture()
	fixture.core.nextExpiry = "2000-01-01T00:00:00Z"
	team := createTeam(t, fixture, "owner", "team", "Expired Invite Team", sportius.JoinPolicyInviteOnly)
	invite, err := fixture.service.CreateInvitation(context.Background(), "owner", sportius.CreateInvitationRequest{
		RequestID:          "invite",
		SpaceID:            team.Profile.SpaceID,
		Kind:               sportius.SpaceKindTeam,
		InviteeDisplayName: "Morgan Member",
	})
	if err != nil {
		t.Fatal(err)
	}
	claimToken := fixture.core.claimToken(invite.InvitationID)
	view, err := fixture.service.GetInvitation(context.Background(), "member", invite.InvitationID, claimToken)
	if err != nil || view.Status != sportius.InvitationStatusExpired {
		t.Fatalf("view=%#v err=%v", view, err)
	}
	_, err = fixture.service.AcceptInvitation(context.Background(), "member", invite.InvitationID, sportius.AcceptInvitationRequest{
		RequestID: "accept", ClaimToken: claimToken,
	})
	var contractError *sportius.Error
	if !errors.As(err, &contractError) || contractError.Code != sportius.ErrorCodeInvitationExpired {
		t.Fatalf("error=%v contract=%#v", err, contractError)
	}
	if len(fixture.core.acceptanceOps) != 0 {
		t.Fatalf("expired invite reached generic acceptance: %#v", fixture.core.acceptanceOps)
	}
	teamView, viewErr := fixture.service.GetTeam(context.Background(), "owner", team.Profile.SpaceID)
	if viewErr != nil {
		t.Fatal(viewErr)
	}
	if len(teamView.Players) != 0 || len(teamView.Staff) != 1 {
		t.Fatalf("expired invite changed roster: %#v", teamView)
	}
}

func TestJoinTeamRejectsMismatchedInvitationBeforeGenericAcceptance(t *testing.T) {
	fixture := newServiceFixture()
	requestedTeam := createTeam(t, fixture, "owner", "team-1", "Requested Team", sportius.JoinPolicyInviteOnly)
	otherTeam := createTeam(t, fixture, "owner", "team-2", "Other Team", sportius.JoinPolicyInviteOnly)
	invite, err := fixture.service.CreateInvitation(context.Background(), "owner", sportius.CreateInvitationRequest{
		RequestID:          "other-invite",
		SpaceID:            otherTeam.Profile.SpaceID,
		Kind:               sportius.SpaceKindTeam,
		InviteeDisplayName: "Morgan Member",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.service.JoinTeam(context.Background(), "member", requestedTeam.Profile.SpaceID, sportius.JoinTeamRequest{
		RequestID: "wrong-target", InvitationID: invite.InvitationID,
		ClaimToken: fixture.core.claimToken(invite.InvitationID),
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want validation error", err)
	}
	if len(fixture.core.acceptanceOps) != 0 {
		t.Fatalf("generic acceptance was called for a mismatched invite: %#v", fixture.core.acceptanceOps)
	}
	view, err := fixture.service.GetInvitation(
		context.Background(), "member", invite.InvitationID, fixture.core.claimToken(invite.InvitationID),
	)
	if err != nil || view.Status != sportius.InvitationStatusPending {
		t.Fatalf("other invite changed: view=%#v err=%v", view, err)
	}
}
