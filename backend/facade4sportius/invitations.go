package facade4sportius

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sportius "github.com/sneat-co/sneat-ext-contracts/sportius"
	"github.com/sneat-co/sportius/backend/models4sportius"
)

func (s *Service) GetInvitation(
	ctx context.Context,
	actorUserID, invitationID, claimToken string,
) (sportius.InvitationView, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.InvitationView{}, err
	}
	if err := validateClaimToken(claimToken); err != nil {
		return sportius.InvitationView{}, err
	}
	resolution, err := s.resolveCoreInvitation(ctx, actorUserID, invitationID, claimToken)
	if err != nil {
		return sportius.InvitationView{}, err
	}
	record, err := s.getInvitationRecord(ctx, invitationID)
	if err != nil {
		return sportius.InvitationView{}, err
	}
	if err = validateCoreInvitationResolution(record, resolution); err != nil {
		return sportius.InvitationView{}, err
	}
	status, err := s.effectiveInvitationStatus(record, resolution)
	if err != nil {
		return sportius.InvitationView{}, err
	}
	if status == sportius.InvitationStatusRevoked && record.Status != sportius.InvitationStatusRevoked {
		record, err = s.markInvitationRevoked(ctx, record)
		if err != nil {
			return sportius.InvitationView{}, err
		}
	}
	spaceName, err := s.invitationSpaceName(ctx, record.Invitation)
	if err != nil {
		return sportius.InvitationView{}, err
	}
	invitation := record.Invitation
	invitation.DeepLink = ""
	return sportius.InvitationView{
		Invitation: invitation,
		SpaceName:  spaceName,
		Status:     status,
		ExpiresAt:  record.ExpiresAt,
	}, nil
}

func (s *Service) AcceptInvitation(
	ctx context.Context,
	actorUserID, invitationID string,
	request sportius.AcceptInvitationRequest,
) (sportius.InvitationAcceptance, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.InvitationAcceptance{}, err
	}
	if err := validateRequestID(request.RequestID); err != nil {
		return sportius.InvitationAcceptance{}, err
	}
	if err := validateClaimToken(request.ClaimToken); err != nil {
		return sportius.InvitationAcceptance{}, err
	}
	resolution, err := s.resolveCoreInvitation(ctx, actorUserID, invitationID, request.ClaimToken)
	if err != nil {
		return sportius.InvitationAcceptance{}, err
	}
	record, err := s.getInvitationRecord(ctx, invitationID)
	if err != nil {
		return sportius.InvitationAcceptance{}, err
	}
	if err = validateCoreInvitationResolution(record, resolution); err != nil {
		return sportius.InvitationAcceptance{}, err
	}
	status, err := s.effectiveInvitationStatus(record, resolution)
	if err != nil {
		return sportius.InvitationAcceptance{}, err
	}
	if status == sportius.InvitationStatusExpired {
		return sportius.InvitationAcceptance{}, invitationExpired()
	}
	if status == sportius.InvitationStatusRevoked {
		if record.Status != sportius.InvitationStatusRevoked {
			if _, err = s.markInvitationRevoked(ctx, record); err != nil {
				return sportius.InvitationAcceptance{}, err
			}
		}
		return sportius.InvitationAcceptance{}, conflictf("invitation was revoked")
	}
	if record.Status == sportius.InvitationStatusAccepted && status == sportius.InvitationStatusAccepted {
		if record.AcceptedByUserID != actorUserID || resolution.Claim.UserID != actorUserID {
			return sportius.InvitationAcceptance{}, conflictf("invitation was already accepted")
		}
		return invitationAcceptance(record), nil
	}
	scope, err := roleScopeForKind(record.Invitation.Kind)
	if err != nil {
		return sportius.InvitationAcceptance{}, err
	}
	roles, err := validateRoles(request.RoleIDs, scope)
	if err != nil {
		return sportius.InvitationAcceptance{}, err
	}
	claim := resolution.Claim
	if status == sportius.InvitationStatusAccepted {
		if claim.UserID != actorUserID {
			return sportius.InvitationAcceptance{}, conflictf("invitation was already accepted")
		}
	} else {
		claim, err = s.core.AcceptInvitation(ctx, CoreAcceptInvitationInput{
			RequestID:    commandRequestKey("invitation-accept:"+invitationID, actorUserID, request.RequestID),
			InvitationID: invitationID,
			ClaimToken:   request.ClaimToken, ActorUserID: actorUserID,
		})
		if err != nil {
			latest, resolveErr := s.resolveCoreInvitation(ctx, actorUserID, invitationID, request.ClaimToken)
			if resolveErr != nil {
				return sportius.InvitationAcceptance{}, coreFailure("sportius.error.invitation_accept", err)
			}
			if validateErr := validateCoreInvitationResolution(record, latest); validateErr != nil {
				return sportius.InvitationAcceptance{}, validateErr
			}
			switch latest.Status {
			case sportius.InvitationStatusAccepted:
				if latest.Claim.UserID != actorUserID {
					return sportius.InvitationAcceptance{}, conflictf("invitation was already accepted")
				}
				claim = latest.Claim
			case sportius.InvitationStatusRevoked:
				_, _ = s.markInvitationRevoked(ctx, record)
				return sportius.InvitationAcceptance{}, conflictf("invitation was revoked")
			case sportius.InvitationStatusExpired:
				return sportius.InvitationAcceptance{}, invitationExpired()
			default:
				if errors.Is(err, ErrForbidden) || errors.Is(err, ErrConflict) {
					return sportius.InvitationAcceptance{}, ErrForbidden
				}
				return sportius.InvitationAcceptance{}, coreFailure("sportius.error.invitation_accept", err)
			}
		}
	}
	participant, err := participantFromInvitationClaim(record.Invitation, actorUserID, claim, roles)
	if err != nil {
		return sportius.InvitationAcceptance{}, err
	}

	switch record.Invitation.Kind {
	case sportius.SpaceKindTeam:
		err = s.acceptTeamInvitation(ctx, actorUserID, record, request.RequestID, participant)
	case sportius.SpaceKindClub:
		err = s.acceptClubInvitation(ctx, actorUserID, record, request.RequestID, participant)
	default:
		err = invalidField("kind", "unsupported invitation space kind")
	}
	if err != nil {
		return sportius.InvitationAcceptance{}, err
	}
	record, err = s.getInvitationRecord(ctx, invitationID)
	if err != nil {
		return sportius.InvitationAcceptance{}, err
	}
	return invitationAcceptance(record), nil
}

func (s *Service) acceptTeamInvitation(
	ctx context.Context,
	actorUserID string,
	invitation models4sportius.InvitationRecord,
	requestID string,
	participant sportius.Participant,
) error {
	team, err := s.getTeamRecord(ctx, invitation.Invitation.SpaceID)
	if err != nil {
		return err
	}
	return mapRepositoryError(s.repository.Update(ctx, func(writer RepositoryWriter) error {
		storedInvitation, ok := writer.GetInvitation(invitation.Invitation.InvitationID)
		if !ok {
			return ErrNotFound
		}
		if storedInvitation.Status == sportius.InvitationStatusAccepted {
			if storedInvitation.AcceptedByUserID != actorUserID {
				return ErrConflict
			}
			return nil
		}
		storedTeam, ok := writer.GetTeam(team.Profile.SpaceID)
		if !ok {
			return ErrNotFound
		}
		participant = reconcileTeamClaimedParticipant(&storedTeam, participant)
		storedTeam.Participants[participant.ContactID] = participant
		storedTeam.MemberUserRoles[actorUserID] = append([]sportius.RoleID(nil), participant.RoleIDs...)
		delete(storedTeam.JoinRequests, actorUserID)
		markInvitationAccepted(&storedInvitation, actorUserID, requestID, participant.RoleIDs)
		writer.PutTeam(storedTeam)
		writer.PutInvitation(storedInvitation)
		return nil
	}))
}

func (s *Service) acceptClubInvitation(
	ctx context.Context,
	actorUserID string,
	invitation models4sportius.InvitationRecord,
	requestID string,
	participant sportius.Participant,
) error {
	club, err := s.getClubRecord(ctx, invitation.Invitation.SpaceID)
	if err != nil {
		return err
	}
	return mapRepositoryError(s.repository.Update(ctx, func(writer RepositoryWriter) error {
		storedInvitation, ok := writer.GetInvitation(invitation.Invitation.InvitationID)
		if !ok {
			return ErrNotFound
		}
		if storedInvitation.Status == sportius.InvitationStatusAccepted {
			if storedInvitation.AcceptedByUserID != actorUserID {
				return ErrConflict
			}
			return nil
		}
		storedClub, ok := writer.GetClub(club.Profile.SpaceID)
		if !ok {
			return ErrNotFound
		}
		participant = reconcileClubClaimedParticipant(&storedClub, participant)
		storedClub.Participants[participant.ContactID] = participant
		storedClub.MemberRoles[actorUserID] = append([]sportius.RoleID(nil), participant.RoleIDs...)
		markInvitationAccepted(&storedInvitation, actorUserID, requestID, participant.RoleIDs)
		writer.PutClub(storedClub)
		writer.PutInvitation(storedInvitation)
		return nil
	}))
}

func (s *Service) getInvitationRecord(ctx context.Context, invitationID string) (models4sportius.InvitationRecord, error) {
	invitationID = strings.TrimSpace(invitationID)
	if invitationID == "" {
		return models4sportius.InvitationRecord{}, invalidField("invitationID", "invitation ID is required")
	}
	var record models4sportius.InvitationRecord
	err := s.repository.View(ctx, func(reader RepositoryReader) error {
		var ok bool
		record, ok = reader.GetInvitation(invitationID)
		if !ok {
			return ErrNotFound
		}
		return nil
	})
	if err != nil {
		return models4sportius.InvitationRecord{}, mapRepositoryError(err)
	}
	return record, nil
}

func (s *Service) invitationSpaceName(ctx context.Context, invitation sportius.Invitation) (string, error) {
	switch invitation.Kind {
	case sportius.SpaceKindTeam:
		team, err := s.getTeamRecord(ctx, invitation.SpaceID)
		if err != nil {
			return "", err
		}
		return team.Profile.Name, nil
	case sportius.SpaceKindClub:
		club, err := s.getClubRecord(ctx, invitation.SpaceID)
		if err != nil {
			return "", err
		}
		return club.Profile.Name, nil
	default:
		return "", invalidField("kind", "unsupported invitation space kind")
	}
}

func (s *Service) validateInvitationTarget(
	ctx context.Context,
	actorUserID string,
	invitationID string,
	claimToken string,
	expectedKind sportius.SpaceKind,
	expectedSpaceID string,
) error {
	if err := validateClaimToken(claimToken); err != nil {
		return err
	}
	resolution, err := s.resolveCoreInvitation(ctx, actorUserID, invitationID, claimToken)
	if err != nil {
		return err
	}
	record, err := s.getInvitationRecord(ctx, invitationID)
	if err != nil {
		return err
	}
	if err = validateCoreInvitationResolution(record, resolution); err != nil {
		return err
	}
	if record.Invitation.Kind != expectedKind || record.Invitation.SpaceID != expectedSpaceID {
		return invalidField("invitationID", "invitation does not belong to the requested space")
	}
	return nil
}

func (s *Service) resolveCoreInvitation(
	ctx context.Context,
	actorUserID, invitationID, claimToken string,
) (CoreInvitationResolution, error) {
	resolution, err := s.core.ResolveInvitation(ctx, CoreResolveInvitationInput{
		InvitationID: invitationID,
		ClaimToken:   claimToken,
		ActorUserID:  actorUserID,
	})
	if err != nil {
		if errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotFound) {
			return CoreInvitationResolution{}, ErrForbidden
		}
		return CoreInvitationResolution{}, coreFailure("sportius.error.invitation_resolve", err)
	}
	switch resolution.Status {
	case sportius.InvitationStatusPending,
		sportius.InvitationStatusAccepted,
		sportius.InvitationStatusRevoked,
		sportius.InvitationStatusExpired:
		return resolution, nil
	default:
		return CoreInvitationResolution{}, coreFailure(
			"sportius.error.invitation_resolve",
			fmt.Errorf("unsupported generic invitation status %q", resolution.Status),
		)
	}
}

func validateCoreInvitationResolution(
	record models4sportius.InvitationRecord,
	resolution CoreInvitationResolution,
) error {
	if resolution.InvitationID != record.Invitation.InvitationID ||
		resolution.SpaceID != record.Invitation.SpaceID ||
		resolution.ContactID != record.Invitation.ContactID {
		return coreFailure(
			"sportius.error.invitation_resolve",
			fmt.Errorf("generic invitation identity does not match Sportius projection"),
		)
	}
	return nil
}

func validateClaimToken(claimToken string) error {
	if strings.TrimSpace(claimToken) == "" {
		return invalidField("claimToken", "invitation claim token is required")
	}
	return nil
}

func (s *Service) effectiveInvitationStatus(
	record models4sportius.InvitationRecord,
	resolution CoreInvitationResolution,
) (sportius.InvitationStatus, error) {
	if resolution.Status != sportius.InvitationStatusPending {
		return resolution.Status, nil
	}
	if record.Status == sportius.InvitationStatusAccepted {
		return "", coreFailure(
			"sportius.error.invitation_resolve",
			fmt.Errorf("generic invitation is pending after Sportius acceptance"),
		)
	}
	if record.ExpiresAt == "" {
		return sportius.InvitationStatusPending, nil
	}
	expiresAt, err := time.Parse(time.RFC3339, record.ExpiresAt)
	if err != nil {
		return "", coreFailure("sportius.error.invitation_expiry", fmt.Errorf("parse expiry: %w", err))
	}
	if !s.now().Before(expiresAt) {
		return sportius.InvitationStatusExpired, nil
	}
	return sportius.InvitationStatusPending, nil
}

func (s *Service) markInvitationRevoked(
	ctx context.Context,
	record models4sportius.InvitationRecord,
) (models4sportius.InvitationRecord, error) {
	err := s.repository.Update(ctx, func(writer RepositoryWriter) error {
		stored, ok := writer.GetInvitation(record.Invitation.InvitationID)
		if !ok {
			return ErrNotFound
		}
		if stored.Status != sportius.InvitationStatusAccepted {
			stored.Status = sportius.InvitationStatusRevoked
			writer.PutInvitation(stored)
		}
		record = stored
		return nil
	})
	return record, err
}

func roleScopeForKind(kind sportius.SpaceKind) (sportius.RoleScope, error) {
	switch kind {
	case sportius.SpaceKindTeam:
		return sportius.RoleScopeTeam, nil
	case sportius.SpaceKindClub:
		return sportius.RoleScopeClub, nil
	default:
		return "", invalidField("kind", "unsupported invitation space kind")
	}
}

func markInvitationAccepted(record *models4sportius.InvitationRecord, actorUserID, requestID string, roles []sportius.RoleID) {
	record.Status = sportius.InvitationStatusAccepted
	record.AcceptedByUserID = actorUserID
	record.AcceptRequestID = requestID
	record.AcceptedRoleIDs = append([]sportius.RoleID(nil), roles...)
}

func invitationAcceptance(record models4sportius.InvitationRecord) sportius.InvitationAcceptance {
	return sportius.InvitationAcceptance{
		InvitationID: record.Invitation.InvitationID,
		SpaceID:      record.Invitation.SpaceID,
		Kind:         record.Invitation.Kind,
		ContactID:    record.Invitation.ContactID,
		RoleIDs:      append([]sportius.RoleID(nil), record.AcceptedRoleIDs...),
	}
}

func participantFromInvitationClaim(
	invitation sportius.Invitation,
	actorUserID string,
	claim CoreInvitationClaim,
	roles []sportius.RoleID,
) (sportius.Participant, error) {
	claim.ContactID = strings.TrimSpace(claim.ContactID)
	claim.UserID = strings.TrimSpace(claim.UserID)
	claim.DisplayName = strings.Join(strings.Fields(claim.DisplayName), " ")
	if claim.ContactID == "" || claim.ContactID != invitation.ContactID {
		return sportius.Participant{}, coreFailure(
			"sportius.error.invitation_accept",
			fmt.Errorf("generic invitation claimed contact %q, expected %q", claim.ContactID, invitation.ContactID),
		)
	}
	if claim.UserID != actorUserID {
		return sportius.Participant{}, coreFailure(
			"sportius.error.invitation_accept",
			fmt.Errorf("generic invitation claimed user %q, expected %q", claim.UserID, actorUserID),
		)
	}
	if claim.DisplayName == "" {
		claim.DisplayName = strings.Join(strings.Fields(invitation.InviteeDisplayName), " ")
	}
	if claim.DisplayName == "" {
		return sportius.Participant{}, coreFailure(
			"sportius.error.invitation_accept",
			fmt.Errorf("generic invitation claim has no display name"),
		)
	}
	return sportius.Participant{
		ContactID:   claim.ContactID,
		UserID:      actorUserID,
		DisplayName: claim.DisplayName,
		RoleIDs:     append([]sportius.RoleID(nil), roles...),
		SpaceMember: true,
	}, nil
}

// reconcileTeamClaimedParticipant keeps the claimed generic contact as the
// Sportius identity and remaps every local contact-ID reference before
// removing a superseded same-user projection.
func reconcileTeamClaimedParticipant(
	team *models4sportius.TeamRecord,
	claimed sportius.Participant,
) sportius.Participant {
	if existing, ok := team.Participants[claimed.ContactID]; ok {
		claimed.RoleIDs = mergeRoles(existing.RoleIDs, claimed.RoleIDs)
		if claimed.DisplayName == "" {
			claimed.DisplayName = existing.DisplayName
		}
	}
	for contactID, existing := range team.Participants {
		if contactID == claimed.ContactID || existing.UserID == "" || existing.UserID != claimed.UserID {
			continue
		}
		claimed.RoleIDs = mergeRoles(existing.RoleIDs, claimed.RoleIDs)
		if claimed.DisplayName == "" {
			claimed.DisplayName = existing.DisplayName
		}
		remapTeamContactReferences(team, contactID, claimed.ContactID)
		delete(team.Participants, contactID)
	}
	return claimed
}

func reconcileClubClaimedParticipant(
	club *models4sportius.ClubRecord,
	claimed sportius.Participant,
) sportius.Participant {
	if existing, ok := club.Participants[claimed.ContactID]; ok {
		claimed.RoleIDs = mergeRoles(existing.RoleIDs, claimed.RoleIDs)
		if claimed.DisplayName == "" {
			claimed.DisplayName = existing.DisplayName
		}
	}
	for contactID, existing := range club.Participants {
		if contactID == claimed.ContactID || existing.UserID == "" || existing.UserID != claimed.UserID {
			continue
		}
		claimed.RoleIDs = mergeRoles(existing.RoleIDs, claimed.RoleIDs)
		if claimed.DisplayName == "" {
			claimed.DisplayName = existing.DisplayName
		}
		remapStringValues(club.ParticipantRequests, contactID, claimed.ContactID)
		delete(club.Participants, contactID)
	}
	return claimed
}

func remapTeamContactReferences(team *models4sportius.TeamRecord, oldContactID, newContactID string) {
	remapStringValues(team.ParticipantRequests, oldContactID, newContactID)
	remapStringValues(team.GuardianRequests, oldContactID, newContactID)
	remapped := make(map[string][]models4sportius.GuardianLink, len(team.GuardianLinks))
	for playerContactID, links := range team.GuardianLinks {
		if playerContactID == oldContactID {
			playerContactID = newContactID
		}
		for _, link := range links {
			if link.PlayerContactID == oldContactID {
				link.PlayerContactID = newContactID
			}
			if link.GuardianContactID == oldContactID {
				link.GuardianContactID = newContactID
			}
			remapped[playerContactID] = append(remapped[playerContactID], link)
		}
	}
	team.GuardianLinks = remapped
}

func remapStringValues(values map[string]string, oldValue, newValue string) {
	for key, value := range values {
		if value == oldValue {
			values[key] = newValue
		}
	}
}

func participantByUserID(participants map[string]sportius.Participant, userID string) (sportius.Participant, bool) {
	for _, participant := range participants {
		if participant.UserID == userID {
			return participant, true
		}
	}
	return sportius.Participant{}, false
}
