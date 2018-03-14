package cookie

import (
	"golang.org/x/net/publicsuffix"
	"net/http"
	"net/http/cookiejar"
)

/**
 * Created by bqh on 2018/3/14.
 * E-mail:M201672845@hust.edu.cn
 */

// 创建 http.CookieJar 类型的值。
func NewCookiejar() http.CookieJar {
	options := &cookiejar.Options{PublicSuffixList: &myPublicSuffixList{}}
	cj, _ := cookiejar.New(options)
	return cj
}

// cookiejar.PublicSuffixList 接口的实现类型。
type myPublicSuffixList struct{}

func (psl *myPublicSuffixList) PublicSuffix(domain string) string {
	suffix, _ := publicsuffix.PublicSuffix(domain)
	return suffix
}

func (psl *myPublicSuffixList) String() string {
	return "Web crawler - public suffix list (rev 1.0) power by 'code.google.com/p/go.net/publicsuffix'"
}
