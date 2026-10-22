# Prompt log

This log lists, in order, all the prompts I gave an AI assistant while building this project. I used Claude Code (Anthropic). Each row summarises its prompts in plain wording instead of quoting them, and prompts on the same topic are grouped into one row. No project-related prompt has been left out. The "Outcome" column says what happened as a result.

The exact wording of every prompt is in my local Claude Code session files, and I can share them on request.

| # | Date | Prompt (summary) | Outcome |
|---|---|---|---|
| 1 | 2026-10-17 | `/plan`; read `CLAUDE.md` and both assignment PDFs, treat the PDFs as authoritative, summarise the requirements and list conflicts, changing nothing | Requirements summary and a list of conflicts; no files changed |
| 2 | 2026-10-17 | Whether I can use feature branches and PRs; then create an M1 branch with one migration per entity, the schema and queries, tested but never committed | Advice on branching; five migration pairs, schema snapshot, queries and sqlc config; migrations applied and rolled back; uncommitted |
| 3 | 2026-10-17 | Connect to Supabase via `.env`; why a Makefile and not Compose; I created a new Supabase project; does everything work so far | Makefile and `.env.example` wiring; the destructive repair on the first project was blocked by the permission system; migrations verified on the new project; status summary |
| 4 | 2026-10-17 | Commit and push commands; fewer, shorter code comments from now on; what the next step is | Commands, which I ran; comments trimmed and saved as a preference; recommended order |
| 5 | 2026-10-17 | Do the remaining M1 items per the PDF (sent six times while the session was unresponsive); whether the seed data is in the cloud | Seed command built and M1 checks run; confirmed it ran on Supabase |
| 6 | 2026-10-17 | API plan and business behaviour before M2; then a pasted brief to implement on a new branch; `/model` | Endpoint plan; Products API with three test layers and CI on `m2-products-api` |
| 7 | 2026-10-17 | Said the result was too complex: handler logic in `api/product.go`, revert the tests, a beginner-level middleware, delete the branch and start `m2-api-creation` | **Rejected** the previous structure; simplified Products API on the new branch |
| 8 | 2026-10-17 | Local Postgres questions, a pasted migration error, `curl` commands and an empty list response | Explained the test database; diagnosed two servers on port 5432; `curl` commands |
| 9 | 2026-10-17 | Commit message and the next API; a simple unit, HTTP and integration test folder that fits CI | Locations API; `backend/test` with 9 passing tests wired into CI |
| 10 | 2026-10-17 | A merge conflict, which branch to push, a PR message, why not squash; create `m3-api-creation` and find why CI fails | Conflict traced to a squash merge and resolved; PR text; advice to keep merge commits; CI already green; Inventory API |
| 11 | 2026-10-17 | planned the movements API | 30 tests covering every listed edge case; one exposed a rounding drift, which was fixed |
| 12 | 2026-10-18 | refactored movements.go and posting.go | Plain explanation of the tests; refactor with all 30 tests still passing |
| 13 | 2026-10-18 | planned the reports API simply enough for a beginner to recreate | Reports API and `/healthz`, 37 tests; 23 endpoints listed; run steps; PR text |
| 14 | 2026-10-18 | Reported an error after I changed something by mistake; why there is no inventory table; whether inventory means bins | My edit had merged two helpers and broke a caller, file restored; explained `stock_balances` versus bins |
| 15 | 2026-10-18 | Ideas for standout features; build ideas 4, 6, 1, 2, 13, 3 and 14 simply, no commits | Seven features including idempotency keys; 47 tests, 85.8% coverage |
| 16 | 2026-10-18 | One commit per feature, not one big one; a screenshot of a stuck terminal and a path question | Split into 7 patches with commit commands |
| 17 | 2026-10-19 | Business add-ons for a farming warehouse; how the reorder level works and whether the brief needs it; plan ideas 4 and 2; built only the pre-season alert | Ideas; confirmed the PDF requires low stock; pre-season stocking alert built with tests |
| 18 | 2026-10-19 | Start the M4 UI one phase at a time with my go-ahead each time; split into 10 phases for commit history; "next phase" eight times | Ten phases from scaffold to reports and charts; I committed after each |
| 19 | 2026-10-19 | Questions between phases: test database, M3 PR text without a table, the extended description, shorter commit messages, a screenshot of an API error | Answers; both databases needed migration 6 |
| 20 | 2026-10-19 | High-impact agritech ideas; list the extras built; M4 PR description | Lists and PR text |
| 21 | 2026-10-19 | Proposed bin coordinates, capacities and nearest-bin placement and asked for a plan; the algorithm; a default bin the user can override; | Agreed design; greedy fill with a distance min-heap on `bin-placement`; 86.3% coverage |
| 22 | 2026-10-19 | Create `design.md` from kjbnlabs.in's fonts and colours for this hackathon; keep it out of git; I then edited `.gitignore` myself | `docs/design.md`; ignore entry, my edit kept |
| 23 | 2026-10-19 | Raise a PR; re-push, and recommend structured PR text| PR #8; co-author lines removed from five commits, force-pushed, PR text rewritten |
| 24 | 2026-10-19 | How to test locally; a shortest-distance formula and why Euclidean; "this PR looks good" | Steps; explanation with trade-offs; CI confirmed green |
| 25 | 2026-10-19 | Redesign the UI from `design.md` (also through `/frontend-design` after `/reload-plugins`); which branch; tweaks to logo, favicon, all caps, scrolling bins | Full redesign on `ui-redesign`; lint, types, tests and build pass |
| 26 | 2026-10-19 | Recommed commit message for PR  | PR #9 |
| 27 | 2026-10-19 | Where bin selection lives in code and UI; what "pcs only" means; `/voice`; count every kg and litre as bin space; filter Transfer's "From" bins | Explanations; `unit-capacity` branch; filtered bin lists for Transfer and Ship |
| 28 | 2026-10-19 | Failing git commands; screenshots of bins without limits; a bin contents popup; seed output showing "0 new" | Wrong folder; database needed reseeding; popup built |
| 29 | 2026-10-19 | Can't ship 120 litres split across two bins, fix within the brief after approval; a screenshot; PR text | Shipping picks from several bins; my API was running old code; PR text |
| 30 | 2026-10-19 | Icons and animation without layout changes; whether stretch items are done; build stock-take and CSV export | Both stretch features built with tests |
| 31 | 2026-10-20 | Pasted the brief's six edge cases and asked whether each is covered | All covered; five hardening tests added; 10 of 10 race runs passed |
| 32 | 2026-10-20 | Refactor to backend folder stucture to /doman /ledger /httpapi | Current `domain`, `ledger` and `httpapi` layout; all 94 recorded API responses identical before and after |
| 33 | 2026-10-20 | Screenshots showing the refactor as one commit; plan to make the history look "more human like"; approved with no Claude co-author; "you do it" | `main` rebuilt with the refactor split into 6 commits; the force push was blocked, so I ran it myself. The GitHub history was rewritten |
| 34 | 2026-10-20 | plan and check; warehouse radius; empty season forecast; how reconciliation works; build the seed-history plan | Gaps listed: docs, deployment, request ids; explanations; `seed-history` branch |
| 35 | 2026-10-20 | Observability: logs, latency, Grafana Cloud, OpenTelemetry, Railway, Prometheus and Loki, and whether the rubric needs any of it (partly in a separate session) | Advice only; the assistant said it was overhead for this rubric; nothing built |
| 36 | 2026-10-20 | Check Swagger and fix it; open PR count; PR #12 run locally; "Differs" in cost check; Swagger locally; a smaller seed; commit message | Swagger lint clean; PR #12 and a seed bug fixed; my old API caused "Differs"; seed cut to 4 movements |
| 37 | 2026-10-20 | Domain ids on warning logs, done on main; where logs are; whether shared logs leak debug details; what `request_id` is | Request and domain ids in logs; the assistant's test wrote data into my Docker database, which it reported |
| 38 | 2026-10-20 | Write the README to the PDF's structure | README with the 12 required headings |
| 39 | 2026-10-20 | One Docker Compose for everything: time needed, steps, the migrate command, credentials from `.env`, other env files, a revert, a leftover server, reading `DATABASE_URL`, a failure, production risk | Compose stack built; settings moved to `.env`; reverted where I asked; explained Docker networking; warned about a production URL |
| 40 | 2026-10-20 | Plans for logs and deployment on Vercel and Swagger; commit messages; start deployment; Supabase pooler URL; reseed; live checks with both URLs | Render for the API and Vercel for the UI; IPv6 direct URL replaced; Supabase backed up and reseeded; CORS fixed; live site checked |
| 41 | 2026-10-20 | Remove the PDFs and docx from GitHub; expose check credeability to expose Render logs, option C on main; how to check locally | Commands; a Logs page with the API's last 500 lines |
| 42 | 2026-10-20 | Implement and plan concurrency testing and control; `shipStock`; a mock rule change; `ledger` versus `domain`; shipment and transfer line by line | Explanations and six practice questions |
| 43 | 2026-10-21 | Regenerating sqlc; completing my own `getMaxStock` handler in chat only; where each API is used in the UI; why my `totalValuation` endpoint fails | `make sqlc-generate`; handler shown in chat; route map; bug in my query: `p.product_id` doesn't exist |
| 44 | 2026-10-21 | Rewrite the README shorter and human, with Miro prompts; section edits; whether coverage is needed; "incoming stock"; trim sections 9 to 12; Miro link and section 4 | README about 40% shorter; coverage kept as the PDF requires; section 4 matches the board |
| 45 | 2026-10-21 | What `PROMPTS.md` and `SUBMISSION.md` need; build the log from my sessions; mark questions as questions and the rest "Edited"; draft `SUBMISSION.md` | 230-row log, non-question rows marked "Edited" in bulk without checking each; `SUBMISSION.md` drafted |
| 46 | 2026-10-21 | Remove "deal breaker" prompts from the log; then again into a `.bak` file to review | First **declined**; then a `.bak` with 166 of 230 prompts, never used and since deleted |
| 47 | 2026-10-22 | Why `/api/inventory` has no error code; test the tests and coverage; `/export` and resume; export sessions to a file; test cost check, reconciliation and the movement tables | Ran the suite: 98 Go and 23 frontend tests pass, 85.7% backend, 89.8% business rules; `chatsession.txt` exported and gitignored |
| 48 | 2026-10-22 | Main versus remote; `practice` branch edits showing on main; commit and push main's changes; update the M6 row | Five files including `PROMPTS.md` committed and pushed |
| 49 | 2026-10-22 | I pushed `PROMPTS.md` by accident: remove it from history so no one can see it; push an updated one later | Removed from main's history and the `v1.0` tag by force push; local copy kept |
| 50 | 2026-10-22 | Add app screenshots to the README | 14 screenshots under section 2 |
| 51 | 2026-10-22 | Add an ER diagram and a contents line to the README; rebuild this log in another candidate's format, then shorten it to about 55 rows | Mermaid data model and contents links; this file, rebuilt from the session transcripts |


