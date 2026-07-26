// Package models4sportius contains Sportius-owned persistence records.
//
// Teams and clubs are not embedded here: their identity and membership live in
// generic Sneat spaces. These records contain only the extension projection
// needed to render Sportius profiles and to make Sportius commands idempotent.
package models4sportius

import sportius "github.com/sneat-co/ext-sportius/backend"

// PersonalProfileRecord is stored under the authenticated user's personal
// Space Sportius extension namespace. UserID remains as the profile subject;
// SpaceID is the canonical ownership and persistence boundary.
type PersonalProfileRecord struct {
	SpaceID string                                      `firestore:"spaceID" json:"spaceID"`
	UserID  string                                      `firestore:"userID" json:"userID"`
	Sports  map[sportius.SportID]sportius.PersonalSport `firestore:"sports" json:"sports"`
}

// TeamSearchRecord is the deliberately public, minimal discovery projection.
// It must never grow ownership, membership, participant, guardian, request, or
// staff fields. Canonical private data remains under the team space.
type TeamSearchRecord struct {
	Brief       sportius.TeamBrief `firestore:"brief" json:"brief"`
	NameKey     string             `firestore:"nameKey" json:"nameKey"`
	SportID     sportius.SportID   `firestore:"sportID" json:"sportID"`
	LocalityKey string             `firestore:"localityKey,omitempty" json:"localityKey,omitempty"`
}

// ClubSearchRecord is the public discovery projection for exact-name lookup.
type ClubSearchRecord struct {
	Brief       sportius.ClubBrief `firestore:"brief" json:"brief"`
	NameKey     string             `firestore:"nameKey" json:"nameKey"`
	LocalityKey string             `firestore:"localityKey,omitempty" json:"localityKey,omitempty"`
	SportIDs    []sportius.SportID `firestore:"sportIDs,omitempty" json:"sportIDs,omitempty"`
}

// TeamRecord is the Sportius projection for a team space.
type TeamRecord struct {
	Profile                        sportius.TeamProfile            `firestore:"profile" json:"profile"`
	CreatedByUserID                string                          `firestore:"createdByUserID" json:"createdByUserID"`
	CreateRequestID                string                          `firestore:"createRequestID" json:"createRequestID"`
	CreateNameKey                  string                          `firestore:"createNameKey" json:"createNameKey"`
	CreateSportID                  sportius.SportID                `firestore:"createSportID" json:"createSportID"`
	CreateFingerprint              string                          `firestore:"createFingerprint" json:"createFingerprint"`
	ProfileVersion                 uint64                          `firestore:"profileVersion" json:"profileVersion"`
	OwnerUserIDs                   map[string]bool                 `firestore:"ownerUserIDs" json:"ownerUserIDs"`
	MemberUserRoles                map[string][]sportius.RoleID    `firestore:"memberUserRoles" json:"memberUserRoles"`
	Participants                   map[string]sportius.Participant `firestore:"participants" json:"participants"`
	ParticipantRequests            map[string]string               `firestore:"participantRequests" json:"participantRequests"`
	ParticipantRequestFingerprints map[string]string               `firestore:"participantRequestFingerprints" json:"participantRequestFingerprints"`
	GuardianRequests               map[string]string               `firestore:"guardianRequests" json:"guardianRequests"`
	GuardianRequestFingerprints    map[string]string               `firestore:"guardianRequestFingerprints" json:"guardianRequestFingerprints"`
	RoleRequestFingerprints        map[string]string               `firestore:"roleRequestFingerprints" json:"roleRequestFingerprints"`
	UpdateRequestFingerprints      map[string]string               `firestore:"updateRequestFingerprints" json:"updateRequestFingerprints"`
	UpdateRequestVersions          map[string]uint64               `firestore:"updateRequestVersions" json:"updateRequestVersions"`
	JoinCommandFingerprints        map[string]string               `firestore:"joinCommandFingerprints" json:"joinCommandFingerprints"`
	LinkRequestFingerprints        map[string]string               `firestore:"linkRequestFingerprints" json:"linkRequestFingerprints"`
	GuardianLinks                  map[string][]GuardianLink       `firestore:"guardianLinks" json:"guardianLinks"`
	JoinRequests                   map[string]JoinRequestRecord    `firestore:"joinRequests" json:"joinRequests"`
}

// GuardianLink is a contact-to-contact relationship projection. The generic
// Sneat linkage written through CorePort remains the system of record.
type GuardianLink struct {
	PlayerContactID    string `firestore:"playerContactID" json:"playerContactID"`
	GuardianContactID  string `firestore:"guardianContactID" json:"guardianContactID"`
	RelationshipRoleID string `firestore:"relationshipRoleID" json:"relationshipRoleID"`
}

// JoinRequestRecord prepares approval-required teams for a later approval
// command without introducing a separate Sports-only approval subsystem.
type JoinRequestRecord struct {
	RequestID   string            `firestore:"requestID" json:"requestID"`
	UserID      string            `firestore:"userID" json:"userID"`
	RoleIDs     []sportius.RoleID `firestore:"roleIDs" json:"roleIDs"`
	Fingerprint string            `firestore:"fingerprint" json:"fingerprint"`
}

// ClubRecord is the Sportius projection for a club space.
type ClubRecord struct {
	Profile                        sportius.ClubProfile            `firestore:"profile" json:"profile"`
	CreatedByUserID                string                          `firestore:"createdByUserID" json:"createdByUserID"`
	CreateRequestID                string                          `firestore:"createRequestID" json:"createRequestID"`
	CreateNameKey                  string                          `firestore:"createNameKey" json:"createNameKey"`
	CreateFingerprint              string                          `firestore:"createFingerprint" json:"createFingerprint"`
	ProfileVersion                 uint64                          `firestore:"profileVersion" json:"profileVersion"`
	OwnerUserIDs                   map[string]bool                 `firestore:"ownerUserIDs" json:"ownerUserIDs"`
	MemberRoles                    map[string][]sportius.RoleID    `firestore:"memberRoles" json:"memberRoles"`
	Participants                   map[string]sportius.Participant `firestore:"participants" json:"participants"`
	ParticipantRequests            map[string]string               `firestore:"participantRequests" json:"participantRequests"`
	ParticipantRequestFingerprints map[string]string               `firestore:"participantRequestFingerprints" json:"participantRequestFingerprints"`
	RoleRequestFingerprints        map[string]string               `firestore:"roleRequestFingerprints" json:"roleRequestFingerprints"`
	UpdateRequestFingerprints      map[string]string               `firestore:"updateRequestFingerprints" json:"updateRequestFingerprints"`
	UpdateRequestVersions          map[string]uint64               `firestore:"updateRequestVersions" json:"updateRequestVersions"`
	TeamSpaceIDs                   map[string]bool                 `firestore:"teamSpaceIDs" json:"teamSpaceIDs"`
	ClubManagerRosterTeamIDs       map[string]bool                 `firestore:"clubManagerRosterTeamIDs" json:"clubManagerRosterTeamIDs"`
}

// InvitationRecord carries Sportius role suggestions while the actual invite
// and token lifecycle remain owned by Sneat's generic invitation subsystem.
type InvitationRecord struct {
	Invitation       sportius.Invitation       `firestore:"invitation" json:"invitation"`
	CreatedBy        string                    `firestore:"createdBy" json:"createdBy"`
	RequestID        string                    `firestore:"requestID" json:"requestID"`
	Status           sportius.InvitationStatus `firestore:"status" json:"status"`
	ExpiresAt        string                    `firestore:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	AcceptedByUserID string                    `firestore:"acceptedByUserID,omitempty" json:"acceptedByUserID,omitempty"`
	AcceptRequestID  string                    `firestore:"acceptRequestID,omitempty" json:"acceptRequestID,omitempty"`
	AcceptedRoleIDs  []sportius.RoleID         `firestore:"acceptedRoleIDs,omitempty" json:"acceptedRoleIDs,omitempty"`
}

// Clone helpers ensure repository callers never mutate stored state through a
// slice, map, or pointer retained from an earlier transaction.
func ClonePersonalProfileRecord(v PersonalProfileRecord) PersonalProfileRecord {
	result := PersonalProfileRecord{
		SpaceID: v.SpaceID,
		UserID:  v.UserID,
		Sports:  make(map[sportius.SportID]sportius.PersonalSport, len(v.Sports)),
	}
	for id, entry := range v.Sports {
		entry.RoleIDs = cloneRoleIDs(entry.RoleIDs)
		result.Sports[id] = entry
	}
	return result
}

func CloneTeamSearchRecord(v TeamSearchRecord) TeamSearchRecord {
	v.Brief = cloneTeamBrief(v.Brief)
	return v
}

func CloneClubSearchRecord(v ClubSearchRecord) ClubSearchRecord {
	v.SportIDs = append([]sportius.SportID(nil), v.SportIDs...)
	return v
}

func CloneTeamRecord(v TeamRecord) TeamRecord {
	result := TeamRecord{
		Profile:                        cloneTeamProfile(v.Profile),
		CreatedByUserID:                v.CreatedByUserID,
		CreateRequestID:                v.CreateRequestID,
		CreateNameKey:                  v.CreateNameKey,
		CreateSportID:                  v.CreateSportID,
		CreateFingerprint:              v.CreateFingerprint,
		ProfileVersion:                 v.ProfileVersion,
		OwnerUserIDs:                   cloneBoolMap(v.OwnerUserIDs),
		MemberUserRoles:                cloneRoleMap(v.MemberUserRoles),
		Participants:                   cloneParticipants(v.Participants),
		ParticipantRequests:            cloneStringMap(v.ParticipantRequests),
		ParticipantRequestFingerprints: cloneStringMap(v.ParticipantRequestFingerprints),
		GuardianRequests:               cloneStringMap(v.GuardianRequests),
		GuardianRequestFingerprints:    cloneStringMap(v.GuardianRequestFingerprints),
		RoleRequestFingerprints:        cloneStringMap(v.RoleRequestFingerprints),
		UpdateRequestFingerprints:      cloneStringMap(v.UpdateRequestFingerprints),
		UpdateRequestVersions:          cloneUint64Map(v.UpdateRequestVersions),
		JoinCommandFingerprints:        cloneStringMap(v.JoinCommandFingerprints),
		LinkRequestFingerprints:        cloneStringMap(v.LinkRequestFingerprints),
		GuardianLinks:                  make(map[string][]GuardianLink, len(v.GuardianLinks)),
		JoinRequests:                   make(map[string]JoinRequestRecord, len(v.JoinRequests)),
	}
	for playerID, links := range v.GuardianLinks {
		result.GuardianLinks[playerID] = append([]GuardianLink(nil), links...)
	}
	for userID, request := range v.JoinRequests {
		request.RoleIDs = cloneRoleIDs(request.RoleIDs)
		result.JoinRequests[userID] = request
	}
	return result
}

func CloneClubRecord(v ClubRecord) ClubRecord {
	return ClubRecord{
		Profile:                        cloneClubProfile(v.Profile),
		CreatedByUserID:                v.CreatedByUserID,
		CreateRequestID:                v.CreateRequestID,
		CreateNameKey:                  v.CreateNameKey,
		CreateFingerprint:              v.CreateFingerprint,
		ProfileVersion:                 v.ProfileVersion,
		OwnerUserIDs:                   cloneBoolMap(v.OwnerUserIDs),
		MemberRoles:                    cloneRoleMap(v.MemberRoles),
		Participants:                   cloneParticipants(v.Participants),
		ParticipantRequests:            cloneStringMap(v.ParticipantRequests),
		ParticipantRequestFingerprints: cloneStringMap(v.ParticipantRequestFingerprints),
		RoleRequestFingerprints:        cloneStringMap(v.RoleRequestFingerprints),
		UpdateRequestFingerprints:      cloneStringMap(v.UpdateRequestFingerprints),
		UpdateRequestVersions:          cloneUint64Map(v.UpdateRequestVersions),
		TeamSpaceIDs:                   cloneBoolMap(v.TeamSpaceIDs),
		ClubManagerRosterTeamIDs:       cloneBoolMap(v.ClubManagerRosterTeamIDs),
	}
}

func CloneInvitationRecord(v InvitationRecord) InvitationRecord {
	v.Invitation.SuggestedRoleIDs = cloneRoleIDs(v.Invitation.SuggestedRoleIDs)
	v.AcceptedRoleIDs = cloneRoleIDs(v.AcceptedRoleIDs)
	return v
}

func cloneTeamProfile(v sportius.TeamProfile) sportius.TeamProfile {
	if v.Age != nil {
		age := *v.Age
		if v.Age.MinAge != nil {
			minAge := *v.Age.MinAge
			age.MinAge = &minAge
		}
		if v.Age.MaxAge != nil {
			maxAge := *v.Age.MaxAge
			age.MaxAge = &maxAge
		}
		v.Age = &age
	}
	v.Location = cloneLocation(v.Location)
	if v.Media != nil {
		media := *v.Media
		v.Media = &media
	}
	if v.Club != nil {
		club := *v.Club
		v.Club = &club
	}
	return v
}

func cloneTeamBrief(v sportius.TeamBrief) sportius.TeamBrief {
	if v.Age != nil {
		age := *v.Age
		if v.Age.MinAge != nil {
			minAge := *v.Age.MinAge
			age.MinAge = &minAge
		}
		if v.Age.MaxAge != nil {
			maxAge := *v.Age.MaxAge
			age.MaxAge = &maxAge
		}
		v.Age = &age
	}
	return v
}

func cloneClubProfile(v sportius.ClubProfile) sportius.ClubProfile {
	v.SportIDs = append([]sportius.SportID(nil), v.SportIDs...)
	v.Location = cloneLocation(v.Location)
	if v.Media != nil {
		media := *v.Media
		v.Media = &media
	}
	return v
}

func cloneLocation(v *sportius.LocationHint) *sportius.LocationHint {
	if v == nil {
		return nil
	}
	result := *v
	if v.Point != nil {
		point := *v.Point
		result.Point = &point
	}
	return &result
}

func cloneParticipants(v map[string]sportius.Participant) map[string]sportius.Participant {
	result := make(map[string]sportius.Participant, len(v))
	for id, participant := range v {
		participant.RoleIDs = cloneRoleIDs(participant.RoleIDs)
		result[id] = participant
	}
	return result
}

func cloneRoleMap(v map[string][]sportius.RoleID) map[string][]sportius.RoleID {
	result := make(map[string][]sportius.RoleID, len(v))
	for id, roles := range v {
		result[id] = cloneRoleIDs(roles)
	}
	return result
}

func cloneRoleIDs(v []sportius.RoleID) []sportius.RoleID {
	return append([]sportius.RoleID(nil), v...)
}

func cloneBoolMap(v map[string]bool) map[string]bool {
	result := make(map[string]bool, len(v))
	for key, value := range v {
		result[key] = value
	}
	return result
}

func cloneStringMap(v map[string]string) map[string]string {
	result := make(map[string]string, len(v))
	for key, value := range v {
		result[key] = value
	}
	return result
}

func cloneUint64Map(v map[string]uint64) map[string]uint64 {
	result := make(map[string]uint64, len(v))
	for key, value := range v {
		result[key] = value
	}
	return result
}
