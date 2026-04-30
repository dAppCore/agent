---
name: onboard
description: Guide new contributors through the codebase
args: [--module]
---

# Interactive Onboarding

This command guides new contributors through the codebase.

## Flow

### 1. Check for Module-Specific Deep Dive

First, check if the user provided a `--module` argument.

- If `args.module` is "tenant":
  - Display the "Tenant Module Deep Dive" section and stop.
- If `args.module` is "admin":
  - Display the "Admin Module Deep Dive" section and stop.
- If `args.module` is "php":
  - Display the "PHP Module Deep Dive" section and stop.
- If `args.module` is not empty but unrecognized, inform the user and show available modules. Then, proceed with the general flow.

### 2. General Onboarding

If no module is specified, display the general onboarding information.

**Welcome Message**
"Welcome to Host UK Monorepo! 👋 Let me help you get oriented."

**Repository Structure**
"This is a federated monorepo with 18 Laravel packages. Each `core-*` directory is an independent git repo."

**Key Modules**
- `core-php`: Foundation framework
- `core-tenant`: Multi-tenancy
- `core-admin`: Admin panel

**Development Commands**
- Run tests: `core go test` / `core php test`
- Format: `core go fmt` / `core php fmt`

### 3. Link to First Task

"Let's find a 'good first issue' for you to work on. You can find them here: https://github.com/host-uk/core-agent/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22"

### 4. Ask User for Interests

Finally, use the `request_user_input` tool to ask the user about their area of interest.

**Prompt:**
"Which area interests you most?
- Backend (PHP/Laravel)
- CLI (Go)
- Frontend (Livewire/Alpine)
- Full stack"

---

## Module Deep Dives

### Tenant Module Deep Dive

**Module**: `core-tenant`
**Description**: Handles all multi-tenancy logic, including tenant identification, database connections, and domain management.
**Key Files**:
- `src/TenantManager.php`: Central class for tenant operations.
- `config/tenant.php`: Configuration options.
**Dependencies**: `core-php`

### Admin Module Deep Dive

**Module**: `core-admin`
**Description**: The admin panel, built with Laravel Nova.
**Key Files**:
- `src/Nova/User.php`: User resource for the admin panel.
- `routes/api.php`: API routes for admin functionality.
**Dependencies**: `core-php`, `core-tenant`

### PHP Module Deep Dive

**Module**: `core-php`
**Description**: The foundation framework, providing shared services, utilities, and base classes. This is the bedrock of all other PHP packages.
**Key Files**:
- `src/ServiceProvider.php`: Registers core services.
- `src/helpers.php`: Global helper functions.
**Dependencies**: None
