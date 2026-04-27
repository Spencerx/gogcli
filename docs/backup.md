---
summary: "Encrypted Google account backups"
read_when:
  - Adding a new gog backup service adapter
  - Changing encrypted backup layout, manifest fields, or age identity handling
  - Debugging backup-gog push, status, or verify
---

# Encrypted Backups

`gog backup` writes Google account data into a Git repository as age-encrypted
JSONL gzip shards. The intended repository is private, for example
`https://github.com/steipete/backup-gog`, but service payloads are encrypted
before Git sees them.

## Commands

Initialize local config, create an age identity if needed, seed the backup repo,
and print the public recipient:

```bash
gog backup init \
  --repo ~/Projects/backup-gog \
  --remote https://github.com/steipete/backup-gog.git
```

Back up Gmail:

```bash
gog backup push --services gmail --account steipete@gmail.com
```

For a bounded smoke run:

```bash
gog backup push --services gmail --account steipete@gmail.com --query 'newer_than:7d' --max 25
```

Inspect cleartext manifest metadata:

```bash
gog backup status
```

Decrypt every shard and verify hashes and row counts:

```bash
gog backup verify
```

Use `--no-push` on `init` or `push` to commit locally without pushing to the
remote.

## Files

Local config:

```text
~/.gog/backup.json
~/.gog/age.key
```

Backup repo:

```text
README.md
manifest.json
data/gmail/<account-hash>/labels.jsonl.gz.age
data/gmail/<account-hash>/messages/YYYY/MM/part-0001.jsonl.gz.age
```

`manifest.json` is intentionally cleartext. It contains format version, export
time, public age recipients, service names, account hashes, shard paths, row
counts, encrypted byte sizes, and plaintext hashes used for verification. It
does not contain email subjects, senders, recipients, bodies, raw message IDs,
or labels.

## Encryption

Backups use the Go `filippo.io/age` library with X25519 age identities. There
is no backup password. The private identity is stored locally:

```text
~/.gog/age.key
```

The matching public recipient starts with `age1...` and is safe to store in
`~/.gog/backup.json` and `manifest.json`. The private `AGE-SECRET-KEY-...`
value must stay local or in a password manager.

For each shard, `gog backup push`:

1. Exports deterministic JSONL rows.
2. Gzip-compresses the JSONL with a fixed gzip timestamp.
3. Encrypts the compressed bytes with age for every configured recipient.
4. Writes only encrypted `*.jsonl.gz.age` files to Git.
5. Writes `manifest.json` with cleartext metadata for status and verification.

`gog backup verify` decrypts each shard with the local age identity, gunzips it,
checks the plaintext SHA-256 hash from the manifest, and verifies row counts.

## Security Boundary

The encrypted shards protect Google content from GitHub and anyone else without
the local age identity. That includes email bodies, subjects, senders,
recipients, raw MIME payloads, labels, Drive filenames, contacts, event titles,
and similar service data.

The manifest is not secret. It leaks operational metadata by design:

- Export time.
- Public age recipients.
- Service names.
- Account hashes.
- Shard paths and month buckets.
- Row counts.
- Encrypted byte sizes.
- Plaintext shard hashes used by `verify`.
- Backup cadence and which shards changed in Git history.

The account hash is not anonymity. It is useful to avoid putting the literal
email address in paths, but someone who can guess the address can compute and
compare the same hash.

Current trust model:

- Confidentiality: good for a private GitHub backup repo as long as
  `~/.gog/age.key` stays private.
- Integrity against random corruption: `age` authentication, gzip decoding,
  plaintext SHA-256, and row-count verification catch damaged shards.
- Integrity against repository writers: limited. Anyone with push access can
  replace encrypted backup data with different data encrypted to the public
  recipient. Keep repo write access restricted and review unexpected commits.
- Key compromise: if `AGE-SECRET-KEY-...` leaks, historical shards in Git
  history may be readable. Rotate recipients, re-encrypt, and treat old Git
  history as exposed unless it is rewritten and all copies are removed.

Future hardening ideas:

- Store only ciphertext hashes in the public manifest and move plaintext hashes
  into encrypted shard metadata.
- Sign manifests or commits with a local signing key so `verify` can prove who
  created the backup, not just that the files are internally consistent.
- Add optional shard padding or disable gzip for deployments that care more
  about size side channels than repository size.

## Gmail Adapter

The first adapter backs up:

- Gmail labels.
- Raw Gmail messages from `users.messages.get(format=raw)`.

Raw message payloads stay base64url encoded inside encrypted JSONL. This
preserves the RFC 2822 message content while keeping the shard format text
friendly.

`--include-spam-trash` defaults to true. Use `--query` and `--max` for bounded
test exports; omit them for a full mailbox scan.

## Adding Services

Keep one backup engine and add service adapters. A service adapter should:

1. Resolve the authenticated account through normal `gog` auth.
2. Export stable rows without writing Google data.
3. Store sensitive identifiers, titles, names, and content inside encrypted
   shards only.
4. Add cleartext manifest counts and account hashes only.
5. Support bounded smoke flags when the service can be huge.

Good next adapters: Calendar, Contacts/People, Tasks, then Drive.
