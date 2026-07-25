package facade4sportius

import (
	"context"
	"strings"

	sportius "github.com/sneat-co/ext-sportius/backend"
	"github.com/sneat-co/sportius/backend/models4sportius"
)

func (s *Service) AddTeamPlayer(ctx context.Context, actorUserID, spaceID string, request sportius.AddPlayerRequest) (sportius.PlayerView, error) {
	roles := mergeRoles([]sportius.RoleID{sportius.RolePlayer}, request.RoleIDs)
	participant, err := s.addTeamParticipant(ctx, actorUserID, spaceID, request.RequestID, request.DisplayName, request.UserID, roles, "player")
	if err != nil {
		return sportius.PlayerView{}, err
	}
	return s.GetTeamPlayer(ctx, actorUserID, spaceID, participant.ContactID)
}

func (s *Service) AddTeamStaff(ctx context.Context, actorUserID, spaceID string, request sportius.AddStaffRequest) (sportius.Participant, error) {
	roles, err := validateRoles(request.RoleIDs, sportius.RoleScopeTeam)
	if err != nil {
		return sportius.Participant{}, err
	}
	if !rolesIncludeStaff(roles) {
		return sportius.Participant{}, invalidField("roleIDs", "at least one staff role is required")
	}
	return s.addTeamParticipant(ctx, actorUserID, spaceID, request.RequestID, request.DisplayName, request.UserID, roles, "staff")
}

func (s *Service) addTeamParticipant(
	ctx context.Context,
	actorUserID, spaceID, requestID, displayName, userID string,
	roles []sportius.RoleID,
	operation string,
) (sportius.Participant, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.Participant{}, err
	}
	if err := validateRequestID(requestID); err != nil {
		return sportius.Participant{}, err
	}
	displayName, err := validateName("participant", displayName)
	if err != nil {
		return sportius.Participant{}, err
	}
	roles, err = validateRoles(roles, sportius.RoleScopeTeam)
	if err != nil {
		return sportius.Participant{}, err
	}
	team, err := s.managedTeam(ctx, actorUserID, spaceID)
	if err != nil {
		return sportius.Participant{}, err
	}
	idempotencyKey := commandRequestKey(operation, actorUserID, requestID)
	fingerprint := commandFingerprint(struct {
		Operation   string
		DisplayName string
		UserID      string
		RoleIDs     []sportius.RoleID
	}{
		Operation: operation, DisplayName: displayName, UserID: strings.TrimSpace(userID),
		RoleIDs: fingerprintRoles(roles),
	})
	if contactID := team.ParticipantRequests[idempotencyKey]; contactID != "" {
		if team.ParticipantRequestFingerprints[idempotencyKey] != fingerprint {
			return sportius.Participant{}, conflictf("request ID was already used with a different participant payload")
		}
		return team.Participants[contactID], nil
	}

	contactID, err := s.core.CreateContact(ctx, CreateContactInput{
		RequestID:   idempotencyKey,
		SpaceID:     spaceID,
		DisplayName: displayName,
		UserID:      strings.TrimSpace(userID),
		ActorUserID: actorUserID,
	})
	if err != nil {
		return sportius.Participant{}, coreFailure("sportius.error.contact_create", err)
	}
	if strings.TrimSpace(contactID) == "" {
		return sportius.Participant{}, notFound("contactID", ErrNotFound)
	}
	if err = s.core.EnsureSpaceMember(ctx, EnsureSpaceMemberInput{
		RequestID:   idempotencyKey + ":member",
		SpaceID:     spaceID,
		UserID:      strings.TrimSpace(userID),
		ContactID:   contactID,
		ActorUserID: actorUserID,
	}); err != nil {
		return sportius.Participant{}, coreFailure("sportius.error.member_add", err)
	}
	participant := sportius.Participant{
		ContactID:   contactID,
		UserID:      strings.TrimSpace(userID),
		DisplayName: displayName,
		RoleIDs:     roles,
		SpaceMember: true,
	}
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		stored, ok := writer.GetTeam(spaceID)
		if !ok {
			return ErrNotFound
		}
		if existingID := stored.ParticipantRequests[idempotencyKey]; existingID != "" {
			if stored.ParticipantRequestFingerprints[idempotencyKey] != fingerprint {
				return ErrConflict
			}
			participant = stored.Participants[existingID]
			return nil
		}
		if stored.ParticipantRequestFingerprints == nil {
			stored.ParticipantRequestFingerprints = make(map[string]string)
		}
		stored.Participants[contactID] = participant
		stored.ParticipantRequests[idempotencyKey] = contactID
		stored.ParticipantRequestFingerprints[idempotencyKey] = fingerprint
		if participant.UserID != "" {
			stored.MemberUserRoles[participant.UserID] = mergeRoles(stored.MemberUserRoles[participant.UserID], roles)
		}
		writer.PutTeam(stored)
		return nil
	}); err != nil {
		return sportius.Participant{}, mapRepositoryError(err)
	}
	return participant, nil
}

func (s *Service) AddClubStaff(ctx context.Context, actorUserID, spaceID string, request sportius.AddStaffRequest) (sportius.Participant, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.Participant{}, err
	}
	if err := validateRequestID(request.RequestID); err != nil {
		return sportius.Participant{}, err
	}
	displayName, err := validateName("staff", request.DisplayName)
	if err != nil {
		return sportius.Participant{}, err
	}
	roles, err := validateRoles(request.RoleIDs, sportius.RoleScopeClub)
	if err != nil {
		return sportius.Participant{}, err
	}
	if !rolesIncludeStaff(roles) {
		return sportius.Participant{}, invalidField("roleIDs", "at least one staff role is required")
	}
	club, err := s.managedClub(ctx, actorUserID, spaceID)
	if err != nil {
		return sportius.Participant{}, err
	}
	key := commandRequestKey("club-staff", actorUserID, request.RequestID)
	fingerprint := commandFingerprint(struct {
		DisplayName string
		UserID      string
		RoleIDs     []sportius.RoleID
	}{
		DisplayName: displayName, UserID: strings.TrimSpace(request.UserID),
		RoleIDs: fingerprintRoles(roles),
	})
	if contactID := club.ParticipantRequests[key]; contactID != "" {
		if club.ParticipantRequestFingerprints[key] != fingerprint {
			return sportius.Participant{}, conflictf("request ID was already used with a different staff payload")
		}
		return club.Participants[contactID], nil
	}
	contactID, err := s.core.CreateContact(ctx, CreateContactInput{
		RequestID: key, SpaceID: spaceID, DisplayName: displayName,
		UserID: strings.TrimSpace(request.UserID), ActorUserID: actorUserID,
	})
	if err != nil {
		return sportius.Participant{}, coreFailure("sportius.error.contact_create", err)
	}
	if err = s.core.EnsureSpaceMember(ctx, EnsureSpaceMemberInput{
		RequestID: key + ":member", SpaceID: spaceID, UserID: strings.TrimSpace(request.UserID),
		ContactID: contactID, ActorUserID: actorUserID,
	}); err != nil {
		return sportius.Participant{}, coreFailure("sportius.error.member_add", err)
	}
	participant := sportius.Participant{
		ContactID: contactID, UserID: strings.TrimSpace(request.UserID), DisplayName: displayName,
		RoleIDs: roles, SpaceMember: true,
	}
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		stored, ok := writer.GetClub(spaceID)
		if !ok {
			return ErrNotFound
		}
		if existingID := stored.ParticipantRequests[key]; existingID != "" {
			if stored.ParticipantRequestFingerprints[key] != fingerprint {
				return ErrConflict
			}
			participant = stored.Participants[existingID]
			return nil
		}
		stored.Participants[contactID] = participant
		if participant.UserID != "" {
			stored.MemberRoles[participant.UserID] = mergeRoles(stored.MemberRoles[participant.UserID], roles)
		}
		if stored.ParticipantRequests == nil {
			stored.ParticipantRequests = make(map[string]string)
		}
		if stored.ParticipantRequestFingerprints == nil {
			stored.ParticipantRequestFingerprints = make(map[string]string)
		}
		stored.ParticipantRequests[key] = contactID
		stored.ParticipantRequestFingerprints[key] = fingerprint
		writer.PutClub(stored)
		return nil
	}); err != nil {
		return sportius.Participant{}, mapRepositoryError(err)
	}
	return participant, nil
}

func (s *Service) GetTeamPlayer(ctx context.Context, actorUserID, spaceID, playerContactID string) (sportius.PlayerView, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.PlayerView{}, err
	}
	access, err := s.core.GetSpaceAccess(ctx, actorUserID, spaceID)
	if err != nil {
		return sportius.PlayerView{}, coreFailure("sportius.error.access_check", err)
	}
	if !access.IsMember && !access.CanManage {
		return sportius.PlayerView{}, ErrForbidden
	}
	team, err := s.getTeamRecord(ctx, spaceID)
	if err != nil {
		return sportius.PlayerView{}, err
	}
	return playerView(team, playerContactID)
}

func (s *Service) LinkGuardian(
	ctx context.Context,
	actorUserID, spaceID, playerContactID string,
	request sportius.LinkGuardianRequest,
) (sportius.PlayerView, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.PlayerView{}, err
	}
	if err := validateRequestID(request.RequestID); err != nil {
		return sportius.PlayerView{}, err
	}
	if strings.TrimSpace(playerContactID) == "" {
		return sportius.PlayerView{}, invalidField("playerContactID", "player contact ID is required")
	}
	relationshipRole := strings.TrimSpace(request.RelationshipRoleID)
	if relationshipRole == "" {
		return sportius.PlayerView{}, invalidField("relationshipRoleID", "guardian relationship role is required")
	}
	team, err := s.managedTeam(ctx, actorUserID, spaceID)
	if err != nil {
		return sportius.PlayerView{}, err
	}
	player, ok := team.Participants[playerContactID]
	if !ok || !hasRole(player.RoleIDs, sportius.RolePlayer) {
		return sportius.PlayerView{}, notFound("playerContactID", ErrNotFound)
	}
	idempotencyKey := commandRequestKey("guardian", actorUserID, request.RequestID)
	guardianID := strings.TrimSpace(request.GuardianContactID)
	displayName := strings.Join(strings.Fields(request.GuardianDisplayName), " ")
	fingerprint := commandFingerprint(struct {
		PlayerContactID   string
		GuardianContactID string
		DisplayName       string
		RelationshipRole  string
	}{
		PlayerContactID: playerContactID, GuardianContactID: guardianID,
		DisplayName: displayName, RelationshipRole: relationshipRole,
	})
	if team.GuardianRequests[idempotencyKey] != "" {
		if team.GuardianRequestFingerprints[idempotencyKey] != fingerprint {
			return sportius.PlayerView{}, conflictf("request ID was already used with a different guardian payload")
		}
		return playerView(team, playerContactID)
	}

	if guardianID == "" {
		displayName, err = validateName("guardian", displayName)
		if err != nil {
			return sportius.PlayerView{}, err
		}
		guardianID, err = s.core.CreateContact(ctx, CreateContactInput{
			RequestID: idempotencyKey, SpaceID: spaceID, DisplayName: displayName, ActorUserID: actorUserID,
		})
		if err != nil {
			return sportius.PlayerView{}, coreFailure("sportius.error.contact_create", err)
		}
	} else if existing, found := team.Participants[guardianID]; found && displayName == "" {
		displayName = existing.DisplayName
	}
	if guardianID == "" {
		return sportius.PlayerView{}, notFound("guardianContactID", ErrNotFound)
	}
	if displayName == "" {
		return sportius.PlayerView{}, invalidField("guardianDisplayName", "guardian display name is required for an unindexed contact")
	}
	if err = s.core.LinkContacts(ctx, LinkContactsInput{
		RequestID: idempotencyKey, SpaceID: spaceID, FromContactID: playerContactID,
		ToContactID: guardianID, RelationshipRole: relationshipRole, ActorUserID: actorUserID,
	}); err != nil {
		return sportius.PlayerView{}, coreFailure("sportius.error.guardian_link", err)
	}
	guardian := sportius.Participant{
		ContactID: guardianID, DisplayName: displayName,
		RoleIDs: []sportius.RoleID{sportius.RoleParentGuardian}, SpaceMember: false,
	}
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		stored, ok := writer.GetTeam(spaceID)
		if !ok {
			return ErrNotFound
		}
		if stored.GuardianRequests[idempotencyKey] != "" {
			if stored.GuardianRequestFingerprints[idempotencyKey] != fingerprint {
				return ErrConflict
			}
			team = stored
			return nil
		}
		if existing, found := stored.Participants[guardianID]; found {
			guardian.UserID = existing.UserID
			guardian.RoleIDs = mergeRoles(existing.RoleIDs, guardian.RoleIDs)
			guardian.SpaceMember = existing.SpaceMember
		}
		stored.Participants[guardianID] = guardian
		stored.GuardianRequests[idempotencyKey] = guardianID
		if stored.GuardianRequestFingerprints == nil {
			stored.GuardianRequestFingerprints = make(map[string]string)
		}
		stored.GuardianRequestFingerprints[idempotencyKey] = fingerprint
		stored.GuardianLinks[playerContactID] = append(stored.GuardianLinks[playerContactID], models4sportius.GuardianLink{
			PlayerContactID: playerContactID, GuardianContactID: guardianID, RelationshipRoleID: relationshipRole,
		})
		writer.PutTeam(stored)
		team = stored
		return nil
	}); err != nil {
		return sportius.PlayerView{}, mapRepositoryError(err)
	}
	return playerView(team, playerContactID)
}

func (s *Service) SetParticipantRoles(
	ctx context.Context,
	actorUserID string,
	kind sportius.SpaceKind,
	spaceID, contactID string,
	request sportius.SetParticipantRolesRequest,
) (sportius.Participant, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.Participant{}, err
	}
	if err := validateRequestID(request.RequestID); err != nil {
		return sportius.Participant{}, err
	}
	if strings.TrimSpace(contactID) == "" {
		return sportius.Participant{}, invalidField("contactID", "participant contact ID is required")
	}
	requestKey := commandRequestKey("participant-roles", actorUserID, request.RequestID)
	switch kind {
	case sportius.SpaceKindTeam:
		roles, err := validateRoles(request.RoleIDs, sportius.RoleScopeTeam)
		if err != nil {
			return sportius.Participant{}, err
		}
		if _, err = s.managedTeam(ctx, actorUserID, spaceID); err != nil {
			return sportius.Participant{}, err
		}
		fingerprint := commandFingerprint(struct {
			Kind      sportius.SpaceKind
			SpaceID   string
			ContactID string
			RoleIDs   []sportius.RoleID
		}{
			Kind: kind, SpaceID: spaceID, ContactID: contactID, RoleIDs: fingerprintRoles(roles),
		})
		var participant sportius.Participant
		err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
			team, ok := writer.GetTeam(spaceID)
			if !ok {
				return ErrNotFound
			}
			participant, ok = team.Participants[contactID]
			if !ok {
				return ErrNotFound
			}
			if existingFingerprint := team.RoleRequestFingerprints[requestKey]; existingFingerprint != "" {
				if existingFingerprint != fingerprint {
					return ErrConflict
				}
				return nil
			}
			participant.RoleIDs = roles
			team.Participants[contactID] = participant
			if participant.UserID != "" {
				team.MemberUserRoles[participant.UserID] = roles
			}
			if team.RoleRequestFingerprints == nil {
				team.RoleRequestFingerprints = make(map[string]string)
			}
			team.RoleRequestFingerprints[requestKey] = fingerprint
			writer.PutTeam(team)
			return nil
		})
		if err != nil {
			return sportius.Participant{}, mapRepositoryError(err)
		}
		return participant, nil
	case sportius.SpaceKindClub:
		roles, err := validateRoles(request.RoleIDs, sportius.RoleScopeClub)
		if err != nil {
			return sportius.Participant{}, err
		}
		if _, err = s.managedClub(ctx, actorUserID, spaceID); err != nil {
			return sportius.Participant{}, err
		}
		fingerprint := commandFingerprint(struct {
			Kind      sportius.SpaceKind
			SpaceID   string
			ContactID string
			RoleIDs   []sportius.RoleID
		}{
			Kind: kind, SpaceID: spaceID, ContactID: contactID, RoleIDs: fingerprintRoles(roles),
		})
		var participant sportius.Participant
		err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
			club, ok := writer.GetClub(spaceID)
			if !ok {
				return ErrNotFound
			}
			participant, ok = club.Participants[contactID]
			if !ok {
				return ErrNotFound
			}
			if existingFingerprint := club.RoleRequestFingerprints[requestKey]; existingFingerprint != "" {
				if existingFingerprint != fingerprint {
					return ErrConflict
				}
				return nil
			}
			participant.RoleIDs = roles
			club.Participants[contactID] = participant
			if participant.UserID != "" {
				club.MemberRoles[participant.UserID] = roles
			}
			if club.RoleRequestFingerprints == nil {
				club.RoleRequestFingerprints = make(map[string]string)
			}
			club.RoleRequestFingerprints[requestKey] = fingerprint
			writer.PutClub(club)
			return nil
		})
		if err != nil {
			return sportius.Participant{}, mapRepositoryError(err)
		}
		return participant, nil
	default:
		return sportius.Participant{}, invalidField("kind", "unsupported participant space kind")
	}
}

func playerView(team models4sportius.TeamRecord, playerContactID string) (sportius.PlayerView, error) {
	player, ok := team.Participants[playerContactID]
	if !ok || !hasRole(player.RoleIDs, sportius.RolePlayer) {
		return sportius.PlayerView{}, notFound("playerContactID", ErrNotFound)
	}
	view := sportius.PlayerView{Player: player, Guardians: []sportius.GuardianLink{}}
	for _, link := range team.GuardianLinks[playerContactID] {
		guardian, ok := team.Participants[link.GuardianContactID]
		if !ok {
			continue
		}
		view.Guardians = append(view.Guardians, sportius.GuardianLink{
			Contact: sportius.ContactBrief{
				ContactID: guardian.ContactID, UserID: guardian.UserID, DisplayName: guardian.DisplayName,
			},
			RelationshipRoleID: link.RelationshipRoleID,
		})
	}
	return view, nil
}
