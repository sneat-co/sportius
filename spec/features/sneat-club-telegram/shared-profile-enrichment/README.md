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
For the MVP, `LocationHint` is the deliberately narrow boundary: locality,
region/country and optional Telegram coordinates may be stored as a discovery
hint, or `TogetheredSpotID` may reference a reusable ToGethered place. Venue
identity, lifecycle and attendance remain owned by ToGethered.

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

### AC: telegram-location-is-supported (verifies REQ:location-reuse)

**Given** a team creator shares Telegram geolocation,
**When** they continue,
**Then** the coordinates are stored in the narrow location hint without creating a
parallel Sportius venue.

### AC: location-is-skippable-and-editable (verifies REQ:location-reuse)

**Given** a creator skips location,
**When** they later add or replace a locality or ToGethered spot reference,
**Then** creation was not blocked and the profile reflects the later edit.

### AC: uploaded-photo-uses-media-reference (verifies REQ:media-reuse)

**Given** a creator uploads a logo through existing Sneat media handling,
**When** the upload succeeds,
**Then** Sportius stores only the returned media reference and renders it on the
team or club card.

### AC: enrichment-copy-is-concise (verifies REQ:location-reuse, REQ:media-reuse)

**Given** an optional location or image step,
**When** Telegram explains its benefit,
**Then** the explanation is at most two short sentences and Skip remains visible.

## Open Questions

None for MVP. Converting locality-only hints into canonical ToGethered spots is
future work and does not change the Sportius contract.

---
*This document follows the https://specscore.md/feature-specification*
