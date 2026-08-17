# BUG_REPRO

## Bug 是什么
模板生命周期链路存在多处协同错误：分类筛选把匹配分类排除，模板激活没有真正更新状态，领券后未累计发行数，概览统计把已领数量按使用记录计算。

## 如何触发
在初始代码中运行 `go test ./internal/service -run TestAnnotationTemplateLifecycleClaimAndStats -count=20`。

## 错误信息
在初始代码中运行 `go test ./internal/service -run TestAnnotationTemplateLifecycleClaimAndStats -count=20`，测试稳定失败，报错包含 `category/keyword list lost active template`。
