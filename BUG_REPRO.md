# BUG_REPRO

## Bug 是什么
文件: internal/store/template_store.go、user_coupon_store.go、usage_record_store.go、redemption_code_store.go、rule_store.go; 符号: Get*/List* 返回值; 机制: 内存仓库把 map 中保存的实体指针原样返回，调用方可以在锁外修改这些指针，绕过 Update 方法和互斥锁，导致仓库内部状态被外部查询结果污染；修复需在各实体 Get/List 返回浅拷贝快照。

## 如何触发
在初始代码中运行 `go test ./internal/service -run TestAnnotationStoreSnapshotsAreImmutable -count=20`。

## 错误信息
在初始代码中运行 `go test ./internal/service -run TestAnnotationStoreSnapshotsAreImmutable -count=20`，测试稳定失败，报错包含 `template pointer mutation leaked into store`。
