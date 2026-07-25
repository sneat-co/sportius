package facade4sportius

import (
	"context"
	"fmt"
	"sort"
	"strings"

	sportius "github.com/sneat-co/ext-sportius/backend"
	"github.com/sneat-co/sportius/backend/models4sportius"
)

func personalSports(record models4sportius.PersonalProfileRecord) []sportius.PersonalSport {
	result := make([]sportius.PersonalSport, 0, len(record.Sports))
	for _, entry := range record.Sports {
		entry.RoleIDs = append([]sportius.RoleID(nil), entry.RoleIDs...)
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SportID < result[j].SportID })
	return result
}

func teamBrief(profile sportius.TeamProfile) sportius.TeamBrief {
	brief := sportius.TeamBrief{
		SpaceID:    profile.SpaceID,
		Name:       profile.Name,
		SportID:    profile.SportID,
		Gender:     profile.Gender,
		JoinPolicy: profile.JoinPolicy,
	}
	if profile.Age != nil {
		age := *profile.Age
		if profile.Age.MinAge != nil {
			value := *profile.Age.MinAge
			age.MinAge = &value
		}
		if profile.Age.MaxAge != nil {
			value := *profile.Age.MaxAge
			age.MaxAge = &value
		}
		brief.Age = &age
	}
	if profile.Location != nil {
		brief.Locality = profile.Location.Locality
	}
	if profile.Club != nil {
		brief.ClubSpaceID = profile.Club.SpaceID
		brief.ClubName = profile.Club.Name
	}
	return brief
}

func clubBrief(profile sportius.ClubProfile) sportius.ClubBrief {
	brief := sportius.ClubBrief{
		SpaceID:        profile.SpaceID,
		Name:           profile.Name,
		PrimarySportID: profile.PrimarySportID,
	}
	if profile.Location != nil {
		brief.Locality = profile.Location.Locality
	}
	return brief
}

func teamSearchRecord(profile sportius.TeamProfile) models4sportius.TeamSearchRecord {
	localityKey := ""
	if profile.Location != nil {
		localityKey = normaliseName(profile.Location.Locality)
	}
	return models4sportius.TeamSearchRecord{
		Brief:       teamBrief(profile),
		NameKey:     normaliseName(profile.Name),
		SportID:     profile.SportID,
		LocalityKey: localityKey,
	}
}

func clubSearchRecord(profile sportius.ClubProfile) models4sportius.ClubSearchRecord {
	localityKey := ""
	if profile.Location != nil {
		localityKey = normaliseName(profile.Location.Locality)
	}
	return models4sportius.ClubSearchRecord{
		Brief:       clubBrief(profile),
		NameKey:     normaliseName(profile.Name),
		LocalityKey: localityKey,
		SportIDs:    append([]sportius.SportID(nil), profile.SportIDs...),
	}
}

func buildTeamView(
	record models4sportius.TeamRecord,
	includeParticipants, canManage bool,
	viewerRoleIDs []sportius.RoleID,
) sportius.TeamView {
	record = models4sportius.CloneTeamRecord(record)
	view := sportius.TeamView{
		Profile:       record.Profile,
		Players:       []sportius.Participant{},
		Staff:         []sportius.Participant{},
		ViewerRoleIDs: append([]sportius.RoleID(nil), viewerRoleIDs...),
		Capabilities:  capabilities(canManage),
	}
	if !includeParticipants {
		return view
	}
	for _, participant := range record.Participants {
		if hasRole(participant.RoleIDs, sportius.RolePlayer) {
			view.Players = append(view.Players, participant)
		}
		if participantIsStaff(participant) {
			view.Staff = append(view.Staff, participant)
		}
	}
	sortParticipants(view.Players)
	sortParticipants(view.Staff)
	return view
}

func participantIsStaff(participant sportius.Participant) bool {
	for _, role := range participant.RoleIDs {
		definition, ok := roleDefinition(role)
		if ok && definition.ImpliesStaff {
			return true
		}
	}
	return false
}

func sortParticipants(values []sportius.Participant) {
	sort.Slice(values, func(i, j int) bool {
		left, right := normaliseName(values[i].DisplayName), normaliseName(values[j].DisplayName)
		if left == right {
			return values[i].ContactID < values[j].ContactID
		}
		return left < right
	})
}

func buildClubView(
	club models4sportius.ClubRecord,
	teams []models4sportius.TeamRecord,
	rosterTeams []models4sportius.TeamRecord,
	includeClubParticipants, canManage bool,
) sportius.ClubView {
	club = models4sportius.CloneClubRecord(club)
	view := sportius.ClubView{
		Profile:      club.Profile,
		Teams:        []sportius.TeamBrief{},
		Staff:        []sportius.Participant{},
		Members:      []sportius.Participant{},
		Capabilities: capabilities(canManage),
	}
	members := make(map[string]sportius.Participant)
	for _, team := range teams {
		view.Teams = append(view.Teams, teamBrief(team.Profile))
	}
	if !includeClubParticipants {
		sortTeamBriefs(view.Teams)
		return view
	}
	for _, participant := range club.Participants {
		addAggregatedParticipant(members, participant)
		if participantIsStaff(participant) {
			view.Staff = append(view.Staff, participant)
		}
	}
	for _, team := range rosterTeams {
		for _, participant := range team.Participants {
			if participant.SpaceMember {
				addAggregatedParticipant(members, participant)
			}
		}
	}
	for _, participant := range members {
		view.Members = append(view.Members, participant)
	}
	sortTeamBriefs(view.Teams)
	sortParticipants(view.Staff)
	sortParticipants(view.Members)
	return view
}

func addAggregatedParticipant(members map[string]sportius.Participant, participant sportius.Participant) {
	key := "contact:" + participant.ContactID
	if participant.UserID != "" {
		key = "user:" + participant.UserID
	}
	existing, ok := members[key]
	if !ok {
		participant.RoleIDs = append([]sportius.RoleID(nil), participant.RoleIDs...)
		members[key] = participant
		return
	}
	existing.RoleIDs = mergeRoles(existing.RoleIDs, participant.RoleIDs)
	existing.SpaceMember = existing.SpaceMember || participant.SpaceMember
	if existing.DisplayName == "" {
		existing.DisplayName = participant.DisplayName
	}
	members[key] = existing
}

func (s *Service) teamView(ctx context.Context, actorUserID, spaceID string) (sportius.TeamView, error) {
	record, err := s.getTeamRecord(ctx, spaceID)
	if err != nil {
		return sportius.TeamView{}, err
	}
	access, err := s.core.GetSpaceAccess(ctx, actorUserID, spaceID)
	if err != nil {
		return sportius.TeamView{}, coreFailure("sportius.error.access_check", err)
	}
	record, err = s.reconcileTeamClubLink(ctx, actorUserID, record)
	if err != nil {
		return sportius.TeamView{}, err
	}
	return buildTeamView(
		record,
		access.IsMember || access.CanManage,
		access.CanManage,
		record.MemberUserRoles[actorUserID],
	), nil
}

func (s *Service) clubView(ctx context.Context, actorUserID, spaceID string) (sportius.ClubView, error) {
	var club models4sportius.ClubRecord
	err := s.repository.View(ctx, func(reader RepositoryReader) error {
		var ok bool
		club, ok = reader.GetClub(spaceID)
		if !ok {
			return fmt.Errorf("%w: club %q", ErrNotFound, spaceID)
		}
		return nil
	})
	if err != nil {
		return sportius.ClubView{}, err
	}
	access, err := s.core.GetSpaceAccess(ctx, actorUserID, spaceID)
	if err != nil {
		return sportius.ClubView{}, coreFailure("sportius.error.access_check", err)
	}
	links, err := s.core.ResolveTeamClubLinks(ctx, ResolveTeamClubLinksInput{
		ClubSpaceID: spaceID, ActorUserID: actorUserID,
	})
	if err != nil {
		return sportius.ClubView{}, coreFailure("sportius.error.space_link_resolve", err)
	}
	teamByID := make(map[string]models4sportius.TeamRecord, len(links))
	err = s.repository.View(ctx, func(reader RepositoryReader) error {
		for _, link := range links {
			if link.ClubSpaceID != spaceID || strings.TrimSpace(link.TeamSpaceID) == "" {
				return fmt.Errorf("%w: invalid authoritative club linkage", ErrConflict)
			}
			if _, seen := teamByID[link.TeamSpaceID]; seen {
				continue
			}
			team, ok := reader.GetTeam(link.TeamSpaceID)
			if !ok {
				continue
			}
			teamByID[link.TeamSpaceID] = team
		}
		return nil
	})
	if err != nil {
		return sportius.ClubView{}, err
	}
	teams := make([]models4sportius.TeamRecord, 0, len(teamByID))
	rosterTeams := make([]models4sportius.TeamRecord, 0, len(teamByID))
	authoritativeIDs := make(map[string]bool, len(teamByID))
	for teamID, team := range teamByID {
		teams = append(teams, team)
		authoritativeIDs[teamID] = true
		if access.CanManage && club.ClubManagerRosterTeamIDs[teamID] {
			rosterTeams = append(rosterTeams, team)
		}
	}
	if err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		stored, ok := writer.GetClub(spaceID)
		if !ok {
			return ErrNotFound
		}
		stored.TeamSpaceIDs = make(map[string]bool, len(authoritativeIDs))
		nextRosterPolicy := make(map[string]bool, len(authoritativeIDs))
		for teamID := range authoritativeIDs {
			stored.TeamSpaceIDs[teamID] = true
			if stored.ClubManagerRosterTeamIDs[teamID] {
				nextRosterPolicy[teamID] = true
			}
		}
		stored.ClubManagerRosterTeamIDs = nextRosterPolicy
		writer.PutClub(stored)
		club = stored
		return nil
	}); err != nil {
		return sportius.ClubView{}, err
	}
	return buildClubView(club, teams, rosterTeams, access.IsMember || access.CanManage, access.CanManage), nil
}

func (s *Service) reconcileTeamClubLink(
	ctx context.Context,
	actorUserID string,
	team models4sportius.TeamRecord,
) (models4sportius.TeamRecord, error) {
	links, err := s.core.ResolveTeamClubLinks(ctx, ResolveTeamClubLinksInput{
		TeamSpaceID: team.Profile.SpaceID, ActorUserID: actorUserID,
	})
	if err != nil {
		return models4sportius.TeamRecord{}, coreFailure("sportius.error.space_link_resolve", err)
	}
	if len(links) > 1 {
		return models4sportius.TeamRecord{}, conflictf("team has multiple club linkages in the MVP view")
	}
	var authoritative *sportius.ClubBrief
	if len(links) == 1 {
		link := links[0]
		if link.TeamSpaceID != team.Profile.SpaceID || strings.TrimSpace(link.ClubSpaceID) == "" {
			return models4sportius.TeamRecord{}, conflictf("invalid authoritative team linkage")
		}
		club, clubErr := s.getClubRecord(ctx, link.ClubSpaceID)
		if clubErr != nil {
			return models4sportius.TeamRecord{}, clubErr
		}
		value := clubBrief(club.Profile)
		authoritative = &value
	}
	if sameClubBrief(team.Profile.Club, authoritative) {
		return team, nil
	}
	err = s.repository.Update(ctx, func(writer RepositoryWriter) error {
		stored, ok := writer.GetTeam(team.Profile.SpaceID)
		if !ok {
			return ErrNotFound
		}
		stored.Profile.Club = authoritative
		writer.PutTeam(stored)
		team = stored
		return nil
	})
	if err != nil {
		return models4sportius.TeamRecord{}, err
	}
	return team, nil
}

func sameClubBrief(first, second *sportius.ClubBrief) bool {
	if first == nil || second == nil {
		return first == nil && second == nil
	}
	return *first == *second
}

func capabilities(canManage bool) sportius.ViewerCapabilities {
	return sportius.ViewerCapabilities{
		CanEdit:               canManage,
		CanInvite:             canManage,
		CanManageParticipants: canManage,
		CanManageLinkages:     canManage,
	}
}

func (s *Service) getTeamRecord(ctx context.Context, spaceID string) (models4sportius.TeamRecord, error) {
	if strings.TrimSpace(spaceID) == "" {
		return models4sportius.TeamRecord{}, invalidf("team space ID is required")
	}
	var record models4sportius.TeamRecord
	err := s.repository.View(ctx, func(reader RepositoryReader) error {
		var ok bool
		record, ok = reader.GetTeam(spaceID)
		if !ok {
			return fmt.Errorf("%w: team %q", ErrNotFound, spaceID)
		}
		return nil
	})
	return record, err
}

func (s *Service) managedTeam(ctx context.Context, actorUserID, spaceID string) (models4sportius.TeamRecord, error) {
	record, err := s.getTeamRecord(ctx, spaceID)
	if err != nil {
		return models4sportius.TeamRecord{}, err
	}
	access, err := s.core.GetSpaceAccess(ctx, actorUserID, spaceID)
	if err != nil {
		return models4sportius.TeamRecord{}, coreFailure("sportius.error.access_check", err)
	}
	if !access.CanManage {
		return models4sportius.TeamRecord{}, fmt.Errorf("%w: team %q", ErrForbidden, spaceID)
	}
	return record, nil
}

func (s *Service) managedClub(ctx context.Context, actorUserID, spaceID string) (models4sportius.ClubRecord, error) {
	record, err := s.getClubRecord(ctx, spaceID)
	if err != nil {
		return models4sportius.ClubRecord{}, err
	}
	access, err := s.core.GetSpaceAccess(ctx, actorUserID, spaceID)
	if err != nil {
		return models4sportius.ClubRecord{}, coreFailure("sportius.error.access_check", err)
	}
	if !access.CanManage {
		return models4sportius.ClubRecord{}, fmt.Errorf("%w: club %q", ErrForbidden, spaceID)
	}
	return record, nil
}

func (s *Service) getClubRecord(ctx context.Context, spaceID string) (models4sportius.ClubRecord, error) {
	if strings.TrimSpace(spaceID) == "" {
		return models4sportius.ClubRecord{}, invalidf("club space ID is required")
	}
	var record models4sportius.ClubRecord
	err := s.repository.View(ctx, func(reader RepositoryReader) error {
		var ok bool
		record, ok = reader.GetClub(spaceID)
		if !ok {
			return fmt.Errorf("%w: club %q", ErrNotFound, spaceID)
		}
		return nil
	})
	if err != nil {
		return models4sportius.ClubRecord{}, err
	}
	return record, nil
}

func (s *Service) findTeamCreation(ctx context.Context, actorUserID, requestID string) (models4sportius.TeamRecord, bool, error) {
	var result models4sportius.TeamRecord
	found := false
	err := s.repository.View(ctx, func(reader RepositoryReader) error {
		for _, team := range reader.ListTeams() {
			if team.CreatedByUserID == actorUserID && team.CreateRequestID == requestID {
				result, found = team, true
				break
			}
		}
		return nil
	})
	return result, found, err
}

func (s *Service) findClubCreation(ctx context.Context, actorUserID, requestID string) (models4sportius.ClubRecord, bool, error) {
	var result models4sportius.ClubRecord
	found := false
	err := s.repository.View(ctx, func(reader RepositoryReader) error {
		for _, club := range reader.ListClubs() {
			if club.CreatedByUserID == actorUserID && club.CreateRequestID == requestID {
				result, found = club, true
				break
			}
		}
		return nil
	})
	return result, found, err
}

func (s *Service) createUserParticipant(
	ctx context.Context,
	userID string,
	spaceID string,
	requestID string,
	roles []sportius.RoleID,
	owner bool,
) (sportius.Participant, error) {
	displayName, err := s.core.UserDisplayName(ctx, userID)
	if err != nil {
		return sportius.Participant{}, coreFailure("sportius.error.user_brief", err)
	}
	displayName = strings.Join(strings.Fields(displayName), " ")
	if displayName == "" {
		displayName = userID
	}
	contactID, err := s.core.CreateContact(ctx, CreateContactInput{
		RequestID:   requestID + ":contact",
		SpaceID:     spaceID,
		DisplayName: displayName,
		UserID:      userID,
		ActorUserID: userID,
	})
	if err != nil {
		return sportius.Participant{}, coreFailure("sportius.error.contact_create", err)
	}
	if strings.TrimSpace(contactID) == "" {
		return sportius.Participant{}, notFound("contactID", ErrNotFound)
	}
	if err = s.core.EnsureSpaceMember(ctx, EnsureSpaceMemberInput{
		RequestID:   requestID + ":member",
		SpaceID:     spaceID,
		UserID:      userID,
		ContactID:   contactID,
		Owner:       owner,
		ActorUserID: userID,
	}); err != nil {
		return sportius.Participant{}, coreFailure("sportius.error.member_add", err)
	}
	return sportius.Participant{
		ContactID:   contactID,
		UserID:      userID,
		DisplayName: displayName,
		RoleIDs:     append([]sportius.RoleID(nil), roles...),
		SpaceMember: true,
	}, nil
}
