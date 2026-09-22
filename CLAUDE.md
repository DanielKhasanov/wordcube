# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A full-stack word-square ("wordcube") solver: a Go backend that exhaustively finds every 5x5 grid where each row and column is a valid word, plus a React frontend that lets a player fill in a partial grid and see matching complete solutions streamed back live.

## Commands

### Backend (`server/`, its own Go module)
- Build: `cd server && go build ./...`
- Test: `cd server && go test ./...`
- Run a single test: `cd server && go test ./psolve/ -run TestBuildPrefixSkipList -v`
- Run the API server: `cd server && go run app/server.go` (serves on `:1323`; expects precomputed solutions at `server/app/data/solutions.textpb`)
- Regenerate the solutions file from a word list: `cd server && go run ./cli -mode find_solutions -word_list app/data/all_words_5.txt -output_dir app/data -num_partitions <N>` (parallelized backtracking search; `-num_partitions` should scale with available CPUs)
- Search precomputed solutions against a saved partial board: `cd server && go run ./cli -mode search -game_state <path> -solutions_file app/data/solutions.textpb -output_dir <dir>`

### Frontend (`site/`, npm/Vite/React/TypeScript, shadcn/ui components)
- Install: `cd site && npm install`
- Dev server: `cd site && npm run dev` (point `VITE_API_BASE_URL` at the running backend, e.g. via `site/.env.local`; see `site/.env.example`)
- Type-check + production build: `cd site && npm run build`
- Lint: `cd site && npm run lint`
- Test: `cd site && npm test` (Vitest + Testing Library; currently covers the `useSolutions` networking hook in `src/hooks/useSolutions.test.ts`)

### Whole-app
- One-time local setup: `./dev-setup.sh`
- Docker build + run (multi-stage: builds frontend, builds Go binary, serves both from one Alpine image on `:1323`): `./build-and-run.sh` or `docker-compose up --build`
- See `DEPLOYMENT.md` for details on the Docker stages and environment variable behavior across dev/prod.

## Architecture

### Solving approach: precompute once, query fast
The expensive part (finding all valid word squares for a given word length) is done **offline**, not per-request:

1. `server/cli -mode find_solutions` builds a `psolve.State` from a word list (`server/psolve/psolve.go`), partitions the search space into `N` independent sub-ranges via `State.Partition` (using big.Int arithmetic over word-index digit sequences so partitions can be resumed/distributed via `Checkpoint` protobufs — see `server/proto/v1/checkpoint.proto`), and runs them concurrently with `server/parallel.Group`. Each partition does a backtracking `Next()`/`addRow()` search: it walks one word per row and, for every row added, advances a per-column `trie.Trie` walker (`server/trie/trie.go`, `WalkIn`/`WalkOut`) to prune any partial grid whose columns can no longer form a dictionary word, plus a prefix-skip-list optimization to jump past whole runs of candidate words that share a doomed prefix.
2. All resulting solutions are serialized once to a `SolutionSet` protobuf text file (`solutions.textpb`, read/written via `server/reader` and `google.golang.org/protobuf/encoding/prototext`).
3. At request time, `server/app/server.go` loads that file into a `psolve.Searcher` (`server/psolve/search.go`), which builds an inverted index (row/col + letter -> set of solution IDs) once at startup. A query board is turned into a small set of keys, the corresponding solution-ID sets are streamed and intersected (`noCopyIntersection`), and only candidates that survive are double-checked exactly (`possibleMatch`) — this makes interactive per-keystroke queries fast without re-running the solver.

### API surface (`server/app/server.go`, Echo framework)
- `POST /solutions` — body `{"board": [][]string}` (5x5, `"_"`/`""` = empty cell); streams matching complete solutions as newline-delimited JSON as they're found, so the client can render results incrementally. CORS is wide open (`Access-Control-Allow-Origin: *`).
- `GET /status` — readiness + total precomputed solution count; the frontend should treat `/solutions` as unavailable until this reports ready, since solution loading/indexing happens in a goroutine at startup.
- Static file serving of the built frontend from `/` (i.e. `static/`, populated by the Docker build).

### Frontend (`site/src`)
`useGameState` owns the 5x5 grid/selection/typing state (click a cell to select row/column, type to fill, Tab to toggle direction, arrows to navigate — see `App.tsx`). `useSolutions` re-POSTs the current board to `/solutions` on every board change and incrementally parses the streamed NDJSON response into state, which `SolutionsSidebar` renders. `GameGrid`/`GameCell` render the board itself; `components/ui/*` are shadcn/ui primitives, not hand-rolled.

### Things to know before changing solving logic
- `psolve.State`'s trie-based pruning depends on `verticalTries[i]` being fed one letter per row in strict top-to-bottom order via `WalkIn`; don't call it out of order or you'll corrupt the walk position (there's no bounds/order checking).
- `Indexes.Int`/`FromInt` encode per-row word indices as digits of a single big.Int in base `len(validWords)` purely so partition boundaries can be represented as one comparable number — if you change how partitioning divides the space, these need to stay consistent with `Partition`.
- `FromCheckpoint` reconstructs state by replaying `addRow` for each row in the checkpoint; it does not check `addRow`'s return value, so replaying a checkpoint whose rows don't actually form a consistent partial word square will silently leave `currentRow` wrong rather than erroring (see the failing `TestCheckpoints/Full_Square` test in `server/psolve/psolve_test.go`). This doesn't affect the real `find_solutions`/`Partition` path since partitions are always built from a fresh, valid search — but be aware of it if you add code that resumes from arbitrary/hand-written checkpoints.

## Repo history note

This repo's git history is unusual: the currently-tracked branch's early history and `master`'s early history were committed as separate, unrelated git objects with matching messages (a Codespaces artifact), and `master` continued on with three additional commits (adding `server/`, `site/`, and the Docker deployment) that other branches never received. `git merge-base` between them returns nothing — there is no common ancestor to merge or rebase against. Treat `master` as canonical; don't attempt to merge other branches into it without checking what's actually different first.
