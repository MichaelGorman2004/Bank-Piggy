package banking

import (
	"context"
	"errors"
	"time"
)

const activityPaginationRestarts = 2

func (s *Service) collectActivity(ctx context.Context, job activityWork, token string) (activityBatch, error) {
	if job.Kind == "investment" {
		return s.collectInvestments(ctx, token)
	}
	for restart := 0; restart <= activityPaginationRestarts; restart++ {
		batch, err := s.collectTransactions(ctx, token, job.Cursor)
		var provider *ProviderError
		if !errors.As(err, &provider) || provider.Code != "activity_pagination_changed" {
			return batch, err
		}
	}
	return activityBatch{}, &ProviderError{Code: "activity_pagination_changed"}
}

func activityWindow() (string, string) {
	now := time.Now().UTC()
	return now.AddDate(0, 0, -activityHistoryDays).Format(time.DateOnly), now.Format(time.DateOnly)
}

func (s *Service) collectTransactions(ctx context.Context, token, cursor string) (activityBatch, error) {
	start, end := activityWindow()
	batch := activityBatch{Cursor: cursor, Start: start, End: end}
	for page := 0; page < maxActivityPages; page++ {
		result, err := s.provider.TransactionsSync(ctx, token, batch.Cursor)
		if err != nil {
			return activityBatch{}, err
		}
		if err := appendTransactionPage(&batch, result); err != nil {
			return activityBatch{}, err
		}
		if !result.HasMore {
			return batch, nil
		}
	}
	return activityBatch{}, errors.New("activity page limit exceeded")
}

func appendTransactionPage(batch *activityBatch, page TransactionPage) error {
	if len(batch.Records)+len(batch.Removed)+len(page.Records)+len(page.Removed) > maxActivityRecords {
		return errors.New("activity record limit exceeded")
	}
	if page.HasMore && (page.NextCursor == "" || page.NextCursor == batch.Cursor) {
		return errors.New("activity cursor did not advance")
	}
	batch.Records = append(batch.Records, page.Records...)
	batch.Removed = append(batch.Removed, page.Removed...)
	batch.Cursor = page.NextCursor
	batch.Complete = page.HistoryComplete
	return nil
}

func (s *Service) collectInvestments(ctx context.Context, token string) (activityBatch, error) {
	start, end := activityWindow()
	batch := activityBatch{Start: start, End: end, Complete: true}
	total := -1
	seen := map[string]bool{}
	for page := 0; page < maxActivityPages; page++ {
		result, err := s.provider.InvestmentTransactions(ctx, token, start, end, len(batch.Records))
		if err != nil {
			return activityBatch{}, err
		}
		if total < 0 {
			total = result.Total
		}
		if err := appendInvestmentPage(&batch, result, total, seen); err != nil {
			return activityBatch{}, err
		}
		if len(batch.Records) == total {
			return batch, nil
		}
	}
	return activityBatch{}, errors.New("investment page limit exceeded")
}

func appendInvestmentPage(batch *activityBatch, page InvestmentPage, total int, seen map[string]bool) error {
	if total < 0 || total > maxActivityRecords || page.Total != total || len(batch.Records)+len(page.Records) > total {
		return errors.New("investment history changed during pagination")
	}
	if len(page.Records) == 0 && len(batch.Records) < total {
		return errors.New("investment history page is incomplete")
	}
	for _, record := range page.Records {
		if err := validateInvestmentRecord(*batch, record, seen); err != nil {
			return err
		}
		seen[record.ProviderID] = true
	}
	batch.Records = append(batch.Records, page.Records...)
	return nil
}

func validateInvestmentRecord(batch activityBatch, record ProviderActivityRecord, seen map[string]bool) error {
	if record.ProviderID == "" || seen[record.ProviderID] {
		return errors.New("duplicate investment activity")
	}
	if string(record.Record.Date) < batch.Start || string(record.Record.Date) > batch.End {
		return errors.New("investment activity outside requested window")
	}
	return nil
}
