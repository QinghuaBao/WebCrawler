# WebCrawler
one web crawler frame based on golang

# 一、介绍

这是一个用go语言实现的网络爬虫框架，本框架的核心在于可定制和可扩展，用户可以根据自己的需要定制各个模块，同时，也给出了一个实现demo供参考。Go语言的初学者也可以通过这个项目熟悉go语言的各种特性，尤其是并发编程。

#二、项目框架介绍

该网络爬虫主要由五个部分组成：调度器、中间件、下载器、分析器、条目处理器。下面分别介绍每部分的功能。
![网络爬虫框架](http://img.blog.csdn.net/20180314160648815?watermark/2/text/aHR0cDovL2Jsb2cuY3Nkbi5uZXQvdTAxMjQxMjY4OQ==/font/5a6L5L2T/fontsize/400/fill/I0JBQkFCMA==/dissolve/70/gravity/SouthEast)

 - **调度器**对整个爬虫框架进行调度，所有消息都要经过调度器，因此同时肩负着缓存和产生系统信息概要的功能。主要包括辅助工具、系统信息概要、缓存、调度主体四个部分；
 - **中间件**承上启下，辅助调度器调度整个系统，主要包括ID生成器、抽象池、停止信号、通道管理；
 - **下载器**接受输入请求，转化为http请求，下载相应的网页，供分析器分析。设计为多线程，可以同时下载多个页面，并且通过实现一个抽象池来管理线程数量；
 - **分析器**接受作为输入的相应，并且还原为HTTP相应，同时分析生成网页中新的HTTP请求（网页中链接）以及要进一步分析的条目。设计为多线程，可以同时处理多个相应，并且通过实现一个抽象池来管理线程数量；
 - **条目处理管道**包括多个条目处理器，接受分析器传递的条目，最后给出处理结果。可以根据自己的需要定制分析方式，后文中将给出一个条目处理管道demo。

#三、数据流

![爬虫数据流](http://img.blog.csdn.net/20180314113910793?watermark/2/text/aHR0cDovL2Jsb2cuY3Nkbi5uZXQvdTAxMjQxMjY4OQ==/font/5a6L5L2T/fontsize/400/fill/I0JBQkFCMA==/dissolve/70/gravity/SouthEast)

输入首次请求的网站之后，下载响应网站形成响应数据传送给分析器，分析器分离其中的链接和要处理的条目，链接形成下一层请求给调度器缓存起来，条目发送给条目处理管道进一步处理，最后给出最终处理结果。

#四、各模块接口设计

##1、调度器

（1）调度器主体  
主要用于启动和停止整个系统，并且从中获取一些系统运行的状态。
```
type Scheduler interface {
	// 开启调度器。
	// 调用该方法会使调度器创建和初始化各个组件。在此之后，调度器会激活爬取流程的执行。
	// 参数channelArgs代表通道参数的容器。
	// 参数poolBaseArgs代表池基本参数的容器。
	// 参数crawlDepth代表了需要被爬取的网页的最大深度值。深度大于此值的网页会被忽略。
	// 参数httpClientGenerator代表的是被用来生成HTTP客户端的函数。
	// 参数respParsers的值应为分析器所需的被用来解析HTTP响应的函数的序列。
	// 参数itemProcessors的值应为需要被置入条目处理管道中的条目处理器的序列。
	// 参数firstHttpReq即代表首次请求。调度器会以此为起始点开始执行爬取流程。
	Start(channelArgs base.ChannelArgs,poolBaseArgs base.PoolBaseArgs,crawlDepth uint32,httpClientGenerator GenHttpClient,respParsers []anlz.ParseResponse,itemProcessors []ipl.ProcessItem,firstHttpReq *http.Request) (err error)
	// 调用该方法会停止调度器的运行。所有处理模块执行的流程都会被中止。
	Stop() bool
	// 判断调度器是否正在运行。
	Running() bool
	// 获得错误通道。调度器以及各个处理模块运行过程中出现的所有错误都会被发送到该通道。
	// 若该方法的结果值为nil，则说明错误通道不可用或调度器已被停止。
	ErrorChan() <-chan error
	// 判断所有处理模块是否都处于空闲状态。
	Idle() bool
	// 获取摘要信息。
	Summary(prefix string) SchedSummary
}
```
（2）系统信息摘要  
通过此接口获取系统的整体情况
```
type SchedSummary interface {
	String() string               // 获得摘要信息的一般表示。
	Detail() string               // 获取摘要信息的详细表示。
	Same(other SchedSummary) bool // 判断是否与另一份摘要信息相同。
}
```
（3）缓存  
一般来说一个页面会有多个链接，这样会导致深一层的请求远多于下载器的下载线程，所以需要先把一部分请求缓存起来，等到下载线程空闲再发送出去。
```
type requestCache interface {
	// 将请求放入请求缓存。
	put(req *base.Request) bool
	// 从请求缓存获取最早被放入且仍在其中的请求。
	get() *base.Request
	// 获得请求缓存的容量。
	capacity() int
	// 获得请求缓存的实时长度，即：其中的请求的即时数量。
	length() int
	// 关闭请求缓存。
	close()
	// 获取请求缓存的摘要信息。
	summary() string
}
```
（4）监控器  
监控整个系统的运行，在所有线程都结束之后及时返回信号，以便于调度器主体终止整个系统。

```
// scheduler代表作为监控目标的调度器。
// intervalNs代表检查间隔时间，单位：纳秒。
// maxIdleCount代表最大空闲计数。
// autoStop被用来指示该方法是否在调度器空闲一段时间（即持续空闲时间，由intervalNs * maxIdleCount得出）之后自行停止调度器。
// detailSummary被用来表示是否需要详细的摘要信息。
// record代表日志记录函数。
func Monitoring(scheduler sched.Scheduler, intervalNs time.Duration, maxIdleCount uint, autoStop bool, detailSummary bool, record Record) <-chan uint64
```

##2、中间件
（1）ID生成器  
生成一个系统中唯一的ID，保证下载器池和分析器池的唯一性。
```
type IdGenerator interface {
	GetUint32() uint32 // 获得一个uint32类型的ID。
}
```

（2）抽象池  
从下载器池和分析器池中抽象出来的票池，方便控制goroutine的数量。其中，Entity可以理解为上一个接口中的唯一ID实体。
```
type Entity interface {
 	Id() uint32
 }

type Pool interface {
	Take() (Entity, error)
	Return(entity Entity) error
	Total() uint32
	Used() uint32
}
```
（3）停止信号  
由于本框架是多线程的，所以需要一个信号来关闭整个系统。
```
type StopSign interface {
	// 置位停止信号。相当于发出停止信号。
	// 如果先前已发出过停止信号，那么该方法会返回false。
	Sign() bool
	// 判断停止信号是否已被发出。
	Signed() bool
	// 重置停止信号。相当于收回停止信号，并清除所有的停止信号处理记录。
	Reset()
	// 处理停止信号。
	// 参数code应该代表停止信号处理方的代号。该代号会出现在停止信号的处理记录中。
	Deal(code string)
	// 获取某一个停止信号处理方的处理计数。该处理计数会从相应的停止信号处理记录中获得。
	DealCount(code string) uint32
	// 获取停止信号被处理的总计数。
	DealTotal() uint32
	// 获取摘要信息。其中应该包含所有的停止信号处理记录。
	Summary() string
}

```

（4）通道管理  
由于本系统是多线程的，所以将通道管理放在一起方便管理，分别管理请求通道、响应通道、条目处理通道、错误通道。
```
type ChannelManager interface {
	// 初始化通道管理器。
	// 参数channelArgs代表通道参数的容器。
	// 参数reset指明是否重新初始化通道管理器。
	Init(channelArgs base.ChannelArgs, reset bool) bool
	// 关闭通道管理器。
	Close() bool
	// 获取请求传输通道。
	ReqChan() (chan base.Request, error)
	// 获取响应传输通道。
	RespChan() (chan base.Response, error)
	// 获取条目传输通道。
	ItemChan() (chan base.Item, error)
	// 获取错误传输通道。
	ErrorChan() (chan error, error)
	// 获取通道管理器的状态。
	Status() ChannelManagerStatus
	// 获取摘要信息。
	Summary() string
}

```
##3、下载器
用于根据请求下载响应的网页。
```
type PageDownloader interface {
	Id() uint32                                        // 获得ID。
	Download(req base.Request) (*base.Response, error) // 根据请求下载网页并返回响应。
}

```
##4、分析器
获取下载器下载的内容，进一步分析，剥离出下一层请求和需要处理的条目。
```
type Analyzer interface {
	Id() uint32 // 获得ID。
	Analyze(respParsers []ParseResponse, resp base.Response) ([]base.Data, []error) // 根据规则分析响应并返回请求和条目。
}
```
##5、条目处理管道
分别处理每一个条目，并且通过管道发送错误信息。
```
type ItemPipeline interface {
	// 发送条目。
	Send(item base.Item) []error
	// FailFast方法会返回一个布尔值。该值表示当前的条目处理管道是否是快速失败的。
	// 这里的快速失败是指：只要对某个条目的处理流程在某一个步骤上出错，
	// 那么条目处理管道就会忽略掉后续的所有处理步骤并报告错误。
	FailFast() bool
	// 设置是否快速失败。
	SetFailFast(failFast bool)
	// 获得已发送、已接受和已处理的条目的计数值。
	// 更确切地说，作为结果值的切片总会有三个元素值。这三个值会分别代表前述的三个计数。
	Count() []uint64
	// 获取正在被处理的条目的数量。
	ProcessingNumber() uint64
	// 获取摘要信息。
	Summary() string
}

```
##6、其他基础结构

```
type Data interface {
	Valid() bool // 数据是否有效。
}

// 请求。
type Request struct {
	httpReq *http.Request // HTTP请求的指针值。
	depth   uint32        // 请求的深度。
}

// 响应。
type Response struct {
	httpResp *http.Response
	depth    uint32      //响应的深度
}

```
#五、具体实现

各接口的具体实现见我的github: https://github.com/hustfoam/WebCrawler
#六、定制demo示例

主要需要实现响应解析函数以及条目处理器函数，其中条目处理函数可以是一系列处理方式，具体见我的github：https://github.com/hustfoam/WebCrawler/blob/master/demo/demo.go

```
type ParseResponse func(httpResp *http.Response, respDepth uint32) ([]base.Data, []error)
type ProcessItem func(item base.Item) (result base.Item, err error)
```

#七、扩展

本框架的初级版可以定制响应解析函数和条目处理函数，根据自己的需求实现。如果有需要的话，下载代码之后更改demo中的ParseResponse、ProcessItem 函数即可定制。更进一步的，可以跟据自己的需求实现上述框架中的接口，改变数据处理方式。

# 参考文献
> *《go语言编程实战》 作者：郝林*  
> *https://github.com/hustfoam/WebCrawler*