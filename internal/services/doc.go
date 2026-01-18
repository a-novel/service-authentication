// Package services implements the business logic layer for authentication operations.
//
// Services orchestrate data access, validation, and external service calls.
// Each service follows the Exec pattern with context-aware request structs.
//
// Key services include:
//   - TokenCreate/TokenRefresh: Authentication token management
//   - CredentialsCreate/Update: User credential lifecycle
//   - ShortCodeCreate/Consume: Verification code handling for email validation and password reset
package services
