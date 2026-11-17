# Releasing txoko

Releases are automated: pushing a version tag runs
[`.github/workflows/release.yml`](.github/workflows/release.yml), which uses
[GoReleaser](https://goreleaser.com) (configured in
[`.goreleaser.yaml`](.goreleaser.yaml)) to:

1. run the tests,
2. build txoko for macOS and Linux, on Intel and ARM,
3. publish a GitHub release with the archives, checksums and a changelog,
4. update the Homebrew cask in [LifeOfAitor/homebrew-tap](https://github.com/LifeOfAitor/homebrew-tap).

## Cutting a release

```sh
git switch main && git pull
make check
make snapshot                    # optional: a full local dry run, nothing published
git tag -a v0.2.0 -m "txoko v0.2.0"
git push origin v0.2.0
```

Then watch the run with `gh run watch` and check the release page.

## Versions

txoko follows [semantic versioning](https://semver.org): `vMAJOR.MINOR.PATCH`.

- **PATCH** (v0.1.1) for fixes,
- **MINOR** (v0.2.0) for new features,
- **MAJOR** for breaking changes. While txoko is `v0.x`, breaking changes bump MINOR.

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org)
(`feat:`, `fix:`, `docs:`…), which is how the changelog groups them.

## One-time setup

- A public repository `LifeOfAitor/homebrew-tap`, where GoReleaser commits the cask.
- A fine-grained personal access token with **Contents: read and write** on
  that repository only, saved in this repository as the Actions secret
  `HOMEBREW_TAP_GITHUB_TOKEN`. The default `GITHUB_TOKEN` can't write to
  another repository.
