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
linked-team people suitable for existing bot pagination/search patterns.

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

## Open Questions

None.

---
*This document follows the https://specscore.md/feature-specification*
