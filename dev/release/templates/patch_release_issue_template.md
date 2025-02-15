<!--
DO NOT COPY THIS ISSUE TEMPLATE MANUALLY. Use `pnpm run release tracking:issues` in the `sourcegraph/sourcegraph` repository.

Arguments:
- $MAJOR
- $MINOR
- $PATCH
- $RELEASE_DATE
- $ONE_WORKING_DAY_AFTER_RELEASE
-->

# $MAJOR.$MINOR.$PATCH patch release

This release is scheduled for **$RELEASE_DATE**.

> **Warning**: To get your commits in `main` included in this patch release, add the `backport-$MAJOR.$MINOR` to the PR to `main`.

## Setup

<!-- Keep in sync with release_issue_template's "Setup" section -->

- [ ] Ensure you have the latest version of the release tooling and configuration by checking out and updating `sourcegraph@main`.
- [ ] Ensure release configuration in [`dev/release/release-config.jsonc`](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/dev/release/release-config.jsonc) on `main` has version $MAJOR.$MINOR.$PATCH selected by using the command:

```shell
pnpm run release release:activate-release
```

## Prepare release

Ensure that all [patch request issues](https://github.com/sourcegraph/sourcegraph/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3Apatch-release-request) are accounted for, and have all relevant commits across relevant repositories listed above in the checklist.

For each of the following repositories you have made changes to, cherry-pick (see snippet below) and check off commits listed above.

- [ ] `sourcegraph/sourcegraph` ([`$MAJOR.$MINOR` release branch](https://github.com/sourcegraph/sourcegraph/tree/$MAJOR.$MINOR) [CI](https://buildkite.com/sourcegraph/sourcegraph/builds?branch=$MAJOR.$MINOR))
- [ ] `sourcegraph/deploy-sourcegraph` ([`$MAJOR.$MINOR` release branch](https://github.com/sourcegraph/deploy-sourcegraph/tree/$MAJOR.$MINOR) [CI](https://buildkite.com/sourcegraph/deploy-sourcegraph/builds?branch=$MAJOR.$MINOR))
- [ ] `sourcegraph/deploy-sourcegraph-docker` ([`$MAJOR.$MINOR` release branch](https://github.com/sourcegraph/deploy-sourcegraph-docker/tree/$MAJOR.$MINOR) [CI](https://buildkite.com/sourcegraph/deploy-sourcegraph-docker/builds?branch=$MAJOR.$MINOR))
- [ ] `sourcegraph/deploy-sourcegraph-helm` ([`$MAJOR.$MINOR` release branch](https://github.com/sourcegraph/deploy-sourcegraph-helm/tree/release/$MAJOR.$MINOR) [CI](https://buildkite.com/sourcegraph/deploy-sourcegraph-helm/builds?branch=release/$MAJOR.$MINOR))

```sh
git checkout $MAJOR.$MINOR
git pull
git cherry-pick <commit0> <commit1> ... # all relevant commits from the default branch
git push origin $MAJOR.$MINOR
```

- [ ] Ensure CI passes on all release branches listed above.

Create and test the first release candidate:

- [ ] Push a new release candidate tag. This command will automatically detect the appropriate release candidate number. This command can be executed as many times as required, and will increment the release candidate number for each subsequent build: :
  ```sh
  pnpm run release release:create-candidate
  ```

> **Note**: Ensure that you've pulled both main and release branches before running this command.

- [ ] Ensure that the following Buildkite pipelines all pass for the `v$MAJOR.$MINOR.$PATCH-rc.1` tag:
  - [ ] [Sourcegraph pipeline](https://buildkite.com/sourcegraph/sourcegraph/builds?branch=v$MAJOR.$MINOR.$PATCH-rc.1)
- [ ] File any failures and regressions in the pipelines as `release-blocker` issues and assign the appropriate teams.

**Note**: You will need to re-check the above pipelines for any subsequent release candidates. You can see the Buildkite logs by tweaking the "branch" query parameter in the URLs to point to the desired release candidate. In general, the URL scheme looks like the following (replacing `N` in the URL): `https://buildkite.com/sourcegraph/sourcegraph/builds?branch=v$MAJOR.$MINOR.$PATCH-rc.N`

Once there is a release candidate available:

- [ ] Create a [Security release approval](https://github.com/sourcegraph/sourcegraph/issues/new/choose#:~:text=Security%20release%20approval) issue and post a message in the [#ask-security](https://sourcegraph.slack.com/archives/C1JH2BEHZ) channel tagging `@security-support`.

## Stage release

<!-- Keep in sync with release_issue_template's "Stage release" section -->

- [ ] Verify the **$MAJOR.$MINOR.$PATCH** section of [CHANGELOG](https://github.com/sourcegraph/sourcegraph/blob/main/CHANGELOG.md) on the `main` is accurate.
- [ ] Ensure security has approved the [Security release approval](https://github.com/sourcegraph/sourcegraph/issues?q=label%3Arelease-blocker+Security+approval+is%3Aopen) issue you created.
- [ ] Promote a release candidate to the final release build. You will need to provide the tag of the release candidate which you would like to promote as an argument. To get a list of available release candidates, you can use:
  ```shell
  pnpm run release release:check-candidate
  ```
  To promote the candidate, use the command:
  ```sh
  pnpm run release release:promote-candidate <tag>
  ```
- [ ] Ensure that the pipeline for the `v$MAJOR.$MINOR.$PATCH` tag has passed: [Sourcegraph pipeline](https://buildkite.com/sourcegraph/sourcegraph/builds?branch=v$MAJOR.$MINOR.$PATCH)
- [ ] Wait for the `v$MAJOR.$MINOR.$PATCH` release Docker images to be available in [Docker Hub](https://hub.docker.com/r/sourcegraph/server/tags)
- [ ] Open PRs that publish the new release and address any action items required to finalize draft PRs (track PR status via the [generated release batch change](https://sourcegraph.sourcegraph.com/organizations/sourcegraph/batch-changes)):
  ```sh
  pnpm run release release:stage
  ```

## Finalize release

<!-- Keep in sync with release_issue_template's "Finalize release" section, except no blog post -->

- [ ] From the [release batch change](https:/sourcegraph.sourcegraph.com/organizations/sourcegraph/batch-changes), merge the release-publishing PRs created previously. Note: some PRs require certain actions performed before merging.
- [ ] **After all the PRs are merged**, perform following checks/actions
  - For [deploy-sourcegraph](https://github.com/sourcegraph/deploy-sourcegraph)
    - [ ] Ensure the [release tag](https://github.com/sourcegraph/deploy-sourcegraph/tags) has been created
  - For [deploy-sourcegraph-docker](https://github.com/sourcegraph/deploy-sourcegraph-docker)
    - [ ] Ensure the [release tag](https://github.com/sourcegraph/deploy-sourcegraph-docker/tags) has been created
  - For [deploy-sourcegraph-helm](https://github.com/sourcegraph/deploy-sourcegraph-helm), also:
    - [ ] Update the [changelog](https://github.com/sourcegraph/deploy-sourcegraph-helm/blob/main/charts/sourcegraph/CHANGELOG.md) to include changes from the patch
    - [ ] Cherry-pick the release-publishing PR from the release branch into `main`
- [ ] Finalize and announce that the release is live:
  ```sh
  pnpm run release release:announce
  ```

## Post-release

- [ ] Close the release:

```shell
  pnpm run release release:close
```

- [ ] Open a PR to update [`dev/release/release-config.jsonc`](https://github.com/sourcegraph/sourcegraph/edit/main/dev/release/release-config.jsonc) after the auto-generated changes above if any.

> **Note:** If another patch release is requested after the release, ask that a [patch request issue](https://github.com/sourcegraph/sourcegraph/issues/new?assignees=&labels=team%2Fdistribution&template=request_patch_release.md) be filled out and approved first.
