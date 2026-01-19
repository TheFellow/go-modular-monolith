# Proposal 002: Missing Event Handlers

**Status:** Proposed
**Type:** Technical Debt / Feature

## 1. Handling `IngredientUpdated` (Unit Changes)

**Problem:**
If an ingredient's Unit changes (e.g., from "Bottle" to "Ounce"), existing recipes that use this ingredient become semantically invalid. A recipe calling for "1.0 of [Ingredient]" now means 1 Ounce instead of 1 Bottle, destroying the recipe balance.

**Proposal:**
Implement a handler for `IngredientUpdated` that protects system integrity.

*   **Strategy A (Blocking):**
    *   The command validator checks if the unit is changing.
    *   If changing, it queries for any `Drink` recipes using this ingredient.
    *   If usage is found, the update is rejected with a `Conflict` error: "Cannot change unit; ingredient is used in X recipes."
*   **Strategy B (Flagging):**
    *   Allow the change.
    *   The `IngredientUpdated` handler finds all affected drinks.
    *   It marks them as `Draft` or `Archived` to remove them from active menus.
    *   It emits an alert/log requiring manual intervention to update the recipe quantities.

**Recommendation:** Strategy A is safer for a strict modular monolith; Strategy B is better for larger teams where blocking updates is disruptive. Given this project's strictness, **Strategy A** is preferred for the Command, or **Strategy B** if implemented purely as a reactive Handler.

## 2. Handling `MenuDeleted` (Order Cleanup)

**Problem:**
Currently, Menus cannot be deleted. If this feature is added, there is a risk of orphaning active Orders that reference the deleted menu.

**Proposal:**
Ensure `MenuDeleted` events trigger necessary cleanup in the `Orders` domain.

*   **Implementation:**
    *   Create a `MenuDeleted` event.
    *   Implement an `Orders` domain handler: `MenuDeletedOrderCanceller`.
*   **Logic:**
    *   When a menu is deleted, find all orders linked to that `MenuID` that are in an active state (`Placed`, `Preparing`).
    *   Transition them to `Cancelled`.
    *   Emit `OrderCancelled` events with a reason code (e.g., "Menu Removed").
    *   (Optional) Notify the user/staff.
