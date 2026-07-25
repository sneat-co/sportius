---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Clubs

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/clubs?op=explore) | [Edit](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/clubs?op=edit) | [Ask question](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/clubs?op=ask) | [Request change](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/clubs?op=request-change) |
**Status:** Draft
**Source Ideas:** sneat-club

## Summary

Independent club spaces with optional sports, staff and linked teams.

## Problem

A club can be created before teams, and can have staff who never belong to a team.

## Behavior

#### REQ: club-creation

Club creation MUST require only a name, create a distinct Sneat space and make the
creator admin/owner. Primary/supported sports, location and image are optional;
clubs MAY be multi-sport or have no sport specified.

#### REQ: club-discovery-and-editing

Club search MUST search clubs only with the same safe name normalisation as teams;
duplicate names are valid. Trusted administrators MUST edit core fields.

#### REQ: club-views

The profile MUST provide Teams, Staff and Members views. Teams are linked spaces;
Staff is club-scoped; Members is a deduplicated aggregation of club people and
linked-team people suitable for existing bot pagination/search patterns. Club-wide
Members is manager-only: every linked team contribution MUST pass authoritative
access/policy checks, expose only a minimal display-name/role brief, omit non-member
guardians and never propagate contact details. A generic team-to-club link alone
MUST NOT grant cross-space roster visibility.

#### REQ: club-next-actions

After creation a user MUST be able to add staff, invite a coach, create a team, link
an existing team or finish. Team creation from a club links the team and leaves the
creator as team admin/owner.

## Acceptance Criteria

### AC: club-can-start-empty (verifies REQ:club-creation)

**Given** a user enters “Limerick Celtics” and skips optional steps,
**When** they create the club,
**Then** a club space exists with no teams and the creator has club admin/owner access.

### AC: club-members-are-deduplicated (verifies REQ:club-views)

**Given** one coach belongs to the club and two linked teams,
**When** an administrator opens Members,
**Then** the coach appears once in the club-wide result.

### AC: club-search-is-club-only (verifies REQ:club-discovery-and-editing)

**Given** a team and club share a normalised name,
**When** a user searches from My Clubs, rejects a candidate and retries,
**Then** only clubs are considered and the user may create a distinct duplicate-name
club after exhausting candidates.

### AC: club-enrichment-and-editing-are-optional (verifies REQ:club-creation, REQ:club-discovery-and-editing)

**Given** a club owner,
**When** they initially skip and later edit primary sport, supported sports,
location, image and name,
**Then** the stable club space is retained and each field remains optional.

### AC: club-sections-remain-distinct (verifies REQ:club-views)

**Given** a club has linked teams, club-only staff and other members,
**When** an authorised manager opens Teams, Staff and Members,
**Then** each view contains the appropriate stable identities without duplicating
people into every team.

### AC: club-members-denies-implicit-roster-access (verifies REQ:club-views)

**Given** a club manager lacks the Sportius roster policy for a linked team,
**When** they open club Members,
**Then** that team’s player/staff briefs and guardian contacts are not returned,
even though the generic team-to-club link exists.

### AC: coach-can-create-team-under-club (verifies REQ:club-next-actions)

**Given** an invited coach is club Staff,
**When** they create a team under the club,
**Then** they become that team’s owner/admin, remain club Staff, and the team is
linked through the generic space relationship.

### AC: existing-team-can-be-linked-from-club (verifies REQ:club-next-actions)

**Given** an administrator can manage an independent team and a club,
**When** they select Link Existing Team,
**Then** the team appears in the club Teams view without changing either space’s
ownership.

## Open Questions

None.

---
*This document follows the https://specscore.md/feature-specification*
