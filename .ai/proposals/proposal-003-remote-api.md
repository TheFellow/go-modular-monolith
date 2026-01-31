# Proposal 003: Remote API & Client-Server Architecture

**Status:** Proposed
**Type:** Major Architecture Evolution

## Goal
Transition `mixology` from a local-only CLI tool to a true Client-Server application. This involves introducing a persistent server process exposing gRPC and REST interfaces, and refactoring the CLI to act as a remote client.

## Motivation
*   **Real-world Realism:** Most modular monoliths are deployed as servers, not CLIs.
*   **Multiple Clients:** Enables future frontends (Web, Mobile, TUI) to connect to the same logic.
*   **Security:** enforcing authentication/authorization at the network boundary.

## Architecture Changes

### 1. The Server Process
Introduce a new entry point (e.g., `cmd/server`) that:
*   Initializes the `app` (Domains, Dispatcher, Store) just like the current CLI.
*   Starts a gRPC server (e.g., `connect-go` or standard `grpc-go`).
*   Starts an HTTP gateway (using `grpc-gateway` or similar) for REST/JSON access.

### 2. Interface Definition (Protobuf)
Define the application surface using Protocol Buffers (`.proto`).
*   **Service Definition:** Map existing Domain Command/Query methods to gRPC services (e.g., `service DrinksService { rpc Create(...) returns (...); }`).
*   **Code Generation:** Use `buf` to manage generation of:
    *   Go server stubs (`pkg/api/gen/...`).
    *   Go client libraries.
    *   OpenAPI (Swagger) documentation.

### 3. The CLI Refactor
Refactor `main/cli` to stop importing `app/` directly.
*   **Current:** `CLI -> Controller -> App Domain -> DAO`
*   **Proposed:** `CLI -> gRPC Client -> Network -> gRPC Server -> App Domain -> DAO`
*   **Impact:** The CLI becomes a "dumb" client. It parses flags, constructs a Protobuf request, sends it, and formats the Protobuf response.

## Technical Strategy

1.  **Tooling:** Adopt `buf` for modern Protobuf management.
2.  **Transport:** Use `connect-go` for a seamless gRPC/HTTP/Web experience.
3.  **Authentication:** Implement interceptors for token-based auth (e.g., JWT) to replace the current process-local "Principal" context.
4.  **Gradual Migration:**
    *   Step 1: Define `.proto` files for one domain (e.g., Drinks).
    *   Step 2: Generate server stubs and wire them to the existing `app` logic.
    *   Step 3: Update the CLI `drinks` command to use the generated client.
    *   Step 4: Repeat for other domains.
