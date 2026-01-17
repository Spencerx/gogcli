#!/usr/bin/env bash
set -euo pipefail

FAST=false
STRICT=false
ACCOUNT=""
SKIP=""
AUTH_SERVICES=""

usage() {
  cat <<'USAGE'
Usage: scripts/live-test.sh [options]

Options:
  --fast              Skip slower tests (docs/sheets/slides)
  --strict            Fail on optional tests (groups/keep/enterprise)
  --account <email>   Account to use (defaults to GOG_IT_ACCOUNT or first auth)
  --skip <list>       Comma-separated skip list (e.g., gmail,drive,docs)
  --auth <services>   Re-auth before running (e.g., all,groups)
  -h, --help          Show this help

Skip keys:
  auth-alias, enable-commands, gmail, drive, docs, sheets, slides,
  calendar, calendar-enterprise, tasks, contacts, people,
  groups, keep, classroom
USAGE
}

while [ $# -gt 0 ]; do
  case "$1" in
    --fast)
      FAST=true
      ;;
    --strict)
      STRICT=true
      ;;
    --account)
      ACCOUNT="$2"
      shift
      ;;
    --skip)
      SKIP="$2"
      shift
      ;;
    --auth)
      AUTH_SERVICES="$2"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown arg: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
 done

SKIP="${SKIP:-${GOG_LIVE_SKIP:-}}"
if [ "$FAST" = true ]; then
  if [ -n "$SKIP" ]; then
    SKIP="$SKIP,docs,sheets,slides"
  else
    SKIP="docs,sheets,slides"
  fi
fi

BIN="${GOG_BIN:-./bin/gog}"
if [ ! -x "$BIN" ]; then
  make build >/dev/null
fi

PY="${PYTHON:-python3}"
if ! command -v "$PY" >/dev/null 2>&1; then
  PY="python"
fi

if [ -z "$ACCOUNT" ]; then
  ACCOUNT="${GOG_IT_ACCOUNT:-}"
fi
if [ -z "$ACCOUNT" ]; then
  acct_json=$($BIN auth list --json)
  ACCOUNT=$($PY -c 'import json,sys; obj=json.load(sys.stdin); print(obj.get("accounts", [{}])[0].get("email", ""))' <<<"$acct_json")
fi
if [ -z "$ACCOUNT" ]; then
  echo "No account available for live tests." >&2
  exit 1
fi

echo "Using account: $ACCOUNT"

if [ -n "$AUTH_SERVICES" ]; then
  $BIN auth add "$ACCOUNT" --services "$AUTH_SERVICES"
fi

skip() {
  local key="$1"
  [ -n "$SKIP" ] || return 1
  IFS=',' read -r -a items <<<"$SKIP"
  for item in "${items[@]}"; do
    if [ "$item" = "$key" ]; then
      return 0
    fi
  done
  return 1
}

extract_id() {
  $PY -c 'import json,sys
obj=json.load(sys.stdin)

def find_id(x):
    if isinstance(x, dict):
        for key in ("id", "draftId", "spreadsheetId", "presentationId", "documentId"):
            if isinstance(x.get(key), str):
                return x[key]
        for v in x.values():
            r=find_id(v)
            if r:
                return r
    if isinstance(x, list):
        for v in x:
            r=find_id(v)
            if r:
                return r
    return ""
print(find_id(obj))' <<<"$1"
}

extract_field() {
  local value="$1"
  local field="$2"
  $PY -c 'import json,sys
obj=json.load(sys.stdin)
key=sys.argv[1]

def find_field(x, k):
    if isinstance(x, dict):
        if k in x and isinstance(x[k], str):
            return x[k]
        for v in x.values():
            r=find_field(v, k)
            if r:
                return r
    if isinstance(x, list):
        for v in x:
            r=find_field(v, k)
            if r:
                return r
    return ""
print(find_field(obj, key))' "$field" <<<"$value"
}

extract_tasklist_id() {
  $PY -c 'import json,sys
obj=json.load(sys.stdin)
for key in ("tasklists","lists","items"):
    if isinstance(obj, dict) and obj.get(key):
        print(obj[key][0].get("id",""))
        sys.exit(0)
print("")' <<<"$1"
}

extract_task_ids() {
  $PY -c 'import json,sys
obj=json.load(sys.stdin)
ids=[]
if isinstance(obj, dict) and "tasks" in obj:
    ids=[t.get("id") for t in obj.get("tasks",[]) if t.get("id")]
elif isinstance(obj, dict) and "task" in obj:
    if obj["task"].get("id"):
        ids=[obj["task"]["id"]]
print("\n".join(ids))' <<<"$1"
}

run_required() {
  local key="$1"
  local label="$2"
  shift 2
  if skip "$key"; then
    echo "==> $label (skipped)"
    return 0
  fi
  echo "==> $label"
  "$@"
}

run_optional() {
  local key="$1"
  local label="$2"
  shift 2
  if skip "$key"; then
    echo "==> $label (skipped)"
    return 0
  fi
  echo "==> $label (optional)"
  if "$@"; then
    echo "ok"
    return 0
  fi
  echo "skipped/failed"
  if [ "$STRICT" = true ]; then
    return 1
  fi
  return 0
}

run_required "time" "time now" "$BIN" time now --json >/dev/null

if ! skip "auth-alias"; then
  alias_name="smoke-$(date +%Y%m%d%H%M%S)"
  run_required "auth-alias" "auth alias set" "$BIN" auth alias set "$alias_name" "$ACCOUNT" --json >/dev/null
  run_required "auth-alias" "auth alias list" "$BIN" auth alias list --json >/dev/null
  run_required "auth-alias" "auth alias unset" "$BIN" auth alias unset "$alias_name" --json >/dev/null
fi

if ! skip "enable-commands"; then
  run_required "enable-commands" "enable-commands allow time" "$BIN" --enable-commands time time now --json >/dev/null
  if $BIN --enable-commands time gmail labels list >/dev/null 2>&1; then
    echo "Expected enable-commands to block gmail, but it succeeded" >&2
    exit 1
  else
    echo "enable-commands block OK"
  fi
fi

if ! skip "gmail"; then
  DRAFT_JSON=$($BIN gmail drafts create --to "$ACCOUNT" --subject "gogcli smoke" --body "smoke" --json)
  DRAFT_ID=$(extract_field "$DRAFT_JSON" draftId)
  [ -n "$DRAFT_ID" ] || { echo "Failed to parse draft id" >&2; exit 1; }
  run_required "gmail" "gmail drafts get" "$BIN" gmail drafts get "$DRAFT_ID" --json >/dev/null
  run_required "gmail" "gmail drafts delete" "$BIN" gmail drafts delete "$DRAFT_ID" --force >/dev/null
fi

if ! skip "drive"; then
  FOLDER_JSON=$($BIN drive mkdir "gogcli-smoke" --json)
  FOLDER_ID=$(extract_id "$FOLDER_JSON")
  [ -n "$FOLDER_ID" ] || { echo "Failed to parse drive folder id" >&2; exit 1; }
  run_required "drive" "drive delete folder" "$BIN" drive delete "$FOLDER_ID" --force >/dev/null
fi

if ! skip "docs"; then
  DOC_JSON=$($BIN docs create "gogcli-smoke" --json)
  DOC_ID=$(extract_id "$DOC_JSON")
  [ -n "$DOC_ID" ] || { echo "Failed to parse doc id" >&2; exit 1; }
  run_required "docs" "drive delete doc" "$BIN" drive delete "$DOC_ID" --force >/dev/null
fi

if ! skip "sheets"; then
  SHEET_JSON=$($BIN sheets create "gogcli-smoke" --json)
  SHEET_ID=$(extract_id "$SHEET_JSON")
  [ -n "$SHEET_ID" ] || { echo "Failed to parse sheet id" >&2; exit 1; }
  run_required "sheets" "drive delete sheet" "$BIN" drive delete "$SHEET_ID" --force >/dev/null
fi

if ! skip "slides"; then
  SLIDES_JSON=$($BIN slides create "gogcli-smoke" --json)
  SLIDES_ID=$(extract_id "$SLIDES_JSON")
  [ -n "$SLIDES_ID" ] || { echo "Failed to parse slides id" >&2; exit 1; }
  run_required "slides" "drive delete slides" "$BIN" drive delete "$SLIDES_ID" --force >/dev/null
fi

if ! skip "calendar"; then
  read -r START END DAY1 DAY2 <<<"$($PY - <<'PY'
import datetime
now=datetime.datetime.now(datetime.timezone.utc).replace(minute=0, second=0, microsecond=0)
start=now + datetime.timedelta(hours=1)
end=start + datetime.timedelta(hours=1)
print(start.strftime('%Y-%m-%dT%H:%M:%SZ'), end.strftime('%Y-%m-%dT%H:%M:%SZ'), start.strftime('%Y-%m-%d'), (start+datetime.timedelta(days=1)).strftime('%Y-%m-%d'))
PY
)"

  EV_JSON=$($BIN calendar create primary --summary "gogcli-smoke" --from "$START" --to "$END" --json)
  EV_ID=$(extract_id "$EV_JSON")
  [ -n "$EV_ID" ] || { echo "Failed to parse calendar event id" >&2; exit 1; }
  run_required "calendar" "calendar event get" "$BIN" calendar event primary "$EV_ID" --json >/dev/null
  run_required "calendar" "calendar propose-time" "$BIN" calendar propose-time primary "$EV_ID" --json >/dev/null
  run_required "calendar" "calendar delete event" "$BIN" calendar delete primary "$EV_ID" --force >/dev/null

  if ! skip "calendar-enterprise"; then
    run_optional "calendar-enterprise" "calendar focus-time" "$BIN" calendar create primary --event-type focus-time --from "$START" --to "$END" --json >/dev/null 2>&1 || true
    run_optional "calendar-enterprise" "calendar out-of-office" "$BIN" calendar create primary --event-type out-of-office --from "$DAY1" --to "$DAY2" --all-day --json >/dev/null 2>&1 || true
    run_optional "calendar-enterprise" "calendar working-location" "$BIN" calendar create primary --event-type working-location --working-location-type office --working-office-label "HQ" --from "$DAY1" --to "$DAY2" --json >/dev/null 2>&1 || true
  fi
fi

if ! skip "tasks"; then
  LIST_JSON=$($BIN tasks lists --json --max 1)
  LIST_ID=$(extract_tasklist_id "$LIST_JSON")
  [ -n "$LIST_ID" ] || { echo "No task list found" >&2; exit 1; }
  TASK_JSON=$($BIN tasks add "$LIST_ID" --title "gogcli-smoke" --due "$DAY1" --repeat daily --repeat-count 2 --json)
  TASK_IDS=$(extract_task_ids "$TASK_JSON")
  [ -n "$TASK_IDS" ] || { echo "Failed to parse task ids" >&2; exit 1; }
  FIRST_TASK_ID=$(echo "$TASK_IDS" | head -n1)
  run_required "tasks" "tasks get" "$BIN" tasks get "$LIST_ID" "$FIRST_TASK_ID" --json >/dev/null
  while IFS= read -r tid; do
    [ -n "$tid" ] && run_required "tasks" "tasks delete" "$BIN" tasks delete "$LIST_ID" "$tid" --force >/dev/null
  done <<<"$TASK_IDS"
fi

if ! skip "contacts"; then
  CONTACT_JSON=$($BIN contacts create --given "gogcli" --family "smoke" --email "gogcli-smoke@example.com" --json)
  CONTACT_ID=$(extract_field "$CONTACT_JSON" resourceName)
  [ -n "$CONTACT_ID" ] || { echo "Failed to parse contact resourceName" >&2; exit 1; }
  run_required "contacts" "contacts delete" "$BIN" contacts delete "$CONTACT_ID" --force >/dev/null
fi

run_required "people" "people me" "$BIN" people me --json >/dev/null

run_optional "groups" "groups list" "$BIN" groups list --json --max 1 >/dev/null 2>&1
run_optional "keep" "keep list" "$BIN" keep list --json >/dev/null 2>&1

run_optional "classroom" "classroom list" "$BIN" classroom courses list --json --max 1 >/dev/null 2>&1

echo "Live tests complete."
