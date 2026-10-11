# Release flow

Release tags always use `vX.Y.Z`. A request for version `0.0.2` means tag
`v0.0.2`. Never create an unprefixed release tag. The scripts currently accept
both forms, so their validation is not a substitute for this convention.

## Read and reconcile

Read `AGENTS.md`, all available project context, and the current handoff before
exploring code. Compare the recorded branch/head with `git status --short`,
`git diff --cached`, `git log -1`, and the remote branch/tag references. Treat
older reports as historical evidence. If local `.docs` files are unavailable,
read `docs/setup.md`, `docs/protocol.md`, `docs/install.md`, and this file.

Identify the exact requested changes and the canonical tag before any mutation.
Keep unrelated working changes outside every commit and release. A dirty local
updater does not become part of a release merely because its tests passed.

## Prepare and approve commits

Search and reuse the existing implementation. Run checks appropriate to the
change and record commands, results, skipped checks, and runtime limits.
Prepare one focused patch per independent task. For each patch, present its
exact files, hunks, and English commit message, then obtain explicit approval.
Approval of one patch does not cover later modifications or additional commits.
Once the approved commit exists, its approval is consumed; do not reuse a
completed checklist entry as authorization for another commit.
Use an existing approval when it covers the unchanged exact patch; do not ask
again for already authorized actions. Tickets are waived for this repository.

Apply only the approved hunks to the index. Check the staged diff, commit, and
push the requested branch. Verify its remote SHA. Do not include other pending
changes, add coauthor attribution, or reset user work.

## Validate the committed release source

Use a clean detached worktree of the committed SHA that will receive the tag.
Do not package the main working directory when it has unrelated changes.
For the approved version, for example `v0.0.2`, run in that clean worktree:

```sh
make test check
bash -n install.sh deploy/install.sh deploy/update.sh deploy/check.sh scripts/release.sh scripts/install_test.sh
bash scripts/release.sh v0.0.2
bash scripts/install_test.sh dist/v0.0.2
(cd dist/v0.0.2 && sha256sum --check checksums.txt)
```

Replace the example tag consistently with the authorized version. Verify that
both archives contain that exact `VERSION`. Bootstrap tests use download and
sudo fixtures; they do not prove publication or install on the host. Cross-builds
prove package generation, not ARM64 runtime support. Run privileged network/storage
tests only in isolated environments. Host installation requires explicit authorization.

## Tag and publish

An explicit instruction to release the version authorizes its canonical tag and
publication. Reuse that authorization; do not ask again merely to push the tag.
Check that the local and remote tag are absent. If either already exists,
inspect its target and release before proceeding; never force-move a released tag.

Create an annotated tag on the verified committed SHA and push that exact ref:

```sh
git tag -a v0.0.2 VERIFIED_SHA -m 'Release v0.0.2'
git push origin refs/tags/v0.0.2
```

Replace `VERIFIED_SHA` with the verified commit and the tag with the authorized
version. The Release workflow runs tests, vet, Bash checks, Linux amd64/arm64
packaging, bootstrap fixtures, and GitHub publication. Identify the run by both
tag and SHA. Wait for its terminal result; a pushed tag is not a completed release.
Do not separately publish duplicate assets while the workflow is running.

If a newly created tag is wrong, stop its workflow and wait until it is terminal
before cleanup, so it cannot publish after deletion. Inspect whether a release
was created. For a correction authorized by the user, remove only the mistaken
release/tag created for that operation; preserve older releases. Publish under
the corrected canonical tag. Do not treat this recovery flow as permission to
delete or retag unrelated releases.

## Verify and hand off

Verify the workflow conclusion, remote tag target, published tag, and release
flags. Require both `saveme_<version>_linux_amd64.tar.gz` and
`saveme_<version>_linux_arm64.tar.gz` plus `checksums.txt`, where asset filenames
omit the leading `v`. Download the actual GitHub assets, check SHA-256, and verify
both `VERSION` files. Test real-network latest/pinned bootstrap using a sudo
fixture if host installation was not authorized. Label fixture evidence accurately.

Report the release link, tag, source SHA, checks performed, and material limits.
Update the same task context, `.docs/flow.md`, `.docs/trace.md`, and the current
handoff checklist; run `python3 .docs/check.py` when available. Record remaining
dirty paths separately. Remove obsolete pending checklist entries or mark them
historical, so another agent does not repeat completed approvals or releases.

## Published source and pending work

`v0.0.2` targets commit `77e097ce208d13c399033f83d7b540efdaa34bf3`. It includes
short durations, adaptive UI, explicit disabled actions, centered heavy countdown
digits, readable confirmation cells, daily-window editing, default Vim navigation,
and `q` navigation outside input screens.

The active-lock updater was not included. In this published source,
`cmd/saved/update.go` still rejects updates whenever a stored lock exists. Local
snapshot/rollback updater changes require their own review, approval, commit,
tests, and release. Revalidate this distinction against the requested tag and
current Git state before making deployment claims.
