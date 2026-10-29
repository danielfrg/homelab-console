package mcp

import sdk "github.com/modelcontextprotocol/go-sdk/mcp"

// restartAnnotation marks the one tool that changes the cluster. A rolling
// restart disrupts but does not destroy, and every call starts a new rollout.
func restartAnnotation() *sdk.ToolAnnotations {
	no := false
	return &sdk.ToolAnnotations{ReadOnlyHint: false, IdempotentHint: false, DestructiveHint: &no, OpenWorldHint: &no}
}
