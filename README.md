> 核心功能来自 github.com/docker/docker/pkg/pubsub

# <div align="center">一个简单的订阅</div>

# 运行
```go
package main

func main() {
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
    
    go p.Publish("abc")
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
```

# License
Apache License Version 2.0, http://www.apache.org/licenses/