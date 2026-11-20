#!/usr/bin/env bash
#
# Runs the CDD service in the foreground. Ctrl+C stops it.
#
# A thin entry point over `cdd start`, kept in bin/ beside it. It works from any
# directory. Options are passed through:
#
#   bin/start.sh            run CDD
#   bin/start.sh --debug    run CDD, logging at DEBUG level
#
# There is no background mode. Running CDD as a service is the job of whatever
# already runs services — a systemd unit, a container runtime, a process
# supervisor — and each of those runs this in the foreground too.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CDD="$HERE/cdd"
[ -x "$CDD" ] || { echo "start: $CDD is missing or not executable" >&2; exit 1; }

exec "$CDD" start "$@"
