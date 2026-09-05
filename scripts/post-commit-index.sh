#!/bin/sh
# Re-index the repo into codebase-memory-mcp after every local commit.
# Requires the codebase-memory-mcp server to be reachable; failures here
# should never block git itself.
REPO_ROOT="$(git rev-parse --show-toplevel)"
echo "[post-commit] codebase index refresh needed for $REPO_ROOT (trigger index_repository via the assistant/CI, not from raw git hook)"
