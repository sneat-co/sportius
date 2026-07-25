---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Conversational Resilience

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/conversational-resilience?op=explore) | [Edit](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/conversational-resilience?op=edit) | [Ask question](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/conversational-resilience?op=ask) | [Request change](https://specscore.studio/app/github.com/sneat-co/sportius/spec/features/sneat-club-telegram/conversational-resilience?op=request-change) |
**Status:** Draft
**Source Ideas:** sneat-club

## Summary

Resumable, localised Telegram journeys that fail safely under stale callbacks,
duplicate updates and backend errors.

## Problem

Telegram delivery is asynchronous and users routinely leave and return. A real
team or club flow must preserve intentional progress without leaking secrets,
duplicating mutations or exposing internal failures.

## Behavior

#### REQ: existing-wizard-lifecycle

Sportius MUST use the existing bot-framework chat-state wizard. Back, Continue
and Skip preserve valid state; leaving and returning resumes it. Explicit Cancel
clears the wizard and all sensitive transient values.

#### REQ: facade-only-mutations

Telegram handlers MUST call the injected Sportius service/facade for domain
queries and mutations. They MUST NOT write Sportius, space, contact, membership,
linkage or invitation records directly.

#### REQ: stale-and-duplicate-safety

Callback data MUST be validated against the active flow and authenticated actor.
Stale/invalid callbacks produce a safe recovery path. Every mutation carries a
stable request ID from chat state so duplicate Telegram delivery is idempotent.
Deleted or remapped spaces return to a safe list/home instead of mutating stale
identity.

#### REQ: safe-errors-and-observability

Unexpected failures MUST be logged with structured operation/request correlation
and no claim tokens or private contact fields. Users receive localised,
non-internal error copy with Retry, Back or Home where recovery is possible.

#### REQ: localised-copy

User-facing copy and catalogue labels MUST use localisation keys with concise
British-English fallback text. User-controlled values MUST be escaped.

## Acceptance Criteria

### AC: wizard-resumes-and-cancels (verifies REQ:existing-wizard-lifecycle)

**Given** a user leaves midway through team creation,
**When** they return,
**Then** the existing chat state resumes at the last valid step; choosing Cancel
clears the wizard, selection and any invitation proof.

### AC: back-and-skip-preserve-valid-state (verifies REQ:existing-wizard-lifecycle)

**Given** a user selected a sport and reached optional enrichment,
**When** they go Back, revise a value, and Skip the next optional step,
**Then** the final command contains the revised value and no stale skipped value.

### AC: duplicate-update-is-idempotent (verifies REQ:stale-and-duplicate-safety)

**Given** Telegram delivers the same create or accept update twice,
**When** both executions use the request ID stored in chat state,
**Then** one domain result exists and the second returns that result without a
duplicate space, contact, membership, linkage or participant.

### AC: stale-callback-recovers-safely (verifies REQ:stale-and-duplicate-safety)

**Given** callback data belongs to a cancelled flow or deleted/remapped space,
**When** it is handled,
**Then** no mutation occurs and the user receives a safe route to the current
Sports home or relevant list.

### AC: bot-accesses-domain-through-facade (verifies REQ:facade-only-mutations)

**Given** a bot-flow test with an injected Sportius fake,
**When** the user completes the flow,
**Then** the expected facade calls are observed and the handler has no direct
persistence or core-module dependency.

### AC: errors-are-redacted-and-retryable (verifies REQ:safe-errors-and-observability)

**Given** a backend failure includes internal detail or an invitation token,
**When** Telegram handles it,
**Then** logs contain only redacted correlated context, user copy exposes neither
detail nor token, and an applicable Retry, Back or Home action is offered.

### AC: localisation-keys-are-complete (verifies REQ:localised-copy)

**Given** every reachable Sportius message and button,
**When** localisation validation runs,
**Then** each has a stable key and British-English fallback, and raw user values
are escaped before rendering.

## Open Questions

None.

---
*This document follows the https://specscore.md/feature-specification*
