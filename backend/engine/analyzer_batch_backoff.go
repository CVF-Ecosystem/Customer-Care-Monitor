package engine

import (
	"context"
	"time"
)

// batchBackoffSleep is a package-private test seam; production preserves sleepCtx cancellation.
var batchBackoffSleep = sleepCtx

func waitBatchBackoff(ctx context.Context, batchHadError bool, consecutiveErrors int) {
	delay := 2 * time.Second
	if batchHadError {
		delay = 10 * time.Second
		if consecutiveErrors >= 3 {
			delay = 30 * time.Second
		}
	}
	batchBackoffSleep(ctx, delay)
}
