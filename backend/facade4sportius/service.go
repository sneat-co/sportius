// Package facade4sportius implements Sportius application commands without
// depending on Telegram, HTTP, Firestore, or generic Sneat implementation
// packages.
package facade4sportius

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sportius "github.com/sneat-co/ext-sportius/backend"
	"github.com/sneat-co/sportius/backend/models4sportius"
)

// Service implements the stable ext-sportius facade.
type Service struct {
	repository Repository
	core       CorePort
	now        func() time.Time
}

var _ sportius.Facade = (*Service)(nil)

func NewService(repository Repository, core CorePort) *Service {
	if repository == nil {
		panic("facade4sportius.NewService: nil repository")
	}
	if core == nil {
		panic("facade4sportius.NewService: nil core port")
	}
	return &Service{repository: repository, core: core, now: time.Now}
}

// NewServiceWithClock is intended for deterministic expiry tests and hosts
// that already provide a clock. A nil clock is rejected like other ports.
func NewServiceWithClock(repository Repository, core CorePort, now func() time.Time) *Service {
	service := NewService(repository, core)
	if now == nil {
		panic("facade4sportius.NewServiceWithClock: nil clock")
	}
	service.now = now
	return service
}

func (s *Service) GetHome(ctx context.Context, actorUserID string) (sportius.SportsHome, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.SportsHome{}, err
	}
	spaces, err := s.core.ListUserSportSpaces(ctx, actorUserID)
	if err != nil {
		return sportius.SportsHome{}, coreFailure("sportius.error.user_spaces", err)
	}
	var home sportius.SportsHome
	err = s.repository.View(ctx, func(reader RepositoryReader) error {
		profile, ok := reader.GetPersonalProfile(actorUserID)
		if ok {
			home.Sports = personalSports(profile)
		}
		seen := make(map[string]bool, len(spaces))
		for _, space := range spaces {
			if strings.TrimSpace(space.SpaceID) == "" ||
				seen[space.SpaceID] ||
				(!space.IsMember && !space.CanManage) {
				continue
			}
			seen[space.SpaceID] = true
			switch space.Kind {
			case sportius.SpaceKindTeam:
				team, found := reader.GetTeam(space.SpaceID)
				if !found {
					continue
				}
				home.Teams = append(home.Teams, teamBrief(team.Profile))
			case sportius.SpaceKindClub:
				club, found := reader.GetClub(space.SpaceID)
				if !found {
					continue
				}
				home.Clubs = append(home.Clubs, clubBrief(club.Profile))
			}
		}
		sortTeamBriefs(home.Teams)
		sortClubBriefs(home.Clubs)
		return nil
	})
	return home, err
}

func (s *Service) GetPersonalProfile(ctx context.Context, actorUserID string) (sportius.PersonalSportsProfile, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.PersonalSportsProfile{}, err
	}
	profile := sportius.PersonalSportsProfile{UserID: actorUserID, Sports: []sportius.PersonalSport{}}
	err := s.repository.View(ctx, func(reader RepositoryReader) error {
		record, ok := reader.GetPersonalProfile(actorUserID)
		if ok {
			profile.Sports = personalSports(record)
		}
		return nil
	})
	return profile, err
}

func (s *Service) PutPersonalSport(ctx context.Context, actorUserID string, sportID sportius.SportID, request sportius.PutPersonalSportRequest) (sportius.PersonalSportsProfile, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.PersonalSportsProfile{}, err
	}
	if err := validateSport(sportID); err != nil {
		return sportius.PersonalSportsProfile{}, err
	}
	roles, err := validateRoles(request.RoleIDs, sportius.RoleScopePersonal)
	if err != nil {
		return sportius.PersonalSportsProfile{}, err
	}
	visibility := request.Visibility
	if visibility == "" {
		visibility = sportius.VisibilityPrivate
	}
	if !validVisibility(visibility) {
		return sportius.PersonalSportsProfile{}, invalidf("unsupported profile visibility %q", visibility)
	}
	record := models4sportius.PersonalProfileRecord{}
	err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		record, _ = writer.GetPersonalProfile(actorUserID)
		if record.Sports == nil {
			record = models4sportius.PersonalProfileRecord{
				UserID: actorUserID,
				Sports: make(map[sportius.SportID]sportius.PersonalSport),
			}
		}
		record.Sports[sportID] = sportius.PersonalSport{
			SportID:    sportID,
			RoleIDs:    roles,
			Visibility: visibility,
		}
		writer.PutPersonalProfile(record)
		return nil
	})
	if err != nil {
		return sportius.PersonalSportsProfile{}, err
	}
	return sportius.PersonalSportsProfile{UserID: actorUserID, Sports: personalSports(record)}, nil
}

func (s *Service) DeletePersonalSport(ctx context.Context, actorUserID string, sportID sportius.SportID) (sportius.PersonalSportsProfile, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.PersonalSportsProfile{}, err
	}
	if err := validateSport(sportID); err != nil {
		return sportius.PersonalSportsProfile{}, err
	}
	record := models4sportius.PersonalProfileRecord{UserID: actorUserID, Sports: make(map[sportius.SportID]sportius.PersonalSport)}
	err := s.repository.Update(ctx, func(writer RepositoryWriter) error {
		if existing, ok := writer.GetPersonalProfile(actorUserID); ok {
			record = existing
			delete(record.Sports, sportID)
			writer.PutPersonalProfile(record)
		}
		return nil
	})
	if err != nil {
		return sportius.PersonalSportsProfile{}, err
	}
	return sportius.PersonalSportsProfile{UserID: actorUserID, Sports: personalSports(record)}, nil
}

func (s *Service) SearchTeams(ctx context.Context, actorUserID string, request sportius.SearchRequest) ([]sportius.TeamBrief, error) {
	if err := validateActor(actorUserID); err != nil {
		return nil, err
	}
	name := normaliseName(request.Name)
	if name == "" {
		return nil, invalidf("team name is required")
	}
	if request.SportID != "" {
		if err := validateSport(request.SportID); err != nil {
			return nil, err
		}
	}
	locality := normaliseName(request.Locality)
	result := make([]sportius.TeamBrief, 0)
	err := s.repository.View(ctx, func(reader RepositoryReader) error {
		for _, team := range reader.FindTeamSearchRecords(TeamSearchFilter{
			NameKey: name, SportID: request.SportID, LocalityKey: locality,
		}) {
			result = append(result, team.Brief)
		}
		sortTeamBriefs(result)
		return nil
	})
	return result, err
}

func (s *Service) CreateTeam(ctx context.Context, actorUserID string, request sportius.CreateTeamRequest) (sportius.TeamView, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.TeamView{}, err
	}
	if err := validateRequestID(request.RequestID); err != nil {
		return sportius.TeamView{}, err
	}
	name, err := validateName("team", request.Name)
	if err != nil {
		return sportius.TeamView{}, err
	}
	if err = validateSport(request.SportID); err != nil {
		return sportius.TeamView{}, err
	}
	creatorRoles, err := validateRoles(request.CreatorRoleIDs, sportius.RoleScopeTeam)
	if err != nil {
		return sportius.TeamView{}, err
	}
	gender, err := validateGender(request.Gender)
	if err != nil {
		return sportius.TeamView{}, err
	}
	age, err := validateAge(request.Age)
	if err != nil {
		return sportius.TeamView{}, err
	}
	location, err := validateLocation(request.Location)
	if err != nil {
		return sportius.TeamView{}, err
	}
	media, err := validateMedia(request.Media)
	if err != nil {
		return sportius.TeamView{}, err
	}
	joinPolicy, err := validateJoinPolicy(request.JoinPolicy)
	if err != nil {
		return sportius.TeamView{}, err
	}
	createFingerprint := commandFingerprint(struct {
		Name           string
		SportID        sportius.SportID
		CreatorRoleIDs []sportius.RoleID
		Gender         sportius.GenderCategory
		Age            *sportius.AgeRange
		Location       *sportius.LocationHint
		Media          *sportius.MediaRef
		JoinPolicy     sportius.JoinPolicy
	}{
		Name: name, SportID: request.SportID, CreatorRoleIDs: fingerprintRoles(creatorRoles),
		Gender: gender, Age: age, Location: location, Media: media, JoinPolicy: joinPolicy,
	})

	if existing, ok, findErr := s.findTeamCreation(ctx, actorUserID, request.RequestID); findErr != nil {
		return sportius.TeamView{}, findErr
	} else if ok {
		if existing.CreateFingerprint != createFingerprint {
			return sportius.TeamView{}, fmt.Errorf("%w: request ID was already used for another team", ErrConflict)
		}
		return s.teamView(ctx, actorUserID, existing.Profile.SpaceID)
	}
	createKey := commandRequestKey("create-team", actorUserID, request.RequestID)

	spaceID, err := s.core.CreateSpace(ctx, CreateSpaceInput{
		RequestID:   createKey,
		Kind:        sportius.SpaceKindTeam,
		Name:        name,
		OwnerUserID: actorUserID,
	})
	if err != nil {
		return sportius.TeamView{}, coreFailure("sportius.error.space_create", err)
	}
	if strings.TrimSpace(spaceID) == "" {
		return sportius.TeamView{}, notFound("spaceID", ErrNotFound)
	}

	participants := make(map[string]sportius.Participant)
	memberRoles := map[string][]sportius.RoleID{actorUserID: creatorRoles}
	if len(creatorRoles) > 0 {
		participant, createErr := s.createUserParticipant(ctx, actorUserID, spaceID, createKey+":creator", creatorRoles, true)
		if createErr != nil {
			return sportius.TeamView{}, createErr
		}
		participants[participant.ContactID] = participant
	}
	record := models4sportius.TeamRecord{
		Profile: sportius.TeamProfile{
			SpaceID:    spaceID,
			Name:       name,
			SportID:    request.SportID,
			Gender:     gender,
			Age:        age,
			Location:   location,
			Media:      media,
			JoinPolicy: joinPolicy,
		},
		CreatedByUserID:                actorUserID,
		CreateRequestID:                request.RequestID,
		CreateNameKey:                  normaliseName(name),
		CreateSportID:                  request.SportID,
		CreateFingerprint:              createFingerprint,
		ProfileVersion:                 1,
		OwnerUserIDs:                   map[string]bool{actorUserID: true},
		MemberUserRoles:                memberRoles,
		Participants:                   participants,
		ParticipantRequests:            make(map[string]string),
		ParticipantRequestFingerprints: make(map[string]string),
		GuardianRequests:               make(map[string]string),
		GuardianRequestFingerprints:    make(map[string]string),
		RoleRequestFingerprints:        make(map[string]string),
		UpdateRequestFingerprints:      make(map[string]string),
		UpdateRequestVersions:          make(map[string]uint64),
		JoinCommandFingerprints:        make(map[string]string),
		LinkRequestFingerprints:        make(map[string]string),
		GuardianLinks:                  make(map[string][]models4sportius.GuardianLink),
		JoinRequests:                   make(map[string]models4sportius.JoinRequestRecord),
	}
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		for _, team := range writer.ListTeams() {
			if team.CreatedByUserID == actorUserID && team.CreateRequestID == request.RequestID {
				if team.CreateFingerprint != record.CreateFingerprint {
					return fmt.Errorf("%w: request ID was already used for another team", ErrConflict)
				}
				record = team
				return nil
			}
		}
		writer.PutTeam(record)
		return nil
	}); err != nil {
		return sportius.TeamView{}, err
	}
	return buildTeamView(record, true, true, record.MemberUserRoles[actorUserID]), nil
}

func (s *Service) GetTeam(ctx context.Context, actorUserID, spaceID string) (sportius.TeamView, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.TeamView{}, err
	}
	if strings.TrimSpace(spaceID) == "" {
		return sportius.TeamView{}, invalidf("team space ID is required")
	}
	return s.teamView(ctx, actorUserID, spaceID)
}

func (s *Service) UpdateTeam(ctx context.Context, actorUserID, spaceID string, request sportius.UpdateTeamRequest) (sportius.TeamView, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.TeamView{}, err
	}
	if err := validateRequestID(request.RequestID); err != nil {
		return sportius.TeamView{}, err
	}
	current, err := s.managedTeam(ctx, actorUserID, spaceID)
	if err != nil {
		return sportius.TeamView{}, err
	}
	if request.ClearAge && request.Age != nil {
		return sportius.TeamView{}, invalidField("age", "age and clearAge cannot be supplied together")
	}
	if request.ClearLocation && request.Location != nil {
		return sportius.TeamView{}, invalidField("location", "location and clearLocation cannot be supplied together")
	}
	if request.ClearMedia && request.Media != nil {
		return sportius.TeamView{}, invalidField("media", "media and clearMedia cannot be supplied together")
	}

	patch := teamProfilePatch{
		clearAge:      request.ClearAge,
		clearLocation: request.ClearLocation,
		clearMedia:    request.ClearMedia,
	}
	if request.Name != nil {
		value, validationErr := validateName("team", *request.Name)
		err = validationErr
		if err != nil {
			return sportius.TeamView{}, err
		}
		patch.name = &value
	}
	if request.SportID != nil {
		if err = validateSport(*request.SportID); err != nil {
			return sportius.TeamView{}, err
		}
		value := *request.SportID
		patch.sportID = &value
	}
	if request.Gender != nil {
		value, validationErr := validateGender(*request.Gender)
		if err = validationErr; err != nil {
			return sportius.TeamView{}, err
		}
		patch.gender = &value
	}
	if !request.ClearAge && request.Age != nil {
		if patch.age, err = validateAge(request.Age); err != nil {
			return sportius.TeamView{}, err
		}
		patch.setAge = true
	}
	if !request.ClearLocation && request.Location != nil {
		if patch.location, err = validateLocation(request.Location); err != nil {
			return sportius.TeamView{}, err
		}
		patch.setLocation = true
	}
	if !request.ClearMedia && request.Media != nil {
		if patch.media, err = validateMedia(request.Media); err != nil {
			return sportius.TeamView{}, err
		}
		patch.setMedia = true
	}
	if request.JoinPolicy != nil {
		value, validationErr := validateJoinPolicy(*request.JoinPolicy)
		if err = validationErr; err != nil {
			return sportius.TeamView{}, err
		}
		patch.joinPolicy = &value
	}
	updateKey := commandRequestKey("team-update", actorUserID, request.RequestID)
	updateFingerprint := commandFingerprint(request)
	completed := false
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		stored, ok := writer.GetTeam(spaceID)
		if !ok {
			return ErrNotFound
		}
		if existing := stored.UpdateRequestFingerprints[updateKey]; existing != "" {
			if existing != updateFingerprint {
				return ErrConflict
			}
			if stored.UpdateRequestVersions[updateKey] != 0 {
				current = stored
				completed = true
			}
			return nil
		}
		if stored.UpdateRequestFingerprints == nil {
			stored.UpdateRequestFingerprints = make(map[string]string)
		}
		if stored.UpdateRequestVersions == nil {
			stored.UpdateRequestVersions = make(map[string]uint64)
		}
		stored.UpdateRequestFingerprints[updateKey] = updateFingerprint
		writer.PutTeam(stored)
		current = stored
		return nil
	}); err != nil {
		return sportius.TeamView{}, mapRepositoryError(err)
	}
	if completed {
		return buildTeamView(current, true, true, current.MemberUserRoles[actorUserID]), nil
	}
	if patch.name != nil && *patch.name != current.Profile.Name {
		if err = s.core.UpdateSpaceName(ctx, UpdateSpaceNameInput{
			RequestID:   updateKey + ":space-name",
			SpaceID:     spaceID,
			Name:        *patch.name,
			ActorUserID: actorUserID,
		}); err != nil {
			return sportius.TeamView{}, coreFailure("sportius.error.space_update", err)
		}
	}
	var updated models4sportius.TeamRecord
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		stored, ok := writer.GetTeam(spaceID)
		if !ok {
			return ErrNotFound
		}
		if stored.UpdateRequestFingerprints[updateKey] != updateFingerprint {
			return ErrConflict
		}
		if stored.UpdateRequestVersions[updateKey] != 0 {
			updated = stored
			return nil
		}
		applyTeamProfilePatch(&stored.Profile, patch)
		stored.ProfileVersion++
		stored.UpdateRequestVersions[updateKey] = stored.ProfileVersion
		writer.PutTeam(stored)
		updated = stored
		return nil
	}); err != nil {
		return sportius.TeamView{}, err
	}
	return buildTeamView(updated, true, true, updated.MemberUserRoles[actorUserID]), nil
}

type teamProfilePatch struct {
	name          *string
	sportID       *sportius.SportID
	gender        *sportius.GenderCategory
	age           *sportius.AgeRange
	location      *sportius.LocationHint
	media         *sportius.MediaRef
	joinPolicy    *sportius.JoinPolicy
	clearAge      bool
	clearLocation bool
	clearMedia    bool
	setAge        bool
	setLocation   bool
	setMedia      bool
}

func applyTeamProfilePatch(profile *sportius.TeamProfile, patch teamProfilePatch) {
	if patch.name != nil {
		profile.Name = *patch.name
	}
	if patch.sportID != nil {
		profile.SportID = *patch.sportID
	}
	if patch.gender != nil {
		profile.Gender = *patch.gender
	}
	if patch.clearAge {
		profile.Age = nil
	} else if patch.setAge {
		profile.Age = patch.age
	}
	if patch.clearLocation {
		profile.Location = nil
	} else if patch.setLocation {
		profile.Location = patch.location
	}
	if patch.clearMedia {
		profile.Media = nil
	} else if patch.setMedia {
		profile.Media = patch.media
	}
	if patch.joinPolicy != nil {
		profile.JoinPolicy = *patch.joinPolicy
	}
}

func (s *Service) JoinTeam(ctx context.Context, actorUserID, spaceID string, request sportius.JoinTeamRequest) (sportius.JoinTeamResponse, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.JoinTeamResponse{}, err
	}
	if err := validateRequestID(request.RequestID); err != nil {
		return sportius.JoinTeamResponse{}, err
	}
	roles, err := validateRoles(request.RoleIDs, sportius.RoleScopeTeam)
	if err != nil {
		return sportius.JoinTeamResponse{}, err
	}
	team, err := s.getTeamRecord(ctx, spaceID)
	if err != nil {
		return sportius.JoinTeamResponse{}, err
	}
	response := sportius.JoinTeamResponse{Team: teamBrief(team.Profile), RoleIDs: roles}
	if request.InvitationID != "" {
		if err = s.validateInvitationTarget(
			ctx,
			actorUserID,
			request.InvitationID,
			request.ClaimToken,
			sportius.SpaceKindTeam,
			spaceID,
		); err != nil {
			return sportius.JoinTeamResponse{}, err
		}
		acceptance, acceptErr := s.AcceptInvitation(ctx, actorUserID, request.InvitationID, sportius.AcceptInvitationRequest{
			RequestID:  request.RequestID + ":accept",
			ClaimToken: request.ClaimToken,
			RoleIDs:    roles,
		})
		if acceptErr != nil {
			return sportius.JoinTeamResponse{}, acceptErr
		}
		if acceptance.Kind != sportius.SpaceKindTeam || acceptance.SpaceID != spaceID {
			return sportius.JoinTeamResponse{}, invalidField("invitationID", "invitation does not belong to this team")
		}
		response.Status = sportius.JoinStatusJoined
		response.RoleIDs = acceptance.RoleIDs
		return response, nil
	}
	joinKey := commandRequestKey("team-join", actorUserID, request.RequestID)
	joinFingerprint := commandFingerprint(struct {
		SpaceID string
		RoleIDs []sportius.RoleID
	}{
		SpaceID: spaceID,
		RoleIDs: fingerprintRoles(roles),
	})
	if existingFingerprint := team.JoinCommandFingerprints[joinKey]; existingFingerprint != "" &&
		existingFingerprint != joinFingerprint {
		return sportius.JoinTeamResponse{}, conflictf("request ID was already used with different join roles")
	}
	access, err := s.core.GetSpaceAccess(ctx, actorUserID, spaceID)
	if err != nil {
		return sportius.JoinTeamResponse{}, coreFailure("sportius.error.access_check", err)
	}
	if access.IsMember {
		currentRoles := team.MemberUserRoles[actorUserID]
		response.Status = sportius.JoinStatusJoined
		response.RoleIDs = mergeRoles(currentRoles, roles)
		participant, hasParticipant := participantByUserID(team.Participants, actorUserID)
		if !hasParticipant {
			participant, err = s.createUserParticipant(
				ctx, actorUserID, spaceID, joinKey+":rebuild", response.RoleIDs, false,
			)
			if err != nil {
				return sportius.JoinTeamResponse{}, err
			}
		} else {
			participant.RoleIDs = mergeRoles(participant.RoleIDs, response.RoleIDs)
			response.RoleIDs = participant.RoleIDs
		}
		if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
			stored, ok := writer.GetTeam(spaceID)
			if !ok {
				return ErrNotFound
			}
			if existingFingerprint := stored.JoinCommandFingerprints[joinKey]; existingFingerprint != "" &&
				existingFingerprint != joinFingerprint {
				return ErrConflict
			}
			response.RoleIDs = mergeRoles(stored.MemberUserRoles[actorUserID], response.RoleIDs)
			stored.MemberUserRoles[actorUserID] = response.RoleIDs
			participant.RoleIDs = mergeRoles(participant.RoleIDs, response.RoleIDs)
			stored.Participants[participant.ContactID] = participant
			if stored.JoinCommandFingerprints == nil {
				stored.JoinCommandFingerprints = make(map[string]string)
			}
			stored.JoinCommandFingerprints[joinKey] = joinFingerprint
			writer.PutTeam(stored)
			return nil
		}); err != nil {
			return sportius.JoinTeamResponse{}, mapRepositoryError(err)
		}
		return response, nil
	}

	switch {
	case team.Profile.JoinPolicy == sportius.JoinPolicyOpen:
		response.Status = sportius.JoinStatusJoined
	case team.Profile.JoinPolicy == sportius.JoinPolicyApprovalRequired:
		response.Status = sportius.JoinStatusRequested
		response.MembershipRequestID = "sportius:team-join:" + spaceID + ":" + joinKey
		if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
			stored, ok := writer.GetTeam(spaceID)
			if !ok {
				return ErrNotFound
			}
			if existingFingerprint := stored.JoinCommandFingerprints[joinKey]; existingFingerprint != "" {
				if existingFingerprint != joinFingerprint {
					return ErrConflict
				}
				if existing := stored.JoinRequests[actorUserID]; existing.Fingerprint == joinFingerprint {
					response.MembershipRequestID = existing.RequestID
				}
				return nil
			}
			if stored.JoinCommandFingerprints == nil {
				stored.JoinCommandFingerprints = make(map[string]string)
			}
			if stored.JoinRequests == nil {
				stored.JoinRequests = make(map[string]models4sportius.JoinRequestRecord)
			}
			stored.JoinRequests[actorUserID] = models4sportius.JoinRequestRecord{
				RequestID:   response.MembershipRequestID,
				UserID:      actorUserID,
				RoleIDs:     roles,
				Fingerprint: joinFingerprint,
			}
			stored.JoinCommandFingerprints[joinKey] = joinFingerprint
			writer.PutTeam(stored)
			return nil
		}); err != nil {
			return sportius.JoinTeamResponse{}, mapRepositoryError(err)
		}
		return response, nil
	default:
		response.Status = sportius.JoinStatusInviteRequired
		return response, nil
	}

	participant, err := s.createUserParticipant(ctx, actorUserID, spaceID, joinKey+":join", roles, false)
	if err != nil {
		return sportius.JoinTeamResponse{}, err
	}
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		stored, ok := writer.GetTeam(spaceID)
		if !ok {
			return ErrNotFound
		}
		if existingFingerprint := stored.JoinCommandFingerprints[joinKey]; existingFingerprint != "" &&
			existingFingerprint != joinFingerprint {
			return ErrConflict
		}
		if currentRoles, joined := stored.MemberUserRoles[actorUserID]; joined {
			response.RoleIDs = mergeRoles(currentRoles, roles)
		} else {
			response.RoleIDs = roles
		}
		stored.MemberUserRoles[actorUserID] = response.RoleIDs
		participant.RoleIDs = response.RoleIDs
		stored.Participants[participant.ContactID] = participant
		delete(stored.JoinRequests, actorUserID)
		if stored.JoinCommandFingerprints == nil {
			stored.JoinCommandFingerprints = make(map[string]string)
		}
		stored.JoinCommandFingerprints[joinKey] = joinFingerprint
		writer.PutTeam(stored)
		return nil
	}); err != nil {
		return sportius.JoinTeamResponse{}, mapRepositoryError(err)
	}
	return response, nil
}

func (s *Service) SearchClubs(ctx context.Context, actorUserID string, request sportius.SearchRequest) ([]sportius.ClubBrief, error) {
	if err := validateActor(actorUserID); err != nil {
		return nil, err
	}
	name := normaliseName(request.Name)
	if name == "" {
		return nil, invalidf("club name is required")
	}
	if request.SportID != "" {
		if err := validateSport(request.SportID); err != nil {
			return nil, err
		}
	}
	locality := normaliseName(request.Locality)
	result := make([]sportius.ClubBrief, 0)
	err := s.repository.View(ctx, func(reader RepositoryReader) error {
		for _, club := range reader.FindClubSearchRecords(ClubSearchFilter{
			NameKey: name, LocalityKey: locality,
		}) {
			if request.SportID != "" &&
				request.SportID != club.Brief.PrimarySportID &&
				!hasSport(club.SportIDs, request.SportID) {
				continue
			}
			result = append(result, club.Brief)
		}
		sortClubBriefs(result)
		return nil
	})
	return result, err
}

func (s *Service) CreateClub(ctx context.Context, actorUserID string, request sportius.CreateClubRequest) (sportius.ClubView, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.ClubView{}, err
	}
	if err := validateRequestID(request.RequestID); err != nil {
		return sportius.ClubView{}, err
	}
	name, err := validateName("club", request.Name)
	if err != nil {
		return sportius.ClubView{}, err
	}
	if request.PrimarySportID != "" {
		if err = validateSport(request.PrimarySportID); err != nil {
			return sportius.ClubView{}, err
		}
	}
	sports, err := validateSports(request.SportIDs)
	if err != nil {
		return sportius.ClubView{}, err
	}
	if request.PrimarySportID != "" {
		sports = mergeSports([]sportius.SportID{request.PrimarySportID}, sports)
	}
	creatorRoles, err := validateRoles(request.CreatorRoleIDs, sportius.RoleScopeClub)
	if err != nil {
		return sportius.ClubView{}, err
	}
	location, err := validateLocation(request.Location)
	if err != nil {
		return sportius.ClubView{}, err
	}
	media, err := validateMedia(request.Media)
	if err != nil {
		return sportius.ClubView{}, err
	}
	createFingerprint := commandFingerprint(struct {
		Name           string
		PrimarySportID sportius.SportID
		SportIDs       []sportius.SportID
		CreatorRoleIDs []sportius.RoleID
		Location       *sportius.LocationHint
		Media          *sportius.MediaRef
	}{
		Name: name, PrimarySportID: request.PrimarySportID,
		SportIDs: fingerprintSports(sports), CreatorRoleIDs: fingerprintRoles(creatorRoles),
		Location: location, Media: media,
	})
	if existing, ok, findErr := s.findClubCreation(ctx, actorUserID, request.RequestID); findErr != nil {
		return sportius.ClubView{}, findErr
	} else if ok {
		if existing.CreateFingerprint != createFingerprint {
			return sportius.ClubView{}, fmt.Errorf("%w: request ID was already used for another club", ErrConflict)
		}
		return s.clubView(ctx, actorUserID, existing.Profile.SpaceID)
	}
	createKey := commandRequestKey("create-club", actorUserID, request.RequestID)
	spaceID, err := s.core.CreateSpace(ctx, CreateSpaceInput{
		RequestID:   createKey,
		Kind:        sportius.SpaceKindClub,
		Name:        name,
		OwnerUserID: actorUserID,
	})
	if err != nil {
		return sportius.ClubView{}, coreFailure("sportius.error.space_create", err)
	}
	if strings.TrimSpace(spaceID) == "" {
		return sportius.ClubView{}, notFound("spaceID", ErrNotFound)
	}
	participants := make(map[string]sportius.Participant)
	memberRoles := map[string][]sportius.RoleID{actorUserID: creatorRoles}
	if len(creatorRoles) > 0 {
		participant, createErr := s.createUserParticipant(ctx, actorUserID, spaceID, createKey+":creator", creatorRoles, true)
		if createErr != nil {
			return sportius.ClubView{}, createErr
		}
		participants[participant.ContactID] = participant
	}
	record := models4sportius.ClubRecord{
		Profile: sportius.ClubProfile{
			SpaceID:        spaceID,
			Name:           name,
			PrimarySportID: request.PrimarySportID,
			SportIDs:       sports,
			Location:       location,
			Media:          media,
		},
		CreatedByUserID:                actorUserID,
		CreateRequestID:                request.RequestID,
		CreateNameKey:                  normaliseName(name),
		CreateFingerprint:              createFingerprint,
		ProfileVersion:                 1,
		OwnerUserIDs:                   map[string]bool{actorUserID: true},
		MemberRoles:                    memberRoles,
		Participants:                   participants,
		ParticipantRequests:            make(map[string]string),
		ParticipantRequestFingerprints: make(map[string]string),
		RoleRequestFingerprints:        make(map[string]string),
		UpdateRequestFingerprints:      make(map[string]string),
		UpdateRequestVersions:          make(map[string]uint64),
		TeamSpaceIDs:                   make(map[string]bool),
		ClubManagerRosterTeamIDs:       make(map[string]bool),
	}
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		for _, club := range writer.ListClubs() {
			if club.CreatedByUserID == actorUserID && club.CreateRequestID == request.RequestID {
				if club.CreateFingerprint != record.CreateFingerprint {
					return fmt.Errorf("%w: request ID was already used for another club", ErrConflict)
				}
				record = club
				return nil
			}
		}
		writer.PutClub(record)
		return nil
	}); err != nil {
		return sportius.ClubView{}, err
	}
	return buildClubView(record, nil, nil, true, true), nil
}

func (s *Service) GetClub(ctx context.Context, actorUserID, spaceID string) (sportius.ClubView, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.ClubView{}, err
	}
	if strings.TrimSpace(spaceID) == "" {
		return sportius.ClubView{}, invalidf("club space ID is required")
	}
	return s.clubView(ctx, actorUserID, spaceID)
}

func (s *Service) UpdateClub(ctx context.Context, actorUserID, spaceID string, request sportius.UpdateClubRequest) (sportius.ClubView, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.ClubView{}, err
	}
	if err := validateRequestID(request.RequestID); err != nil {
		return sportius.ClubView{}, err
	}
	current, err := s.managedClub(ctx, actorUserID, spaceID)
	if err != nil {
		return sportius.ClubView{}, err
	}
	if request.ClearPrimarySport && request.PrimarySportID != nil {
		return sportius.ClubView{}, invalidField("primarySportID", "primarySportID and clearPrimarySport cannot be supplied together")
	}
	if !request.ReplaceSportIDs && request.SportIDs != nil {
		return sportius.ClubView{}, invalidField("sportIDs", "replaceSportIDs is required when sportIDs is supplied")
	}
	if request.ClearLocation && request.Location != nil {
		return sportius.ClubView{}, invalidField("location", "location and clearLocation cannot be supplied together")
	}
	if request.ClearMedia && request.Media != nil {
		return sportius.ClubView{}, invalidField("media", "media and clearMedia cannot be supplied together")
	}
	patch := clubProfilePatch{
		clearPrimarySport: request.ClearPrimarySport,
		replaceSportIDs:   request.ReplaceSportIDs,
		clearLocation:     request.ClearLocation,
		clearMedia:        request.ClearMedia,
	}
	if request.Name != nil {
		value, validationErr := validateName("club", *request.Name)
		err = validationErr
		if err != nil {
			return sportius.ClubView{}, err
		}
		patch.name = &value
	}
	if !request.ClearPrimarySport && request.PrimarySportID != nil {
		if *request.PrimarySportID != "" {
			if err = validateSport(*request.PrimarySportID); err != nil {
				return sportius.ClubView{}, err
			}
		}
		value := *request.PrimarySportID
		patch.primarySportID = &value
	}
	if request.ReplaceSportIDs {
		if patch.sportIDs, err = validateSports(request.SportIDs); err != nil {
			return sportius.ClubView{}, err
		}
	}
	if !request.ClearLocation && request.Location != nil {
		if patch.location, err = validateLocation(request.Location); err != nil {
			return sportius.ClubView{}, err
		}
		patch.setLocation = true
	}
	if !request.ClearMedia && request.Media != nil {
		if patch.media, err = validateMedia(request.Media); err != nil {
			return sportius.ClubView{}, err
		}
		patch.setMedia = true
	}
	updateKey := commandRequestKey("club-update", actorUserID, request.RequestID)
	updateFingerprint := commandFingerprint(request)
	completed := false
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		stored, ok := writer.GetClub(spaceID)
		if !ok {
			return ErrNotFound
		}
		if existing := stored.UpdateRequestFingerprints[updateKey]; existing != "" {
			if existing != updateFingerprint {
				return ErrConflict
			}
			if stored.UpdateRequestVersions[updateKey] != 0 {
				current = stored
				completed = true
			}
			return nil
		}
		if stored.UpdateRequestFingerprints == nil {
			stored.UpdateRequestFingerprints = make(map[string]string)
		}
		if stored.UpdateRequestVersions == nil {
			stored.UpdateRequestVersions = make(map[string]uint64)
		}
		stored.UpdateRequestFingerprints[updateKey] = updateFingerprint
		writer.PutClub(stored)
		current = stored
		return nil
	}); err != nil {
		return sportius.ClubView{}, mapRepositoryError(err)
	}
	if completed {
		return s.clubView(ctx, actorUserID, spaceID)
	}
	if patch.name != nil && *patch.name != current.Profile.Name {
		if err = s.core.UpdateSpaceName(ctx, UpdateSpaceNameInput{
			RequestID:   updateKey + ":space-name",
			SpaceID:     spaceID,
			Name:        *patch.name,
			ActorUserID: actorUserID,
		}); err != nil {
			return sportius.ClubView{}, coreFailure("sportius.error.space_update", err)
		}
	}
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		stored, ok := writer.GetClub(spaceID)
		if !ok {
			return ErrNotFound
		}
		if stored.UpdateRequestFingerprints[updateKey] != updateFingerprint {
			return ErrConflict
		}
		if stored.UpdateRequestVersions[updateKey] != 0 {
			return nil
		}
		applyClubProfilePatch(&stored.Profile, patch)
		stored.ProfileVersion++
		stored.UpdateRequestVersions[updateKey] = stored.ProfileVersion
		writer.PutClub(stored)
		return nil
	}); err != nil {
		return sportius.ClubView{}, err
	}
	return s.clubView(ctx, actorUserID, spaceID)
}

type clubProfilePatch struct {
	name              *string
	primarySportID    *sportius.SportID
	sportIDs          []sportius.SportID
	location          *sportius.LocationHint
	media             *sportius.MediaRef
	clearPrimarySport bool
	replaceSportIDs   bool
	clearLocation     bool
	clearMedia        bool
	setLocation       bool
	setMedia          bool
}

func applyClubProfilePatch(profile *sportius.ClubProfile, patch clubProfilePatch) {
	if patch.name != nil {
		profile.Name = *patch.name
	}
	if patch.clearPrimarySport {
		profile.PrimarySportID = ""
	} else if patch.primarySportID != nil {
		profile.PrimarySportID = *patch.primarySportID
	}
	if patch.replaceSportIDs {
		profile.SportIDs = append([]sportius.SportID(nil), patch.sportIDs...)
	}
	if profile.PrimarySportID != "" {
		profile.SportIDs = mergeSports([]sportius.SportID{profile.PrimarySportID}, profile.SportIDs)
	}
	if patch.clearLocation {
		profile.Location = nil
	} else if patch.setLocation {
		profile.Location = patch.location
	}
	if patch.clearMedia {
		profile.Media = nil
	} else if patch.setMedia {
		profile.Media = patch.media
	}
}

func (s *Service) LinkTeamToClub(ctx context.Context, actorUserID string, request sportius.LinkTeamToClubRequest) (sportius.ClubView, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.ClubView{}, err
	}
	if err := validateRequestID(request.RequestID); err != nil {
		return sportius.ClubView{}, err
	}
	team, err := s.managedTeam(ctx, actorUserID, request.TeamSpaceID)
	if err != nil {
		return sportius.ClubView{}, err
	}
	club, err := s.managedClub(ctx, actorUserID, request.ClubSpaceID)
	if err != nil {
		return sportius.ClubView{}, err
	}
	linkRequestKey := commandRequestKey("team-club-link", actorUserID, request.RequestID)
	linkFingerprint := commandFingerprint(struct {
		TeamSpaceID string
		ClubSpaceID string
	}{
		TeamSpaceID: team.Profile.SpaceID,
		ClubSpaceID: club.Profile.SpaceID,
	})
	// Reserve the actor/request key before crossing into generic linkage
	// storage. This closes the concurrent-reuse window: a failed core write can
	// be retried with the same pair, but the request can never be repurposed.
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		for _, candidate := range writer.ListTeams() {
			if fingerprint := candidate.LinkRequestFingerprints[linkRequestKey]; fingerprint != "" &&
				fingerprint != linkFingerprint {
				return ErrConflict
			}
		}
		storedTeam, ok := writer.GetTeam(team.Profile.SpaceID)
		if !ok {
			return ErrNotFound
		}
		if storedTeam.LinkRequestFingerprints == nil {
			storedTeam.LinkRequestFingerprints = make(map[string]string)
		}
		storedTeam.LinkRequestFingerprints[linkRequestKey] = linkFingerprint
		writer.PutTeam(storedTeam)
		return nil
	}); err != nil {
		return sportius.ClubView{}, mapRepositoryError(err)
	}
	links, err := s.core.ResolveTeamClubLinks(ctx, ResolveTeamClubLinksInput{
		TeamSpaceID: team.Profile.SpaceID, ActorUserID: actorUserID,
	})
	if err != nil {
		return sportius.ClubView{}, coreFailure("sportius.error.space_link_resolve", err)
	}
	hasAuthoritativeLink := false
	for _, link := range links {
		if link.TeamSpaceID != team.Profile.SpaceID || strings.TrimSpace(link.ClubSpaceID) == "" {
			return sportius.ClubView{}, conflictf("invalid authoritative team linkage")
		}
		if link.ClubSpaceID != club.Profile.SpaceID {
			return sportius.ClubView{}, fmt.Errorf("%w: team is already linked to club %q", ErrConflict, link.ClubSpaceID)
		}
		hasAuthoritativeLink = true
	}
	if !hasAuthoritativeLink {
		if err = s.core.LinkSpaces(ctx, LinkSpacesInput{
			RequestID:   linkRequestKey,
			FromSpaceID: team.Profile.SpaceID,
			ToSpaceID:   club.Profile.SpaceID,
			Role:        "club",
			ActorUserID: actorUserID,
		}); err != nil {
			return sportius.ClubView{}, coreFailure("sportius.error.space_link", err)
		}
	}
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		storedTeam, teamOK := writer.GetTeam(team.Profile.SpaceID)
		storedClub, clubOK := writer.GetClub(club.Profile.SpaceID)
		if !teamOK || !clubOK {
			return ErrNotFound
		}
		for _, candidate := range writer.ListTeams() {
			if fingerprint := candidate.LinkRequestFingerprints[linkRequestKey]; fingerprint != "" &&
				fingerprint != linkFingerprint {
				return ErrConflict
			}
		}
		brief := clubBrief(storedClub.Profile)
		storedTeam.Profile.Club = &brief
		storedClub.TeamSpaceIDs[storedTeam.Profile.SpaceID] = true
		if storedClub.ClubManagerRosterTeamIDs == nil {
			storedClub.ClubManagerRosterTeamIDs = make(map[string]bool)
		}
		storedClub.ClubManagerRosterTeamIDs[storedTeam.Profile.SpaceID] = true
		writer.PutTeam(storedTeam)
		writer.PutClub(storedClub)
		return nil
	}); err != nil {
		return sportius.ClubView{}, err
	}
	return s.clubView(ctx, actorUserID, club.Profile.SpaceID)
}

func (s *Service) CreateInvitation(ctx context.Context, actorUserID string, request sportius.CreateInvitationRequest) (sportius.Invitation, error) {
	if err := validateActor(actorUserID); err != nil {
		return sportius.Invitation{}, err
	}
	if err := validateRequestID(request.RequestID); err != nil {
		return sportius.Invitation{}, err
	}
	request.ContactID = strings.TrimSpace(request.ContactID)
	request.InviteeDisplayName = strings.Join(strings.Fields(request.InviteeDisplayName), " ")
	if (request.ContactID == "") == (request.InviteeDisplayName == "") {
		return sportius.Invitation{}, invalidField("contactID", "provide exactly one of contact ID or invitee display name")
	}
	if request.InviteeDisplayName != "" {
		var err error
		request.InviteeDisplayName, err = validateName("invitee", request.InviteeDisplayName)
		if err != nil {
			return sportius.Invitation{}, err
		}
	}
	var scope sportius.RoleScope
	switch request.Kind {
	case sportius.SpaceKindTeam:
		scope = sportius.RoleScopeTeam
		if _, err := s.managedTeam(ctx, actorUserID, request.SpaceID); err != nil {
			return sportius.Invitation{}, err
		}
	case sportius.SpaceKindClub:
		scope = sportius.RoleScopeClub
		if _, err := s.managedClub(ctx, actorUserID, request.SpaceID); err != nil {
			return sportius.Invitation{}, err
		}
	default:
		return sportius.Invitation{}, invalidf("unsupported invitation space kind %q", request.Kind)
	}
	roles, err := validateRoles(request.SuggestedRoleIDs, scope)
	if err != nil {
		return sportius.Invitation{}, err
	}
	invitationKey := commandRequestKey("invitation", actorUserID, request.RequestID)
	var existing models4sportius.InvitationRecord
	found := false
	if err = s.repository.View(ctx, func(reader RepositoryReader) error {
		existing, found = reader.FindInvitationByRequest(actorUserID, request.RequestID)
		return nil
	}); err != nil {
		return sportius.Invitation{}, err
	}
	if found {
		if existing.Invitation.SpaceID != request.SpaceID || existing.Invitation.Kind != request.Kind {
			return sportius.Invitation{}, fmt.Errorf("%w: request ID was already used for another invitation", ErrConflict)
		}
		if !sameRoles(existing.Invitation.SuggestedRoleIDs, roles) {
			return sportius.Invitation{}, fmt.Errorf("%w: request ID was already used with different invitation roles", ErrConflict)
		}
		if request.ContactID != "" && existing.Invitation.ContactID != request.ContactID {
			return sportius.Invitation{}, fmt.Errorf("%w: request ID was already used for another invitation contact", ErrConflict)
		}
		if request.InviteeDisplayName != "" && existing.Invitation.InviteeDisplayName != request.InviteeDisplayName {
			return sportius.Invitation{}, fmt.Errorf("%w: request ID was already used for another invitee", ErrConflict)
		}
		coreInvite, coreErr := s.core.CreateInvitation(ctx, CoreInvitationInput{
			RequestID:   invitationKey,
			SpaceID:     request.SpaceID,
			ContactID:   existing.Invitation.ContactID,
			ActorUserID: actorUserID,
		})
		if coreErr != nil {
			return sportius.Invitation{}, coreFailure("sportius.error.invitation_create", coreErr)
		}
		if coreInvite.InvitationID != existing.Invitation.InvitationID {
			return sportius.Invitation{}, fmt.Errorf(
				"%w: generic invitation ID %q does not match persisted ID %q",
				ErrConflict, coreInvite.InvitationID, existing.Invitation.InvitationID,
			)
		}
		result := existing.Invitation
		result.DeepLink = coreInvite.DeepLink
		return result, nil
	}

	contactID := request.ContactID
	inviteeDisplayName := request.InviteeDisplayName
	if contactID != "" {
		contact, contactErr := s.core.GetSpaceContact(ctx, GetSpaceContactInput{
			SpaceID: request.SpaceID, ContactID: contactID, ActorUserID: actorUserID,
		})
		if contactErr != nil {
			switch {
			case errors.Is(contactErr, ErrNotFound):
				return sportius.Invitation{}, notFound("contactID", contactErr)
			case errors.Is(contactErr, ErrForbidden):
				return sportius.Invitation{}, ErrForbidden
			default:
				return sportius.Invitation{}, coreFailure("sportius.error.contact_get", contactErr)
			}
		}
		if strings.TrimSpace(contact.ContactID) != contactID {
			return sportius.Invitation{}, coreFailure(
				"sportius.error.contact_get",
				fmt.Errorf("generic contact lookup returned %q for %q", contact.ContactID, contactID),
			)
		}
		inviteeDisplayName, err = validateName("invitee", contact.DisplayName)
		if err != nil {
			return sportius.Invitation{}, coreFailure("sportius.error.contact_get", err)
		}
	} else {
		contactID, err = s.core.CreateContact(ctx, CreateContactInput{
			RequestID:   invitationKey + ":contact",
			SpaceID:     request.SpaceID,
			DisplayName: inviteeDisplayName,
			ActorUserID: actorUserID,
		})
		if err != nil {
			return sportius.Invitation{}, coreFailure("sportius.error.contact_create", err)
		}
		if strings.TrimSpace(contactID) == "" {
			return sportius.Invitation{}, coreFailure("sportius.error.contact_create", ErrNotFound)
		}
	}
	coreInvite, err := s.core.CreateInvitation(ctx, CoreInvitationInput{
		RequestID:   invitationKey,
		SpaceID:     request.SpaceID,
		ContactID:   contactID,
		ActorUserID: actorUserID,
	})
	if err != nil {
		return sportius.Invitation{}, coreFailure("sportius.error.invitation_create", err)
	}
	if strings.TrimSpace(coreInvite.InvitationID) == "" || strings.TrimSpace(coreInvite.DeepLink) == "" {
		return sportius.Invitation{}, notFound("invitationID", ErrNotFound)
	}
	invitation := sportius.Invitation{
		InvitationID:       coreInvite.InvitationID,
		SpaceID:            request.SpaceID,
		Kind:               request.Kind,
		ContactID:          contactID,
		InviteeDisplayName: inviteeDisplayName,
		SuggestedRoleIDs:   roles,
		DeepLink:           coreInvite.DeepLink,
	}
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		if stored, ok := writer.FindInvitationByRequest(actorUserID, request.RequestID); ok {
			if stored.Invitation.InvitationID != coreInvite.InvitationID {
				return fmt.Errorf(
					"%w: generic invitation ID %q does not match concurrently persisted ID %q",
					ErrConflict, coreInvite.InvitationID, stored.Invitation.InvitationID,
				)
			}
			if stored.Invitation.SpaceID != request.SpaceID ||
				stored.Invitation.Kind != request.Kind ||
				stored.Invitation.ContactID != contactID ||
				stored.Invitation.InviteeDisplayName != inviteeDisplayName ||
				!sameRoles(stored.Invitation.SuggestedRoleIDs, roles) {
				return ErrConflict
			}
			invitation = stored.Invitation
			invitation.DeepLink = coreInvite.DeepLink
			return nil
		}
		storedInvitation := invitation
		storedInvitation.DeepLink = ""
		writer.PutInvitation(models4sportius.InvitationRecord{
			Invitation: storedInvitation,
			CreatedBy:  actorUserID,
			RequestID:  request.RequestID,
			Status:     sportius.InvitationStatusPending,
			ExpiresAt:  coreInvite.ExpiresAt,
		})
		return nil
	}); err != nil {
		return sportius.Invitation{}, err
	}
	return invitation, nil
}
