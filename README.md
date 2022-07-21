> 核心功能来自 github.com/docker/docker/pkg/pubsub

# <div align="center">一个简单的订阅</div>

# 运行
```go
func main() {
    s := NewPublisherServer()
    go s.Sub("hello")
    s.Pub("hello", fmt.Sprintf("啊哈哈哈 golang channel %d", i))
    s.Close("hello")
}
```

# License
Apache License Version 2.0, http://www.apache.org/licenses/