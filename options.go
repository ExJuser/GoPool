package GoPool

import (
	"time"
)

type Option func(opts *Options)

func loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}
	return opts
}

// Options 一些实例化协程池时的可选参数
type Options struct {
	//距离上一次使用间隔超过 ExpiryDuration 的worker被认为是过期
	ExpiryDuration time.Duration
	//标志在初始化协程池时是否提前分配内存
	PreAlloc bool
	//因为向协程池提交任务被阻塞的最大协程数量
	MaxBlockingTasks int
	//非阻塞式 不会因为向协程池提交任务被阻塞 直接返回错误
	NonBlocking  bool
	PanicHandler func(interface{})
	Logger       Logger
}

func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

func WithExpiryDuration(expiryDuration time.Duration) Option {
	return func(opts *Options) {
		opts.ExpiryDuration = expiryDuration
	}
}

func WithPreAlloc(preAlloc bool) Option {
	return func(opts *Options) {
		opts.PreAlloc = preAlloc
	}
}
func WithMaxBlockingTasks(maxBlockingTasks int) Option {
	return func(opts *Options) {
		opts.MaxBlockingTasks = maxBlockingTasks
	}
}

func WithNonBlocking(nonBlocking bool) Option {
	return func(opts *Options) {
		opts.NonBlocking = nonBlocking
	}
}

func WithPanicHandler(panicHandler func(interface{})) Option {
	return func(opts *Options) {
		opts.PanicHandler = panicHandler
	}
}

func WithLogger(logger Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}
