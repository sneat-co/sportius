---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Team-to-Club Linkage

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/team-club-linkage?op=explore) | [Edit](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/team-club-linkage?op=edit) | [Ask question](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/team-club-linkage?op=ask) | [Request change](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/team-club-linkage?op=request-change) |
**Status:** Draft
**Source Ideas:** sneat-club

## Summary

Teams and clubs relate through the platform’s generic, many-to-many-capable
space-linkage mechanism; the MVP presents at most one club for a team.

## Problem

An informal team may later join a club, while clubs can begin before teams.

## Behavior

#### REQ: generic-linkage-source-of-truth

Team-to-club affiliation MUST be written and read through generic space linkage,
not a bespoke club-id field as its sole source of truth.

#### REQ: single-club-mvp

The Telegram UI MUST offer Select Club, Create Club and No Club for Now after team
creation, and expose zero or one club for that team. The domain MUST retain generic
many-to-many compatibility for later use.

#### REQ: linkage-authorisation

Linking from either team or club context MUST require management access to both
spaces and MUST NOT grant ownership or roster access. Generic linkage reads are
authoritative over Sportius caches. Relinking MUST preserve the UI’s zero-or-one
club invariant until a future many-to-many product decision.

## Acceptance Criteria

### AC: newly-created-club-is-linked (verifies REQ:generic-linkage-source-of-truth, REQ:single-club-mvp)

**Given** a newly created independent team,
**When** its creator creates a club from the affiliation step,
**Then** a club space is created and a generic team-to-club linkage is persisted.

### AC: independent-team-is-valid (verifies REQ:single-club-mvp)

**Given** a team creator chooses No Club for Now,
**When** the team profile opens,
**Then** the team is usable with no club linkage.

### AC: existing-club-can-be-selected (verifies REQ:single-club-mvp, REQ:linkage-authorisation)

**Given** an administrator can manage an independent team and an existing club,
**When** they select that club from the team flow,
**Then** the generic linkage is visible from both profiles and neither ownership
record changes.

### AC: linkage-requires-both-sides (verifies REQ:linkage-authorisation)

**Given** an actor manages only one of the two spaces,
**When** they try to link the team and club from either context,
**Then** the command is rejected without writing a linkage or granting access.

### AC: generic-linkage-overrides-stale-cache (verifies REQ:generic-linkage-source-of-truth)

**Given** a Sportius profile cache names one club but generic linkages name another
or none,
**When** the team profile is read,
**Then** the generic linkage result is returned and the cache is reconciled or
cleared.

### AC: relinking-keeps-single-club-ui (verifies REQ:single-club-mvp)

**Given** a team already has a club in the MVP,
**When** an administrator selects another club,
**Then** the flow requires replacing the existing affiliation rather than exposing
two clubs, while the stored relation remains merge/reconciliation capable.

## Open Questions

The host’s exact generic cross-space linkage write adapter remains an
approval-gated core integration decision; Sportius will not substitute a bespoke
authoritative club ID.

---
*This document follows the https://specscore.md/feature-specification*
