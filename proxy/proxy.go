package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"api-gateway/config"
)

// ReverseProxy 实现反向代理功能的核心结构
type ReverseProxy struct {
	config   *config.Config
	handlers map[string]http.Handler
	// 将来可以添加中间件链、插件管理器等组件
}

// NewReverseProxy 创建一个新的反向代理实例
func NewReverseProxy(cfg *config.Config) *ReverseProxy {
	proxy := &ReverseProxy{
		config:   cfg,
		handlers: make(map[string]http.Handler),
	}

	// 初始化所有路由处理器
	proxy.initHandlers()

	return proxy
}

// 初始化路由处理器
func (p *ReverseProxy) initHandlers() {
	for _, svc := range p.config.Services {
		// 非常重要：创建一个局部变量来存储每次循环的服务信息
		// 这样每个闭包都会捕获自己的服务配置副本，而不是共享循环变量
		service := svc // 创建服务配置的局部副本

		targetURL, err := url.Parse(service.Target)
		if err != nil {
			log.Printf("解析目标URL失败 %s: %v", service.Target, err)
			continue
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// 自定义Director函数以支持路径修改和请求头设置
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			// 保存原始路径用于日志
			originalPath := req.URL.Path
			log.Printf("DEBUG: 处理请求 - 原始路径: %s, 服务路径: %s, StripPath: %v",
				originalPath, service.Path, service.StripPath)

			// 非常重要：在调用originalDirector之前处理路径
			// 路径处理 - 是否去除前缀路径
			if service.StripPath {
				log.Printf("DEBUG: StripPath=true, 开始路径处理")
				if strings.HasPrefix(req.URL.Path, service.Path) {
					// 去除路径前缀
					newPath := strings.TrimPrefix(req.URL.Path, service.Path)
					// 确保路径始终以/开头
					if newPath == "" || newPath[0] != '/' {
						newPath = "/" + newPath
					}
					log.Printf("DEBUG: 路径前缀匹配成功! %s -> %s", req.URL.Path, newPath)
					req.URL.Path = newPath
					log.Printf("路径重写 [调用原始Director前]: %s -> %s (服务: %s)",
						originalPath, newPath, service.Name)
				} else {
					log.Printf("警告：请求路径 %s 未以 %s 开头，无法移除前缀 (服务: %s)",
						req.URL.Path, service.Path, service.Name)
				}
			} else {
				log.Printf("DEBUG: StripPath=false, 保留原始路径: %s", req.URL.Path)
			}

			log.Printf("DEBUG: 路径处理完成，调用原始Director前的路径: %s", req.URL.Path)
			// 在路径处理之后调用原始Director
			originalDirector(req)
			log.Printf("DEBUG: 原始Director调用后的路径: %s", req.URL.Path)

			// 设置请求头
			for key, value := range service.Headers {
				req.Header.Set(key, value)
			}

			// 打印完整的转发URL
			log.Printf("转发详情: %s %s -> %s%s", req.Method, originalPath, targetURL.Host, req.URL.Path)
		}

		// 添加错误处理
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("代理请求错误 %s: %v", r.URL.Path, err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("代理请求失败"))
		}

		p.handlers[service.Path] = proxy
		log.Printf("已注册服务 [%s] 路径: %s -> %s (strip_path=%v)",
			service.Name, service.Path, service.Target, service.StripPath)
	}
}

// ServeHTTP 实现http.Handler接口
func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 路径匹配 - 查找最长匹配的路径
	var bestMatch string
	var bestHandler http.Handler

	log.Printf("DEBUG: 收到请求，开始路径匹配: %s", r.URL.Path)
	for path, handler := range p.handlers {
		if strings.HasPrefix(r.URL.Path, path) {
			log.Printf("DEBUG: 路径前缀匹配: %s 前缀 %s", r.URL.Path, path)
			// 找到最长匹配
			if len(path) > len(bestMatch) {
				bestMatch = path
				bestHandler = handler
				log.Printf("DEBUG: 更新最佳匹配: %s", bestMatch)
			}
		}
	}

	if bestHandler != nil {
		// 在这里可以添加请求前处理的钩子 (将来的中间件功能)

		log.Printf("接收请求: %s，匹配路径: %s", r.URL.Path, bestMatch)
		bestHandler.ServeHTTP(w, r)

		// 在这里可以添加请求后处理的钩子 (将来的中间件功能)
		return
	}

	// 没有匹配的处理器
	log.Printf("未找到匹配服务: %s", r.URL.Path)
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("服务未找到"))
}
