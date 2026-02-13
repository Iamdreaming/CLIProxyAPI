## Why

Operators lack a centralized way to review upstream vendor error logs, making incident triage slow and inconsistent. Providing a list-based view enables fast inspection and troubleshooting now that vendor integrations are growing.

## What Changes

- Add a new API capability to query upstream vendor error logs with list-style pagination and filters.
- Add management handler(s) to return error log list entries.
- Add storage/query support for retrieving error log records.

## Capabilities

### New Capabilities
- `vendor-error-log-list`: List and filter upstream vendor error logs for management/operations use.

### Modified Capabilities
-

## Impact

- New management API endpoint(s) for error log listing.
- Storage/query layer additions for error log retrieval.
- Potential updates to auth/permissions and monitoring/observability surfaces.
