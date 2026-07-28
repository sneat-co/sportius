# AI agent guidance

This is the Sportius implementation repository. The public extension contract
lives in `sneat-co/ext-sportius`; Telegram presentation lives in
`sneat-co/sneat-bots`.

Keep these boundaries:

- Business decisions and orchestration belong in `backend/`, behind the
  `ext-sportius` facade.
- Bot handlers render and navigate; they must not persist Sportius data
  directly.
- Teams and clubs are independent Sneat spaces linked through the generic
  linkage system.
- Do not import another extension implementation. Express host or
  cross-extension needs as ports.
- ToGethered owns places and attendance, GameBoard owns games/scoring, and
  MatchUps owns competitions.
- Public web UI is out of scope for the Telegram MVP. Do not extend the
  generated web scaffold unless a web initiative explicitly requests it.

Backend and extension standards:

- https://github.com/sneat-co/sneat-specs/blob/main/standards/extension-backend-architecture.md
- https://github.com/sneat-co/sneat-libs/blob/main/docs/extension-standards/README.md
