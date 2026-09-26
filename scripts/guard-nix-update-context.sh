#!/bin/sh
# Refuse to run the privileged Nix update job outside the intended
# repository, refs, and events. The context arrives through step-level
# env: expressions because the runner's ATOMGIT_* environment variables
# do not carry the documented owner/repo, refs/heads/..., or event-name
# values, while a job-level "if:" cannot combine comparisons with && or
# || without failing the whole run at startup. on.push.branches cannot
# cover workflow_dispatch, which runs the workflow file from whichever
# branch was selected.
#
# Expected environment (injected by the workflow step):
#   GUARD_REPOSITORY  from ${{ atomgit.repository }}
#   GUARD_REF         from ${{ atomgit.ref }}
#   GUARD_EVENT       from ${{ atomgit.event_name }}
set -eu

repository=${GUARD_REPOSITORY:-}
ref=${GUARD_REF:-}
event=${GUARD_EVENT:-}

case "$repository" in
  hust-open-atom-club/atomgit-cli) ;;
  *)
    echo "Refusing to run for repository '${repository}'." >&2
    exit 1
    ;;
esac

case "$ref" in
  refs/heads/main|refs/heads/test|refs/heads/nix-update) ;;
  *)
    echo "Refusing to run for ref '${ref}'." >&2
    exit 1
    ;;
esac

case "$event" in
  # AtomGit push runs report "Push"; retain the documented lowercase name.
  push|Push|workflow_dispatch) ;;
  *)
    echo "Refusing to run for event '${event}'." >&2
    exit 1
    ;;
esac
