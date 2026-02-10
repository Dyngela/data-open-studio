package lib

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// Status represents the execution status of a node
type Status string

const (
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Progress represents a progress update for a node
type Progress struct {
	NodeID   int    `json:"nodeId"`
	NodeName string `json:"nodeName"`
	Status   Status `json:"status"`
	RowCount int64  `json:"rowCount"`
	Message  string `json:"message"`
}

// NewProgress creates a new progress update
func NewProgress(nodeID int, nodeName string, status Status, rowCount int64, message string) Progress {
	return Progress{
		NodeID:   nodeID,
		NodeName: nodeName,
		Status:   status,
		RowCount: rowCount,
		Message:  message,
	}
}

// ProgressFunc is a function that reports progress for a node
type ProgressFunc func(Progress)

// ProgressReporter sends progress updates via NATS
type ProgressReporter struct {
	conn    *nats.Conn
	subject string
	noop    bool
}

// NewProgressReporter creates a new NATS-based progress reporter.
// Best-effort: if NATS connection fails, returns a no-op reporter (never fails the job).
func NewProgressReporter(natsURL, tenantID string, jobID uint) *ProgressReporter {
	subject := fmt.Sprintf("tenant.%s.job.%d.progress", tenantID, jobID)

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Printf("WARNING: NATS connection failed (%s), progress reporting disabled: %v", natsURL, err)
		return &ProgressReporter{noop: true, subject: subject}
	}

	log.Printf("NATS connected, publishing progress to subject: %s", subject)
	return &ProgressReporter{
		conn:    nc,
		subject: subject,
	}
}

// Close drains and closes the NATS connection
func (r *ProgressReporter) Close() {
	if r.noop || r.conn == nil {
		return
	}
	if err := r.conn.Drain(); err != nil {
		log.Printf("NATS drain error: %v", err)
	}
}

// ReportFunc returns a ProgressFunc that publishes updates to NATS
func (r *ProgressReporter) ReportFunc() ProgressFunc {
	if r.noop {
		return func(p Progress) {
			log.Printf("progress (no-op): node=%d status=%s rows=%d", p.NodeID, p.Status, p.RowCount)
		}
	}

	return func(p Progress) {
		data, err := json.Marshal(p)
		if err != nil {
			log.Printf("progress marshal error: %v", err)
			return
		}
		if err := r.conn.Publish(r.subject, data); err != nil {
			log.Printf("progress publish error: %v", err)
		}
	}
}

func PrettyPrintStruct(s interface{}, sep string) string {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return "input is not a struct"
	}

	t := v.Type()
	var parts []string

	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)
		fieldType := t.Field(i)

		// Priority: 'db' tag -> Field Name
		label := fieldType.Tag.Get("db")
		if label == "" {
			label = fieldType.Name
		}

		valStr := "null"

		// Handle sql.Null* types via reflection
		if fieldValue.Kind() == reflect.Struct {
			isValidField := fieldValue.FieldByName("Valid")
			if isValidField.IsValid() && isValidField.Bool() {
				// Index 0 is typically the internal value (String, Int32, etc.)
				actualValue := fieldValue.Field(0).Interface()

				if tm, ok := actualValue.(time.Time); ok {
					valStr = tm.Format("2006-01-02 15:04:05")
				} else {
					valStr = fmt.Sprintf("%v", actualValue)
				}
			}
		} else {
			// Handle standard types (string, int, etc.)
			valStr = fmt.Sprintf("%v", fieldValue.Interface())
		}

		parts = append(parts, fmt.Sprintf("%s: %s", label, valStr))
	}

	// Wrap in markers so you can see start/end in a dense log
	sep = " | "
	return "» " + strings.Join(parts, sep) + " «"
}
