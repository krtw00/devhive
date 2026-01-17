# Parallel Development Worker Role Template

## Critical: Work Start Procedure

**Before starting work, always execute the following:**

```bash
# 1. Verify environment
pwd
git branch --show-current

# 2. Report work start to devhive
devhive worker start --task "Describe your task here"

# 3. Check unread messages
devhive msg unread

# 4. Check overall status
devhive status
```

---

## Basic Rules

### Work Scope
- Focus only on assigned tasks
- Do not modify unrelated files
- Ask PM via devhive if anything is unclear

### Prohibited Actions
- Do not commit directly to main branch
- Do not modify other workers' files

---

## DevHive Commands

### Worker Status
```bash
# Start work
devhive worker start --task "Task description"

# Update current task
devhive worker task "Current task description"

# Report error
devhive worker error "Error description"

# Report completion
devhive worker complete
```

### Messaging
```bash
# Send to specific worker
devhive msg send <worker-name> "Message content"

# Broadcast to all workers
devhive msg broadcast "Message content"

# Check unread messages
devhive msg unread
```

### Status Check
```bash
# View overall sprint status
devhive status
```

---

## Communication Guidelines

### When to Check Messages
- Before starting work
- Before modifying shared files
- Periodically during work

### When to Send Messages
1. **Modifying shared code** → Notify affected workers
2. **Found a bug** → Broadcast to all
3. **Need API/interface changes** → Message relevant worker
4. **Work is blocked** → Message PM

---

## Commit Rules

```bash
git commit -m "$(cat <<'EOF'
type(scope): Brief description

Detailed description (if needed)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

### Commit Types
| Type | Usage |
|------|-------|
| feat | New feature |
| fix | Bug fix |
| test | Add/modify tests |
| refactor | Refactoring |
| docs | Documentation |

---

## Completion Flow

1. Implementation complete
2. Tests pass
3. Linting passes
4. Commit created
5. **Report completion:**

```bash
devhive worker complete
```

### On Error
```bash
devhive worker error "Description of the problem"
devhive msg send pm "Need assistance with: ..."
```

---

## Customization

Add project-specific sections below:

- Tech stack
- Test commands
- Lint commands
- Project-specific rules
