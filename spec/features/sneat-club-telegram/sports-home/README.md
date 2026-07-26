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
buttons from the authenticated user’s Sports data. It MUST escape user-entered
text and truncate long collections into a compact Telegram card.

#### REQ: empty-state-actions

An empty summary MUST offer Add Sport, Find/Create Team and Find/Create Club paths.

#### REQ: entry-and-navigation

Sports MUST be reachable through normal Sneat bot navigation and `/sports`.
My Sports, My Teams and My Clubs MUST each open a detailed entity-appropriate
view. Home listings MUST be derived from authoritative generic space access and
MUST NOT treat a stale Sportius participant projection as membership.
Personal and family Space cards SHOULD expose the same Sports entry point.
Those buttons are navigation only: a family entry MUST NOT aggregate or reveal
members' team/club memberships unless each affiliation has been explicitly
shared with or linked to the family Space.

## Acceptance Criteria

### AC: home-renders-summary (verifies REQ:compact-summary)

**Given** a user with Basketball, one team and one club,
**When** they open Sports,
**Then** the card names all three categories and offers My Sports, My Teams and My Clubs.

### AC: home-renders-empty-actions (verifies REQ:empty-state-actions)

**Given** a user with no Sports records,
**When** they open Sports,
**Then** they can start every supported first-use path.

### AC: sports-has-both-entry-points (verifies REQ:entry-and-navigation)

**Given** an authenticated Sneat bot user,
**When** they choose Sports from normal navigation or send `/sports`,
**Then** both paths render the same compact Sports home.

### AC: summaries-are-safe-and-compact (verifies REQ:compact-summary)

**Given** many teams including one with Telegram markup characters in its name,
**When** Sports home renders,
**Then** the name is escaped, the message remains within Telegram limits, and a
detail action exposes entries omitted from the compact summary.

### AC: home-uses-authoritative-access (verifies REQ:entry-and-navigation)

**Given** a stale Sportius participant projection for a space the actor cannot
access,
**When** Sports home renders,
**Then** that team or club is absent.

### AC: space-cards-open-sports-without-family-disclosure (verifies REQ:entry-and-navigation)

**Given** an actor can open personal and family Space cards and has a private team
membership,
**When** either card offers Sports,
**Then** the actor can open the Sports section and the family card alone does not
publish or aggregate that private membership for other family members.

## Open Questions

None.

---
*This document follows the https://specscore.md/feature-specification*
