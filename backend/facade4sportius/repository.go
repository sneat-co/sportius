package facade4sportius

import (
	"context"

	sportius "github.com/sneat-co/ext-sportius/backend"
	"github.com/sneat-co/sportius/backend/models4sportius"
)

// Repository provides atomic read and write units over Sportius projections.
// A Firestore adapter can implement this interface without exposing Firestore
// or dalgo types to the application service.
type Repository interface {
	View(ctx context.Context, fn func(RepositoryReader) error) error
	Update(ctx context.Context, fn func(RepositoryWriter) error) error
}

type RepositoryReader interface {
	GetPersonalProfile(userID string) (models4sportius.PersonalProfileRecord, bool)
	GetTeam(spaceID string) (models4sportius.TeamRecord, bool)
	ListTeams() []models4sportius.TeamRecord
	FindTeamSearchRecords(filter TeamSearchFilter) []models4sportius.TeamSearchRecord
	GetClub(spaceID string) (models4sportius.ClubRecord, bool)
	ListClubs() []models4sportius.ClubRecord
	FindClubSearchRecords(filter ClubSearchFilter) []models4sportius.ClubSearchRecord
	GetInvitation(invitationID string) (models4sportius.InvitationRecord, bool)
	FindInvitationByRequest(actorUserID, requestID string) (models4sportius.InvitationRecord, bool)
}

type TeamSearchFilter struct {
	NameKey     string
	SportID     sportius.SportID
	LocalityKey string
}

type ClubSearchFilter struct {
	NameKey     string
	LocalityKey string
}

type RepositoryWriter interface {
	RepositoryReader
	PutPersonalProfile(profile models4sportius.PersonalProfileRecord)
	PutTeam(team models4sportius.TeamRecord)
	PutClub(club models4sportius.ClubRecord)
	PutInvitation(invitation models4sportius.InvitationRecord)
}
