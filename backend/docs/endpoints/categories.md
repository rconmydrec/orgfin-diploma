# Categories Endpoints

HTTP handlers for managing user-defined expense and income categories, including flat listing, grouped tree view, creation, update, and soft deletion.

## Table of Contents

- [GET /categories/](#get-categories)
- [GET /categories/grouped/](#get-categoriesgrouped)
- [POST /categories/](#post-categories)
- [PUT /categories/:id/](#put-categoriesid)
- [DELETE /categories/:id/](#delete-categoriesid)

---

## GET /categories/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/categories/handler.go`
**Service**: `internal/services/categories/service.go`

### Request

No request body.

### Response

**Success**: HTTP 200

```json
[
  {
    "id": 1,
    "userId": 42,
    "name": "(-) Groceries",
    "parentId": null,
    "isIncome": false,
    "createdAt": "2024-01-15T10:00:00",
    "updatedAt": "2024-01-15T10:00:00",
    "children": []
  }
]
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | "Missing authorization header" |
| Internal error | 500 | "Failed to get categories" |

### Business Logic

- Returns a flat array of all non-deleted categories belonging to the authenticated user.
- Category names are prefixed: `(+)` for income categories, `(-)` for expense categories.
- Child category names include `>>` hierarchy notation (e.g., `(-) Food >> Groceries`).
- Results are ordered by `is_income` descending, then alphabetically by name.
- Children within each parent are sorted alphabetically.
- The `children` array is always returned as empty `[]` in this flat-list endpoint.
- Handler delegates to the categories service, which builds the flat list from repository data.

### Known Gaps / TODOs

- Missing `is_active` check on the user. Inactive/deactivated users can still retrieve categories.

### Tests

- `TestGetCategoriesSuccess` — 200 with correct flat list structure
- `TestGetCategoriesUnauthorized` — 401 when no auth token provided
- `TestGetCategoriesDBError` — 500 when repository returns a database error
- `TestGetCategoriesSuccess` (unit) — 200, categories present in response
- `TestGetCategoriesWithChildrenSorting` — verifies children are sorted alphabetically

---

## GET /categories/grouped/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/categories/handler.go`
**Service**: `internal/services/categories/service.go`

### Request

No request body.

### Response

**Success**: HTTP 200

```json
{
  "income": [
    {
      "id": 2,
      "name": "Salary",
      "children": []
    }
  ],
  "expenses": [
    {
      "id": 1,
      "name": "Groceries",
      "children": [
        { "id": 3, "name": "Supermarket", "children": [] }
      ]
    }
  ]
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | "Missing authorization header" |
| Internal error | 500 | "Failed to get categories" |

### Business Logic

- Returns a tree structure grouped into `income` and `expenses` top-level arrays.
- Category names are NOT prefixed (unlike the flat-list endpoint).
- Only non-deleted categories are returned.
- Parent categories contain their children nested under the `children` field.
- When no categories of a given type exist, the corresponding key may be `null` rather than `[]`.

### Known Gaps / TODOs

- Missing `is_active` check on the user.
- When no categories exist for a type, `income` or `expenses` may be `null` instead of an empty array `[]`.

### Tests

- `TestGetGroupedCategoriesSuccess` — 200 with income/expenses structure
- `TestGetGroupedCategoriesUnauthorized` — 401 when no auth token provided
- `TestGetGroupedCategoriesDBError` — 500 when repository returns a database error
- `TestGetGroupedCategoriesSuccess` (unit) — 200 with correct structure
- `TestGetGroupedCategoriesWithChildren` — children are included in parent nodes

---

## POST /categories/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/categories/handler.go`
**Service**: `internal/services/categories/service.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| name | string | Yes | Category name; `validate:"required"` |
| isIncome | bool | Yes | True for income, false for expense |
| parentId | *int | No | Parent category ID for subcategories |

### Response

**Success**: HTTP 201

```json
{
  "id": 5,
  "userId": 42,
  "name": "Utilities",
  "parentId": null,
  "isIncome": false,
  "createdAt": "2024-01-15T10:00:00",
  "updatedAt": "2024-01-15T10:00:00",
  "children": []
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | "Missing authorization header" |
| Validation failure (missing name) | 422 | "Validation failed" |
| Invalid JSON body | 422 | "Invalid request data" |
| Internal error | 500 | "Failed to create category" |

### Business Logic

- Always creates a new category record regardless of any `id` field sent in the body.
- Sets `is_deleted = false` explicitly in the INSERT statement.
- Duplicate names are allowed; no uniqueness check is performed.
- Parent existence is not validated; setting a non-existent `parentId` is accepted.

### Known Gaps / TODOs

- Missing `is_active` check on the user.

### Tests

- `TestCreateExpenseCategorySuccess` — 201 for expense category
- `TestCreateIncomeCategorySuccess` — 201 with `isIncome: true`
- `TestCreateCategoryMissingName` — 422 when name is absent
- `TestCreateCategoryUnauthorized` — 401 when no auth token
- `TestCreateSubcategorySuccess` — 201 with a valid parentId
- `TestCreateCategoryInvalidJSON` — 422 for malformed body
- `TestCreateCategoryWithLongName` — 201 with a very long name
- `TestCreateCategoryWithSpecialCharacters` — 201 with special characters in name
- `TestCreateCategoryBindError` — 422 on bind failure
- `TestCreateCategoryValidateError` — 422 on validation failure
- `TestCreateCategoryDBError` — 500 on repository error
- `TestCreateCategorySuccess` (unit) — 201 with correct response

---

## PUT /categories/:id/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/categories/handler.go`
**Service**: `internal/services/categories/service.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| name | string | No | Category name (validation not enforced on update — see Known Gaps) |
| isIncome | bool | No | Income/expense flag |
| parentId | *int | No | Parent category ID |

Path parameter `:id` is the category ID to update.

### Response

**Success**: HTTP 200

```json
{
  "id": 5,
  "userId": 42,
  "name": "Updated Name",
  "parentId": null,
  "isIncome": false,
  "createdAt": "2024-01-15T10:00:00",
  "updatedAt": "2024-01-15T10:30:00",
  "children": []
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | "Missing authorization header" |
| Invalid path ID | 400 | "Invalid category ID" |
| Invalid JSON body | 422 | "Invalid request data" |
| Category not found | 404 | "Category not found" |
| Category belongs to another user | 403 | "Access denied" |
| Internal error | 500 | "Failed to update category" |

### Business Logic

- Parses `:id` from the path; returns 400 if it is not a valid integer.
- Fetches the existing category by ID with `is_deleted = false` filter.
- Checks that `category.UserID == authenticatedUserID`; returns 403 if not.
- Updates all mutable fields (name, parentId, isIncome) with the values from the request body.

### Known Gaps / TODOs

- Missing `is_active` check on the user.
- `c.Validate` is not called on update requests, so an empty `name` is silently accepted.
- Any database error from `GetByID` (including connection errors) is reported as 404 "Category not found" instead of distinguishing `sql.ErrNoRows` from other errors.

### Tests

- `TestUpdateCategorySuccess` — 200 with updated fields
- `TestUpdateCategoryNotFound` — 404 for non-existent category
- `TestUpdateCategoryOtherUser` — 403 when category belongs to another user
- `TestUpdateCategoryUnauthorized` — 401 when no auth token
- `TestUpdateCategoryInvalidID` — 400 for non-integer path ID
- `TestUpdateCategoryInvalidID` (unit) — 400 for invalid ID
- `TestUpdateCategoryBindError` — 422 on bind failure
- `TestUpdateCategoryNotFound` (unit) — 404 on repository no-rows
- `TestUpdateCategoryAccessDenied` — 403 on ownership mismatch
- `TestUpdateCategoryDBError` — 500 on repository error
- `TestUpdateCategorySuccess` (unit) — 200 with correct response

---

## DELETE /categories/:id/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/categories/handler.go`
**Service**: `internal/services/categories/service.go`

### Request

No request body. Path parameter `:id` is the category ID to delete.

### Response

**Success**: HTTP 200

```json
{
  "id": 5,
  "userId": 42,
  "name": "Utilities",
  "parentId": null,
  "isIncome": false,
  "createdAt": "2024-01-15T10:00:00",
  "updatedAt": "2024-01-15T10:45:00",
  "children": []
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | "Missing authorization header" |
| Invalid path ID | 400 | "Invalid category ID" |
| Category not found | 404 | "Category not found" |
| Category belongs to another user | 403 | "Access denied" |
| Internal error | 500 | "Failed to delete category" |

### Business Logic

- Parses `:id` from the path; returns 400 if it is not a valid integer.
- Fetches the category by ID with `is_deleted = false` filter.
- Checks that `category.UserID == authenticatedUserID`; returns 403 if not.
- Performs a soft delete by setting `is_deleted = true`.
- Returns the deleted category object in the response.
- Child categories are NOT cascaded — orphan subcategories remain in the database.

### Known Gaps / TODOs

- Missing `is_active` check on the user.
- Any database error from `GetByID` (including connection errors) is reported as 404 "Category not found" instead of distinguishing `sql.ErrNoRows` from other errors.

### Tests

- `TestDeleteCategorySuccess` — 200 with deleted category object
- `TestDeleteCategoryNotFound` — 404 for non-existent category
- `TestDeleteCategoryOtherUser` — 403 when category belongs to another user
- `TestDeleteCategoryUnauthorized` — 401 when no auth token
- `TestDeleteCategoryInvalidID` — 400 for non-integer path ID
- `TestDeleteCategoryNotFound` (unit) — 404 on repository no-rows
- `TestDeleteCategoryAccessDenied` — 403 on ownership mismatch
- `TestDeleteCategoryDBError` — 500 on repository error
- `TestDeleteCategorySuccess` (unit) — 200 with correct response
