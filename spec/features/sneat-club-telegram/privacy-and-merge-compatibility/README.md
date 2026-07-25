---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Privacy and Future Space-Merge Compatibility

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/privacy-and-merge-compatibility?op=explore) | [Edit](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/privacy-and-merge-compatibility?op=edit) | [Ask question](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/privacy-and-merge-compatibility?op=ask) | [Request change](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/privacy-and-merge-compatibility?op=request-change) |
**Status:** Draft
**Source Ideas:** sneat-club

## Summary

Protect personal profile privacy now and retain records that can be reconciled by a
future generic same-type space merge.

## Problem

Membership is operational information, profile disclosure is user-controlled, and
duplicate real-world organisations are unavoidable.

## Behavior

#### REQ: privacy-boundaries

Personal sport visibility MUST be user-controlled and independent from team/club
membership. Guardian links MUST NOT disclose team data or grant access. Bot handlers
MUST validate identity, callback state, space access and invitation validity.

#### REQ: merge-aware-records

Sportius MUST use stable ids and relations rather than globally unique names. Its
extension records MUST be reconcilable if generic merging remaps spaces; no feature
may assume a permanent sole historical identity or irreconcilable singleton link.

#### REQ: conflict-inventory

Implementation documentation MUST identify future resolution rules for conflicting
names, age/gender, club links, participant roles, contacts, owners, images and
locations. Generic merge infrastructure is out of scope.

## Acceptance Criteria

### AC: membership-does-not-publish-profile (verifies REQ:privacy-boundaries)

**Given** a user joins a Football team while Football is hidden or absent on their
personal profile,
**When** another member views the team,
**Then** the membership is available according to team access while the user’s
personal Football profile entry is not automatically exposed or created.

### AC: duplicates-remain-reconcilable (verifies REQ:merge-aware-records, REQ:conflict-inventory)

**Given** two separately created teams with the same name and overlapping people,
**When** their records are inspected for a future generic merge,
**Then** they can be related by stable ids and documented conflicts can be resolved
without relying on name uniqueness or deleting historical relations.

## Open Questions

Generic merge semantics belong to Sneat core and require separate approval.

---
*This document follows the https://specscore.md/feature-specification*
