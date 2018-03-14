package itempipeline

import "github.com/WebCrawler/base"

/**
 * Created by bqh on 2018/3/10.
 * E-mail:M201672845@hust.edu.cn
 */

// 被用来处理条目的函数类型。
type ProcessItem func(item base.Item) (result base.Item, err error)