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
immediately; approval-required creates the existing request flow; invite-only
requires a valid invitation.

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

## Open Questions

Ownership transfer/coach claim is a follow-up; the model must permit multiple owners.

---
*This document follows the https://specscore.md/feature-specification*
