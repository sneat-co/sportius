---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Shared Profile Enrichment

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/shared-profile-enrichment?op=explore) | [Edit](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/shared-profile-enrichment?op=edit) | [Ask question](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/shared-profile-enrichment?op=ask) | [Request change](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/shared-profile-enrichment?op=request-change) |
**Status:** Draft
**Source Ideas:** sneat-club

## Summary

Optional locality and visual identity for team and club profiles.

## Problem

Names alone are insufficient for recognition and discovery, but precision and images
must never block first use.

## Behavior

#### REQ: location-reuse

Teams and clubs MUST allow Telegram geolocation, typed town/city/country, or Skip.
Sportius MUST reuse an existing generic location abstraction or reference a
ToGethered spot through linkage; it MUST NOT introduce a competing venue model.

#### REQ: media-reuse

Teams and clubs MUST offer a logo/photo upload or Skip using existing Sneat media
handling. The media reference, not a new binary subsystem, is stored in the profile.

## Acceptance Criteria

### AC: locality-without-coordinates (verifies REQ:location-reuse)

**Given** a team creator,
**When** they enter “Limerick, Ireland”,
**Then** that locality is saved in the selected shared location representation without
requiring coordinates.

### AC: image-is-skippable (verifies REQ:media-reuse)

**Given** a club creator without an image,
**When** they choose Skip,
**Then** the club is created and its profile has no media reference.

## Open Questions

Resolve the concrete ToGethered adapter only after repository contract review.

---
*This document follows the https://specscore.md/feature-specification*
