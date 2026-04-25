#!/bin/bash
# One-time script: create Hobbies and Goals groups, move items into them.
# Run from your local terminal: bash setup_groups.sh
# Delete this file after running.

BASE="https://playground-flywheel.fly.dev"

echo "==> Creating Hobbies group..."
HOBBIES_RESP=$(curl -s -X POST "$BASE/api/groups" \
  -H "Content-Type: application/json" \
  -d '{"name":"Hobbies"}')
echo "$HOBBIES_RESP"

echo "==> Creating Goals group..."
GOALS_RESP=$(curl -s -X POST "$BASE/api/groups" \
  -H "Content-Type: application/json" \
  -d '{"name":"Goals"}')
echo "$GOALS_RESP"

echo ""
echo "==> Fetching group IDs..."
GROUPS=$(curl -s "$BASE/api/groups")
echo "$GROUPS"

HOBBIES_ID=$(echo "$GROUPS" | python3 -c "import sys,json; gs=json.load(sys.stdin); print(next(g['id'] for g in gs if g['name']=='Hobbies'))")
GOALS_ID=$(echo "$GROUPS" | python3 -c "import sys,json; gs=json.load(sys.stdin); print(next(g['id'] for g in gs if g['name']=='Goals'))")

echo "Hobbies ID: $HOBBIES_ID"
echo "Goals ID:   $GOALS_ID"

echo ""
echo "==> Fetching items..."
ITEMS=$(curl -s "$BASE/api/items")

move_item() {
  local NAME="$1"
  local GROUP_ID="$2"
  local ID=$(echo "$ITEMS" | python3 -c "
import sys,json
items=json.load(sys.stdin)
match=next((it for it in items if it['name'].lower()=='$NAME'.lower()), None)
print(match['id'] if match else '')
")
  if [ -z "$ID" ]; then
    echo "  WARN: item '$NAME' not found"
  else
    RESULT=$(curl -s -X PATCH "$BASE/api/items/$ID" \
      -H "Content-Type: application/json" \
      -d "{\"group_id\":$GROUP_ID}")
    echo "  $NAME (id=$ID) -> group $GROUP_ID: $RESULT"
  fi
}

echo ""
echo "==> Moving items into Hobbies (id=$HOBBIES_ID)..."
move_item "gardening" "$HOBBIES_ID"
move_item "fishing" "$HOBBIES_ID"
move_item "dancing" "$HOBBIES_ID"
move_item "music" "$HOBBIES_ID"

echo ""
echo "==> Moving items into Goals (id=$GOALS_ID)..."
move_item "own a home" "$GOALS_ID"
move_item "fallback income" "$GOALS_ID"
move_item "full-stack" "$GOALS_ID"
move_item "non-django" "$GOALS_ID"

echo ""
echo "Done. Delete this file: rm setup_groups.sh"
