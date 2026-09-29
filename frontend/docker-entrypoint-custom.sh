#!/bin/sh
set -eu
escaped_key=$(printf '%s' "${YANDEX_MAPS_API_KEY:-}" | sed 's/[\\"$`]/\\&/g')
printf 'window.APP_CONFIG = { YANDEX_MAPS_API_KEY: "%s" };\n' "$escaped_key" > /usr/share/nginx/html/config.js
exec /docker-entrypoint.sh "$@"
