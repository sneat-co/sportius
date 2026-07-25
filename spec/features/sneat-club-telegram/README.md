---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Sneat Club Telegram MVP

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram?op=explore) | [Edit](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram?op=edit) | [Ask question](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram?op=ask) | [Request change](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram?op=request-change) |
**Status:** Draft
**Source Ideas:** sneat-club

## Summary

The first production surface for **Sneat Club** (`sneat.club`): a compact Telegram
experience in the existing Sneat bot. Sportius owns organisational Sports data and
orchestration; the bot is presentation only. A club and a team are separate Sneat
spaces. GameBoard owns games, scoring and match participation; ToGethered remains
the owner of reusable places/venues and is referenced rather than duplicated.

## Contents

| Child | Description |
|---|---|
| [sports-home](sports-home/README.md) | Entry card and navigation |
| [personal-sports-profile](personal-sports-profile/README.md) | User-controlled sports and general roles |
| [teams](teams/README.md) | Team discovery, creation, joining and editing |
| [clubs](clubs/README.md) | Club discovery, creation, staff and membership views |
| [participants-and-invitations](participants-and-invitations/README.md) | Players, staff, guardians and invitations |
| [shared-profile-enrichment](shared-profile-enrichment/README.md) | Location and photos/logos |
| [team-club-linkage](team-club-linkage/README.md) | Generic space linkage for team affiliation |
| [privacy-and-merge-compatibility](privacy-and-merge-compatibility/README.md) | Privacy boundaries and future merge safety |
| [conversational-resilience](conversational-resilience/README.md) | Resumable, localised and failure-safe Telegram flows |

## Problem

Amateur teams and clubs need to start in any order—by declaring a sport, finding a
team, creating a team, creating a club or accepting an invite—without requiring a
formal club hierarchy or a web dashboard.

## Behavior

This is an umbrella. Child specifications own behaviour and acceptance criteria.

## Acceptance Criteria

Not defined here (umbrella feature — acceptance criteria live on child features).

## Not Doing / Out of Scope

Competitions, fixtures, scoring, payments, medical data, public web pages,
fine-grained RBAC, and generic space merging.

## Open Questions

None. The MVP stores a narrow locality/coordinate hint or a ToGethered spot
reference; Sportius does not own venues or place lifecycle.

---
*This document follows the https://specscore.md/feature-specification*
