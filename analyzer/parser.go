package analyzer

import (
	"github.com/WebCrawler/base"
	"net/http"
)

/**
 * Created by bqh on 2018/3/10.
 * E-mail:M201672845@hust.edu.cn
 */

// 被用于解析HTTP响应的函数类型。
type ParseResponse func(httpResp *http.Response, respDepth uint32) ([]base.Data, []error)
