package facade4sportius

import (
	"context"
	"sort"
	"strings"

	sportius "github.com/sneat-co/ext-sportius/backend"
)

var _ sportius.TeamRosterAuthority = (*Service)(nil)

// ResolveTwoPlayerRoster verifies the Sportius player projection against the
// host's authoritative generic Space membership and returns exactly two active
// authenticated players. It never trusts a stale Sportius projection alone.
func (s *Service) ResolveTwoPlayerRoster(ctx context.Context, request sportius.TwoPlayerRosterRequest) (sportius.TwoPlayerRosterSnapshot, error) {
	spaceID := strings.TrimSpace(request.TeamSpaceID)
	if spaceID == "" {
		return sportius.TwoPlayerRosterSnapshot{}, invalidField("teamSpaceID", "team space ID is required")
	}
	rosterPort, ok := s.core.(TeamRosterPort)
	if !ok {
		return sportius.TwoPlayerRosterSnapshot{}, coreFailure("sportius.error.team_roster_unavailable", ErrNotFound)
	}
	team, err := s.getTeamRecord(ctx, spaceID)
	if err != nil {
		return sportius.TwoPlayerRosterSnapshot{}, err
	}
	members, err := rosterPort.ListSpaceMembers(ctx, spaceID)
	if err != nil {
		return sportius.TwoPlayerRosterSnapshot{}, coreFailure("sportius.error.team_roster_members", err)
	}
	memberUsers := make(map[string]bool, len(members))
	for _, member := range members {
		if userID := strings.TrimSpace(member.UserID); userID != "" {
			memberUsers[userID] = true
		}
	}

	players := make([]sportius.TwoPlayerRosterMember, 0, 2)
	seenUsers := make(map[string]bool, 2)
	for _, participant := range team.Participants {
		userID := strings.TrimSpace(participant.UserID)
		if !participant.SpaceMember || !hasRole(participant.RoleIDs, sportius.RolePlayer) || userID == "" {
			continue
		}
		if !memberUsers[userID] || seenUsers[userID] {
			continue
		}
		seenUsers[userID] = true
		players = append(players, sportius.TwoPlayerRosterMember{
			UserID: userID, ContactID: strings.TrimSpace(participant.ContactID),
		})
	}
	if len(players) != 2 {
		return sportius.TwoPlayerRosterSnapshot{}, conflictf("team must have exactly two current authenticated player members")
	}
	sort.Slice(players, func(i, j int) bool {
		if players[i].UserID == players[j].UserID {
			return players[i].ContactID < players[j].ContactID
		}
		return players[i].UserID < players[j].UserID
	})
	snapshot := sportius.TwoPlayerRosterSnapshot{
		SchemaVersion: sportius.TwoPlayerRosterSchemaVersion,
		TeamSpaceID:   spaceID,
		Players:       players,
	}
	snapshot.Version = commandFingerprint(struct {
		SchemaVersion string
		TeamSpaceID   string
		Players       []sportius.TwoPlayerRosterMember
	}{snapshot.SchemaVersion, snapshot.TeamSpaceID, snapshot.Players})
	if expected := strings.TrimSpace(request.ExpectedVersion); expected != "" && expected != snapshot.Version {
		return sportius.TwoPlayerRosterSnapshot{}, conflictf("team roster changed since acceptance")
	}
	return cloneTwoPlayerRosterSnapshot(snapshot), nil
}

func cloneTwoPlayerRosterSnapshot(snapshot sportius.TwoPlayerRosterSnapshot) sportius.TwoPlayerRosterSnapshot {
	snapshot.Players = append([]sportius.TwoPlayerRosterMember(nil), snapshot.Players...)
	return snapshot
}
