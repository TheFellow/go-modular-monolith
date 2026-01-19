# Engineering Onboarding: The Mixology Architecture

Welcome to the **Mixology** codebase. This project is a reference implementation of a production-grade **Modular Monolith** in Go. It demonstrates how to build scalable, maintainable systems without the premature complexity of microservices.

## 1. High-Level Architecture

The system is designed around **strict boundaries** and **unidirectional flow**.

### The "Spine": Middleware Pipelines
Everything enters the system through a pipeline. We don't just "call functions."
*   **Query Pipeline:** optimized for reading. Fast, simple, often cached.
*   **Command Pipeline:** optimized for writing. Heavy, transactional, audit-logged.

**Why?** This guarantees that every operation—whether it's adding a drink or checking inventory—automatically gets logging, metrics, tracing, and authorization without the domain developer writing a single line of boilerplate code.

### The "Brain": Modular Domains (`app/domains/*`)
Business logic lives here. Each domain (e.g., `drinks`, `inventory`, `orders`) is self-contained.
*   **Rule:** Domains **never** import other domains' internals (DAOs, commands).
*   **Rule:** Domains communicate **only** via the public API (Queries) or Events.

## 2. Core Concepts & Patterns

### Event-Driven Communication
We use a **Dispatcher** pattern to decouple modules.
*   **"Fat" Events:** Events carry the *entire* state of the entity (e.g., `DrinkCreated` contains the full `Drink` struct), not just an ID. This prevents "chatty" callbacks where listeners immediately have to query back for data.
*   **Cascading Consistency:** When an `Ingredient` is deleted, we don't just delete the row. An event fires. The `Drinks` domain hears it and soft-deletes recipes using that ingredient. The `Menu` domain hears *that* and removes the drink. The system self-heals.

### Rich Domain Models
We avoid "anemic" models.
*   **Typed IDs:** We don't use `string` for IDs. We use `cedar.EntityUID` throughout the stack—from the CLI flag down to the database row. This prevents accidental ID swapping (e.g., passing a UserID to a DrinkID function).
*   **Boundary Enforcement:** We don't write defensive code inside models (e.g., `if ID == ""`). We enforce validity at the input boundary. If a Model exists in the system, it is valid by definition.

### Persistence: Embedded & Transactional
We use **bstore** (backed by bbolt) for embedded, ACID-compliant storage.
*   **Why?** It keeps the demo zero-dependency while behaving like a real database (transactions, indexes).
*   **Pattern:** All write operations run inside a `UnitOfWork`. If an event handler fails, the entire transaction (command + events) rolls back.

### Authorization: ABAC & "Dual-State"
We use **Cedar** for Attribute-Based Access Control.
*   **The "Dual-State" Check:** Authorization is tricky in a mutable system. If you change a "Wine" (which you *can* edit) to a "Cocktail" (which you *can't*), when do we check?
    *   **IN Check:** Are you allowed to touch the *current* state?
    *   **OUT Check:** Are you allowed to create the *resulting* state?
*   This happens automatically in the middleware.

## 3. Developer Workflow

### Adding a New Feature
1.  **Define the Model:** Start in `app/<domain>/models`. What does the data look like?
2.  **Define the Persistence:** Update `internal/dao`. mapping the model to a `bstore` row.
3.  **Implement the Logic:**
    *   **Reading?** Add a `Query`. Return `*Model` (use `nil` for not found).
    *   **Writing?** Add a `Command`. Take a full model, return a full model.
4.  **Expose it:** Wire it up in `module.go`.

### Key Conventions
*   **Return Pointers:** methods return `*Drink`, not `Drink`. This allows clear `nil` checks for "Not Found" rather than checking for empty structs.
*   **Soft Deletes:** We rarely hard-delete. Use `DeletedAt`. The framework handles filtering these out by default.
*   **Observability:** Don't add logs to your business logic. The middleware handles request logging. Only log warnings/errors that are specific to internal logic flow.

## 4. Where to Start?
*   **`app/drinks/`**: The reference domain. It implements the standard patterns cleanest.
*   **`pkg/middleware/`**: The infrastructure code. Look here to understand how auth and transactions work.
*   **`main/cli/`**: The entry point. See how commands are constructed and executed.