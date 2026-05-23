# ⚔️ 梦幻西游物价追踪系统

**梦幻西游商品价格记录与分析系统** — 跟踪《梦幻西游》游戏中商品价格走势的 Web 应用，支持 AI 图片识别自动录入价格。

---

## 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| 后端 | Go (1.25) + SQLite | 纯 Go 实现，无 CGO 依赖 |
| 前端 | Vue3 + Element Plus + ECharts | Vite 构建，Pinia 状态管理 |
| 认证 | JWT + bcrypt | 24 小时令牌，服务端无状态 |
| OCR | OpenAI 兼容 API | 调用 AI 模型识别游戏截图 |
| 数据库 | SQLite | `modernc.org/sqlite`，单文件存储 |
| 部署 | 前端嵌入 Go 二进制 | 单文件部署，监听 `0.0.0.0:8081` |

---

## 项目结构

```
xyq_product/
├── server/                   # Go 后端
│   ├── main.go               # 入口，路由注册，中间件
│   ├── go.mod
│   ├── database/
│   │   ├── db.go             # 数据库初始化 + 建表
│   │   └── seed.go           # Excel 导入 + 初始数据
│   │   ├── handlers/
│   │   │   ├── auth.go           # 登录认证
│   │   │   ├── category.go       # 分类查询
│   │   │   ├── product.go        # 商品 CRUD
│   │   │   ├── price.go          # 价格 CRUD + 批量 + 走势
│   │   │   ├── ocr.go            # AI 图片识别
│   │   │   └── settings.go       # 系统设置 + 密码修改
│   ├── middleware/
│   │   └── auth.go           # JWT 中间件
│   ├── models/
│   │   └── models.go         # 数据结构定义
│   └── static/               # 前端构建产物（嵌入二进制）
├── web/                      # Vue3 前端
│   ├── src/
│   │   ├── api/index.js      # API 客户端封装
│   │   ├── router/index.js   # 路由（登录/总览/商品管理/走势）
│   │   ├── stores/user.js    # Pinia 用户状态
│   │   ├── views/
│   │   │   ├── Login.vue     # 登录页
│   │   │   ├── Dashboard.vue # 价格总览（分类 Tab + 搜索）
│   │   │   ├── Products.vue  # 商品管理（CRUD + OCR）
│   │   │   └── Trend.vue     # 走势分析（ECharts 折线图）
│   │   └── components/
│   │       ├── OcrDialog.vue        # 拍照识别录入对话框
│   │       ├── PriceTrendDialog.vue  # 价格走势弹窗（ECharts 折线图）
│   │       ├── SettingsDialog.vue    # 系统设置弹窗（OCR 配置 + 密码修改）
│   │       ├── PriceTable.vue
│   │       ├── ProductForm.vue
│   │       └── TrendChart.vue
│   └── vite.config.js        # 代理 + 构建输出配置
├── data/mhxy.db              # SQLite 数据库文件
├── build.sh                  # 一键构建脚本
├── test.sh                   # API 自动化测试
├── 梦幻将军物价表.xlsx       # Excel 数据源
└── xyq_product               # 编译后的二进制
```

---

## 数据库模型

系统使用 5 张 SQLite 表：

- **categories** — 商品分类（消耗品、炼妖石、宝石...共 13 类）
- **products** — 商品（关联分类，`name` 唯一约束，含备注）
- **prices** — 价格记录（商品+日期联合唯一）
- **admins** — 管理员（bcrypt 密码哈希）
- **settings** — 系统配置（key-value 存储，如 OCR 地址/模型）

---

## API 接口一览

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| POST | `/api/login` | 否 | 管理员登录 |
| GET | `/api/categories` | 否 | 获取分类列表 |
| GET | `/api/products` | 否 | 商品列表（支持分类/关键词筛选） |
| POST | `/api/products` | 是 | 新增商品 |
| PUT | `/api/products/:id` | 是 | 编辑商品 |
| DELETE | `/api/products/:id` | 是 | 删除商品 |
| GET | `/api/products/:id/prices` | 否 | 获取商品价格记录 |
| POST | `/api/prices` | 是 | 添加价格 |
| DELETE | `/api/prices/:id` | 是 | 删除价格 |
| POST | `/api/prices/batch` | 是 | 批量录入价格 |
| GET | `/api/trend` | 否 | 价格走势数据 |
| POST | `/api/ocr` | 是 | AI 图片识别商品价格 |
| GET | `/api/settings` | 是 | 获取系统设置 |
| PUT | `/api/settings` | 是 | 更新系统设置 |
| PUT | `/api/settings/password` | 是 | 修改管理员密码 |

---

## 核心功能

1. **价格总览 Dashboard** — 按分类 Tab 卡片网格展示商品最近3次价格，点击查看走势
2. **商品管理** — 增删改查，分类筛选，查看/管理价格历史
3. **走势分析** — ECharts 折线图，支持多商品对比
4. **AI 图片识别** — 上传游戏截图，AI 自动识别商品名称和价格，未匹配商品自动创建，批量录入价格
5. **系统设置** — 在线配置 OCR 地址/模型、修改管理员密码
6. **Excel 数据导入** — 从 `梦幻将军物价表.xlsx` 导入 132+ 商品、7 个日期价格数据

---

## 设计亮点

- **前端嵌入二进制** — `//go:embed static/*` 将前端构建产物编译进 Go 可执行文件，单文件部署
- **OCR 容错处理** — 后端 + 前端双重 JSON 解析，兼容 AI 模型的不同输出格式
- **事务批量导入** — Excel 导入和批量价格录入均使用数据库事务
- **CORS + 恢复中间件** — 内置跨域支持和 panic 恢复

---

## 快速开始

### 一键构建

```bash
./build.sh
```

该脚本会依次：
1. 安装前端依赖并构建（输出到 `server/static/`）
2. 编译 Go 后端
3. 从 Excel 初始化数据库

### 手动启动

```bash
# 启动开发模式（前后端分离）
cd web && npm run dev    # 前端 http://localhost:5173
cd server && go run .    # 后端 http://localhost:8081

# 生产模式（单文件）
./xyq_product --port 8081 --db data/mhxy.db
```

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--port` | `8081` | 服务端口 |
| `--db` | `data/mhxy.db` | 数据库路径 |
| `--ocr-endpoint` | `http://192.168.0.105:7890/v1/chat/completions` | OCR AI 模型地址 |
| `--ocr-model` | `glm-ocr` | OCR AI 模型名称 |
| `--init-db` | `false` | 初始化数据库并导入 Excel |
| `--xlsx` | `""` | Excel 文件路径 |

### 管理员账号

- 用户名: `admin`
- 密码: `123456`

---

## 待改进问题

- JWT 密钥硬编码在 `middleware/auth.go` 中
- 管理员初始密码以固定 bcrypt hash 形式写在 `database/db.go` 中
- OCR 服务地址默认连接到内网 `192.168.0.105:7890`（可通过命令行参数覆盖）

---

## 更新日志

详见 [CHANGELOG.md](./CHANGELOG.md)。
