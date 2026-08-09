#!/bin/sh
# Alternating check: fails on odd invocations, passes on even.
# State file tracks invocation count.
STATE_FILE="${1:-.tmp/alternating-check.state}"
mkdir -p "$(dirname "$STATE_FILE")"
COUNT=0
if [ -f "$STATE_FILE" ]; then
    COUNT=$(cat "$STATE_FILE")
fi
COUNT=$((COUNT + 1))
printf '%d' "$COUNT" > "$STATE_FILE"
if [ $((COUNT % 2)) -eq 0 ]; then
    exit 0
else
    exit 1
fi
