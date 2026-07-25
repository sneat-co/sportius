---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Sports Home

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/sports-home?op=explore) | [Edit](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/sports-home?op=edit) | [Ask question](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/sports-home?op=ask) | [Request change](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/sports-home?op=request-change) |
**Status:** Draft
**Source Ideas:** sneat-club

## Summary

Compact Telegram entry point for a user’s sports, teams and clubs.

## Problem

Users need a useful starting point that does not mimic a web dashboard.

## Behavior

#### REQ: compact-summary

The bot MUST render My Sports, My Teams and My Clubs summaries and navigation
buttons from the authenticated user’s Sports data.

#### REQ: empty-state-actions

An empty summary MUST offer Add Sport, Find/Create Team and Find/Create Club paths.

## Acceptance Criteria

### AC: home-renders-summary (verifies REQ:compact-summary)

**Given** a user with Basketball, one team and one club,
**When** they open Sports,
**Then** the card names all three categories and offers My Sports, My Teams and My Clubs.

### AC: home-renders-empty-actions (verifies REQ:empty-state-actions)

**Given** a user with no Sports records,
**When** they open Sports,
**Then** they can start every supported first-use path.

## Open Questions

None.

---
*This document follows the https://specscore.md/feature-specification*
