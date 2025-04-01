package sheet

import (
	"errors"
	"github.com/avast/retry-go/v4"
	"google.golang.org/api/googleapi"
	"log"
	"time"
)

func Retry[T any](fn retry.RetryableFuncWithData[T]) (T, error) {
	return retry.DoWithData(fn,
		retry.OnRetry(func(n uint, err error) {
			log.Printf("(#%d/10) Rate limit exceeded, retrying...\n", n+1)
		}),
		retry.RetryIf(func(err error) bool {
			var x *googleapi.Error
			if errors.As(err, &x) {
				return x.Code == 429
			}
			return false
		}),
		retry.Delay(5*time.Second),
		retry.Attempts(10),
	)
}
