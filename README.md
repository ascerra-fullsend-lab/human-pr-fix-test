# human-pr-fix-test

E2E test repository for verifying fix agent gating on human-authored PRs.

## Test plan

1. PR without `fullsend-fix` label → review agent runs, fix agent does NOT auto-trigger
2. PR with `fullsend-fix` label → review agent runs, fix agent DOES auto-trigger

