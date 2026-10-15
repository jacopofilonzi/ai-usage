# AGENTS.md

Guidelines for AI coding agents (and people) working in this repository.
What lives where and how the pieces fit together is in [STRUCTURE.md](STRUCTURE.md): read it first,
instead of re-reading the whole repo.

These guidelines come from a shared template: https://github.com/jacopofilonzi/agents-guideline.
See "Updating these guidelines" below to align them with the latest version.

<!--
Template: "Workflow", "Code" and "Testing" are my default style; keep them unless the project needs
something different. Fill in or delete every <PLACEHOLDER> section, then delete these comments.
-->

## What it is

<PROJECT>: <one sentence>. Stack: <language + framework + versions>. Targets: <Linux server / Windows, macOS, Linux / …>.

## Workflow

- **Non-trivial changes start with a plan.** For a feature, refactor, new dependency or anything touching
  several files: read what you need, then present the plan (what changes, which files, trade-offs, open
  questions) and wait for my approval before editing. Small, obvious fixes can go straight in. If the plan
  changes significantly along the way, stop and ask again.
- **Parallelize independent modules.** When the work needs several modules and you judge it feasible, first
  define and commit the shared constructs yourself (types, interfaces, API contracts, config, shared
  helpers), then launch several agents in parallel, one per module, each with a clear scope and the files
  it owns. Afterwards, review and integrate their output and run `make check` on the whole. If the modules
  are too coupled to split cleanly, do the work sequentially instead.
- When something is genuinely my decision (UX, user-visible naming, data formats, breaking changes), ask
  instead of guessing. Otherwise pick the sensible default and say what you picked.
- Run commands from the repo root through the Makefile (`make help` lists them). A new recurring command
  goes into the Makefile too.
- **While developing, use the framework's dev tools** (`make dev`: dev server, hot reload, `wails3 dev`, watch
  mode…) instead of building and launching the app after every change. Build and run the real artifact
  (binary, installer, Docker image) only when I say the dev tools don't pick up a specific change, and only
  for that check; then go back to the dev server.
- **Always run `make check` before declaring work done.** It must pass with zero errors and zero warnings.
  If you couldn't run something (Docker, a live API, a platform you're not on), say so instead of claiming it works.
- Makefile recipes must work in both `sh` and `cmd.exe` (GNU make on Windows falls back to `cmd.exe`): only
  `cd dir && command`, no inline `VAR=x cmd`, no `rm`/`cp`/`mkdir -p`; use `node -e` (or the project's own
  language) for file operations.
- Every new env var goes in the config module **and** is documented in `.env.example` (and in
  `docker-compose.yml` if relevant for deployment). Never commit `.env` or secrets.

### Commits

- Small, step-by-step commits as the work progresses, never a single commit at the end. Conventional
  prefixes: `feat:`, `fix:`, `refactor:`, `chore:`, `docs:`, `test:`.
- **Exception, experiments:** while we're trying things out (styling, layouts, alternative approaches),
  don't commit. Commit only once I've picked the version to keep.
- Mockups and design alternatives are never committed: they only serve to choose; the chosen one lives in the code.
- Push, tag and release only when I ask.

### Keep the agent docs in sync, in the same change

`STRUCTURE.md` and this file exist so agents don't have to re-read the whole repo. When you add, move,
rename or delete a file or module, or change an API endpoint, env var, invariant, data format or workflow
command, update `STRUCTURE.md` / `AGENTS.md` (and `README.md` for user-facing changes) in the same commit
or right after. Before declaring work done, check that every path, name and value they mention for the
areas you touched still matches the code.

### Updating these guidelines

When I ask you to check for new guidelines, fetch the latest template from
https://github.com/jacopofilonzi/agents-guideline (raw files:
`https://raw.githubusercontent.com/jacopofilonzi/agents-guideline/main/{AGENTS,CLAUDE,STRUCTURE}.md`)
and align this project's copies with it:

- Bring in new and changed generic rules (workflow, commits, code, testing…) and drop the ones removed
  upstream, without asking.
- Keep everything project-specific: filled-in placeholders, invariants, recipes, pitfalls, the whole
  content of `STRUCTURE.md`, and rules this project added or deliberately changed.
- Ask only when an upstream change conflicts with the project's current style or guidelines (e.g. it
  contradicts a project-specific rule or a convention the code follows); list each conflict with both versions.
- Then summarize what changed and commit it as `docs: sync agent guidelines`.

## Invariants — don't break these

<!-- Rules that aren't obvious from the code and would cause real damage if broken. Say WHY when it isn't obvious. Examples: -->
- <Secrets/tokens are never logged or returned; they live in a redacted `Secret` type.>
- <Optional services (cache, Redis…) never fail a request: errors degrade to a miss; the server starts without them.>
- <Bump `<CACHE_KEY_VERSION>` whenever a cached payload shape changes.>
- <When backend API models change, update the frontend types in the same change.>
- <Durable data (SQLite, user files) is not cache: formats and IDs stay stable and backward compatible.>

## Code

- Everything is written in English (code, comments, commits, docs), except the language packs.
- Keep modules small and focused: one folder per area; split a file when it grows several responsibilities.
- Comments explain *why*, not *what*. Put the design of a non-trivial module in its top-level doc comment
  (upstream formats, invariants, data layout).
- Never ignore errors: wrap them with context. No `unwrap()`/`expect()`/unchecked casts on external data
  (network, files, user input).
- Use the project's logger, never `println!`/`console.log`/`fmt.Println` in core logic.
- OS-specific code is isolated in one place (`platform/`, `_windows`/`_linux`/`_darwin` files); no
  branching on the OS elsewhere. Build paths with the language's path join, never by concatenating `/`.
- Reuse shared clients and connections (HTTP client, DB pool) instead of creating them per request.
- **Dependencies are recent majors whose APIs differ from most online examples:** check the docs for the
  installed version before using an API. Ask before adding a new dependency.
  <List pinned or unusual versions here, e.g. "axum 0.8 (`{param}` path syntax)", "TypeScript pinned to 6.x: don't upgrade".>
- Deleting user data means moving it to the trash / soft delete, unless agreed otherwise.

### UI

<!-- Delete if the project has no UI. -->
- Every UI string goes through i18n, with both the English and the Italian key (`<DEFAULT_LANG>` is the
  default); the type check must catch a missing key.
- Svelte 5 runes only (`$state`, `$derived`, `$effect`, `$props`): no legacy stores or `export let`.
- Tailwind 4: reusable classes are `@utility` in the global CSS.
- Never hard-code base paths or the API prefix: build URLs from the shared base constant.

## Testing

- Unit tests live next to the code. Test parsing against fixtures copied from real responses, never live
  requests; live checks, if any, are a separate opt-in command (e.g. `make smoke`).
- End-to-end checks: drive the built app with a headless browser (Playwright with the system Chrome/Edge;
  install `playwright-core` in a scratch dir, not in the repo), unless the project has its own e2e setup.

## Versions and releases

<!-- Delete if the project isn't versioned. -->
- Versions are `x.y.z`: `x` major (only when I decide), `y` feature, `z` fix. A feature bumps `y` and resets `z`.
- The version lives in <FILES>: change it only with `<scripts/set-version.sh x.y.z>`, which updates them all.
- Tags and releases only when I ask.

## <Recipe, e.g. "Adding a <thing>">

<!-- Step-by-step for extension points that will be repeated (new provider, new step type, new integration…). -->
1. <Create `<path>` implementing `<Trait>`.>
2. <Register it in `<file>`.>
3. <Add tests with fixtures.>

## Known pitfalls

- The main dev machine is Windows 11; shells are PowerShell and Git Bash. Commands and scripts must work there.
- **Git Bash rewrites env values that look like paths** (`BASE_PATH=/x` becomes `C:/Program Files/Git/x`):
  prefix the command with `MSYS2_ENV_CONV_EXCL='*'`, or set the value in `.env`.
- A running `.exe` locks its binary on Windows: stop it before rebuilding.
- <Project-specific gotchas: framework quirks, things that look like bugs but are intentional.>
