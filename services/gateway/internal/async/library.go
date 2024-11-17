package async

import (
	"context"
	"github.com/RohanPoojary/gomq"
	"github.com/muhomorfus/ds-lab-02/services/gateway/internal/clients/library"
	"github.com/muhomorfus/ds-lab-02/services/gateway/internal/models"
	"log/slog"
	"net/http"
	"time"
)

func ReturnBookRetry(broker gomq.Broker, libraryClient *library.ClientWithResponses) {
	poller := broker.Subscribe(gomq.ExactMatcher("library.return_book.retry"))

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

			for time.Now().Sub(retry.Time) <= 10*time.Second {
			}

			resp, err := libraryClient.ReturnBookWithResponse(context.Background(), retry.LibraryUID, retry.BookUUID, library.ReturnBookJSONRequestBody{
				Condition: library.ReturnBookRequestCondition(retry.Condition),
			})
			if err != nil {
				broker.Publish("library.return_book.retry", models.Retry{
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
				broker.Publish("library.return_book.retry", models.Retry{
					LibraryUID: retry.LibraryUID,
					BookUUID:   retry.BookUUID,
					Condition:  retry.Condition,
					Violations: retry.Violations,
					Username:   retry.Username,
					Time:       time.Now(),
				})
				continue
			}

			if resp.JSON200 == nil {
				continue
			}

			if resp.JSON200.Violation {
				broker.Publish("rating.save_violations.retry", models.Retry{
					LibraryUID: retry.LibraryUID,
					BookUUID:   retry.BookUUID,
					Condition:  retry.Condition,
					Violations: retry.Violations + 1,
					Username:   retry.Username,
					Time:       time.Now(),
				})
				continue
			}

			broker.Publish("rating.save_violations.retry", models.Retry{
				LibraryUID: retry.LibraryUID,
				BookUUID:   retry.BookUUID,
				Condition:  retry.Condition,
				Violations: retry.Violations,
				Username:   retry.Username,
				Time:       time.Now(),
			})
		}
	}()
}
