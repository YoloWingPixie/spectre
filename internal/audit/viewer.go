package audit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const localHost = "127.0.0.1"
const feedbackTokenHeader = "X-Audit-Token"
const feedbackBodyLimit = 1024 * 1024

type viewer struct {
	URL      string
	server   *http.Server
	done     chan struct{}
	serveErr error
	release  func() error
	once     sync.Once
	closeErr error
}

func startViewer(options auditOptions, port int) (*viewer, error) {
	if port < 0 || port > 65535 {
		return nil, invalid("Port must be an integer from 0 to 65535")
	}
	c, err := loadAudit(options)
	if err != nil {
		return nil, err
	}
	options.Project = c.Root
	options.AuditID = c.Audit.ID
	token, err := uuid()
	if err != nil {
		return nil, err
	}
	render := func() ([]byte, error) {
		current, err := loadAudit(options)
		if err != nil {
			return nil, err
		}
		doc, err := readReport(current.Paths.Report, true)
		if err != nil {
			return nil, err
		}
		if err := doc.ready(); err != nil {
			return nil, err
		}
		images, err := collectImages(doc.Report, current.Directory, "", true)
		if err != nil {
			return nil, err
		}
		doc.Report.RepoRoot = current.Root
		return renderReport(doc.Report, images, &viewerConnection{token, doc.Revision})
	}
	if _, err := render(); err != nil {
		return nil, err
	}
	release, err := acquireLock(c.Directory + "/.viewer.lock")
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", localHost, port))
	if err != nil {
		return nil, errors.Join(err, release())
	}
	origin := "http://" + listener.Addr().String()
	v := &viewer{URL: origin + "/", done: make(chan struct{}), release: release}
	v.server = &http.Server{ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		err := func() error {
			navigation := r.Method == http.MethodGet && r.URL.Path == "/" && r.Header.Get("Sec-Fetch-Mode") == "navigate" && r.Header.Get("Sec-Fetch-Dest") == "document"
			if r.Host != listener.Addr().String() || r.Header.Get("Origin") != "" && r.Header.Get("Origin") != origin || r.Header.Get("Sec-Fetch-Site") == "cross-site" && !navigation {
				return &auditError{http.StatusForbidden, "Only the local viewer origin is allowed"}
			}
			if r.URL.Path == "/" && r.Method == http.MethodGet {
				page, err := render()
				if err != nil {
					return err
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, err = w.Write(page)
				return err
			}
			if r.URL.Path != "/feedback" {
				return missing("Not found")
			}
			if r.Header.Get(feedbackTokenHeader) != token {
				return &auditError{http.StatusForbidden, "Invalid viewer token"}
			}
			switch r.Method {
			case http.MethodGet:
				result, err := getFeedback(options)
				if err != nil {
					return err
				}
				return sendJSON(w, http.StatusOK, result)
			case http.MethodPut:
				if r.Header.Get("Origin") != origin {
					return &auditError{http.StatusForbidden, "Feedback writes require the viewer origin"}
				}
				mediaType, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
				if strings.TrimSpace(mediaType) != "application/json" {
					return &auditError{http.StatusUnsupportedMediaType, "Expected application/json"}
				}
				data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, feedbackBodyLimit))
				if err != nil {
					var limit *http.MaxBytesError
					if errors.As(err, &limit) {
						return &auditError{http.StatusRequestEntityTooLarge, "Feedback request is too large"}
					}
					return &auditError{http.StatusBadRequest, "Cannot read feedback request"}
				}
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
					return &auditError{http.StatusBadRequest, "Invalid JSON request"}
				}
				for _, key := range []string{"findingId", "content", "expectedRevision", "reportRevision"} {
					if _, ok := fields[key]; !ok {
						return invalid("%s is required", key)
					}
				}
				var input feedbackUpdate
				if err := json.Unmarshal(data, &input); err != nil {
					return invalid("Invalid feedback request: %v", err)
				}
				result, err := saveFeedback(options, input)
				if err != nil {
					return err
				}
				return sendJSON(w, http.StatusOK, result)
			default:
				return &auditError{http.StatusMethodNotAllowed, "Method not allowed"}
			}
		}()
		if err != nil {
			code := http.StatusInternalServerError
			message := "The operation failed. Saved data was not replaced."
			var problem *auditError
			if errors.As(err, &problem) {
				code = problem.status
				message = problem.message
			}
			_ = sendJSON(w, code, map[string]string{"error": message})
		}
	})}
	go func() { v.serveErr = v.server.Serve(listener); close(v.done) }()
	return v, nil
}
func (v *viewer) close() error {
	v.once.Do(func() {
		err := v.server.Close()
		<-v.done
		if errors.Is(v.serveErr, http.ErrServerClosed) {
			v.serveErr = nil
		}
		v.closeErr = errors.Join(err, v.serveErr, v.release())
	})
	return v.closeErr
}
func sendJSON(w http.ResponseWriter, status int, value any) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, err = w.Write(data)
	return err
}
