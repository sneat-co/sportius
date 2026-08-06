package facade4sportius

import (
	"context"

	sportius "github.com/sneat-co/ext-sportius/backend"
)

// CorePort is the narrow boundary from Sportius business logic to generic
// Sneat infrastructure. Its implementation belongs in the host composition
// root; Sportius must not import generic space, contact, linkage, membership,
// or invitation implementation packages directly.
//
// Mutation methods receive a stable RequestID and must be idempotent. This
// lets the service safely retry when a core write succeeded but updating the
// Sportius projection failed.
type CorePort interface {
	CreateSpace(ctx context.Context, input CreateSpaceInput) (spaceID string, err error)
	UpdateSpaceName(ctx context.Context, input UpdateSpaceNameInput) error
	EnsureSpaceMember(ctx context.Context, input EnsureSpaceMemberInput) error
	CreateContact(ctx context.Context, input CreateContactInput) (contactID string, err error)
	GetSpaceContact(ctx context.Context, input GetSpaceContactInput) (CoreContact, error)
	LinkContacts(ctx context.Context, input LinkContactsInput) error
	LinkSpaces(ctx context.Context, input LinkSpacesInput) error
	ResolveTeamClubLinks(ctx context.Context, input ResolveTeamClubLinksInput) ([]CoreTeamClubLink, error)
	CreateInvitation(ctx context.Context, input CoreInvitationInput) (CoreInvitation, error)
	ResolveInvitation(ctx context.Context, input CoreResolveInvitationInput) (CoreInvitationResolution, error)
	AcceptInvitation(ctx context.Context, input CoreAcceptInvitationInput) (CoreInvitationClaim, error)
	UserDisplayName(ctx context.Context, userID string) (string, error)
	GetPersonalSpaceID(ctx context.Context, actorUserID string) (spaceID string, err error)
	GetSpaceAccess(ctx context.Context, actorUserID, spaceID string) (SpaceAccess, error)
	ListUserSportSpaces(ctx context.Context, actorUserID string) ([]UserSportSpaceAccess, error)
}

// TeamRosterPort is an optional, privileged read port for a competition host.
// It deliberately lives beside CorePort instead of extending it: existing
// Sportius hosts continue to compile until they opt in, while a competition
// that needs an authoritative roster fails closed when the host has not wired
// this capability.
//
// The returned set is generic Space membership, not a Sportius projection.
// The host MUST return every current member identity it can authoritatively
// resolve for the Space.
type TeamRosterPort interface {
	ListSpaceMembers(ctx context.Context, spaceID string) ([]CoreSpaceMember, error)
}

// CoreSpaceMember is the minimum generic membership identity needed to verify
// a competition roster. UserID is required for competition entrants; ContactID
// is retained so hosts that support contact-only memberships can expose their
// stable identity without granting them entry to an authenticated competition.
type CoreSpaceMember struct {
	UserID    string
	ContactID string
}

type CreateSpaceInput struct {
	RequestID   string
	Kind        sportius.SpaceKind
	Name        string
	OwnerUserID string
}

type UpdateSpaceNameInput struct {
	RequestID   string
	SpaceID     string
	Name        string
	ActorUserID string
}

type EnsureSpaceMemberInput struct {
	RequestID   string
	SpaceID     string
	UserID      string
	ContactID   string
	Owner       bool
	ActorUserID string
}

type CreateContactInput struct {
	RequestID   string
	SpaceID     string
	DisplayName string
	UserID      string
	ActorUserID string
}

// GetSpaceContactInput resolves an invitation target through generic contact
// access rules. Sportius must not accept an arbitrary contact ID merely
// because the caller can manage the Sportius projection.
type GetSpaceContactInput struct {
	SpaceID     string
	ContactID   string
	ActorUserID string
}

// CoreContact is the minimum generic contact identity Sportius needs for an
// invitation. The generic contact remains the source of truth.
type CoreContact struct {
	ContactID   string
	UserID      string
	DisplayName string
}

type LinkContactsInput struct {
	RequestID        string
	SpaceID          string
	FromContactID    string
	ToContactID      string
	RelationshipRole string
	ActorUserID      string
}

type LinkSpacesInput struct {
	RequestID   string
	FromSpaceID string
	ToSpaceID   string
	Role        string
	ActorUserID string
}

// ResolveTeamClubLinksInput reads generic space-to-space linkages
// authoritatively. Exactly one filter is normally supplied: TeamSpaceID when
// rendering a team, ClubSpaceID when rendering a club.
type ResolveTeamClubLinksInput struct {
	TeamSpaceID string
	ClubSpaceID string
	ActorUserID string
}

// CoreTeamClubLink is derived by the host adapter from generic linkages.
// Cross-team roster visibility is deliberately kept in the Sportius-owned
// ClubManagerRosterTeamIDs policy; a generic linkage alone grants no contact
// visibility.
type CoreTeamClubLink struct {
	TeamSpaceID string
	ClubSpaceID string
}

type CoreInvitationInput struct {
	RequestID   string
	SpaceID     string
	ContactID   string
	ActorUserID string
}

type CoreInvitation struct {
	InvitationID string
	DeepLink     string
	ExpiresAt    string
}

type CoreAcceptInvitationInput struct {
	RequestID    string
	InvitationID string
	ClaimToken   string
	ActorUserID  string
}

type CoreResolveInvitationInput struct {
	InvitationID string
	ClaimToken   string
	ActorUserID  string
}

// CoreInvitationResolution represents generic invitation state. The generic
// invitation subsystem is authoritative for revocation and claims; the local
// Sportius record is a recoverable roles/status projection. Implementations
// MUST validate ClaimToken before returning any invitation metadata or status.
type CoreInvitationResolution struct {
	InvitationID string
	SpaceID      string
	ContactID    string
	Status       sportius.InvitationStatus
	Claim        CoreInvitationClaim
}

// CoreInvitationClaim is returned by generic invitation acceptance after it
// has claimed the invitation's existing target contact for the authenticated
// user. Sportius adds role metadata to this identity; it must not create a
// second contact.
type CoreInvitationClaim struct {
	ContactID   string
	UserID      string
	DisplayName string
}

// SpaceAccess comes from generic Sneat access, which is authoritative.
// Sportius projections must never grant access to contact data or mutations.
type SpaceAccess struct {
	IsMember  bool
	CanManage bool
}

// UserSportSpaceAccess is the authoritative generic-space membership index
// used by Sports home. Sportius member maps are projections and are never used
// to decide whether a team or club belongs on the current user's home.
type UserSportSpaceAccess struct {
	SpaceID   string
	Kind      sportius.SpaceKind
	IsMember  bool
	CanManage bool
}
