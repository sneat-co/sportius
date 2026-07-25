# Sportius

Sportius is the Sneat extension behind [Sneat Club](https://sneat.club). It
owns persistent sporting organisations and profiles: personal sports, teams,
clubs, participant roles, rosters, guardians, join policy, invitations, and
team-to-club affiliation.

The first production surface is the existing Sneat Telegram bot. Telegram
presentation lives in [`sneat-bots`](https://github.com/sneat-co/sneat-bots);
it calls the facade implemented here through the stable
[`ext-sportius`](https://github.com/sneat-co/ext-sportius) contract.

## Domain boundary

- A team and a club are independent Sneat spaces.
- Generic space linkages are authoritative for team-to-club affiliation.
- ToGethered owns reusable places, venues, attendance, and activity intent.
  Sportius stores a narrow location hint and may reference a ToGethered spot.
- GameBoard owns games, participation in a game, and scoring.
- MatchUps owns competitions, leagues, cups, divisions, and fixtures.
- Sportus is a separate legacy wind-sports/equipment extension. Its staged
  retirement is tracked in Backstage; it is not renamed into Sportius.

## Repository layout

```text
backend/                # Go business facade, validation, repositories and ports
spec/                   # SpecScore feature definitions
apps/sportius-app/      # Reserved web composition root; not in Telegram MVP
libs/extensions/sportius/
  runtime/              # Reserved web runtime package
landings/               # Reserved product landing scaffold
```

The MVP is specified under
[`spec/features/sneat-club-telegram`](spec/features/sneat-club-telegram/README.md).
The cross-repository implementation plan and living initiative record are in
[`sneat-co/backstage`](https://github.com/sneat-co/backstage).

## Backend

`backend/` is the Go module `github.com/sneat-co/sportius/backend`. It
implements `github.com/sneat-co/ext-sportius/backend.Facade` behind repository
and host-platform ports. The Sneat host supplies adapters for spaces, contacts,
membership, invitations, and generic linkages.

```sh
cd backend
go test ./...
go vet ./...
```

Public DTOs, catalogues, and facade interfaces remain in `ext-sportius`; no
contract source is copied here.

## Web status

The generated Angular and landing packages are deliberately not a production
surface in this initiative. Public web onboarding, dashboards, profiles, SEO,
and short links are deferred. Automatic web deployment is disabled until a
separate web initiative replaces the generated scaffold with real Sportius UI.
