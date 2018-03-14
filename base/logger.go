package base

import "github.com/WebCrawler/logging"

/**
 * Created by bqh on 2018/3/10.
 * E-mail:M201672845@hust.edu.cn
 */

// 创建日志记录器。
func NewLogger() logging.Logger {
	return logging.NewSimpleLogger()
}
