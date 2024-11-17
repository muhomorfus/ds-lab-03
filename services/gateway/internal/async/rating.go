package async

import (
	"context"
	"github.com/RohanPoojary/gomq"
	"github.com/muhomorfus/ds-lab-02/services/gateway/internal/clients/rating"
	"github.com/muhomorfus/ds-lab-02/services/gateway/internal/models"
	"log/slog"
	"net/http"
	"time"
)

func SaveViolationsRetry(broker gomq.Broker, ratingClient *rating.ClientWithResponses) {
	poller := broker.Subscribe(gomq.ExactMatcher("rating.save_violations.retry"))

	go func() {
		for {
			value, ok := poller.Poll()
			if !ok {
				return
			}

			retry, ok := value.(models.Retry)
			if !ok {
				slog.Error("invalid library return_book.retry message type")
				continue
			}

			slog.Info("poller message from rating.save_violations.retry", "message", retry)

			for time.Now().Sub(retry.Time) <= 10*time.Second {
			}

			resp, err := ratingClient.SaveViolationsWithResponse(context.Background(), &rating.SaveViolationsParams{
				Count:     retry.Violations,
				XUserName: retry.Username,
			})
			if err != nil {
				slog.Info("error request, need to retry message")
				broker.Publish("rating.save_violations.retry", models.Retry{
					LibraryUID: retry.LibraryUID,
					BookUUID:   retry.BookUUID,
					Condition:  retry.Condition,
					Violations: retry.Violations,
					Username:   retry.Username,
					Time:       time.Now(),
				})
				continue
			}

			if resp.StatusCode() >= http.StatusInternalServerError {
				slog.Info("request status 500, need to retry message")
				broker.Publish("rating.save_violations.retry", models.Retry{
					LibraryUID: retry.LibraryUID,
					BookUUID:   retry.BookUUID,
					Condition:  retry.Condition,
					Violations: retry.Violations,
					Username:   retry.Username,
					Time:       time.Now(),
				})
				continue
			}
		}
	}()
}
