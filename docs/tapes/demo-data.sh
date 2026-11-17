#!/usr/bin/env bash
# Builds a throwaway activity history for the README screenshots, so they
# never show anyone's real data: a temporary repository with a few commits a
# day over the last two weeks, imported into the database at $TXOKO_DB.
set -euo pipefail
: "${TXOKO_DB:?set TXOKO_DB to a throwaway database path}"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
repo="$work/project"

# Keep the user's git config (and any global hooks) out of this.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q "$repo"
git -C "$repo" config user.email demo@example.com
git -C "$repo" config user.name Demo

n=0
for day in $(seq 13 -1 0); do
	for i in $(seq 1 $(((day * 7 + 3) % 4 + 1))); do
		when=$(date -v-"${day}"d -v-"${i}"H +%Y-%m-%dT%H:%M:%S)
		GIT_AUTHOR_DATE="$when" GIT_COMMITTER_DATE="$when" \
			git -C "$repo" commit -q --allow-empty -m "work $((n += 1))"
	done
done

rm -f "$TXOKO_DB"
txoko import --days 30 "$repo" >/dev/null
