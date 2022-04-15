#!/usr/bin/env bash

echo "$(echo '<rubrikk>'; cat $1)" > "$1"
echo "</rubrikk>" >> "$1"