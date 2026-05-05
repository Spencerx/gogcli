---
title: Overview
permalink: /
description: "gog is a single Go CLI for Gmail, Calendar, Drive, Docs, Sheets, Slides, Forms, Apps Script, Contacts, Tasks, and Workspace admin — built for terminals, scripts, CI, and coding agents."
---

# gog

A script-friendly Google Workspace CLI. One binary, every major Google API, predictable output for terminals, shell pipelines, CI, and coding agents.

## Why gog

- **One CLI, every API.** Gmail, Calendar, Drive, Docs, Sheets, Slides, Forms, Apps Script, Contacts, People, Tasks, Classroom, Chat, Groups, Keep, and Workspace Admin under one binary.
- **Stable output.** `--json` to stdout for scripts, `--plain` TSV when you need to `awk`, human progress on stderr so pipes stay clean.
- **Multi-account, multi-client.** Many Google accounts and OAuth client projects in one config. OAuth, direct access tokens, ADC, and Workspace service accounts all supported.
- **Built for agents.** Runtime allow/deny lists (`--enable-commands`, `--disable-commands`, `--gmail-no-send`) and baked safety-profile binaries for sandboxes that should not be able to broaden their own permissions.
- **Read-only audits.** Drive `tree`, `du`, `inventory`; Contacts `dedupe` preview; raw API JSON dumps without ever mutating remote state.
- **Generated reference.** Every command has a Markdown page produced from `gog schema --json`.

## Pick your path

- **Trying it.** Read [Install](install.md), then [Quickstart](quickstart.md). Five minutes from `brew install` to your first authenticated query.
- **Wiring up an agent.** Read [Safety Profiles](safety-profiles.md) and the bundled [`gog` agent skill](https://github.com/steipete/gogcli/blob/main/.agents/skills/gog/SKILL.md). Lock the binary down before you hand it to a model.
- **Running Workspace at scale.** Read [Auth Clients](auth-clients.md) for service accounts, named OAuth clients, and domain-wide delegation.
- **Backing up an account.** Read [Backup](backup.md) before pointing `gog backup push` at a busy mailbox.
- **Looking up a flag.** Open the [Command Index](commands/) — every subcommand has a generated page.

## Status

`gog` is actively developed; the [CHANGELOG](https://github.com/steipete/gogcli/blob/main/CHANGELOG.md) has the most recent shipping log. The API surface is intentionally not 1:1 with `gmcli`/`gccli`/`gdcli` and there is no migration tooling — `gog` is the new CLI, not a port.

## Out of scope

- MCP server (this is a CLI).
- Hosted runtime, web UI, or GUI.
- Importing legacy `~/.gmcli`, `~/.gccli`, `~/.gdcli` state.

Released under the [MIT license](https://github.com/steipete/gogcli/blob/main/LICENSE). Not affiliated with Google. Google is a trademark of Google LLC.
