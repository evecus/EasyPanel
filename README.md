# EasyPanel - NAS 导航面板

## 项目结构

```
easypanel/
├── main.go                          # 程序入口
├── go.mod                           # Go 模块定义
├── internal/
│   ├── config/config.go             # 配置管理
│   ├── handler/handler.go           # API 处理器
│   └── middleware/auth.go           # JWT 认证中间件
└── web/templates/index.html         # 前端页面（嵌入到二进制中）
```

## 一键构建步骤（Linux）

```bash
# 1. 创建目录结构
mkdir -p easypanel/{internal/{config,handler,middleware},web/templates}

# 2. 将各文件放到对应位置后执行：
cd easypanel
go mod tidy
go build -o easypanel .

# 3. 运行
./easypanel
```

## 功能说明

- 首次运行自动生成 easypanel.yaml 配置文件（端口3000，admin/admin）
- 登录页面：账号密码认证，JWT 会话
- 主面板：显示主机名、时间、日期
- 应用图标：文字图标 或 图片图标（支持URL和本地上传）
- 鼠标悬停图标：显示添加/编辑/删除按钮
- 拖拽排序应用
- 设置面板：账户、面板个性化、壁纸、用户管理
- 上传文件存储在 ./uploads/ 目录

## 默认账号
- 账号：admin
- 密码：admin
