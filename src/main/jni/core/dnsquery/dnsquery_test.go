package dnsquery

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/metacubex/mihomo/component/resolver"
	D "github.com/miekg/dns"
)

type fakeService struct {
	reply func(msg *D.Msg) (*D.Msg, error)
}

func (s fakeService) ServeMsg(_ context.Context, msg *D.Msg) (*D.Msg, error) {
	return s.reply(msg)
}

func question(name string, qtype uint16) *D.Msg {
	m := new(D.Msg)
	m.SetQuestion(D.Fqdn(name), qtype)
	return m
}

func TestRecordingServiceRecordsAnswerAndPassesReplyThrough(t *testing.T) {
	log := NewLog(10)
	svc := &recordingService{next: fakeService{reply: func(msg *D.Msg) (*D.Msg, error) {
		r := new(D.Msg)
		r.SetReply(msg)
		r.Answer = []D.RR{
			&D.CNAME{Hdr: D.RR_Header{Name: "example.com.", Rrtype: D.TypeCNAME}, Target: "edge.example.net."},
			&D.A{Hdr: D.RR_Header{Name: "edge.example.net.", Rrtype: D.TypeA}, A: net.IPv4(93, 184, 216, 34)},
		}
		return r, nil
	}}, log: log}

	reply, err := svc.ServeMsg(context.Background(), question("example.com", D.TypeA))
	if err != nil || reply == nil || len(reply.Answer) != 2 {
		t.Fatalf("reply not passed through: %v %v", reply, err)
	}

	got := log.Snapshot()
	if len(got) != 1 {
		t.Fatalf("want 1 record, got %d", len(got))
	}
	q := got[0]
	if q.Domain != "example.com" || q.Type != "A" || q.Rcode != "NOERROR" || q.Error != "" {
		t.Fatalf("unexpected record: %+v", q)
	}
	if len(q.Answers) != 2 || q.Answers[0] != "edge.example.net" || q.Answers[1] != "93.184.216.34" {
		t.Fatalf("unexpected answers: %v", q.Answers)
	}
	if q.Time.IsZero() || q.Delay < 0 {
		t.Fatalf("missing timing: %+v", q)
	}
}

func TestRecordingServiceRecordsError(t *testing.T) {
	log := NewLog(10)
	svc := &recordingService{next: fakeService{reply: func(*D.Msg) (*D.Msg, error) {
		return nil, errors.New("all DNS requests failed")
	}}, log: log}

	_, err := svc.ServeMsg(context.Background(), question("blocked.test", D.TypeAAAA))
	if err == nil {
		t.Fatal("error not passed through")
	}
	q := log.Snapshot()[0]
	if q.Type != "AAAA" || q.Error != "all DNS requests failed" || q.Rcode != "" || len(q.Answers) != 0 {
		t.Fatalf("unexpected record: %+v", q)
	}
}

func TestLogKeepsNewestFirstWithinCapacity(t *testing.T) {
	log := NewLog(3)
	for _, name := range []string{"a.test", "b.test", "c.test", "d.test", "e.test"} {
		log.Add(Query{Domain: name})
	}
	got := log.Snapshot()
	want := []string{"e.test", "d.test", "c.test"}
	if len(got) != len(want) {
		t.Fatalf("want %d records, got %d", len(want), len(got))
	}
	for i, name := range want {
		if got[i].Domain != name {
			t.Fatalf("record %d: want %s, got %s", i, name, got[i].Domain)
		}
	}

	log.Clear()
	if len(log.Snapshot()) != 0 {
		t.Fatal("clear left records behind")
	}
}

func TestInstallWrapsOnce(t *testing.T) {
	log := NewLog(10)
	prev := resolver.DefaultService
	t.Cleanup(func() { resolver.DefaultService = prev })

	resolver.DefaultService = nil
	Install(log)
	if resolver.DefaultService != nil {
		t.Fatal("recorder installed without a DNS service")
	}

	inner := fakeService{reply: func(msg *D.Msg) (*D.Msg, error) { return msg, nil }}
	resolver.DefaultService = inner
	Install(log)
	Install(log)
	rec, ok := resolver.DefaultService.(*recordingService)
	if !ok {
		t.Fatalf("service not wrapped: %T", resolver.DefaultService)
	}
	if _, nested := rec.next.(*recordingService); nested {
		t.Fatal("service wrapped twice")
	}
}
