# Project Manager (PM) Role Template

## Overview

The PM coordinates parallel development workers, monitors progress, handles reviews, and manages merges.

---

## Work Start Procedure

```bash
# 1. Set project environment
export DEVHIVE_PROJECT=<project-name>

# 2. Check overall status
devhive status

# 3. Check unread messages
devhive msg unread

# 4. Monitor events
devhive watch
```

---

## Key Responsibilities

1. **Sprint Management** - Initialize sprints, register workers
2. **Progress Monitoring** - Track worker status and blockers
3. **Communication Hub** - Relay information between workers
4. **Review & Merge** - Review completed work and merge branches
5. **Problem Resolution** - Help blocked workers

---

## DevHive Commands

### Sprint Management
```bash
# Initialize sprint
devhive init <sprint-id>

# Complete sprint
devhive sprint complete
```

### Worker Management
```bash
# Register workers
devhive worker register <name> <branch> --role <role> --worktree <path>

# Check worker status
devhive worker show <name>

# View all workers
devhive status
```

### Monitoring
```bash
# Real-time monitoring
devhive watch

# Filter by type
devhive watch --filter=message
devhive watch --filter=worker

# View event history
devhive events --limit 50
```

### Messaging
```bash
# Send to specific worker
devhive msg send <worker-name> "Message"

# Broadcast to all
devhive msg broadcast "Message"

# Check messages
devhive msg unread
```

---

## Monitoring Checklist

### Periodic Checks
- [ ] All workers have status updates
- [ ] No workers stuck in "blocked" or "error" state
- [ ] Unread messages are addressed
- [ ] No conflicting file changes between workers

### Before Merge
- [ ] Worker reported completion via `devhive worker complete`
- [ ] Tests pass
- [ ] Code review completed
- [ ] No conflicts with other branches

---

## Common Scenarios

### Worker Reports Error
```bash
# Check error details
devhive worker show <worker-name>

# View recent events
devhive events --limit 10

# Send assistance
devhive msg send <worker-name> "How can I help?"
```

### Coordinating Shared File Changes
```bash
# Broadcast to all workers
devhive msg broadcast "Worker-A will modify shared/utils.ts. Please wait."

# After completion
devhive msg broadcast "shared/utils.ts changes complete. You may proceed."
```

### Sprint Status Report
```bash
# Get overall status
devhive status

# Get detailed events
devhive events --limit 100
```

---

## Merge Workflow

### 1. Verify Completion
```bash
devhive worker show <worker-name>
# Confirm status is "completed"
```

### 2. Review Changes
```bash
cd <worktree-path>
git log --oneline -10
git diff main..HEAD
```

### 3. Merge to Main
```bash
git checkout main
git merge --no-ff <feature-branch> -m "Merge: <description>"
```

### 4. Notify Workers
```bash
devhive msg broadcast "Merged <worker-name>'s changes to main"
```

---

## Troubleshooting

| Issue | Action |
|-------|--------|
| Worker not responding | Check `devhive worker show`, send message |
| Merge conflict | Coordinate with affected workers |
| Worker stuck on error | Review error details, provide guidance |
| Communication breakdown | Use `devhive msg broadcast` for clarity |
