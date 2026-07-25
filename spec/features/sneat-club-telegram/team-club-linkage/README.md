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

## Acceptance Criteria

### AC: newly-created-club-is-linked (verifies REQ:generic-linkage-source-of-truth, REQ:single-club-mvp)

**Given** a newly created independent team,
**When** its creator creates a club from the affiliation step,
**Then** a club space is created and a generic team-to-club linkage is persisted.

### AC: independent-team-is-valid (verifies REQ:single-club-mvp)

**Given** a team creator chooses No Club for Now,
**When** the team profile opens,
**Then** the team is usable with no club linkage.

## Open Questions

Existing linkage role names and authorisation are resolved by facade integration.

---
*This document follows the https://specscore.md/feature-specification*
