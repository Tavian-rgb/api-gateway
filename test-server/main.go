package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("请指定端口号，例如: go run main.go 3000")
	}

	port := os.Args[1]

	// API服务处理函数
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("服务器收到请求: %s %s", r.Method, r.URL.Path)
		fmt.Fprintf(w, "这是测试服务器，端口: %s，路径: %s\n", port, r.URL.Path)

		// 打印请求头
		fmt.Fprintln(w, "\n请求头:")
		for name, values := range r.Header {
			fmt.Fprintf(w, "%s: %s\n", name, values[0])
		}
	})

	// 启动HTTP服务器
	log.Printf("测试服务器启动在端口 %s", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("无法启动服务器: %v", err)
	}
}
