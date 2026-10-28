package core

// Quota giới hạn số giây dùng mỗi ngày của một domain; Day là ngày (YYYY-MM-DD theo zone) mà UsedSec thuộc về.
type Quota struct {
	LimitSec int64  `json:"limit_sec"`
	UsedSec  int64  `json:"used_sec"`
	Day      string `json:"day"`
}

const (
	MinQuotaSec = 60
	MaxQuotaSec = 1440 * 60
)

func (q *Quota) Clone() *Quota {
	if q == nil {
		return nil
	}
	quota := *q
	return &quota
}

// Exhausted là true khi đã dùng hết giới hạn; quota nil luôn là false.
func (q *Quota) Exhausted() bool {
	return q != nil && q.UsedSec >= q.LimitSec
}

func (q *Quota) Validate() error {
	const zero = 0
	if q == nil {
		return nil
	}
	if q.LimitSec < MinQuotaSec || q.LimitSec > MaxQuotaSec || q.UsedSec < zero {
		return ErrInvalid
	}
	return nil
}
