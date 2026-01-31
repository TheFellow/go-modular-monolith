# Proposal 001: Architectural Refactorings

**Status:** Reviewed
**Type:** Refactoring

## Decision Summary

| Proposal | Decision | Rationale |
|----------|----------|-----------|
| 1. Value Object Extraction | **Accepted** → Sprint 035 | Real limitation: no unit conversion exists |
| 2. Query Model Separation | **Declined** | Already implemented via CLI surfaces DTOs |

## 1. Value Object Extraction (Formal Units)

**Current State:**
Primitives like `float64` and simple string labels are used for quantities and units (e.g., `Quantity: 1.5`, `Unit: "oz"`). This relies on implicit knowledge and convention, making the system prone to conversion errors (e.g., accidentally adding milliliters to ounces).

**Proposal:**
Introduce formalized Value Objects for units of measurement.

*   **Implementation:** Create a `pkg/units` or `app/kernel/units` package.
*   **Features:**
    *   Distinct types for `Volume` (ml, oz, cl) and `Mass` (g, kg, lb).
    *   Automatic conversion logic (e.g., `func (v Volume) ToOunces() float64`).
    *   Arithmetic operations that enforce type safety (prevent adding `Mass` to `Volume`).
*   **Impact:**
    *   **Inventory:** Stock levels become strictly typed.
    *   **Recipes:** Ingredient requirements are unambiguous.
    *   **Costing:** Price-per-unit calculations become more robust.

## 2. Query Model Separation (CQRS Light)

**Current State:**
Read operations (Queries) return the exact same Domain Models used for Write operations (Commands). While simple, this limits flexibility. "List" views often require aggregated data (e.g., a drink list that includes "In Stock" status or "Current Margin"), forcing the frontend/CLI to stitch data together or the Domain Model to become bloated with view-only fields.

**Proposal:**
Adopt a "CQRS Light" approach by introducing dedicated Read Models (DTOs) for specific views.

*   **Implementation:**
    *   Keep Domain Models (`models.Drink`) pure, focused on business rules and persistence identity.
    *   Create View Models (e.g., `views.DrinkSummary`, `views.MenuDetail`).
*   **Strategy:**
    *   **Option A (Join-on-Read):** Query handlers compose these views by fetching data from multiple repositories/tables on the fly.
    *   **Option B (Materialized Views):** For high-traffic views, event handlers could update a denormalized "Read Store" optimized for specific queries.
*   **Impact:**
    *   Decouples the "Shape of Data on Disk" from the "Shape of Data on Screen".
    *   Allows optimizing list queries without polluting the core domain logic.
