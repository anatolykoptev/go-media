package youtube

import "testing"

// DisableKkdai must actually remove tier 1 — a field that builds the backend
// anyway would leave the uninterruptible goja path reachable while looking
// configured.
func TestDisableKkdaiSkipsTier1(t *testing.T) {
	if New(Config{DisableKkdai: true}).kkd != nil {
		t.Fatal("DisableKkdai=true still constructed the kkdai backend")
	}
	if New(Config{}).kkd == nil {
		t.Fatal("default config lost the kkdai backend")
	}
}
