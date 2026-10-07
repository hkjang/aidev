#!/bin/sh
# npm adds an ancestor node_modules/.bin containing Node 20; pin the installed Node 22 for local checks.
export PATH="/home/hkjang/.nvm/versions/node/v22.23.1/bin:$PATH"
exec /bin/sh "$@"
