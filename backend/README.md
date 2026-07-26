# Sportius backend

Sportius domain and application module. The Go module is rooted in `backend/`
at `github.com/sneat-co/sportius/backend`.

The stable public facade, DTOs, catalogues and role codes live in
`github.com/sneat-co/ext-sportius/backend`. This module implements that facade.
It does not import Telegram, HTTP, Firestore, `sneat-go-core`,
`sneat-core-modules`, or another extension implementation.

Generic Sneat work crosses `CorePort`: spaces, contacts, space membership,
contact and space linkages, invitations, and user briefs. Its production
adapter belongs in the host composition root. Sportius persistence crosses
`Repository`; `MemoryRepository` is a copy-on-write transactional adapter for
unit and bot-flow tests. `DalgoRepository` is the production adapter.

The dalgo adapter stores canonical data at:

- `/spaces/{personalSpaceID}/ext/sportius` for the user-controlled personal
  profile;
- `/spaces/{spaceID}/ext/sportius` for a team or club projection.

The former `/users/{uid}/ext/sportius` profile is a read-only migration source.
The first successful profile mutation writes the complete record to the
personal Space; the old record can be retired after rollout verification.

Exact-name discovery projections are stored under
`/ext/sportius/teams/{spaceID}` and `/ext/sportius/clubs/{spaceID}`.
They contain public brief fields and normalised equality-query keys only.
Ownership, user IDs, participants, guardians, staff, join requests and
idempotency metadata remain in canonical space extension records.
Sportius invitation role/status metadata is stored under
`/ext/sportius/invitations/{invitationID}`; the generic invitation and token
lifecycle remains behind `CorePort`.

Invitations are contact-first, matching Sneat's personal invitation model. An
inviter either selects an accessible space contact or supplies a display name
from which the host creates a non-member contact. That contact ID is passed to
the generic invitation. Generic acceptance claims the same contact and returns
its identity; Sportius then attaches the confirmed participant roles to that
contact without creating a duplicate.

Inspection and acceptance require the generic invitation's opaque claim token.
The token is passed only to the core adapter for proof validation. Neither it
nor the creation-only deep link is persisted by Sportius, and inspection
responses defensively strip the deep link. Authoritative generic status
resolution permits same-user recovery when generic acceptance succeeded but a
Sportius projection write failed.

## Packages

- `const4sportius`: stable extension ID.
- `models4sportius`: Sportius-owned projection records and defensive clones.
- `facade4sportius`: facade service, validation, repository and core ports,
  in-memory and dalgo repositories, view aggregation, and tests.

## Build & test

The module pins an immutable `ext-sportius/backend` pseudo-version and builds
standalone. The repository workspace may still be used while co-developing a
new contract revision.

```bash
cd backend
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

## Consistency boundary

Core mutation methods receive a stable request ID and must be idempotent. This
handles a retry where the generic Sneat write succeeded but the Sportius
projection write failed. Actor-scoped request keys and canonical payload
fingerprints reject accidental reuse for a different command. Profile updates
apply validated field patches to the latest record in the repository
transaction, so an unrelated concurrent update is not overwritten. The
generic name write and Sportius projection still span two stores; stable,
profile-versioned core request IDs make retries safe, but atomic cross-store
compare-and-swap remains host infrastructure rather than a Sportius concern.

Team and club names are deliberately non-unique;
stable space IDs are identity. The MVP service exposes zero-or-one club per
team while persisting the relationship through the generic space-linkage port.
Embedded team/club linkage fields are rebuildable caches: profile reads resolve
generic linkages and reconcile stale or missing projections. Linked-team roster
aggregation is restricted to generic club managers and also requires an
explicit Sportius club-manager roster policy. Ordinary club members never
receive cross-team contacts.

Generic Sneat space access is authoritative for every management mutation and
viewer capability, and for the teams and clubs shown on Sports home. Sportius
membership/owner projections are query aids only.
`My Teams` and `My Clubs` are derived views, not copied lists in a user,
personal-Space or family-Space settings document. Opening Sports from a family
Space is navigation and does not disclose every family member's memberships.
A future family Sports aggregate must use explicit generic linkages or another
equally explicit sharing grant; absence of such a link means private.
Public profile browsing never returns player, staff, guardian or club-member
contacts to an actor who is not a generic space member.

An approval-required join creates only a Sportius pending request: it does not
create a contact, participant, or generic space membership. An approval command
is intentionally deferred until the host's generic membership approval
contract is selected; Sportius must not invent a parallel approval authority.
Generic invitation revocation likewise remains host-owned. This implementation
does not hide private APIs behind Telegram handlers.
