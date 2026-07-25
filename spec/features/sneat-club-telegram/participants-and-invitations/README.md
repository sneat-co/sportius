---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Participants and Invitations

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/participants-and-invitations?op=explore) | [Edit](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/participants-and-invitations?op=edit) | [Ask question](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/participants-and-invitations?op=ask) | [Request change](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/participants-and-invitations?op=request-change) |
**Status:** Draft
**Source Ideas:** sneat-club

## Summary

Explicit participant roles, contact-based players/guardians, staff and reusable
Telegram invitations.

## Problem

People, contacts, space members, participant roles and permissions overlap but are
not the same thing.

## Behavior

#### REQ: participant-roles

Team and club participant roles MUST use stable codes, allow multiple values and be
separate from personal roles and Sneat space permissions. Team staff includes coach,
assistant-coach, manager, administrator, organiser, volunteer, official and other.

#### REQ: players-and-staff

An administrator MUST add a player with display name only and add staff. For this
MVP a player is also made a team space member; the team role record MUST remain
separable so future contact-only players are possible. Coaches/assistant coaches are
space members; no team is required to have a coach.

#### REQ: guardian-linkage

Guardians MUST be contacts linked to player contacts by generic contact linkage;
one guardian may link to several players. A guardian MUST NOT gain team space access
solely from that relationship and is shown through the player profile.

#### REQ: invitations

Invitations MUST reuse the canonical Invitus personal-contact invitation flow, not
the legacy Debtus referral/invite store or the generic mass-invite claim path. The
inviter selects an existing accessible space contact or supplies a display name so
a non-member contact is created before issuing the invitation. The generic invite
claims that same contact and its space membership; Sportius stores optional
suggested participant roles separately.

Invitees may accept, remove or add Sportius roles (including none). Acceptance MUST
apply those roles to the contact claimed by Invitus and MUST NOT create a second
contact. Invalid or expired invitations create neither membership nor Sportius
participant projections. Inspection and acceptance MUST validate the opaque
claim proof from the delivery link (currently the Invitus PIN); an invitation
ID alone is insufficient. The full delivery link is returned only to the
creator and MUST NOT be persisted in Sportius projections or callback data.

## Acceptance Criteria

### AC: display-name-player (verifies REQ:players-and-staff)

**Given** a team administrator,
**When** they add player “Mia”,
**Then** a player contact and MVP team-space membership are created without surname,
birth date, phone or email.

### AC: guardian-is-not-member (verifies REQ:guardian-linkage)

**Given** a player and parent contact,
**When** staff link the parent as guardian,
**Then** the player profile shows the guardian relationship and the parent has no
team-space membership.

### AC: invite-roles-remain-editable (verifies REQ:invitations)

**Given** an invitation suggested as Player and Coach,
**When** the invitee removes Coach and accepts,
**Then** they join with Player only and may later adjust participant roles.

### AC: invite-claims-existing-contact (verifies REQ:invitations)

**Given** “Mia” is a non-member team contact with a pending personal invitation,
**When** Mia accepts through Telegram and confirms no Sportius role,
**Then** Invitus binds her user to that contact, generic membership is created,
Sportius records the same contact identity, and no duplicate Mia contact exists.

### AC: invite-requires-delivery-proof (verifies REQ:invitations)

**Given** a valid pending invitation ID,
**When** an actor inspects or accepts it without the delivery claim token or with
an incorrect token,
**Then** no participant, membership or invitation projection changes and the
token is not logged, rendered or placed in Telegram callback data.

## Open Questions

None.

---
*This document follows the https://specscore.md/feature-specification*
