// Package-level helpers for Worker Deployment Version metadata.
package dashboard

import (
	"context"
	"fmt"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

// metaKeyPizzaVersion is the version-metadata key each worker upserts at startup
// to publish its friendly shape label (v1/v2/v3) for its Build ID.
const metaKeyPizzaVersion = "pizzaVersion"

// decodeVersionLabel extracts the friendly label from a version's metadata map.
// Returns "" when the key is absent or cannot be decoded (caller falls back to
// CreateTime ordering).
func decodeVersionLabel(metadata map[string]*commonpb.Payload) string {
	p, ok := metadata[metaKeyPizzaVersion]
	if !ok {
		return ""
	}
	var label string
	if err := converter.GetDefaultDataConverter().FromPayload(p, &label); err != nil {
		return ""
	}
	return label
}

// fetchVersionLabel describes one version and decodes its pizzaVersion label.
// Returns "" (no error) when the version has no such metadata yet.
func fetchVersionLabel(ctx context.Context, c client.Client, deploymentName, buildID string) (string, error) {
	h := c.WorkerDeploymentClient().GetHandle(deploymentName)
	desc, err := h.DescribeVersion(ctx, client.WorkerDeploymentDescribeVersionOptions{BuildID: buildID})
	if err != nil {
		return "", fmt.Errorf("describe version %q: %w", buildID, err)
	}
	return decodeVersionLabel(desc.Info.Metadata), nil
}
