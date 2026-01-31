# Proposal 004: Demonstrating Maintainability via Adjacent Features

**Status:** Proposed
**Goal:** Prove the core virtue of the Modular Monolith architecture—**maintainability**—by implementing features that demonstrate "Additive Change" (The Open/Closed Principle applied to Architecture). We want to show that we can add complex business value with near-zero risk of breaking existing core flows.

## 1. The Loyalty Program (Purely Additive)

**Concept:** A system to track customer points and tiers based on order history.

*   **The Feature:**
    *   Customers earn 10 points for every dollar spent.
    *   Accumulating points unlocks "Tiers" (Bronze, Silver, Gold).
*   **The Maintainability Demo:**
    *   We create a new domain `app/domains/loyalty`.
    *   It registers an event handler for `OrderCompleted` (from the `orders` domain).
    *   **Crucial Point:** We do *not* touch a single line of code in the `orders` domain logic. The `Orders` module doesn't even know `Loyalty` exists.
    *   **Outcome:** Adding this major feature has **zero regression risk** for the critical "Place Order" path.

## 2. Smart Procurement (Workflow Extension)

**Concept:** Automating the restocking process when inventory runs low.

*   **The Feature:**
    *   Define "Suppliers" and map them to Ingredients.
    *   When an ingredient drops below its `LowStockThreshold` (triggered by `StockAdjusted`), automatically generate a `PurchaseOrder`.
*   **The Maintainability Demo:**
    *   We creates a new domain `app/domains/procurement`.
    *   It listens to `StockAdjusted` events from `inventory`.
    *   **Crucial Point:** `Inventory` remains focused solely on *tracking* what we have. It doesn't become bloated with *ordering* logic. The complexity of "Purchase Orders" (status, approval, delivery) is completely isolated in the new domain.
    *   **Outcome:** Demonstrates how to extend a business process (Inventory Management) without making the original "God Class" (or "God Module") larger.

## 3. Dynamic Pricing / "Happy Hour" (Safe Inter-Domain Coupling)

**Concept:** Variable pricing based on time or rules.

*   **The Feature:**
    *   Define rules: "Fridays 5pm-7pm: 50% off Cocktails".
    *   Menu display reflects the *current* effective price.
*   **The Maintainability Demo:**
    *   Create a `marketing` domain to manage these rules.
    *   Refactor `Menu` queries to ask `Marketing`: *"What is the effective price adjustment for Drink X?"*
    *   **Crucial Point:** This demonstrates **Safe Synchronous Coupling**.
        *   The `Menu` data (persistence) remains the "Base Price".
        *   The `Menu` display (query) composes data from `Marketing`.
        *   If `Marketing` logic changes (new complex rules), `Menu` storage is unaffected.
    *   **Outcome:** Shows how to separate "State" (Base Price) from "Policy" (Current Price) to keep the core domain stable while enabling volatile business rules.
