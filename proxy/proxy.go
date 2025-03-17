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
		targetURL, err := url.Parse(svc.Target)
		if err != nil {
			log.Printf("解析目标URL失败 %s: %v", svc.Target, err)
			continue
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// 自定义Director函数以支持路径修改和请求头设置
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)

			// 路径处理 - 是否去除前缀路径
			if svc.StripPath {
				path := req.URL.Path
				if strings.HasPrefix(path, svc.Path) {
					req.URL.Path = strings.TrimPrefix(path, svc.Path)
					// 确保路径始终以/开头
					if req.URL.Path == "" || req.URL.Path[0] != '/' {
						req.URL.Path = "/" + req.URL.Path
					}
				}
			}

			// 设置请求头
			for key, value := range svc.Headers {
				req.Header.Set(key, value)
			}
		}

		// 添加错误处理
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("代理请求错误 %s: %v", r.URL.Path, err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("代理请求失败"))
		}

		p.handlers[svc.Path] = proxy
		log.Printf("已注册服务 [%s] 路径: %s -> %s", svc.Name, svc.Path, svc.Target)
	}
}

// ServeHTTP 实现http.Handler接口
func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 路径匹配 - 查找最长匹配的路径
	var bestMatch string
	var bestHandler http.Handler

	for path, handler := range p.handlers {
		if strings.HasPrefix(r.URL.Path, path) {
			// 找到最长匹配
			if len(path) > len(bestMatch) {
				bestMatch = path
				bestHandler = handler
			}
		}
	}

	if bestHandler != nil {
		// 在这里可以添加请求前处理的钩子 (将来的中间件功能)

		log.Printf("转发请求 %s -> %s", r.URL.Path, bestMatch)
		bestHandler.ServeHTTP(w, r)

		// 在这里可以添加请求后处理的钩子 (将来的中间件功能)
		return
	}

	// 没有匹配的处理器
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("服务未找到"))
}
