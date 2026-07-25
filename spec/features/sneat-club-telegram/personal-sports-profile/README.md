---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Personal Sports Profile

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/personal-sports-profile?op=explore) | [Edit](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/personal-sports-profile?op=edit) | [Ask question](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/personal-sports-profile?op=ask) | [Request change](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/personal-sports-profile?op=request-change) |
**Status:** Draft
**Source Ideas:** sneat-club

## Summary

A user-controlled list of sports and descriptive, general roles independent of team
or club membership.

## Problem

Interest, personal identity and operational membership have different privacy and
permission meanings.

## Behavior

#### REQ: personal-sport-entry

Users MUST be able to add, hide or remove multiple catalogue-backed sports without
joining a team; storage MUST use stable sport identifiers.

#### REQ: role-multiselect

For each sport, users MUST be able to toggle zero or more stable-code general roles
in Telegram and continue. Initial roles include player, coach, assistant-coach,
manager, organiser, volunteer, parent-guardian and other; labels are localised.
The first keyboard SHOULD show a small common subset and progressively disclose
the rest rather than presenting the whole catalogue at once.

#### REQ: affiliation-follow-up

After adding a sport, the bot MUST offer team search, team creation or Skip; it MUST
NOT force a team or expose the sport publicly by default.

## Acceptance Criteria

### AC: roles-are-multiple-and-descriptive (verifies REQ:role-multiselect)

**Given** a user adding Basketball,
**When** they select Player and Coach then continue,
**Then** both stable role codes are saved to their Basketball profile entry and no
space permission changes.

### AC: profile-does-not-require-team (verifies REQ:personal-sport-entry, REQ:affiliation-follow-up)

**Given** a user adds Football and skips affiliation,
**When** the wizard completes,
**Then** Football remains on their private profile and no team exists or is joined.

### AC: role-selection-may-be-empty (verifies REQ:role-multiselect)

**Given** a user adding Basketball,
**When** they continue without selecting a role,
**Then** Basketball is saved with an empty role list.

### AC: profile-entry-is-user-controlled (verifies REQ:personal-sport-entry)

**Given** a user has several personal sports,
**When** they hide one, unhide it and later remove it,
**Then** only that profile entry changes and no team or club membership changes.

### AC: catalogue-values-are-stable-and-localised (verifies REQ:personal-sport-entry, REQ:role-multiselect)

**Given** the same profile is rendered in two supported locales,
**When** catalogue labels are translated,
**Then** the stored sport and role codes remain unchanged.

### AC: affiliation-actions-continue-the-selected-sport (verifies REQ:affiliation-follow-up)

**Given** Basketball was just added,
**When** the user chooses Search Team or Create Team,
**Then** the next flow is constrained to Basketball and can complete without
changing profile visibility.

## Open Questions

None.

---
*This document follows the https://specscore.md/feature-specification*
