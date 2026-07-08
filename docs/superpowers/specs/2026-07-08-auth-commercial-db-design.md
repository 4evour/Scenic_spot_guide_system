# Auth And Commercial Database Design

## Context

The scenic guide project already has partial account infrastructure:

- Go backend supports user registration, username/password login, logout, current-user lookup, admin user management, guest login, and guest upgrade.
- Vue auth store supports guest sessions and guest upgrade.
- The login page only exposes username/password login and guest continuation.
- Self-service password change is not exposed as a dedicated user endpoint.
- Local development can use SQLite, while deployment already has PostgreSQL support through configuration and `docker-compose.yml`.

The target is a real scenic-area commercial deployment, not a demo-only setup.

## Decisions

1. Visitors can enter without creating an account.
2. Ordinary registration succeeds and returns the user to the login page for manual login.
3. Guest upgrade stays seamless: a guest becomes a registered visitor without losing the current user ID, session, avatar preference, or chat history.
4. Registration requires `username` and `password`; `email` remains optional and reserved for later email binding.
5. Usernames must remain globally unique.
6. Production deployments must use PostgreSQL, not SQLite.
7. A self-service password change endpoint must be added.
8. Production database management must include explicit migrations, backups, retention policy, indexes, and observability.

## User Experience

The login screen should support three clear paths:

- Continue as guest: starts or restores a guest session and routes to the visitor experience.
- Login: existing users enter username and password.
- Register: new users enter username, password, and optional email. On success they return to the login form.

Guest visitors should see a lightweight "register account" entry inside the visitor experience. Upgrading from that entry uses the current guest session and keeps the existing data attached to the same account record.

The account area should expose:

- Current account status: guest, visitor, or admin.
- Register account action for guests.
- Change password action for registered visitors and admins.
- Logout action.

## Backend API Design

Existing routes remain:

- `POST /api/v1/auth/guest-login`
- `POST /api/v1/auth/upgrade-guest`
- `POST /api/v1/register`
- `POST /api/v1/login`
- `POST /api/v1/logout`
- `GET /api/v1/user/me`

Add one self-service password endpoint:

```http
PUT /api/v1/user/password
Content-Type: application/json

{
  "current_password": "OldPassword123",
  "new_password": "NewPassword123"
}
```

Rules:

- Requires authentication.
- Rejects guest users.
- Verifies `current_password` with bcrypt before changing the password.
- Validates `new_password` with the existing password policy.
- Stores only the bcrypt hash.
- Keeps the current session valid after a successful change.

Registration behavior:

- `POST /api/v1/register` creates a `visitor` account.
- The response does not set an auth cookie.
- The frontend returns to the login form and asks the user to log in.

Guest upgrade behavior:

- `POST /api/v1/auth/upgrade-guest` keeps the current user record and changes its role from `guest` to `visitor`.
- The backend clears the guest token and display name, sets the new username/password/email fields, and refreshes the auth cookie.
- Existing chat sessions and messages continue to point at the same `user_id`.

## Data Model

The existing `users` table remains the account source of truth:

- `username`: required, unique.
- `password`: bcrypt hash.
- `email`: optional for now.
- `role`: `admin`, `visitor`, or `guest`.
- `guest_token`: set only for guest users.
- `display_name`: used for guest display names.
- `preferred_avatar_id`: user preference.

No new account table is required for the first version.

## Commercial Database Plan

Development may continue using SQLite. Production must use PostgreSQL.

Production requirements:

- Use PostgreSQL as the only production database driver.
- Keep database credentials in environment variables or deployment secrets.
- Use persistent database storage that is independent of application containers.
- Add explicit migration scripts before production release.
- Treat GORM `AutoMigrate` as a development convenience, not the only production migration mechanism.
- Configure database connection pool limits per deployment size.
- Add regular backups and at least one documented restore drill.

Recommended retention:

- User accounts: retain until user deletion or legal/commercial policy requires removal.
- Avatar preferences: retain with the account.
- Chat sessions and messages: retain 180 or 365 days by configuration.
- Interaction logs: retain 90 or 180 days by configuration.
- Aggregated dashboard stats: retain longer than raw logs.

Recommended indexes to verify or add:

- `users.username`
- `users.guest_token`
- `chat_sessions.user_id, chat_sessions.last_active_at`
- `chat_messages.user_id, chat_messages.created_at`
- `interaction_logs.user_id, interaction_logs.created_at`
- `interaction_logs.source, interaction_logs.created_at`

The first commercial version should not introduce sharding, read replicas, or a separate analytics database. PostgreSQL plus indexes, retention, backups, and monitoring is the right first step.

## Error Handling

User-facing errors should be specific enough to guide correction without leaking sensitive detail:

- Login failure: keep the existing generic username/password error.
- Duplicate username: tell the user the username is already used.
- Weak password: return the existing password policy message.
- Wrong current password: return a password verification error.
- Guest upgrade from non-guest account: return a clear "only guest accounts can upgrade" error.

API errors must keep the project's existing `code/message/data` response shape.

## Security

The implementation must preserve the current security posture:

- Store auth in HttpOnly cookies.
- Keep CSRF protection for state-changing authenticated endpoints.
- Keep rate limits on login, registration, and guest login.
- Hash passwords with bcrypt.
- Do not log raw passwords.
- Do not send password hashes to the frontend.

Additional production expectations:

- Configure a strong JWT secret.
- Use secure cookies under HTTPS.
- Add database backups and restore testing before production handoff.
- Avoid committing real database credentials.

## Testing

Backend tests:

- Register visitor with valid username/password.
- Reject duplicate username.
- Reject weak password.
- Guest login creates or restores a guest account.
- Guest upgrade preserves user ID and changes role to visitor.
- Guest upgrade rejects duplicate username.
- Password change rejects guests.
- Password change rejects wrong current password.
- Password change accepts valid current password and new password.
- Old password no longer works after password change.

Frontend checks:

- Login page shows guest, login, and register paths.
- Register success returns to login instead of auto-login.
- Guest route access still works without a prior account.
- Guest upgrade closes the modal and refreshes current user state.
- Password change form handles validation and backend errors.

Production checks:

- PostgreSQL configuration starts successfully through Docker Compose.
- Migrations apply on an empty database.
- Migrations are idempotent in CI or local verification.
- Backup and restore instructions are documented.

## Implementation Scope

In scope for the first implementation:

- Login page registration entry.
- Guest-first visitor experience.
- Guest upgrade polish.
- Self-service password endpoint.
- Frontend password change UI.
- API documentation update.
- Production database guidance documentation.
- Focused backend and frontend verification.

Out of scope for the first implementation:

- Email verification.
- Forgot password by email.
- OAuth/social login.
- Two-factor authentication.
- Multi-tenant scenic-area account isolation.
- Separate analytics warehouse.
- Read replicas or database sharding.

## Approval Gate

This design should be reviewed before implementation. After approval, the next step is an implementation plan that breaks the work into backend API, frontend auth UI, database documentation, and verification tasks.
