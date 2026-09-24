package service

import "github.com/blueship581/cybuildprice/backend/internal/constants"

// AlertMatch decides whether a newly submitted offer fires a subscription.
// An alert fires only when:
//   - the new price reaches the configured drop from the subscription baseline, and
//   - the new price is not higher than the subscriber's target price.
func AlertMatch(baseline, target, newPrice, dropPercent float64) bool {
	if baseline <= 0 || newPrice <= 0 {
		return false
	}
	requiredPrice := baseline * (1 - dropPercent/constants.PercentBase)
	return newPrice <= requiredPrice && newPrice <= target
}

// DropPercent returns the percentage decrease of newPrice against baseline.
func DropPercent(baseline, newPrice float64) float64 {
	if baseline <= 0 {
		return 0
	}
	return (baseline - newPrice) / baseline * constants.PercentBase
}
