# Documentation Rules for AI Agents

This document defines rules for creating, updating, and organizing documentation. All AI agents MUST follow these rules.

## MECE Principle

All documentation MUST be **MECE** (Mutually Exclusive, Collectively Exhaustive):

- **Mutually Exclusive**: No overlapping content between documents
- **Collectively Exhaustive**: All information is documented somewhere

## Document Hierarchy

```
CLAUDE.md              # Entry point - overview, links to docs/
docs/
├── INDEX.md           # Document map - when to read which doc
├── BACKEND.md         # Go backend ONLY
├── FRONTEND.md        # Vue/Nuxt frontend ONLY
├── API.md             # REST endpoints ONLY
├── DATABASE.md        # Schema/queries ONLY
├── DEPLOYMENT.md      # Docker/K8s ONLY
├── openapi.yaml       # Machine-readable API spec
├── DOCUMENTATION_RULES.md  # This file
└── {FEATURE}.md       # Feature-specific docs (as needed)
```

## Document Ownership (MECE Boundaries)

| Document | Owns | Does NOT Own |
|----------|------|--------------|
| CLAUDE.md | Project overview, quick reference, dev rules | Implementation details |
| INDEX.md | Document navigation, when-to-read guide | Actual content |
| BACKEND.md | Go code structure, interfaces, patterns | API specs, DB schema |
| FRONTEND.md | Vue components, composables, pages | API specs, backend logic |
| API.md | Endpoints, request/response schemas | Implementation details |
| DATABASE.md | Tables, columns, indexes, queries | API specs, code patterns |
| DEPLOYMENT.md | Docker, K8s, env vars, health checks | Code implementation |

## When to Create New Document

Create a new `docs/{NAME}.md` when:

1. **New bounded context**: Feature with distinct domain (e.g., `BILLING.md`, `NOTIFICATIONS.md`)
2. **Cross-cutting concern**: Affects multiple layers (e.g., `SECURITY.md`, `LOGGING.md`)
3. **External integration**: New third-party service (e.g., `STRIPE_INTEGRATION.md`)

Do NOT create new document when:
- Content fits existing document's ownership
- Information is temporary (use comments instead)
- Content is code-specific (use code comments)

## Document Format Rules

### File Naming

```
docs/{NAME}.md

NAME: UPPERCASE, underscore for multi-word
Examples: BACKEND.md, DATABASE.md, STRIPE_INTEGRATION.md
```

### Required Sections

Every document MUST have:

```markdown
# {Title}

{One-line description}

## Quick Reference

{Table or list of key information}

## {Main Content Sections}

{Structured content}
```

### Formatting Rules

| Element | Format | Example |
|---------|--------|---------|
| Headers | H2 for main, H3 for sub | `## Section`, `### Subsection` |
| Lists | Bullets for unordered | `- Item` |
| Code | Fenced with language | ` ```go ` |
| Tables | For structured data | `| Col | Col |` |
| Paths | Code format | `` `backend/internal/` `` |
| Commands | Code block | ` ```bash ` |

### Content Rules

1. **No prose**: Use tables, lists, code blocks
2. **No redundancy**: Reference other docs, don't copy
3. **Specific over general**: Include exact values, paths, commands
4. **Examples required**: Every concept needs a concrete example
5. **Cross-reference**: Link to related docs with `[text](path)`

### Table Format

```markdown
| Column | Type | Description |
|--------|------|-------------|
| value | type | brief desc |
```

### Code Block Format

```markdown
```language
// Comment explaining what this does
actual_code_here
```
```

## Update Rules

### When to Update

- **Code change**: Update corresponding doc immediately
- **New feature**: Add to relevant doc or create new
- **Bug fix**: Update if behavior changed
- **Deprecation**: Mark with `[DEPRECATED]` prefix

### Update Process

1. Identify which doc owns the content (see MECE boundaries)
2. Update ONLY that document
3. If cross-cutting, update INDEX.md references
4. Verify no duplication introduced

### Conflict Resolution

If content could belong to multiple docs:

1. Identify primary ownership (where is the "source of truth"?)
2. Place full content in primary doc
3. Add brief reference in secondary docs: `See [DOC.md](DOC.md#section)`

## Self-Documentation Directive

When AI agent encounters undocumented area:

### Step 1: Check Existing Docs

```
1. Read docs/INDEX.md
2. Identify relevant document
3. Search for existing content
```

### Step 2: Determine Action

| Situation | Action |
|-----------|--------|
| Content exists elsewhere | Add cross-reference |
| Fits existing doc | Add to existing doc |
| New bounded context | Create new doc |
| Temporary/specific | Add code comment only |

### Step 3: Create/Update

Follow format rules above. Minimum viable doc:

```markdown
# {Feature Name}

{One sentence description}

## Overview

{What it does, when to use}

## Implementation

{Key files, interfaces, patterns}

## Usage

{Commands, examples}
```

### Step 4: Update References

1. Add to `docs/INDEX.md` document map
2. Add to `CLAUDE.md` if affects dev workflow

## Template: New Feature Document

```markdown
# {FEATURE_NAME}

{One-line description of what this feature does.}

## Quick Reference

| Item | Value |
|------|-------|
| Primary Files | `path/to/files` |
| Dependencies | list |
| Status | active/deprecated/experimental |

## Overview

{2-3 sentences explaining purpose and scope}

## Architecture

{How it fits into the system}

## Implementation

### Key Components

| Component | File | Purpose |
|-----------|------|---------|
| Name | `path` | What it does |

### Interfaces

```go
type Interface interface {
    Method() error
}
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `VAR` | `value` | What it controls |

## Usage

### Basic Example

```bash
command example
```

### Common Operations

| Operation | Command/Code |
|-----------|--------------|
| Do X | `how to do X` |

## Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| Problem | Why | Fix |

## Related Documents

- [Related Doc](./RELATED.md)
```

## Validation Checklist

Before committing documentation:

- [ ] Content is in correct document (MECE)
- [ ] No duplication with other docs
- [ ] Tables used for structured data
- [ ] Code examples included
- [ ] Cross-references added where needed
- [ ] INDEX.md updated if new doc
