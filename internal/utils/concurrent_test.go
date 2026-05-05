package utils

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestConcurrentExecutor_Execute(t *testing.T) {
	logger := zap.NewNop()
	progressMgr := NewProgressManager(logger)

	executor := NewConcurrentExecutor(logger, 2, 1, 1*time.Second, progressMgr, "test", "测试操作")

	items := []string{"item1", "item2", "item3", "item4", "item5"}

	// 成功的测试函数
	successFunc := func(item string) (interface{}, error) {
		return "result_" + item, nil
	}

	results, err := executor.Execute(items, successFunc)
	if err != nil {
		t.Errorf("执行应该成功: %v", err)
	}

	if len(results) != len(items) {
		t.Errorf("期望 %d 个结果, 实际 %d", len(items), len(results))
	}

	// 验证结果
	for i, result := range results {
		if result.Item != items[i] {
			t.Errorf("期望项目 %s, 实际 %s", items[i], result.Item)
		}
		if result.Error != nil {
			t.Errorf("结果不应该有错误: %v", result.Error)
		}
	}
}

func TestConcurrentExecutor_ExecuteWithFailures(t *testing.T) {
	logger := zap.NewNop()
	progressMgr := NewProgressManager(logger)

	executor := NewConcurrentExecutor(logger, 2, 1, 1*time.Second, progressMgr, "test", "测试操作")

	items := []string{"item1", "item2", "item3"}

	// 总是失败的测试函数
	failFunc := func(item string) (interface{}, error) {
		return nil, errors.New("模拟失败")
	}

	results, err := executor.ExecuteWithFilter(items, failFunc)
	if err == nil {
		t.Error("执行应该失败")
	}

	if results != nil {
		t.Error("失败时应该返回nil结果")
	}
}

func TestConcurrentExecutor_ExecuteWithPartialFailures(t *testing.T) {
	logger := zap.NewNop()
	progressMgr := NewProgressManager(logger)

	executor := NewConcurrentExecutor(logger, 2, 1, 1*time.Second, progressMgr, "test", "测试操作")

	items := []string{"good1", "bad", "good2"}

	// 部分成功的测试函数
	partialFunc := func(item string) (interface{}, error) {
		if item == "bad" {
			return nil, errors.New("模拟失败")
		}
		return "result_" + item, nil
	}

	results, err := executor.ExecuteWithFilter(items, partialFunc)
	if err != nil {
		t.Errorf("部分成功时不应返回错误: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("期望 2 个成功结果, 实际 %d", len(results))
	}

	// 验证成功的项目
	successfulItems := make(map[string]bool)
	for _, result := range results {
		successfulItems[result.Item] = true
	}

	if !successfulItems["good1"] || !successfulItems["good2"] {
		t.Error("应该返回good1和good2的结果")
	}

	if successfulItems["bad"] {
		t.Error("不应该返回bad的结果")
	}
}

func TestConcurrentExecutor_ExecuteWithStats(t *testing.T) {
	logger := zap.NewNop()
	progressMgr := NewProgressManager(logger)

	executor := NewConcurrentExecutor(logger, 2, 1, 1*time.Second, progressMgr, "test", "统计测试")

	items := []string{"item1", "item2", "item3"}

	workFunc := func(item string) (interface{}, error) {
		time.Sleep(10 * time.Millisecond) // 模拟工作
		return "result_" + item, nil
	}

	stats, err := executor.ExecuteWithStats(items, workFunc)
	if err != nil {
		t.Errorf("执行应该成功: %v", err)
	}

	if stats.TotalItems != len(items) {
		t.Errorf("期望总数 %d, 实际 %d", len(items), stats.TotalItems)
	}

	if stats.Successful != len(items) {
		t.Errorf("期望成功数 %d, 实际 %d", len(items), stats.Successful)
	}

	if stats.SuccessRate != 100.0 {
		t.Errorf("期望成功率 100.0, 实际 %.1f", stats.SuccessRate)
	}

	if stats.Operations != "统计测试" {
		t.Errorf("期望操作名 '统计测试', 实际 '%s'", stats.Operations)
	}

	if stats.Duration <= 0 {
		t.Error("执行时间应该大于0")
	}
}

func TestConcurrentExecutor_ExecuteWithCallback(t *testing.T) {
	logger := zap.NewNop()
	progressMgr := NewProgressManager(logger)

	executor := NewConcurrentExecutor(logger, 2, 1, 1*time.Second, progressMgr, "test", "回调测试")

	items := []string{"item1", "item2", "item3"}
	var callbackResults []ExecuteResult

	workFunc := func(item string) (interface{}, error) {
		return "result_" + item, nil
	}

	callback := func(result ExecuteResult) {
		callbackResults = append(callbackResults, result)
	}

	err := executor.ExecuteWithCallback(items, workFunc, callback)
	if err != nil {
		t.Errorf("执行应该成功: %v", err)
	}

	if len(callbackResults) != len(items) {
		t.Errorf("期望 %d 个回调结果, 实际 %d", len(items), len(callbackResults))
	}
}

func TestConcurrentExecutor_Retry(t *testing.T) {
	logger := zap.NewNop()
	progressMgr := NewProgressManager(logger)

	executor := NewConcurrentExecutor(logger, 2, 3, 10*time.Millisecond, progressMgr, "test", "重试测试")

	items := []string{"item1"}
	attemptCount := 0

	workFunc := func(item string) (interface{}, error) {
		attemptCount++
		if attemptCount < 3 {
			return nil, errors.New("模拟失败")
		}
		return "success", nil
	}

	results, err := executor.Execute(items, workFunc)
	if err != nil {
		t.Errorf("最终应该成功: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("期望 1 个结果, 实际 %d", len(results))
	}

	if attemptCount != 3 {
		t.Errorf("期望 3 次尝试, 实际 %d", attemptCount)
	}
}

func TestConcurrentExecutor_PanicRecovery(t *testing.T) {
	logger := zap.NewNop()
	progressMgr := NewProgressManager(logger)

	executor := NewConcurrentExecutor(logger, 1, 1, 1*time.Second, progressMgr, "test", "panic测试")

	items := []string{"panic_item"}

	workFunc := func(item string) (interface{}, error) {
		panic("模拟panic")
	}

	results, err := executor.Execute(items, workFunc)
	if err == nil {
		t.Error("应该因为panic而失败")
	}

	if len(results) != 0 {
		t.Error("panic后应该没有成功结果")
	}
}