package endpoint

import (
	"net/url"
	"strings"
)

// NewEndpoint 创建一个新的端点URL
func NewEndpoint(scheme, host string) *url.URL {
	return &url.URL{Scheme: scheme, Host: host}
}

// ParseEndpoint 从多个端点中解析出指定scheme的端点
func ParseEndpoint(endpoints []string, scheme string) (string, error) {
	for _, e := range endpoints {
		u, err := url.Parse(e)
		if err != nil {
			return "", err
		}

		if u.Scheme == scheme {
			return u.Host, nil
		}
	}
	return "", nil
}

// ParseEndpoints 解析多个端点URLs
func ParseEndpoints(endpoints []string) ([]*url.URL, error) {
	var urls []*url.URL
	for _, e := range endpoints {
		u, err := url.Parse(e)
		if err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, nil
}

// Scheme 获取端点URL的scheme，可选择是否使用安全连接
// 例如：scheme="http",isSecure=true 将得到 "https"
func Scheme(scheme string, isSecure bool) string {
	if isSecure {
		return scheme + "s"
	}
	return scheme
}

// NormalizeEndpoint 标准化端点URL
// 确保端点有scheme，默认使用defaultScheme
func NormalizeEndpoint(endpoint string, defaultScheme string) (string, error) {
	if endpoint == "" {
		return "", nil
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = defaultScheme + "://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
