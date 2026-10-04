// Package dnsquery keeps a bounded in-memory history of the DNS queries that
// reach mihomo's resolver.DefaultService (TUN DNS hijack, the dns outbound).
// The dns.listen server holds its own service reference and is not recorded.
package dnsquery

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/metacubex/mihomo/component/resolver"
	D "github.com/miekg/dns"
)

type Query struct {
	Domain  string    `json:"domain"`
	Type    string    `json:"type"`
	Answers []string  `json:"answers"`
	Rcode   string    `json:"rcode,omitempty"`
	Error   string    `json:"error,omitempty"`
	Delay   int64     `json:"delay"`
	Time    time.Time `json:"time"`
}

type Log struct {
	mu      sync.Mutex
	entries []Query
	next    int
	full    bool
}

func NewLog(capacity int) *Log {
	return &Log{entries: make([]Query, capacity)}
}

func (l *Log) Add(q Query) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries[l.next] = q
	l.next = (l.next + 1) % len(l.entries)
	if l.next == 0 {
		l.full = true
	}
}

// Snapshot returns the recorded queries, newest first.
func (l *Log) Snapshot() []Query {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := l.next
	if l.full {
		n = len(l.entries)
	}
	out := make([]Query, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, l.entries[(l.next-i+len(l.entries))%len(l.entries)])
	}
	return out
}

func (l *Log) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	clear(l.entries)
	l.next = 0
	l.full = false
}

type recordingService struct {
	next resolver.Service
	log  *Log
}

func (s *recordingService) ServeMsg(ctx context.Context, msg *D.Msg) (*D.Msg, error) {
	start := time.Now()
	reply, err := s.next.ServeMsg(ctx, msg)
	if len(msg.Question) > 0 {
		s.log.Add(newQuery(msg.Question[0], reply, err, start))
	}
	return reply, err
}

// Install wraps the current DNS service; executor.ApplyConfig replaces it, so
// call this after every apply.
func Install(log *Log) {
	svc := resolver.DefaultService
	if svc == nil {
		return
	}
	if _, ok := svc.(*recordingService); ok {
		return
	}
	resolver.DefaultService = &recordingService{next: svc, log: log}
}

func newQuery(q D.Question, reply *D.Msg, err error, start time.Time) Query {
	query := Query{
		Domain:  strings.TrimSuffix(q.Name, "."),
		Type:    D.Type(q.Qtype).String(),
		Answers: []string{},
		Delay:   time.Since(start).Milliseconds(),
		Time:    start,
	}
	if err != nil {
		query.Error = err.Error()
	}
	if reply == nil {
		return query
	}
	query.Rcode = D.RcodeToString[reply.Rcode]
	for _, rr := range reply.Answer {
		query.Answers = append(query.Answers, answerValue(rr))
	}
	return query
}

func answerValue(rr D.RR) string {
	switch r := rr.(type) {
	case *D.A:
		return r.A.String()
	case *D.AAAA:
		return r.AAAA.String()
	case *D.CNAME:
		return strings.TrimSuffix(r.Target, ".")
	default:
		return strings.TrimSpace(strings.TrimPrefix(rr.String(), rr.Header().String()))
	}
}
