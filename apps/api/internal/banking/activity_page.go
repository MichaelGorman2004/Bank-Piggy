package banking

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"

	"bankpiggy/api/internal/budget"
)

const activityReadPageSize = 200

type activityCursor struct {
	Month       string `json:"m"`
	Scope       string `json:"s"`
	Fingerprint string `json:"f"`
	ID          string `json:"i"`
}

func pageActivity(result *budget.ActivityResponse, encoded string) error {
	fingerprint, err := activityFingerprint(*result)
	if err != nil {
		return err
	}
	start, err := activityPageStart(*result, encoded, fingerprint)
	if err != nil {
		return err
	}
	end := min(start+activityReadPageSize, len(result.Records))
	if end < len(result.Records) {
		cursor := activityCursor{result.Month, result.Scope, fingerprint, result.Records[end-1].ID}
		data, err := json.Marshal(cursor)
		if err != nil {
			return err
		}
		value := base64.RawURLEncoding.EncodeToString(data)
		result.NextCursor = &value
	}
	result.Records = result.Records[start:end]
	return nil
}

func activityPageStart(result budget.ActivityResponse, encoded, fingerprint string) (int, error) {
	if encoded == "" {
		return 0, nil
	}
	if len(encoded) > 1024 {
		return 0, ErrInvalid
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return 0, ErrInvalid
	}
	var cursor activityCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return 0, ErrInvalid
	}
	if !validActivityCursor(cursor, result) {
		return 0, ErrInvalid
	}
	if cursor.Fingerprint != fingerprint {
		return 0, ErrConflict
	}
	for index, record := range result.Records {
		if record.ID == cursor.ID {
			return index + 1, nil
		}
	}
	return 0, ErrInvalid
}

func validActivityCursor(cursor activityCursor, result budget.ActivityResponse) bool {
	return cursor.Month == result.Month && cursor.Scope == result.Scope && cursor.ID != ""
}

func activityFingerprint(result budget.ActivityResponse) (string, error) {
	hash := sha256.New()
	if err := json.NewEncoder(hash).Encode(result); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
