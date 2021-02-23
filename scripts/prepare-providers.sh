#!/usr/bin/env bash

jq -n \
  --arg ghk "$GITHUB_KEY" --arg ghs "$GITHUB_SECRET" \
  --arg glk "$GITLAB_KEY" --arg gls "$GITLAB_SECRET" \
  --arg gok "$GOOGLE_KEY" --arg gos "$GOOGLE_SECRET" \
  '{
    github: {key: $ghk, secret: $ghs},
    gitlab: {key: $glk, secret: $gls},
    google: {key: $gok, secret: $gos}
  }' > providers.json