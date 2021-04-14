# Pull Request Process

This document explains the process and best practices for submitting a PR to the [OpenFunction project](https://github.com/OpenFunction/OpenFunction). It should serve as a reference for all contributors, and be useful especially to new and infrequent submitters.

- [Before You Submit a PR](#before-you-submit-a-pr)
  - [Run Local Verifications](#run-local-verifications)
- [The PR Submit Process](#the-pr-submit-process)
  - [Write Release Notes if Needed](#write-release-notes-if-needed)
  - [The Testing and Merge Workflow](#the-testing-and-merge-workflow)
  - [Marking Unfinished Pull Requests](#marking-unfinished-pull-requests)
  - [Comment Commands Reference](#comment-commands-reference)
  - [Automation](#automation)
  - [How the e2e Tests Work](#how-the-e2e-tests-work)

## Before You Submit a PR

This guide is for contributors who already have a PR to submit. If you're looking for information on setting up your developer environment and writing code to contribute to OpenFunction, see the [development guide](development-workflow.md).

**Make sure your PR adheres to our best practices. These include following project conventions, making small PRs, and commenting thoroughly. Please read the more detailed section on [Best Practices for Faster Reviews](#best-practices-for-faster-reviews) at the end of this document.**

### Run Local Verifications

You can run these local verifications before you submit your PR to predict the pass or fail of continuous integration.

## The PR Submit Process

Merging a PR requires the following steps to be completed before the PR is merged automatically. For details about each step, see the [The Testing and Merge Workflow](#the-testing-and-merge-workflow) section below.

- Make the PR
- Release notes - do one of the following:
  - Add notes in the release notes block, or
  - Update the release note label
- Pass all tests
- Get a `/lgtm` from a reviewer
- Get approval from an owner

If your PR meets all of the steps above, it will enter the submit queue to be merged. When it is next in line to be merged, the tests will run a second time. If all tests pass, the PR will be merged automatically.

### Write Release Notes if Needed

Release notes are required for any PR with user-visible changes, such as bug-fixes, feature additions, and output format changes.

If you don't add release notes in the PR template, the `do-not-merge/release-note-label-needed` label is added to your PR automatically after you create it. There are a few ways to update it.

To add a release-note section to the PR description:

For PRs with a release note:

    ```release-note
    Your release note here
    ```

For PRs that require additional action from users switching to the new release, include the string "action required" (case insensitive) in the release note:

    ```release-note
    action required: your release note here
    ```

For PRs that don't need to be mentioned at release time, just write "NONE" (case insensitive):

    ```release-note
    NONE
    ```

The `/release-note-none` comment command can still be used as an alternative to writing "NONE" in the release-note block if it is left empty.

To see how to format your release notes, view the [PR template](https://github.com/) for a brief example. PR titles and body comments can be modified at any time prior to the release to make them friendly for release notes.
// PR template TODO

Release notes apply to PRs on the master branch. For cherry-pick PRs, see the [cherry-pick instructions](cherry-picks.md). The only exception to these rules is when a PR is not a cherry-pick and is targeted directly to the non-master branch.  In this case, a `release-note-*` label is required for that non-master PR.

// cherry-pick TODO

Now that your release notes are in shape, let's look at how the PR gets tested and merged.

### The Testing and Merge Workflow

The OpenFunction merge workflow uses comments to run tests and labels for merging PRs.

NOTE: For pull requests that are in progress but not ready for review, prefix the PR title with `WIP` or `[WIP]` and track any remaining TODOs in a checklist in the pull request description.

Here's the process the PR goes through on its way from submission to merging:

1. Make the pull request
2. `@o8x-merge-robot` assigns reviewers //TODO

If you're **not** a member of the OpenFunction organization:

1. Reviewer/OpenFunction member checks that the PR is safe to test. If so, they comment `/ok-to-test`
2. Reviewer suggests edits
3. Push edits to your PR branch
4. Repeat the prior two steps as needed
5. (Optional) Some reviewers prefer that you squash commits at this step
6. Owner is assigned and will add the `/approve` label to the PR

If you are a member, or a member comments `/ok-to-test`, the PR will be considered to be trusted. Then the pre-submit tests will run:

1. Automatic tests run. See the current list of tests on the PR
2. If tests fail, resolve issues by pushing edits to your PR branch
3. If the failure is a flake, anyone on trusted PRs can comment `/retest` to rerun failed tests

Once the tests pass, all failures are commented as flakes, or the reviewer adds the labels `lgtm` and `approved`, the PR enters the final merge queue. The merge queue is needed to make sure no incompatible changes have been introduced by other PRs since the tests were last run on your PR.

Either the [on call contributor](on-call-rotations.md) will manage the merge queue manually. //TODO

1. The PR enters the merge queue
2. The merge queue triggers a test re-run with the comment `/test all [submit-queue is verifying that this PR is safe to merge]`
    2.1. Author has signed the CLA (`cncf-cla: yes` label added to PR)
    2.2. No changes made since last `lgtm` label applied
3. If tests fail, resolve issues by pushing edits to your PR branch
4. If the failure is a flake, anyone can comment `/retest` if the PR is trusted
5. If tests pass, the merge queue automatically merges the PR

That's the last step. Your PR is now merged.

### Marking Unfinished Pull Requests

If you want to solicit reviews before the implementation of your pull request is complete, you should hold your pull request to ensure that the merge queue does not pick it up and attempt to merge it. There are two methods to achieve this:

1. You may add the `/hold` or `/hold cancel` comment commands
2. You may add or remove a `WIP` or `[WIP]` prefix to your pull request title

The GitHub robots will add and remove the `do-not-merge/hold` label as you use the comment commands and the `do-not-merge/work-in-progress` label as you edit your title. While either label is present, your pull request will not be considered for merging.

### Comment Commands Reference//TODO

[The commands doc]() contains a reference for all comment commands. //TODO

### Automation//TODO

The OpenFunction developer community uses a variety of automation to manage pull requests.  This automation is described in detail [in the automation doc](automation.md). //TODO

### How the e2e Tests Work//TODO

The  tests will post the status results to the PR. If an e2e test fails,
`@o8x-ci-robot` will comment on the PR with the test history and the
comment-command to re-run that test. e.g.

> The following tests failed, say /retest to rerun them all.

