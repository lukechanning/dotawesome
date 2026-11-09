#!/bin/sh
set -eu

WOW_ROOT=${WOW_ROOT:-/mnt/hive/WoW}
WOW_REPO=${WOW_REPO:-$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)}
WOW_ENV=${WOW_ENV:-$WOW_ROOT/server/config/realm.env}
WOW_COMPOSE=$WOW_REPO/components/wow/compose.yaml
DOCKER_HOST=${DOCKER_HOST:-unix:///run/user/$(id -u)/radium-wow-docker.sock}
export WOW_ROOT WOW_REPO WOW_ENV WOW_COMPOSE DOCKER_HOST

case "$WOW_ROOT" in
	/mnt/hive/WoW|/mnt/hive/WoW/*) ;;
	*) echo "wow: WOW_ROOT must be beneath /mnt/hive/WoW" >&2; exit 1 ;;
esac

require_radium() {
	[ "$(hostname -s)" = radium ] || { echo "wow: this command runs on radium" >&2; exit 1; }
	[ -w "$WOW_ROOT" ] || { echo "wow: $WOW_ROOT is not writable" >&2; exit 1; }
}

load_realm_env() {
	[ -r "$WOW_ENV" ] || { echo "wow: missing $WOW_ENV; copy realm.env.example and populate it" >&2; exit 1; }
	set -a
	# shellcheck disable=SC1090
	. "$WOW_ENV"
	set +a
	case "${WOW_TAILSCALE_IP:-}" in CHANGE_ME|'') echo "wow: configure WOW_TAILSCALE_IP" >&2; exit 1;; esac
	case "${WOW_DB_ROOT_PASSWORD:-}" in CHANGE_ME|'') echo "wow: configure WOW_DB_ROOT_PASSWORD" >&2; exit 1;; esac
}

compose() {
	docker compose --env-file "$WOW_ENV" -f "$WOW_COMPOSE" "$@"
}
