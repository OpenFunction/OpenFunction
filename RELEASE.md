# Releases

This page describes the release process and the currently planned schedule for upcoming releases as well as the respective release shepherd.

## Release schedule

| release series | date  (year-month-day) | release shepherd          |
|----------------|------------------------|---------------------------|
| v0.1.0         | 2021-05-17             | Benjamin Huo @benjaminhuo |
| v0.2.0         | 2021-06-30             | Benjamin Huo @benjaminhuo |
| v0.3.0         | 2021-08-19             | Laminar @tpiperatgod      |
| v0.3.1         | 2021-08-27             | Wanjun Lei @wanjunlei     |
| v0.4.0         | 2021-10-19             | Wanjun Lei @wanjunlei     |
| v0.5.0         | 2021-12-31             | Benjamin Huo @benjaminhuo |
| v0.6.0-rc      | 2022-03-08             | Benjamin Huo @benjaminhuo |
| v0.6.0         | 2022-03-21             | Benjamin Huo @benjaminhuo |
| v0.7.0-rc      | 2022-08-11             | Wrongerror @wrongerror    |
| v0.7.0         | 2022-08-16             | Wrongerror @wrongerror    |
| v0.8.0-rc      | 2022-10-14             | Wrongerror @wrongerror    |
| v0.8.0         | 2022-10-21             | Wrongerror @wrongerror    |
| v0.8.1-rc      | 2022-11-23             | Wrongerror @wrongerror    |
| v0.8.1         | 2022-12-01             | Wrongerror @wrongerror    |
| v1.0.0-rc      | 2023-02-23             | Wrongerror @wrongerror    |
| v1.0.0         | 2023-03-08             | Wrongerror @wrongerror    |
| v1.1.0         | 2023-05-30             | Wanjun Lei @wanjunlei     |
| v1.1.1         | 2023-06-14             | Wanjun Lei @wanjunlei     |
| v1.2.0         | 2023-09-22             | Wrongerror @wrongerror    |


# How to cut a new release

> This guide is strongly based on the [Prometheus release instructions](https://github.com/prometheus/prometheus/blob/master/RELEASE.md).

## Branch management and versioning strategy

We use [Semantic Versioning](http://semver.org/).

We maintain a separate branch for each minor release, named `release-<major>.<minor>`, e.g. `release-1.1`, `release-2.0`.

The usual flow is to merge new features and changes into the main branch and to merge bug fixes into the latest release branch. Bug fixes are then merged into main from the latest release branch. The main branch should always contain all commits from the latest release branch.

If a bug fix got accidentally merged into main, cherry-pick commits have to be created in the latest release branch, which then have to be merged back into main. Try to avoid that situation.

Maintaining the release branches for older minor releases happens on a best effort basis.

## Prepare your release

For a major or minor release, start working in the `main` branch. For a patch release, start working in the minor release branch you want to patch (e.g. `release-0.1` if you're releasing `v0.1.1`).

Add an entry for the new version to the `CHANGELOG.md` file. Entries in the `CHANGELOG.md` should be in this order:

* `[CHANGE]`
* `[FEATURE]`
* `[ENHANCEMENT]`
* `[BUGFIX]`

Create a PR for the change log.

## Publish the new release

For new minor and major releases, create the `release-<major>.<minor>` branch starting at the PR merge commit.
From now on, all work happens on the `release-<major>.<minor>` branch.

Bump the version in the `VERSION` file.

Regenerate bundle.yaml based on latest code by `make manifests` and then commit the changed bundle.yaml to the `release-<major>.<minor>` branch:

```bash
make manifests
git add ./
git commit -s -m "regenerate bundle.yaml"
git push
```

Tag the new release with a tag named `v<major>.<minor>.<patch>`, e.g. `v2.1.3`. Note the `v` prefix. You can do the tagging on the commandline:

```bash
tag="$(< VERSION)"
git tag -a "${tag}" -m "${tag}"
git push origin "${tag}"
```
Commit all the changes.

The corresponding container image will be built and pushed to [docker hub](https://hub.docker.com/repository/docker/openfunction) automatically by CI once the release tag is added.

Go to https://github.com/OpenFunction/OpenFunction/releases/new, associate the new release with the before pushed tag, paste in changes made to `CHANGELOG.md`, add file `config/bundle.yaml`, file `config/strategy/build-strategy.yaml`, file `config/domain/default-domain.yaml` and then click "Publish release".

For patch releases, submit a pull request to merge back the release branch into the `main` branch.

