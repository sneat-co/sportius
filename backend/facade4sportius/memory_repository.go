package facade4sportius

import (
	"context"
	"sync"

	"github.com/sneat-co/sportius/backend/models4sportius"
)

// MemoryRepository is a concurrency-safe adapter used by facade and bot-flow
// tests. Update callbacks are copy-on-write, so a failed callback never leaves
// partial state behind.
type MemoryRepository struct {
	mu    sync.RWMutex
	state memoryState
}

type memoryState struct {
	profiles              map[string]models4sportius.PersonalProfileRecord
	teams                 map[string]models4sportius.TeamRecord
	teamSearchRecords     map[string]models4sportius.TeamSearchRecord
	clubs                 map[string]models4sportius.ClubRecord
	clubSearchRecords     map[string]models4sportius.ClubSearchRecord
	invitations           map[string]models4sportius.InvitationRecord
	invitationRequestKeys map[string]string
}

type memoryTx struct {
	state *memoryState
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{state: newMemoryState()}
}

func newMemoryState() memoryState {
	return memoryState{
		profiles:              make(map[string]models4sportius.PersonalProfileRecord),
		teams:                 make(map[string]models4sportius.TeamRecord),
		teamSearchRecords:     make(map[string]models4sportius.TeamSearchRecord),
		clubs:                 make(map[string]models4sportius.ClubRecord),
		clubSearchRecords:     make(map[string]models4sportius.ClubSearchRecord),
		invitations:           make(map[string]models4sportius.InvitationRecord),
		invitationRequestKeys: make(map[string]string),
	}
}

func (r *MemoryRepository) View(ctx context.Context, fn func(RepositoryReader) error) error {
	if err := contextError(ctx); err != nil {
		return mapRepositoryError(err)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return mapRepositoryError(fn(memoryTx{state: &r.state}))
}

func (r *MemoryRepository) Update(ctx context.Context, fn func(RepositoryWriter) error) error {
	if err := contextError(ctx); err != nil {
		return mapRepositoryError(err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	next := cloneMemoryState(r.state)
	if err := fn(memoryTx{state: &next}); err != nil {
		return mapRepositoryError(err)
	}
	if err := contextError(ctx); err != nil {
		return mapRepositoryError(err)
	}
	r.state = next
	return nil
}

func (tx memoryTx) GetPersonalProfile(userID string) (models4sportius.PersonalProfileRecord, bool) {
	value, ok := tx.state.profiles[userID]
	if !ok {
		return models4sportius.PersonalProfileRecord{}, false
	}
	return models4sportius.ClonePersonalProfileRecord(value), ok
}

func (tx memoryTx) GetTeam(spaceID string) (models4sportius.TeamRecord, bool) {
	value, ok := tx.state.teams[spaceID]
	if !ok {
		return models4sportius.TeamRecord{}, false
	}
	return models4sportius.CloneTeamRecord(value), ok
}

func (tx memoryTx) ListTeams() []models4sportius.TeamRecord {
	result := make([]models4sportius.TeamRecord, 0, len(tx.state.teams))
	for _, value := range tx.state.teams {
		result = append(result, models4sportius.CloneTeamRecord(value))
	}
	return result
}

func (tx memoryTx) FindTeamSearchRecords(filter TeamSearchFilter) []models4sportius.TeamSearchRecord {
	result := make([]models4sportius.TeamSearchRecord, 0)
	for _, value := range tx.state.teamSearchRecords {
		if value.NameKey != filter.NameKey ||
			(filter.SportID != "" && value.SportID != filter.SportID) ||
			(filter.LocalityKey != "" && value.LocalityKey != filter.LocalityKey) {
			continue
		}
		result = append(result, models4sportius.CloneTeamSearchRecord(value))
	}
	return result
}

func (tx memoryTx) GetClub(spaceID string) (models4sportius.ClubRecord, bool) {
	value, ok := tx.state.clubs[spaceID]
	if !ok {
		return models4sportius.ClubRecord{}, false
	}
	return models4sportius.CloneClubRecord(value), ok
}

func (tx memoryTx) ListClubs() []models4sportius.ClubRecord {
	result := make([]models4sportius.ClubRecord, 0, len(tx.state.clubs))
	for _, value := range tx.state.clubs {
		result = append(result, models4sportius.CloneClubRecord(value))
	}
	return result
}

func (tx memoryTx) FindClubSearchRecords(filter ClubSearchFilter) []models4sportius.ClubSearchRecord {
	result := make([]models4sportius.ClubSearchRecord, 0)
	for _, value := range tx.state.clubSearchRecords {
		if value.NameKey != filter.NameKey ||
			(filter.LocalityKey != "" && value.LocalityKey != filter.LocalityKey) {
			continue
		}
		result = append(result, models4sportius.CloneClubSearchRecord(value))
	}
	return result
}

func (tx memoryTx) GetInvitation(invitationID string) (models4sportius.InvitationRecord, bool) {
	value, ok := tx.state.invitations[invitationID]
	if !ok {
		return models4sportius.InvitationRecord{}, false
	}
	return models4sportius.CloneInvitationRecord(value), ok
}

func (tx memoryTx) FindInvitationByRequest(actorUserID, requestID string) (models4sportius.InvitationRecord, bool) {
	id, ok := tx.state.invitationRequestKeys[requestKey(actorUserID, requestID)]
	if !ok {
		return models4sportius.InvitationRecord{}, false
	}
	return tx.GetInvitation(id)
}

func (tx memoryTx) PutPersonalProfile(profile models4sportius.PersonalProfileRecord) {
	tx.state.profiles[profile.UserID] = models4sportius.ClonePersonalProfileRecord(profile)
}

func (tx memoryTx) PutTeam(team models4sportius.TeamRecord) {
	tx.state.teams[team.Profile.SpaceID] = models4sportius.CloneTeamRecord(team)
	tx.state.teamSearchRecords[team.Profile.SpaceID] = teamSearchRecord(team.Profile)
}

func (tx memoryTx) PutClub(club models4sportius.ClubRecord) {
	tx.state.clubs[club.Profile.SpaceID] = models4sportius.CloneClubRecord(club)
	tx.state.clubSearchRecords[club.Profile.SpaceID] = clubSearchRecord(club.Profile)
}

func (tx memoryTx) PutInvitation(invitation models4sportius.InvitationRecord) {
	value := models4sportius.CloneInvitationRecord(invitation)
	value.Invitation.DeepLink = ""
	tx.state.invitations[value.Invitation.InvitationID] = value
	tx.state.invitationRequestKeys[requestKey(value.CreatedBy, value.RequestID)] = value.Invitation.InvitationID
}

func cloneMemoryState(state memoryState) memoryState {
	next := newMemoryState()
	for id, value := range state.profiles {
		next.profiles[id] = models4sportius.ClonePersonalProfileRecord(value)
	}
	for id, value := range state.teams {
		next.teams[id] = models4sportius.CloneTeamRecord(value)
	}
	for id, value := range state.teamSearchRecords {
		next.teamSearchRecords[id] = models4sportius.CloneTeamSearchRecord(value)
	}
	for id, value := range state.clubs {
		next.clubs[id] = models4sportius.CloneClubRecord(value)
	}
	for id, value := range state.clubSearchRecords {
		next.clubSearchRecords[id] = models4sportius.CloneClubSearchRecord(value)
	}
	for id, value := range state.invitations {
		next.invitations[id] = models4sportius.CloneInvitationRecord(value)
	}
	for key, id := range state.invitationRequestKeys {
		next.invitationRequestKeys[key] = id
	}
	return next
}

func contextError(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func requestKey(actorUserID, requestID string) string {
	return actorUserID + "\x00" + requestID
}
