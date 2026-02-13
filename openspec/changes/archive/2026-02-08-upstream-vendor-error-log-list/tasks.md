## 1. API Surface

- [x] 1.1 Define management route(s) for vendor error log listing and wire into router
- [x] 1.2 Add request/response models for list pagination and filters

## 2. Storage and Query

- [x] 2.1 Add storage interface method for listing vendor error logs with filters
- [x] 2.2 Implement backend query for list retrieval with pagination and indexes

## 3. Handler Implementation

- [x] 3.1 Implement management handler to validate filters and call storage query
- [x] 3.2 Enforce management permissions for the error log list endpoint

## 4. Tests and Validation

- [x] 4.1 Add handler tests for pagination, vendor filter, and time range filter
- [x] 4.2 Add access control test for unauthorized requests
