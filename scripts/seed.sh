#!/usr/bin/env bash
#
# Seed script for Mixology CLI
#
# Bootstraps sample data from JSON files in scripts/data/:
#   - ingredients.json: Common bar ingredients with stock levels
#   - drinks.json: Classic cocktail recipes
#
# Usage:
#   ./scripts/seed.sh
#
# Prerequisites:
#   - mixology binary must be in PATH or current directory
#   - jq must be installed (brew install jq)
#   - Database must be initialized
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="$SCRIPT_DIR/data"
MIXOLOGY="${MIXOLOGY:-./mixology}"

# Check dependencies
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required. Install with: brew install jq"
    exit 1
fi

if ! command -v "$MIXOLOGY" &> /dev/null && [[ ! -x "$MIXOLOGY" ]]; then
    echo "Error: mixology binary not found. Build it first:"
    echo "  go build -o mixology ./main"
    exit 1
fi

echo "=== Mixology Seed Script ==="
echo ""

# Associative array to store ingredient key -> ID mappings
declare -A INGREDIENT_IDS

# -----------------------------------------------------------------------------
# Create Ingredients
# -----------------------------------------------------------------------------

echo "Creating ingredients..."

while IFS= read -r ingredient; do
    key=$(echo "$ingredient" | jq -r '.key')
    name=$(echo "$ingredient" | jq -r '.name')
    category=$(echo "$ingredient" | jq -r '.category')
    unit=$(echo "$ingredient" | jq -r '.unit')
    description=$(echo "$ingredient" | jq -r '.description')

    payload=$(echo "$ingredient" | jq -c '{
        name,
        category,
        unit,
        description: (.description // "")
    }')
    id=$(echo "$payload" | "$MIXOLOGY" ingredients create --stdin)

    INGREDIENT_IDS["$key"]="$id"
    echo "  $name: $id"
done < <(jq -c '.[]' "$DATA_DIR/ingredients.json")

echo "  Created ${#INGREDIENT_IDS[@]} ingredients"

# -----------------------------------------------------------------------------
# Set Inventory Levels
# -----------------------------------------------------------------------------

echo ""
echo "Setting inventory levels..."

while IFS= read -r ingredient; do
    key=$(echo "$ingredient" | jq -r '.key')
    quantity=$(echo "$ingredient" | jq -r '.stock.quantity')
    cost=$(echo "$ingredient" | jq -r '.stock.cost')

    payload=$(echo "$ingredient" | jq -c --arg id "${INGREDIENT_IDS[$key]}" '{
        ingredient_id: $id,
        quantity: .stock.quantity,
        unit: .unit,
        cost_per_unit: .stock.cost
    }')
    echo "$payload" | "$MIXOLOGY" inventory set --stdin > /dev/null
done < <(jq -c '.[]' "$DATA_DIR/ingredients.json")

echo "  Inventory stocked"

# -----------------------------------------------------------------------------
# Create Drinks
# -----------------------------------------------------------------------------

echo ""
echo "Creating drinks..."

DRINK_IDS=()

while IFS= read -r drink; do
    name=$(echo "$drink" | jq -r '.name')

    # Transform ingredient keys to IDs in the recipe
    transformed=$(echo "$drink" | jq -c '
        .recipe.ingredients = [.recipe.ingredients[] | {
            ingredient_id: .key,
            amount: .amount,
            unit: .unit
        } | del(.key)]
    ')

    # Replace key placeholders with actual IDs
    for key in "${!INGREDIENT_IDS[@]}"; do
        transformed=$(echo "$transformed" | sed "s/\"ingredient_id\": \"$key\"/\"ingredient_id\": \"${INGREDIENT_IDS[$key]}\"/g")
    done

    id=$(echo "$transformed" | "$MIXOLOGY" drinks create --stdin)
    DRINK_IDS+=("$id")
    echo "  $name: $id"
done < <(jq -c '.[]' "$DATA_DIR/drinks.json")

# -----------------------------------------------------------------------------
# Create Menu
# -----------------------------------------------------------------------------

echo ""
echo "Creating menu..."

MENU=$(printf '%s' '{"name":"Classic Cocktails"}' | "$MIXOLOGY" menu create --stdin)
echo "  Menu: $MENU"

for drink_id in "${DRINK_IDS[@]}"; do
    "$MIXOLOGY" menu add-drink --menu-id "$MENU" --drink-id "$drink_id" > /dev/null
done

"$MIXOLOGY" menu publish --id "$MENU" > /dev/null
echo "  Menu published with ${#DRINK_IDS[@]} drinks"

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------

echo ""
echo "=== Seed Complete ==="
echo ""
echo "Created:"
echo "  - ${#INGREDIENT_IDS[@]} ingredients"
echo "  - ${#DRINK_IDS[@]} classic cocktails"
echo "  - 1 published menu"
echo ""
echo "View the menu with cost analysis:"
echo "  $MIXOLOGY menu show --id $MENU --costs --target-margin 0.7"
echo ""
echo "List all drinks:"
echo "  $MIXOLOGY drinks list"
echo ""
echo "Check inventory:"
echo "  $MIXOLOGY inventory list"
