---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Teams

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/teams?op=explore) | [Edit](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/teams?op=edit) | [Ask question](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/teams?op=ask) | [Request change](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/teams?op=request-change) |
**Status:** Draft
**Source Ideas:** sneat-club

## Summary

Find, create, join, browse and edit independent team spaces.

## Problem

Teams are not unique by name and can exist before, after or without a club or coach.

## Behavior

#### REQ: team-space

Each created team MUST be its own Sneat space with a Sportius team profile; its
creator MUST receive the existing owner/admin membership capability independently of
their participant roles.

#### REQ: team-creation

Creation MUST require only a name and sport (ask sport only when context lacks one),
then allow gender category, structured min/max age, location and logo/photo to be
skipped. Duplicate names MUST be permitted.

#### REQ: team-discovery

Team search MUST search teams only and normalise case, outer/repeated whitespace and
safe punctuation. It MUST show a candidate for confirmation and offer retry or
creation on rejection/no match.

#### REQ: join-roles-and-policy

A joiner MUST choose zero or more team participant roles. Open policy joins
immediately; approval-required persists a pending request but its approve/reject
operator UI is pilot-deferred; invite-only requires a valid invitation. A pending
request MUST NOT create membership.

#### REQ: team-profile-editing

Trusted administrators MUST be able to browse Players and Staff and edit core
profile fields (name, sport, gender, age, location and image).

#### REQ: profile-consent

When create/join introduces a sport absent from the personal profile, the user MUST
be asked whether to add it; no automatic profile mutation is allowed.

## Acceptance Criteria

### AC: minimal-team-created (verifies REQ:team-space, REQ:team-creation)

**Given** a user creates “Park Court Basketball” for Basketball and skips enrichment,
**When** creation completes,
**Then** a distinct team space exists, the creator is an admin/owner, and optional
gender, age, location and image are unset.

### AC: duplicate-name-is-valid (verifies REQ:team-creation, REQ:team-discovery)

**Given** “Celtics” already exists,
**When** a user rejects that search candidate and creates another “Celtics”,
**Then** a second stable-space-id team is created without overwriting the first.

### AC: joins-follow-policy (verifies REQ:join-roles-and-policy)

**Given** open, approval-required and invite-only teams,
**When** a user selects Player and Coach and attempts each join,
**Then** the open join succeeds, the approval flow requests approval, and invite-only
rejects the attempt without a valid invitation.

### AC: team-profile-consent (verifies REQ:profile-consent)

**Given** a user creates a Hockey team but has no Hockey profile entry,
**When** they choose No at the profile prompt,
**Then** the team membership remains and Hockey is not added to the profile.

### AC: my-teams-supports-many-affiliations (verifies REQ:team-profile-editing)

**Given** a user has several roles across several teams in the same sport,
**When** they open My Teams and a team profile,
**Then** each accessible team appears once and the selected team shows all their
participant roles plus Players and Staff actions.

### AC: search-confirm-reject-and-retry (verifies REQ:team-discovery)

**Given** multiple teams have an equivalent normalised name,
**When** a user confirms one candidate, rejects another, or exhausts the results,
**Then** confirmation opens that stable team ID and rejection offers another
candidate, a new search, or team creation.

### AC: joins-allow-zero-or-one-role (verifies REQ:join-roles-and-policy)

**Given** an open team,
**When** one user joins with Player and another joins with no participant role,
**Then** both become members and their distinct role selections are retained.

### AC: approval-request-is-not-membership (verifies REQ:join-roles-and-policy)

**Given** an approval-required team,
**When** a non-member submits a join request,
**Then** a pending request is recorded, no generic membership or participant is
created, and Telegram explains that approval management is not yet in the pilot.

### AC: team-creation-uses-or-asks-sport (verifies REQ:team-creation)

**Given** one creator starts from Basketball and another starts from My Teams,
**When** each creates a team,
**Then** the first reuses Basketball and the second is asked to select a sport.

### AC: structured-age-and-gender-are-optional (verifies REQ:team-creation)

**Given** team creators provide min-only, max-only, both, or neither age bound and
choose or skip gender,
**When** each team is created,
**Then** the stored structured values and human-readable labels match each choice.

### AC: team-core-fields-are-editable (verifies REQ:team-profile-editing)

**Given** a trusted team administrator,
**When** they edit name, sport, gender, age, location and image,
**Then** the team profile reflects each change while retaining the stable space ID.

### AC: profile-consent-yes-is-explicit (verifies REQ:profile-consent)

**Given** a user joins a Basketball team without a Basketball profile entry,
**When** they explicitly choose Yes at the profile prompt,
**Then** Basketball is added privately; it is not added before that choice.

## Open Questions

Ownership transfer/coach claim is a follow-up; the model must permit multiple owners.

---
*This document follows the https://specscore.md/feature-specification*
