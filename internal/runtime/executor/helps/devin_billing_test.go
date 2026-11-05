package helps

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func buildDevinUsageBytes(apiProvider uint64, usedProviderPlan uint64) []byte {
	var b []byte
	b = protowire.AppendTag(b, 3, protowire.VarintType)
	b = protowire.AppendVarint(b, 6)
	b = protowire.AppendTag(b, 6, protowire.VarintType)
	b = protowire.AppendVarint(b, apiProvider)
	if usedProviderPlan != 0 {
		b = protowire.AppendTag(b, 13, protowire.VarintType)
		b = protowire.AppendVarint(b, usedProviderPlan)
	}
	return b
}

func TestParseDevinUsageBillingFields(t *testing.T) {
	usage := parseDevinUsageField(buildDevinUsageBytes(87, 1))
	if usage.APIProvider != 87 {
		t.Fatalf("APIProvider = %d, want 87", usage.APIProvider)
	}
	if !usage.UsedProviderPlan {
		t.Fatal("UsedProviderPlan = false, want true")
	}

	quota := parseDevinUsageField(buildDevinUsageBytes(66, 0))
	if quota.APIProvider != 66 || quota.UsedProviderPlan {
		t.Fatalf("quota usage = (%d, %t), want (66, false)", quota.APIProvider, quota.UsedProviderPlan)
	}
}

func TestMergeDevinBillingKeepsLatestProviderAndPlanFlag(t *testing.T) {
	dst := &DevinUsage{APIProvider: 66}
	MergeDevinBilling(dst, &DevinUsage{APIProvider: 87, UsedProviderPlan: true})
	MergeDevinBilling(dst, &DevinUsage{})
	if dst.APIProvider != 87 || !dst.UsedProviderPlan {
		t.Fatalf("merged = (%d, %t), want (87, true)", dst.APIProvider, dst.UsedProviderPlan)
	}
	MergeDevinBilling(nil, dst)
	MergeDevinBilling(dst, nil)
}

func TestDevinBillingSource(t *testing.T) {
	cases := []struct {
		usage *DevinUsage
		want  string
	}{
		{nil, "unknown"},
		{&DevinUsage{}, "unknown"},
		{&DevinUsage{APIProvider: 87, UsedProviderPlan: true}, "provider_plan"},
		{&DevinUsage{APIProvider: 66}, "devin_quota"},
	}
	for _, tc := range cases {
		if got := DevinBillingSource(tc.usage); got != tc.want {
			t.Errorf("DevinBillingSource(%+v) = %q, want %q", tc.usage, got, tc.want)
		}
	}
}

func TestDevinAPIProviderName(t *testing.T) {
	if got := DevinAPIProviderName(87); got != "OPENAI_SIWC" {
		t.Errorf("DevinAPIProviderName(87) = %q", got)
	}
	if got := DevinAPIProviderName(66); got != "66" {
		t.Errorf("DevinAPIProviderName(66) = %q", got)
	}
}
