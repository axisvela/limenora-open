// Package sdk provides the official Apache-2.0 Go SDK for LIMENORA.
// It re-exports the public surface from the AGPL-3.0 core package
// github.com/axisrobo/LIMENORA/backend/pkg/limenora.
package sdk

import (
	"github.com/axisrobo/LIMENORA/backend/pkg/limenora"
)

// Version is the SDK version.
const Version = limenora.Version

// Role is a gateway deployment role.
type Role = limenora.Role

const (
	RoleEdge       = limenora.RoleEdge
	RoleEnterprise = limenora.RoleEnterprise
	RoleControl    = limenora.RoleControl
)

// Info returns a stable GatewayInfo struct.
func Info(role Role) limenora.GatewayInfo {
	return limenora.Info(role)
}
