package helps

import "strconv"

// devinAPIProviderNames maps observed ModelUsageStats.api_provider values to the
// names Devin's Connect JSON encoding reports. Unknown values are logged raw.
var devinAPIProviderNames = map[uint64]string{
	87: "OPENAI_SIWC",
}

// DevinAPIProviderName returns a readable name for a ModelUsageStats.api_provider value.
func DevinAPIProviderName(provider uint64) string {
	if name, ok := devinAPIProviderNames[provider]; ok {
		return name
	}
	return strconv.FormatUint(provider, 10)
}

// MergeDevinBilling copies billing attribution from a later usage frame into dst.
// The provider plan flag is sticky: once any frame reports it, the turn was paid
// by the linked plan.
func MergeDevinBilling(dst, src *DevinUsage) {
	if dst == nil || src == nil {
		return
	}
	if src.APIProvider != 0 {
		dst.APIProvider = src.APIProvider
	}
	if src.UsedProviderPlan {
		dst.UsedProviderPlan = true
	}
}

// DevinBillingSource classifies who paid for a Devin turn: the linked provider
// plan (for example a ChatGPT subscription), Devin's own quota, or unknown when
// the response carried no billing attribution.
func DevinBillingSource(usage *DevinUsage) string {
	if usage == nil || (usage.APIProvider == 0 && !usage.UsedProviderPlan) {
		return "unknown"
	}
	if usage.UsedProviderPlan {
		return "provider_plan"
	}
	return "devin_quota"
}
