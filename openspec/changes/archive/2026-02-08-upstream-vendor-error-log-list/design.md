## Context

The system integrates multiple upstream vendors and logs error events, but operators lack a centralized, queryable view for those error logs. The proposal introduces a list-based management capability with pagination and filters. Existing storage and management APIs need to expose this data without changing existing behavior for other endpoints.

## Goals / Non-Goals

**Goals:**
- Provide a management API endpoint that lists upstream vendor error logs with pagination and basic filtering.
- Support storage/query access patterns needed for listing error log records efficiently.
- Ensure access control aligns with existing management permissions.

**Non-Goals:**
- Real-time streaming, alerting, or notification features.
- Editing or deleting error logs.
- UI/frontend changes beyond exposing the API.

## Decisions

- **Introduce a dedicated management endpoint for error log listing.**
  - Rationale: Separates operational/admin concerns from core runtime APIs and matches existing management routing patterns.
  - Alternatives: Add to existing vendor endpoints; rejected to avoid mixing concerns and permissions.

- **Define a normalized list response with pagination metadata.**
  - Rationale: Consistent with other list endpoints and enables efficient clients.
  - Alternatives: Cursor-only without metadata; rejected due to poorer operator UX and harder debugging.

- **Add storage query methods targeted at list retrieval with filters.**
  - Rationale: Keeps query logic centralized and testable, supports different backends.
  - Alternatives: Inline SQL in handlers; rejected to maintain separation of concerns.

## Risks / Trade-offs

- **[Performance under high log volume] →** Use indexed query fields (e.g., vendor ID, time range) and pagination defaults.
- **[Permission gaps] →** Reuse existing management auth middleware and add tests for access control.
- **[Schema mismatch across storage backends] →** Define a canonical error log model and map backend fields explicitly.
