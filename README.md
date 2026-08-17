# Coupon Center — 优惠券中心

纯 Go 标准库实现的优惠券中心后端服务，零第三方依赖，开箱即跑。

## 业务说明

管理优惠券的完整生命周期：**模板创建 → 激活发放 → 用户领取 → 下单锁定 → 核销抵扣 → 统计报表**。

- **优惠券模板**：定义券的面额、类型（满减/折扣）、库存、有效期、每人限领。
- **用户券**：用户领取的券实例，状态机 `unused → locked → used`，支持过期扫描。
- **使用记录**：核销后产生的不可变记录，用于统计与对账。
- **规则**：可配置的核销限制（每日限次、单笔上限、品类排除），支持全局或按模板生效。
- **兑换码**：通过码值兑换优惠券，支持单个/批量创建、作废。

> 所有金额字段单位为人民币「分」（int64），避免浮点精度问题。

## 运行

```bash
cd origin
go run ./cmd/server
# 默认监听 :8080，可通过 PORT / ADDR 环境变量修改
```

环境变量：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| PORT | 8080 | 监听端口 |
| ADDR | :PORT | 完整监听地址（优先于 PORT） |
| MAX_PAGE_SIZE | 100 | 分页最大条数 |
| LOG_LEVEL | info | 日志级别：debug/info/warn/error |

## API 一览

### 优惠券模板

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/templates | 创建模板（草稿态） |
| GET | /api/templates | 列表（支持 status/type/category/keyword 筛选 + 分页） |
| GET | /api/templates/{id} | 详情 |
| PUT | /api/templates/{id} | 更新可编辑字段 |
| DELETE | /api/templates/{id} | 删除（已有领券则拒绝） |
| POST | /api/templates/{id}/transition | 状态流转（draft→active⇄paused→archived） |
| POST | /api/templates/batch-archive | 批量归档 |

### 用户券

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/coupons/claim | 领取优惠券 |
| GET | /api/coupons | 列表（支持 user_id/template_id/status 筛选 + 分页） |
| GET | /api/coupons/{id} | 详情 |
| POST | /api/coupons/{id}/lock | 下单锁定 |
| POST | /api/coupons/{id}/unlock | 取消锁定 |
| POST | /api/coupons/{id}/redeem | 核销（需先锁定到该订单） |
| POST | /api/coupons/expire-scan | 过期扫描 |

### 使用记录

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/usage-records | 列表（支持 user_id/template_id/order_id/from/to 筛选 + 分页） |
| GET | /api/usage-records/{id} | 详情 |

### 规则

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/rules | 创建规则 |
| GET | /api/rules | 列表（支持 type/status/template_id 筛选 + 分页） |
| GET | /api/rules/{id} | 详情 |
| PUT | /api/rules/{id} | 更新 |
| DELETE | /api/rules/{id} | 删除 |

### 兑换码

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/redeem-codes | 创建单个兑换码 |
| POST | /api/redeem-codes/batch | 批量生成兑换码 |
| GET | /api/redeem-codes | 列表（支持 template_id/status/keyword 筛选 + 分页） |
| GET | /api/redeem-codes/{id} | 详情 |
| POST | /api/redeem-codes/redeem | 输入码值兑换优惠券 |
| POST | /api/redeem-codes/{id}/void | 作废 |
| DELETE | /api/redeem-codes/{id} | 删除（已兑换的不可删） |

### 统计

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/stats/overview | 全局概览（领券/核销/抵扣总额/核销率） |
| GET | /api/stats/by-template | 按模板分组统计 |
| GET | /api/stats/by-day | 按天分组统计 |
| GET | /api/stats/top-users | 用户核销排行（?n=10） |

### 健康检查

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /healthz | 健康检查 |

## 统一响应格式

```json
{"code": 0, "message": "ok", "data": ...}
```

错误码映射：400 参数校验失败 / 404 记录不存在 / 409 状态冲突或唯一性冲突 / 500 内部错误。

## 工程结构

```
origin/
├── go.mod
├── README.md
├── cmd/server/main.go          # 入口：配置加载、依赖装配、优雅关闭
├── internal/
│   ├── app/app.go              # 依赖装配 store -> service -> handler
│   ├── config/config.go        # 环境变量配置
│   ├── model/                  # 领域模型 + 状态机 + 校验
│   ├── store/                  # Store 接口 + 内存实现
│   ├── service/                # 业务逻辑 + 统计
│   └── handler/                # HTTP 路由 + 处理器
└── pkg/
    ├── httpx/httpx.go          # 统一响应、分页、JSON 解析
    ├── idgen/idgen.go          # Hex ID + base62 短码
    └── logger/logger.go        # 分级日志
```

## 测试

```bash
go test ./...
```
