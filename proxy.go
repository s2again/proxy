package main

import (
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// 反向代理结构体
type Proxy struct {
	target             *url.URL
	cache              *LRUCache
	cacheValidDuration time.Duration
}

// 创建新的反向代理
func NewProxy(targetURL string, cache *LRUCache, cacheDuration time.Duration) *Proxy {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic("Failed to parse target URL:" + err.Error())
	}
	return &Proxy{
		target:             target,
		cache:              cache,
		cacheValidDuration: cacheDuration,
	}
}

// 反向代理处理函数
func (p *Proxy) ReverseProxy(c *gin.Context) {
	cacheKey := c.Request.URL.Path

	if cachedData, cachedHeader, ok := cache.Get(cacheKey); ok {
		// 如果缓存存在，直接返回缓存数据和头部信息
		for key, values := range cachedHeader {
			for _, value := range values {
				c.Header(key, value)
			}
		}
		c.Data(http.StatusOK, c.ContentType(), cachedData)
		return
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(p.target)
	reverseProxy.Transport = &ProxyTransport{http.DefaultTransport} // 通过Transport删除X-Forwarded-For Header
	reverseProxy.Director = func(req *http.Request) {
		req.Host = p.target.Host // 设置请求的Host为p.target.Host
		req.URL.Scheme = p.target.Scheme
		req.URL.Host = p.target.Host // 设置请求的Host为p.target.Host
		req.URL.Path = singleJoiningSlash(p.target.Path, c.Request.URL.Path)
		req.URL.RawPath = p.target.RawPath
		req.URL.RawQuery = c.Request.URL.RawQuery
		req.URL.Fragment = p.target.Fragment
		req.Method = c.Request.Method
		req.Body = c.Request.Body
		req.GetBody = c.Request.GetBody
		req.ContentLength = c.Request.ContentLength
		req.TransferEncoding = c.Request.TransferEncoding
		req.Close = c.Request.Close
		req.Header = make(http.Header, len(c.Request.Header))
		for k, v := range c.Request.Header {
			req.Header[k] = v
		}
	}
	reverseProxy.ModifyResponse = func(resp *http.Response) error {
		// 只缓存状态码200的响应
		if resp.StatusCode != http.StatusOK {
			return nil
		}
		// 将响应信息保存到缓存块
		resp.Header.Del("Cache-control") // 不保存Cache-control信息
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return nil
		}
		cache.Put(cacheKey, respBody, resp.Header, int(resp.ContentLength), time.Now().Add(p.cacheValidDuration))

		// 将响应头部信息返回给客户端
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}
		// 将响应体写入到客户端
		c.Data(http.StatusOK, c.ContentType(), respBody)
		return nil
	}
	reverseProxy.ServeHTTP(c.Writer, c.Request)
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

type ProxyTransport struct {
	http.RoundTripper
}

func (t *ProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Del("X-Forwarded-For")
	return t.RoundTripper.RoundTrip(req)
}
