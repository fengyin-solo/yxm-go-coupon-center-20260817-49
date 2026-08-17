# BUG_REPRO

## Bug 是什么
文件: internal/service/template_service.go、user_coupon_service.go、rule_service.go、redemption_code_service.go 与 internal/service/service.go; 符号: ListTemplates/ListUserCoupons/ListRules/ListRedeemCodes 缺少统一分页参数校验; 机制: 这些方法直接用 `(page-1)*size` 计算切片下标，page 或 size 为 0/负数时产生负下标或异常分页范围，导致 slice bounds panic；正确做法是在服务层入口统一拒绝非法分页参数。

## 如何触发
在初始代码中运行 `go test ./internal/service -run TestAnnotationPaginationRejectsInvalidServiceInput -count=20`。

## 错误信息
在初始代码中运行 `go test ./internal/service -run TestAnnotationPaginationRejectsInvalidServiceInput -count=20`，测试因 `slice bounds out of range` panic 稳定失败。
