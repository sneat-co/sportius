package facade4sportius

import (
	"context"
	"reflect"

	"github.com/dal-go/dalgo/dal"
	dalrecord "github.com/dal-go/record"
	sportius "github.com/sneat-co/sneat-ext-contracts/sportius"
	"github.com/sneat-co/sportius/backend/models4sportius"
)

const (
	usersCollection       = "users"
	spacesCollection      = "spaces"
	extCollection         = "ext"
	teamsIndexCollection  = "teams"
	clubsIndexCollection  = "clubs"
	invitationsCollection = "invitations"
)

// DalgoRepository persists canonical extension documents at:
//
//	/spaces/{spaceID}/ext/sportius
//
// Search projections and invitation metadata live under the extension-owned
// subtree /ext/sportius/{teams|clubs|invitations}/{id}. Generic Sneat spaces,
// contacts, memberships, linkages and invitation tokens are never stored here.
// The former /users/{uid}/ext/sportius profile is read only as a lazy-migration
// fallback; every subsequent profile write targets the personal Space.
type DalgoRepository struct {
	db dal.DB
}

var _ Repository = (*DalgoRepository)(nil)

func NewDalgoRepository(db dal.DB) *DalgoRepository {
	if db == nil {
		panic("facade4sportius.NewDalgoRepository: nil DB")
	}
	return &DalgoRepository{db: db}
}

func (r *DalgoRepository) View(ctx context.Context, fn func(RepositoryReader) error) error {
	return mapRepositoryError(r.db.RunReadonlyTransaction(ctx, func(ctx context.Context, tx dal.ReadTransaction) error {
		reader := &dalgoRepositoryTx{ctx: ctx, reader: tx}
		if err := fn(reader); err != nil {
			return err
		}
		return reader.err
	}))
}

func (r *DalgoRepository) Update(ctx context.Context, fn func(RepositoryWriter) error) error {
	return mapRepositoryError(r.db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		writer := &dalgoRepositoryTx{ctx: ctx, reader: tx, writer: tx}
		if err := fn(writer); err != nil {
			return err
		}
		return writer.err
	}))
}

type spaceExtensionDBO struct {
	Kind     sportius.SpaceKind                     `firestore:"kind" json:"kind"`
	Personal *models4sportius.PersonalProfileRecord `firestore:"personal,omitempty" json:"personal,omitempty"`
	Team     *models4sportius.TeamRecord            `firestore:"team,omitempty" json:"team,omitempty"`
	Club     *models4sportius.ClubRecord            `firestore:"club,omitempty" json:"club,omitempty"`
}

type dalgoRepositoryTx struct {
	ctx    context.Context
	reader dal.ReadSession
	writer dal.ReadwriteSession
	err    error
}

func (tx *dalgoRepositoryTx) GetPersonalProfile(ref PersonalProfileRef) (models4sportius.PersonalProfileRecord, bool) {
	var extension spaceExtensionDBO
	if tx.get(spaceExtensionKey(ref.SpaceID), &extension) && extension.Personal != nil {
		return models4sportius.ClonePersonalProfileRecord(*extension.Personal), true
	}
	if tx.err != nil {
		return models4sportius.PersonalProfileRecord{}, false
	}
	var legacy models4sportius.PersonalProfileRecord
	if !tx.get(legacyProfileKey(ref.UserID), &legacy) {
		return models4sportius.PersonalProfileRecord{}, false
	}
	return models4sportius.ClonePersonalProfileRecord(legacy), true
}

func (tx *dalgoRepositoryTx) GetTeam(spaceID string) (models4sportius.TeamRecord, bool) {
	var value spaceExtensionDBO
	if !tx.get(spaceExtensionKey(spaceID), &value) || value.Kind != sportius.SpaceKindTeam || value.Team == nil {
		return models4sportius.TeamRecord{}, false
	}
	return models4sportius.CloneTeamRecord(*value.Team), true
}

func (tx *dalgoRepositoryTx) ListTeams() []models4sportius.TeamRecord {
	if tx.err != nil {
		return nil
	}
	q := dal.From(extensionCollectionRef(teamsIndexCollection)).NewQuery().
		SelectIntoRecord(func() dalrecord.Record {
			return dalrecord.NewRecordWithIncompleteKey(teamsIndexCollection, reflect.String, new(models4sportius.TeamSearchRecord))
		})
	records, err := dal.ExecuteQueryAndReadAllToRecords(tx.ctx, q, tx.reader)
	if err != nil {
		tx.err = err
		return nil
	}
	result := make([]models4sportius.TeamRecord, 0, len(records))
	for _, record := range records {
		search := record.Data().(*models4sportius.TeamSearchRecord)
		if value, ok := tx.GetTeam(search.Brief.SpaceID); ok {
			result = append(result, value)
		}
	}
	return result
}

func (tx *dalgoRepositoryTx) FindTeamSearchRecords(filter TeamSearchFilter) []models4sportius.TeamSearchRecord {
	if tx.err != nil {
		return nil
	}
	qb := dal.From(extensionCollectionRef(teamsIndexCollection)).NewQuery().
		WhereField("nameKey", dal.Equal, filter.NameKey)
	if filter.SportID != "" {
		qb = qb.WhereField("sportID", dal.Equal, filter.SportID)
	}
	if filter.LocalityKey != "" {
		qb = qb.WhereField("localityKey", dal.Equal, filter.LocalityKey)
	}
	q := qb.SelectIntoRecord(func() dalrecord.Record {
		return dalrecord.NewRecordWithIncompleteKey(teamsIndexCollection, reflect.String, new(models4sportius.TeamSearchRecord))
	})
	records, err := dal.ExecuteQueryAndReadAllToRecords(tx.ctx, q, tx.reader)
	if err != nil {
		tx.err = err
		return nil
	}
	result := make([]models4sportius.TeamSearchRecord, 0, len(records))
	for _, record := range records {
		result = append(result, models4sportius.CloneTeamSearchRecord(*record.Data().(*models4sportius.TeamSearchRecord)))
	}
	return result
}

func (tx *dalgoRepositoryTx) GetClub(spaceID string) (models4sportius.ClubRecord, bool) {
	var value spaceExtensionDBO
	if !tx.get(spaceExtensionKey(spaceID), &value) || value.Kind != sportius.SpaceKindClub || value.Club == nil {
		return models4sportius.ClubRecord{}, false
	}
	return models4sportius.CloneClubRecord(*value.Club), true
}

func (tx *dalgoRepositoryTx) ListClubs() []models4sportius.ClubRecord {
	if tx.err != nil {
		return nil
	}
	q := dal.From(extensionCollectionRef(clubsIndexCollection)).NewQuery().
		SelectIntoRecord(func() dalrecord.Record {
			return dalrecord.NewRecordWithIncompleteKey(clubsIndexCollection, reflect.String, new(models4sportius.ClubSearchRecord))
		})
	records, err := dal.ExecuteQueryAndReadAllToRecords(tx.ctx, q, tx.reader)
	if err != nil {
		tx.err = err
		return nil
	}
	result := make([]models4sportius.ClubRecord, 0, len(records))
	for _, record := range records {
		search := record.Data().(*models4sportius.ClubSearchRecord)
		if value, ok := tx.GetClub(search.Brief.SpaceID); ok {
			result = append(result, value)
		}
	}
	return result
}

func (tx *dalgoRepositoryTx) FindClubSearchRecords(filter ClubSearchFilter) []models4sportius.ClubSearchRecord {
	if tx.err != nil {
		return nil
	}
	qb := dal.From(extensionCollectionRef(clubsIndexCollection)).NewQuery().
		WhereField("nameKey", dal.Equal, filter.NameKey)
	if filter.LocalityKey != "" {
		qb = qb.WhereField("localityKey", dal.Equal, filter.LocalityKey)
	}
	q := qb.SelectIntoRecord(func() dalrecord.Record {
		return dalrecord.NewRecordWithIncompleteKey(clubsIndexCollection, reflect.String, new(models4sportius.ClubSearchRecord))
	})
	records, err := dal.ExecuteQueryAndReadAllToRecords(tx.ctx, q, tx.reader)
	if err != nil {
		tx.err = err
		return nil
	}
	result := make([]models4sportius.ClubSearchRecord, 0, len(records))
	for _, record := range records {
		result = append(result, models4sportius.CloneClubSearchRecord(*record.Data().(*models4sportius.ClubSearchRecord)))
	}
	return result
}

func (tx *dalgoRepositoryTx) GetInvitation(invitationID string) (models4sportius.InvitationRecord, bool) {
	var value models4sportius.InvitationRecord
	if !tx.get(invitationKey(invitationID), &value) {
		return models4sportius.InvitationRecord{}, false
	}
	return models4sportius.CloneInvitationRecord(value), true
}

func (tx *dalgoRepositoryTx) FindInvitationByRequest(actorUserID, requestID string) (models4sportius.InvitationRecord, bool) {
	if tx.err != nil {
		return models4sportius.InvitationRecord{}, false
	}
	q := dal.From(extensionCollectionRef(invitationsCollection)).NewQuery().
		SelectIntoRecord(func() dalrecord.Record {
			return dalrecord.NewRecordWithIncompleteKey(invitationsCollection, reflect.String, new(models4sportius.InvitationRecord))
		})
	records, err := dal.ExecuteQueryAndReadAllToRecords(tx.ctx, q, tx.reader)
	if err != nil {
		tx.err = err
		return models4sportius.InvitationRecord{}, false
	}
	for _, record := range records {
		value := record.Data().(*models4sportius.InvitationRecord)
		if value.CreatedBy == actorUserID && value.RequestID == requestID {
			return models4sportius.CloneInvitationRecord(*value), true
		}
	}
	return models4sportius.InvitationRecord{}, false
}

func (tx *dalgoRepositoryTx) PutPersonalProfile(profile models4sportius.PersonalProfileRecord) {
	value := models4sportius.ClonePersonalProfileRecord(profile)
	tx.set(spaceExtensionKey(profile.SpaceID), spaceExtensionDBO{Personal: &value})
}

func (tx *dalgoRepositoryTx) PutTeam(team models4sportius.TeamRecord) {
	value := models4sportius.CloneTeamRecord(team)
	tx.set(spaceExtensionKey(team.Profile.SpaceID), spaceExtensionDBO{Kind: sportius.SpaceKindTeam, Team: &value})
	tx.set(extensionItemKey(teamsIndexCollection, team.Profile.SpaceID), teamSearchRecord(team.Profile))
}

func (tx *dalgoRepositoryTx) PutClub(club models4sportius.ClubRecord) {
	value := models4sportius.CloneClubRecord(club)
	tx.set(spaceExtensionKey(club.Profile.SpaceID), spaceExtensionDBO{Kind: sportius.SpaceKindClub, Club: &value})
	tx.set(extensionItemKey(clubsIndexCollection, club.Profile.SpaceID), clubSearchRecord(club.Profile))
}

func (tx *dalgoRepositoryTx) PutInvitation(invitation models4sportius.InvitationRecord) {
	value := models4sportius.CloneInvitationRecord(invitation)
	value.Invitation.DeepLink = ""
	tx.set(invitationKey(value.Invitation.InvitationID), value)
}

func (tx *dalgoRepositoryTx) get(key *dalrecord.Key, destination any) bool {
	if tx.err != nil {
		return false
	}
	record := dalrecord.NewRecordWithData(key, destination)
	if err := tx.reader.Get(tx.ctx, record); err != nil {
		if dalrecord.IsNotFound(err) {
			return false
		}
		tx.err = err
		return false
	}
	return record.Exists()
}

func (tx *dalgoRepositoryTx) set(key *dalrecord.Key, value any) {
	if tx.err != nil {
		return
	}
	if tx.writer == nil {
		tx.err = ErrForbidden
		return
	}
	tx.err = tx.writer.Set(tx.ctx, dalrecord.NewRecordWithData(key, value))
}

func legacyProfileKey(userID string) *dalrecord.Key {
	return dalrecord.NewKeyWithParentAndID(dalrecord.NewKeyWithID(usersCollection, userID), extCollection, sportius.ExtensionID)
}

func spaceExtensionKey(spaceID string) *dalrecord.Key {
	return dalrecord.NewKeyWithParentAndID(dalrecord.NewKeyWithID(spacesCollection, spaceID), extCollection, sportius.ExtensionID)
}

func extensionRootKey() *dalrecord.Key {
	return dalrecord.NewKeyWithID(extCollection, sportius.ExtensionID)
}

func extensionCollectionRef(collection string) dal.CollectionRef {
	return dal.NewCollectionRef(collection, "", extensionRootKey())
}

func extensionItemKey(collection, id string) *dalrecord.Key {
	return dalrecord.NewKeyWithParentAndID(extensionRootKey(), collection, id)
}

func invitationKey(invitationID string) *dalrecord.Key {
	return extensionItemKey(invitationsCollection, invitationID)
}
