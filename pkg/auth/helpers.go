package auth

import (
	"context"
)

// HasPermission checks if user has a specific permission
// This is a placeholder for future permission-based authorization
func HasPermission(ctx context.Context, permission string) bool {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return false
	}

	// For now, map permissions to roles
	// In the future, this could check against a permission database
	switch permission {
	case "projects:create", "projects:update":
		return claims.HasAnyRole("ANALYST", "ADMIN")
	case "projects:delete", "users:manage":
		return claims.HasRole("ADMIN")
	case "projects:read":
		return claims.HasAnyRole("VIEWER", "ANALYST", "ADMIN")
	default:
		return false
	}
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) string {
	userID, _ := GetUserIDFromContext(ctx)
	return userID
}

// GetUserRole retrieves user role from context
func GetUserRole(ctx context.Context) string {
	role, _ := GetUserRoleFromContext(ctx)
	return role
}

// GetUserGroups retrieves user groups from context
func GetUserGroups(ctx context.Context) []string {
	groups, _ := GetUserGroupsFromContext(ctx)
	return groups
}

// GetUserEmail retrieves user email from context
func GetUserEmail(ctx context.Context) string {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return ""
	}
	return claims.Email
}

// IsAuthenticated checks if request is authenticated
func IsAuthenticated(ctx context.Context) bool {
	_, ok := GetClaimsFromContext(ctx)
	return ok
}

// IsAdmin checks if user has ADMIN role
func IsAdmin(ctx context.Context) bool {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return false
	}
	return claims.HasRole("ADMIN")
}

// IsAnalyst checks if user has ANALYST role
func IsAnalyst(ctx context.Context) bool {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return false
	}
	return claims.HasRole("ANALYST")
}

// IsViewer checks if user has VIEWER role
func IsViewer(ctx context.Context) bool {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return false
	}
	return claims.HasRole("VIEWER")
}

// CanAccessResource checks if user can access a resource
// This is a helper for resource-based authorization
func CanAccessResource(ctx context.Context, resourceOwnerID string) bool {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return false
	}

	// Admin can access all resources
	if claims.HasRole("ADMIN") {
		return true
	}

	// User can access their own resources
	if claims.UserID == resourceOwnerID {
		return true
	}

	return false
}

// RequirePermission is a helper function to check permission in handlers
func RequirePermission(ctx context.Context, permission string) error {
	if !HasPermission(ctx, permission) {
		return ErrInsufficientPermissions
	}
	return nil
}

// RequireRoleFunc is a helper function to check role in handlers
func RequireRoleFunc(ctx context.Context, role string) error {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return ErrTokenNotFound
	}

	if !claims.HasRole(role) {
		return ErrInsufficientPermissions
	}

	return nil
}

// RequireAnyRoleFunc is a helper function to check any role in handlers
func RequireAnyRoleFunc(ctx context.Context, roles ...string) error {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return ErrTokenNotFound
	}

	if !claims.HasAnyRole(roles...) {
		return ErrInsufficientPermissions
	}

	return nil
}
