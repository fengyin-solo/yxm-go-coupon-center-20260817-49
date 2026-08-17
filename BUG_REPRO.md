# BUG_REPRO

## Bug 是什么
兑换码生命周期链路被破坏：过期判断恒为未过期，按码值查询误用 ID 比较，兑换成功后没有完整写回 redeemed 状态、兑换用户和时间，领券侧还绕过模板结束时间校验。

## 如何触发
在初始代码中运行 `go test ./internal/service -run TestAnnotationRedeemCodeLifecycle -count=20`。

## 错误信息
在初始代码中运行 `go test ./internal/service -run TestAnnotationRedeemCodeLifecycle -count=20`，测试稳定失败，报错包含 `redeem live code: 记录不存在`。
