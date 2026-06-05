package dashboard

import (
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

func TestDecodeVersionLabel(t *testing.T) {
	dc := converter.GetDefaultDataConverter()
	p, err := dc.ToPayload("v2")
	if err != nil {
		t.Fatalf("ToPayload: %v", err)
	}
	if got := decodeVersionLabel(map[string]*commonpb.Payload{}); got != "" {
		t.Errorf("empty metadata = %q, want \"\"", got)
	}
	if got := decodeVersionLabel(map[string]*commonpb.Payload{metaKeyPizzaVersion: p}); got != "v2" {
		t.Errorf("decoded = %q, want v2", got)
	}
}
