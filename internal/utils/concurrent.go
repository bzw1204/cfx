package utils

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConcurrentExecutor 通用并发执行器
// 用于执行需要并发处理的任务，支持进度回调、错误处理和重试机制
type ConcurrentExecutor struct {
	logger         *zap.Logger
	maxWorkers     int
	retry          int
	retryDelay     time.Duration
	progressMgr    *ProgressManager
	taskID         string
	operationName  string
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	Item   string      // 处理的项
	Result interface{} // 结果数据
	Error  error       // 错误信息
}

// WorkFunc 工作函数类型
type WorkFunc func(item string) (interface{}, error)

// NewConcurrentExecutor 创建新的并发执行器
func NewConcurrentExecutor(logger *zap.Logger, maxWorkers, retry int, retryDelay time.Duration, progressMgr *ProgressManager, taskID, operationName string) *ConcurrentExecutor {
	return &ConcurrentExecutor{
		logger:        logger,
		maxWorkers:    maxWorkers,
		retry:         retry,
		retryDelay:    retryDelay,
		progressMgr:   progressMgr,
		taskID:        taskID,
		operationName: operationName,
	}
}

// Execute 执行并发任务
// items: 要处理的项列表
// workFunc: 处理单个项的函数
// 返回: 成功的结果列表，如果有任何失败则返回错误
// 重试策略: 每轮只重试上一轮失败的节点，已成功的不重复执行
func (ce *ConcurrentExecutor) Execute(items []string, workFunc WorkFunc) ([]ExecuteResult, error) {
	ce.logger.Info("starting_concurrent_execution",
		zap.String("operation", ce.operationName),
		zap.Int("total_items", len(items)),
		zap.Int("max_workers", ce.maxWorkers))

	allResults := make([]ExecuteResult, 0, len(items))
	remaining := items

	for attempt := 1; attempt <= ce.retry; attempt++ {
		ce.logger.Info("execution_attempt",
			zap.Int("attempt", attempt),
			zap.String("operation", ce.operationName),
			zap.Int("remaining_items", len(remaining)))

		results, _ := ce.executeAttempt(remaining, workFunc)
		allResults = append(allResults, results...)

		// 全部成功，提前结束
		if len(allResults) >= len(items) {
			ce.logger.Info("execution_success",
				zap.String("operation", ce.operationName),
				zap.Int("successful_results", len(allResults)))
			return allResults, nil
		}

		// 收集本轮失败的节点用于重试
		successSet := make(map[string]bool, len(results))
		for _, r := range results {
			successSet[r.Item] = true
		}
		failed := make([]string, 0, len(remaining)-len(results))
		for _, item := range remaining {
			if !successSet[item] {
				failed = append(failed, item)
			}
		}
		remaining = failed

		if attempt < ce.retry && len(remaining) > 0 {
			ce.logger.Warn("execution_retry",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", ce.retry),
				zap.Int("failed_count", len(remaining)),
				zap.Duration("retry_delay", ce.retryDelay),
				zap.String("operation", ce.operationName))
			time.Sleep(ce.retryDelay)
		}
	}

	if len(allResults) > 0 {
		ce.logger.Info("execution_partial_success",
			zap.String("operation", ce.operationName),
			zap.Int("successful_results", len(allResults)),
			zap.Int("total_items", len(items)))
		return allResults, nil
	}

	return nil, fmt.Errorf("operation %s failed after %d attempts", ce.operationName, ce.retry)
}

// executeAttempt 执行单次尝试
func (ce *ConcurrentExecutor) executeAttempt(items []string, workFunc WorkFunc) ([]ExecuteResult, error) {
	total := len(items)
	completed := 0
	lastProgressUpdate := time.Now()

	// 信号量控制并发数
	sem := make(chan struct{}, ce.maxWorkers)
	resultChan := make(chan ExecuteResult, total)
	var wg sync.WaitGroup

	// 启动工作goroutines
	for _, item := range items {
		wg.Add(1)
		go func(itemStr string) {

defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					ce.logger.Warn("goroutine_panic_recovered",
						zap.String("item", itemStr),
						zap.Any("panic", r),
						zap.String("operation", ce.operationName))
					resultChan <- ExecuteResult{Item: itemStr, Error: fmt.Errorf("panic: %v", r)}
				}
			}()

			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := workFunc(itemStr)
			resultChan <- ExecuteResult{Item: itemStr, Result: result, Error: err}
		}(item)
	}

	// 收集结果
	var results []ExecuteResult
	var failedResults []ExecuteResult

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		if result.Error != nil {
			failedResults = append(failedResults, result)
		} else {
			results = append(results, result)
		}

		completed++

		// 更新进度（限制更新频率）
		now := time.Now()
		if ce.progressMgr != nil && (now.Sub(lastProgressUpdate) >= time.Second || completed == total) {
			ce.progressMgr.UpdateProgress(ce.taskID, ce.operationName, completed, total, "")
			lastProgressUpdate = now
		}
	}

	ce.logger.Info("attempt_completed",
		zap.String("operation", ce.operationName),
		zap.Int("completed", completed),
		zap.Int("successful", len(results)),
		zap.Int("failed", len(failedResults)))

	return results, nil
}

// ExecuteWithFilter 执行并发任务并过滤结果
// 只返回成功的结果，如果全部失败则返回错误
func (ce *ConcurrentExecutor) ExecuteWithFilter(items []string, workFunc WorkFunc) ([]ExecuteResult, error) {
	results, err := ce.Execute(items, workFunc)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("all items failed for operation %s", ce.operationName)
	}

	return results, nil
}

// ExecuteWithCallback 执行并发任务，每个完成时调用回调函数
func (ce *ConcurrentExecutor) ExecuteWithCallback(items []string, workFunc WorkFunc, callback func(result ExecuteResult)) error {
	total := len(items)
	completed := 0
	lastProgressUpdate := time.Now()

	sem := make(chan struct{}, ce.maxWorkers)
	resultChan := make(chan ExecuteResult, total)
	var wg sync.WaitGroup

	for _, item := range items {
		wg.Add(1)
		go func(itemStr string) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					ce.logger.Warn("goroutine_panic_recovered",
						zap.String("item", itemStr),
						zap.Any("panic", r))
					resultChan <- ExecuteResult{Item: itemStr, Error: fmt.Errorf("panic: %v", r)}
				}
			}()

			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := workFunc(itemStr)
			execResult := ExecuteResult{Item: itemStr, Result: result, Error: err}
			resultChan <- execResult
		}(item)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		if callback != nil {
			callback(result)
		}

		completed++
		now := time.Now()
		if ce.progressMgr != nil && (now.Sub(lastProgressUpdate) >= time.Second || completed == total) {
			ce.progressMgr.UpdateProgress(ce.taskID, ce.operationName, completed, total, "")
			lastProgressUpdate = now
		}
	}

	return nil
}

// GetStats 获取执行统计信息
type ExecutionStats struct {
	TotalItems    int           `json:"total_items"`
	Completed     int           `json:"completed"`
	Successful    int           `json:"successful"`
	Failed        int           `json:"failed"`
	SuccessRate   float64       `json:"success_rate"`
	Duration      time.Duration `json:"duration"`
	Operations    string        `json:"operation"`
	AvgTimePerItem time.Duration `json:"avg_time_per_item"`
}

// ExecuteWithStats 执行并返回统计信息
func (ce *ConcurrentExecutor) ExecuteWithStats(items []string, workFunc WorkFunc) (*ExecutionStats, error) {
	startTime := time.Now()

	results, err := ce.Execute(items, workFunc)

	duration := time.Since(startTime)
	successful := len(results)
	failed := len(items) - successful
	successRate := 0.0
	if len(items) > 0 {
		successRate = float64(successful) / float64(len(items)) * 100
	}

	avgTimePerItem := time.Duration(0)
	if len(items) > 0 {
		avgTimePerItem = duration / time.Duration(len(items))
	}

	stats := &ExecutionStats{
		TotalItems:     len(items),
		Completed:      len(items),
		Successful:     successful,
		Failed:         failed,
		SuccessRate:    successRate,
		Duration:       duration,
		Operations:     ce.operationName,
		AvgTimePerItem: avgTimePerItem,
	}

	ce.logger.Info("execution_stats",
		zap.Int("total", stats.TotalItems),
		zap.Int("successful", stats.Successful),
		zap.Int("failed", stats.Failed),
		zap.Float64("success_rate", stats.SuccessRate),
		zap.Duration("duration", stats.Duration),
		zap.String("operation", stats.Operations))

	return stats, err
}