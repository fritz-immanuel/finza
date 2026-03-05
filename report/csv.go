package report

import (
	"bytes"
	"encoding/csv"
	"strconv"

	"github.com/yourusername/moneytracker/domain"
)

func BuildCSV(entries []domain.Entry) ([]byte, error) {
	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)

	if err := w.Write([]string{"id", "date", "time", "amount", "currency", "type", "category", "description", "raw_text", "message_id"}); err != nil {
		return nil, err
	}

	for _, e := range entries {
		row := []string{
			strconv.FormatInt(e.ID, 10),
			e.Timestamp.UTC().Format("2006-01-02"),
			e.Timestamp.UTC().Format("15:04:05"),
			strconv.FormatInt(e.Amount, 10),
			e.Currency,
			string(e.Type),
			e.Category,
			e.Description,
			e.RawText,
			strconv.Itoa(e.MessageID),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}
