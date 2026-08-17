# BUG_REPRO

## Bug 是什么
核销规则与统计链路存在组合缺陷：全局规则不再作用于模板，订单抵扣上限条件方向错误且模板上限被绕过，异常抵扣校验被关闭，统计订单金额误累计为抵扣金额。

## 如何触发
在初始代码中运行 `go test ./internal/service -run TestAnnotationUsageRulesAndStats -count=20`。

## 错误信息
在初始代码中运行 `go test ./internal/service -run TestAnnotationUsageRulesAndStats -count=20`，测试稳定失败，报错包含 `excluded category was allowed`。
