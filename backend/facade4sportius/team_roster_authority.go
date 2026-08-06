package facade4sportius

import (
	"context"
	"sort"
	"strings"

	sportius "github.com/sneat-co/ext-sportius/backend"
)

// TeamRosterAuthority is the narrow server-to-server capability a competition
// host needs. It is intentionally separate from the user-facing Facade: no
// caller-supplied actor can use it to discover a private team roster.
//
// A future non-breaking ext-sportius contract can publish this exact interface
// and DTOs for other extensions to consume through a host-wired port.
type TeamRosterAuthority interface {
	ResolveTwoPlayerRoster(ctx context.Context, request TwoPlayerRosterRequest) (TwoPlayerRosterSnapshot, error)
}

const twoPlayerRosterSchemaVersion = "sportius.team-roster.v1"

// TwoPlayerRosterRequest identifies a team and, after registration, carries
// the snapshot version previously accepted by the competition. A non-empty
// ExpectedVersion rejects a changed or stale roster rather than silently
// replacing it.
type TwoPlayerRosterRequest struct {
	TeamSpaceID     string
	ExpectedVersion string
}

// TwoPlayerRosterSnapshot is a deterministic, immutable-at-return-time record
// of the exactly two authenticated players accepted for a competition entry.
// Its Version is a SHA-256 fingerprint of the schema, team and stable player
// identities. Consumers persist this value and pass it back before progressing
// an entry, which makes membership changes fail closed.
type TwoPlayerRosterSnapshot struct {
	SchemaVersion string                  `json:"schemaVersion"`
	TeamSpaceID   string                  `json:"teamSpaceID"`
	Players       []TwoPlayerRosterMember `json:"players"`
	Version       string                  `json:"version"`
}

// TwoPlayerRosterMember is deliberately identity-only. It contains neither
// participant roles nor mutable profile data, so a display-name edit cannot
// invalidate an otherwise unchanged competition acceptance.
type TwoPlayerRosterMember struct {
	UserID    string `json:"userID"`
	ContactID string `json:"contactID"`
}

var _ TeamRosterAuthority = (*Service)(nil)

// ResolveTwoPlayerRoster verifies the Sportius player projection against the
// host's authoritative generic Space membership and returns exactly two active
// authenticated players. It never trusts a stale Sportius projection alone.
func (s *Service) ResolveTwoPlayerRoster(ctx context.Context, request TwoPlayerRosterRequest) (TwoPlayerRosterSnapshot, error) {
	spaceID := strings.TrimSpace(request.TeamSpaceID)
	if spaceID == "" {
		return TwoPlayerRosterSnapshot{}, invalidField("teamSpaceID", "team space ID is required")
	}
	rosterPort, ok := s.core.(TeamRosterPort)
	if !ok {
		return TwoPlayerRosterSnapshot{}, coreFailure("sportius.error.team_roster_unavailable", ErrNotFound)
	}
	team, err := s.getTeamRecord(ctx, spaceID)
	if err != nil {
		return TwoPlayerRosterSnapshot{}, err
	}
	members, err := rosterPort.ListSpaceMembers(ctx, spaceID)
	if err != nil {
		return TwoPlayerRosterSnapshot{}, coreFailure("sportius.error.team_roster_members", err)
	}
	memberUsers := make(map[string]bool, len(members))
	for _, member := range members {
		if userID := strings.TrimSpace(member.UserID); userID != "" {
			memberUsers[userID] = true
		}
	}

	players := make([]TwoPlayerRosterMember, 0, 2)
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
		players = append(players, TwoPlayerRosterMember{
			UserID: userID, ContactID: strings.TrimSpace(participant.ContactID),
		})
	}
	if len(players) != 2 {
		return TwoPlayerRosterSnapshot{}, conflictf("team must have exactly two current authenticated player members")
	}
	sort.Slice(players, func(i, j int) bool {
		if players[i].UserID == players[j].UserID {
			return players[i].ContactID < players[j].ContactID
		}
		return players[i].UserID < players[j].UserID
	})
	snapshot := TwoPlayerRosterSnapshot{
		SchemaVersion: twoPlayerRosterSchemaVersion,
		TeamSpaceID:   spaceID,
		Players:       players,
	}
	snapshot.Version = commandFingerprint(struct {
		SchemaVersion string
		TeamSpaceID   string
		Players       []TwoPlayerRosterMember
	}{snapshot.SchemaVersion, snapshot.TeamSpaceID, snapshot.Players})
	if expected := strings.TrimSpace(request.ExpectedVersion); expected != "" && expected != snapshot.Version {
		return TwoPlayerRosterSnapshot{}, conflictf("team roster changed since acceptance")
	}
	return cloneTwoPlayerRosterSnapshot(snapshot), nil
}

func cloneTwoPlayerRosterSnapshot(snapshot TwoPlayerRosterSnapshot) TwoPlayerRosterSnapshot {
	snapshot.Players = append([]TwoPlayerRosterMember(nil), snapshot.Players...)
	return snapshot
}
