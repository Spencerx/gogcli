# gog Docs

`gog` is a single CLI for Google Workspace automation: Gmail, Calendar, Drive,
Docs, Sheets, Slides, Contacts, Tasks, People, Forms, Apps Script, Groups, Admin,
Keep, and related agent workflows.

## Start Here

- Install and authenticate from the repository
  [README](https://github.com/steipete/gogcli#readme).
- Read [Install and Runtime Packages](install.md) when installing from
  Homebrew, Docker, GitHub releases, Windows ZIPs, or source.
- Read [Auth Clients](auth-clients.md) when setting up OAuth clients, service
  accounts, or Workspace domain-wide delegation.
- Read [Command Guards and Baked Safety Profiles](safety-profiles.md) when
  running `gog` from agents or automation.
- Read the bundled [`gog` agent skill](../.agents/skills/gog/SKILL.md) when an
  agent needs safe auth preflight, JSON-first output, or guarded Workspace
  automation patterns.
- Read [Sheets Tables](sheets-tables.md) when creating or inspecting Google
  Sheets structured tables.
- Open the [Command Index](commands/README.md) for generated docs for every CLI
  command.

## Feature Pages

- [Install and Runtime Packages](install.md)
- [Auth Clients](auth-clients.md)
- [Command Guards and Baked Safety Profiles](safety-profiles.md)
- [Raw API Dumps](raw-api.md)
- [Raw API Sensitive Field Audit](raw-audit.md)
- [Gmail Workflows](gmail-workflows.md)
- [Gmail watch](watch.md)
- [Email Tracking](email-tracking.md)
- [Drive Audits](drive-audits.md)
- [Contacts Dedupe Preview](contacts-dedupe.md)
- [Contacts JSON Update](contacts-json-update.md)
- [Google Docs Editing](docs-editing.md)
- [Sheets Tables](sheets-tables.md)
- [Sheets Formatting](sheets-formatting.md)
- [Slides from Markdown](slides-markdown.md)
- [Slides Template Replacement](slides-template-replacement.md)
- [Backups](backup.md)
- [Date and Time Input Formats](dates.md)

## Common Paths

```bash
gog auth add you@gmail.com --services gmail,calendar,drive
gog gmail search 'newer_than:7d' --max 10
gog gmail get <messageId> --sanitize-content --json
gog calendar events --today
gog drive ls --max 20
```

## Command Docs

Every command page under `docs/commands/` is generated from
`gog schema --json`. Do not hand-edit generated command pages. After changing
commands, flags, aliases, arguments, or help text, run:

```bash
make docs-commands
```

`make docs-check` verifies that every schema command has a generated page and
that required feature pages are present and linked from this overview.

Then build the GitHub Pages site locally:

```bash
make docs-site
open dist/docs-site/index.html
```

The site is intentionally static: no framework, no package install, and no
client-side dependency beyond a small navigation script embedded by the builder.
