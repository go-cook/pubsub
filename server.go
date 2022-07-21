package pubsub

import (
	"fmt"
	"sync"
	"time"
)

type PublisherServer struct {
	Bucket map[string]*Publisher
	m      sync.RWMutex
}

func NewPublisherServer() *PublisherServer {
	return &PublisherServer{
		Bucket: make(map[string]*Publisher),
	}
}

// Sub 订阅主题
func (s *PublisherServer) Sub(topic string) {
	p := s.newPublisher(topic)
	c := p.SubscribeTopic(func(v interface{}) bool {
		if _, ok := v.(string); ok {
			return true
		}
		return false
	})
	for v := range c {
		// TODO 业务处理
		fmt.Println(v)
	}
}

// Pub 向主题发送
func (s *PublisherServer) Pub(topic string, v interface{}) {
	p := s.newPublisher(topic)
	p.Publish(v)
}

// Close 关闭订阅
func (s *PublisherServer) Close(topic string) {
	p := s.newPublisher(topic)
	p.Close()
	delete(s.Bucket, topic)
}

// newPublisher 获取主题订阅
func (s *PublisherServer) newPublisher(topic string) *Publisher {
	s.m.RLock()
	defer s.m.RUnlock()
	p := s.Bucket[topic]
	if p == nil {
		p = NewPublisher(100*time.Millisecond, 10)
		s.Bucket[topic] = p
	}
	return p
}
