package pubsub

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSendToOneSub(t *testing.T) {
	p := NewPublisher(100*time.Millisecond, 10)
	c := p.Subscribe()

	p.Publish("hi")

	msg := <-c
	if msg.(string) != "hi" {
		t.Fatalf("expected message hi but received %v", msg)
	} else {
		fmt.Println("msg", msg)
	}
}

func TestSendToMultipleSubs(t *testing.T) {
	p := NewPublisher(100*time.Millisecond, 10)
	var subs []chan interface{}
	subs = append(subs, p.Subscribe(), p.Subscribe(), p.Subscribe())

	p.Publish("hi")

	for _, c := range subs {
		msg := <-c
		if msg.(string) != "hi" {
			t.Fatalf("expected message hi but received %v", msg)
		}
	}
}

func TestEvictOneSub(t *testing.T) {
	p := NewPublisher(100*time.Millisecond, 10)
	s1 := p.Subscribe()
	s2 := p.Subscribe()

	p.Evict(s1)
	p.Publish("hi")
	if _, ok := <-s1; ok {
		t.Fatal("expected s1 to not receive the published message")
	}

	msg := <-s2
	if msg.(string) != "hi" {
		t.Fatalf("expected message hi but received %v", msg)
	}
}

func TestClosePublisher(t *testing.T) {
	p := NewPublisher(100*time.Millisecond, 10)
	var subs []chan interface{}
	subs = append(subs, p.Subscribe(), p.Subscribe(), p.Subscribe())
	p.Close()

	for _, c := range subs {
		if _, ok := <-c; ok {
			t.Fatal("expected all subscriber channels to be closed")
		}
	}
}

const sampleText = "test"

type testSubscriber struct {
	dataCh chan interface{}
	ch     chan error
}

func (s *testSubscriber) Wait() error {
	return <-s.ch
}

func newTestSubscriber(p *Publisher) *testSubscriber {
	ts := &testSubscriber{
		dataCh: p.Subscribe(),
		ch:     make(chan error),
	}
	go func() {
		for data := range ts.dataCh {
			s, ok := data.(string)
			if !ok {
				ts.ch <- fmt.Errorf("Unexpected type %T", data)
				break
			}
			if s != sampleText {
				ts.ch <- fmt.Errorf("Unexpected text %s", s)
				break
			}
		}
		close(ts.ch)
	}()
	return ts
}

// for testing with -race
func TestPubSubRace(t *testing.T) {
	p := NewPublisher(0, 1024)
	var subs []*testSubscriber
	for j := 0; j < 50; j++ {
		subs = append(subs, newTestSubscriber(p))
	}
	for j := 0; j < 1000; j++ {
		p.Publish(sampleText)
	}
	time.AfterFunc(1*time.Second, func() {
		for _, s := range subs {
			p.Evict(s.dataCh)
		}
	})
	for _, s := range subs {
		s.Wait()
	}
}

func BenchmarkPubSub(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p := NewPublisher(0, 1024)
		var subs []*testSubscriber
		for j := 0; j < 50; j++ {
			subs = append(subs, newTestSubscriber(p))
		}
		b.StartTimer()
		for j := 0; j < 1000; j++ {
			p.Publish(sampleText)
		}
		time.AfterFunc(1*time.Second, func() {
			for _, s := range subs {
				p.Evict(s.dataCh)
			}
		})
		for _, s := range subs {
			if err := s.Wait(); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func TestRunPubSub(t *testing.T) {
	s := NewPublisherServer()

	fmt.Println("server running...")

	go s.Sub("hello")
	// 多个订阅者
	go s.Sub("golang")
	go s.Sub("php")

	time.Sleep(time.Millisecond * 10)

	go func() {
		i := 0
		for {
			i++
			s.Pub("hello", fmt.Sprintf("hello channel %d", i))
			if i > 10 {
				break
			}
		}
	}()

	go s.Pub("golang", "golang 123")
	go s.Pub("php", "php 123")
	time.Sleep(time.Millisecond * 100)
	s.Close("hello")
	s.Close("golang")
	s.Close("php")
}

func TestTCPPubSub(t *testing.T) {
	listen, err := net.Listen("tcp", "0.0.0.0:9898")
	if err != nil {
		panic(err)
	}
	fmt.Println("server running...")
	p := GetPublisherInstance()
	for {
		conn, err := listen.Accept()
		if err != nil {
			continue
		}
		go handleRequest(conn, p)
	}
}

var Conns = make(map[string]net.Conn)

func handleRequest(conn net.Conn, p *Publisher) {
	for {
		bytes, _, _ := bufio.NewReader(conn).ReadLine()
		fmt.Println(fmt.Sprintf("request string: [%s]", string(bytes)))
		content := strings.Split(string(bytes), " ")
		if content[0] == "subscribe" {
			topic := content[1]
			addr := conn.RemoteAddr().String()
			Conns[addr+topic] = conn
			go func() {
				c := p.SubscribeTopic(func(v interface{}) bool {
					if s, ok := v.(string); ok && strings.Contains(s, topic) {
						return true
					}
					return false
				})
				for v := range c {
					for k, conn2 := range Conns {
						if k == addr+topic {
							conn2.Write([]byte(fmt.Sprintf("%s topic: %v", topic, v)))
						}
					}
				}

			}()
		} else if content[0] == "publish" {
			topic := content[1]
			go p.Publish(topic + content[2] + "\n")
		} else if content[1] == "quit" {
			topic := content[1]
			c := p.SubscribeTopic(func(v interface{}) bool {
				if s, ok := v.(string); ok && strings.Contains(s, topic) {
					return true
				}
				return false
			})
			p.Evict(c)
			break
		} else {
			fmt.Println("common chat " + string(bytes))
			break
		}
	}
}

type Event struct {
	mu  sync.Mutex
	pub *Publisher
}

func New() *Event {
	return &Event{
		pub: NewPublisher(100*time.Millisecond, bufferSize),
	}
}

func (e *Event) Subscribe() (chan interface{}, func()) {
	e.mu.Lock()

	l := e.pub.Subscribe()
	e.mu.Unlock()

	cancel := func() {
		e.Evict(l)
	}
	return l, cancel
}

func (e *Event) Evict(l chan interface{}) {
	e.pub.Evict(l)
}

func (e *Event) SubscribeTopic() chan interface{} {
	e.mu.Lock()

	var topic func(m interface{}) bool
	topic = func(m interface{}) bool { return true }

	var ch chan interface{}
	if topic != nil {
		ch = e.pub.SubscribeTopic(topic)
	} else {
		// Subscribe to all events if there are no filters
		ch = e.pub.Subscribe()
	}

	e.mu.Unlock()
	return ch
}

type Message struct {
	ID      string
	Status  string
	Content string
	Time    int64
}

func (e *Event) PublishMessage(jm Message) {
	e.mu.Lock()
	e.mu.Unlock()
	e.pub.Publish(jm)
}

func (e *Event) SubscribersCount() int {
	return e.pub.Len()
}

func TestEvent(t *testing.T) {
	e := New()
	go func() {
		c := e.SubscribeTopic()

		for v := range c {
			fmt.Println(v)
		}
	}()
	time.Sleep(10)
	m := Message{
		ID:      "1",
		Status:  "success",
		Content: "Hello",
		Time:    time.Now().Unix(),
	}
	e.PublishMessage(m)
}

func TestRPCEvent(t *testing.T) {
	p := NewPublisher(100*time.Microsecond, 10)
	golang := p.SubscribeTopic(func(v interface{}) bool {
		if key, ok := v.(string); ok {
			if strings.HasPrefix(key, "golang:") {
				return true
			}
		}
		return false
	})

	docker := p.SubscribeTopic(func(v interface{}) bool {
		if key, ok := v.(string); ok {
			if strings.HasPrefix(key, "docker:") {
				return true
			}
		}
		return false
	})

	go p.Publish("wang")
	go p.Publish("golang: https://golang.org")
	go p.Publish("docker: https://www.docker.com")

	time.Sleep(time.Second * 2)

	go func() {
		fmt.Println("golang topic:", <-golang)
	}()

	go func() {
		fmt.Println("docker topic:", <-docker)
	}()

	time.Sleep(time.Second * 3)
	fmt.Println("end")
}

func TestStringSlice(t *testing.T) {
	str := "subscribe hello value123"
	content := strings.Split(str, " ")
	fmt.Println(content[0])
}
