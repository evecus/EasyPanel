<div align="center">

<svg width="80" height="80" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <linearGradient id="grad" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
      <stop offset="0%" stop-color="#a855f7"/>
      <stop offset="100%" stop-color="#ec4899"/>
    </linearGradient>
  </defs>
  <rect width="32" height="32" rx="8" fill="url(#grad)"/>
  <rect x="7" y="9.5" width="18" height="3" rx="1.5" fill="white"/>
  <rect x="7" y="14.5" width="13" height="3" rx="1.5" fill="white" fill-opacity="0.75"/>
  <rect x="7" y="19.5" width="15" height="3" rx="1.5" fill="white" fill-opacity="0.5"/>
</svg>

# EasyPanel

**一个简洁优雅的自托管应用导航面板**

支持多用户 · 7种主题色 · 自定义壁纸 · 农历时钟 · Docker 部署

</div>

---

## ✨ 功能

- 🖥️ 应用导航网格，支持文字/图片图标，拖拽排序
- 🎨 7 种主题色 + 自定义壁纸（支持上传或 URL）
- 🕐 时钟组件（时间 / 日期 / 星期 / 农历）
- 👥 多用户管理，各账号独立
- 🔓 公开 / 私有访问模式（公开模式无需登录可浏览，编辑仍需登录）
- 📐 可调节图标大小、圆角、间距、字体
- 💾 数据导入 / 导出备份
- 🌐 中英文双语

---

## 🚀 快速开始

### 方式一：直接下载二进制（推荐）

前往 [Releases](../../releases) 页面下载对应平台的二进制文件：

| 文件 | 平台 |
|------|------|
| `easypanel-linux-amd64` | Linux x86_64 |
| `easypanel-linux-arm64` | Linux ARM64（树莓派等） |

```bash
# 下载后赋予执行权限
chmod +x easypanel-linux-amd64

# 运行
./easypanel-linux-amd64
```

访问 `http://你的IP:3088`，默认账号 `admin` / `admin`。

---

### 方式二：Docker 部署

```bash
docker run -d \
  --name easypanel \
  --restart unless-stopped \
  -p 3088:3088 \
  -v /root/data:/app/data \
  -v /root/config:/app/config \
  \
  # ── 读取宿主机系统信息 ──
  -v /proc:/host/proc:ro \
  -v /sys:/host/sys:ro \
  -e HOST_PROC=/host/proc \
  -e HOST_SYS=/host/sys \
  \
  # ── 进程管理（读取宿主机进程） ──
  --pid=host \
  \
  # ── 网络信息（读取宿主机网卡） ──
  --network=host \
  \
  # ── Docker 管理（操作宿主机 Docker） ──
  -v /var/run/docker.sock:/var/run/docker.sock \
  \
  # ── 温度传感器 ──
  -v /sys/class/thermal:/sys/class/thermal:ro \
  \
  evecus/easypanel:latest
```

**挂载目录说明：**

| 容器路径 | 说明 |
|---------|------|
| `/app/data` | 应用数据、设置、上传的图片 |
| `/app/config` | 账号配置、JWT 密钥、端口设置 |

> ⚠️ 不挂载目录时数据会在容器删除后丢失

#### Docker Compose

```yaml
services:
  easypanel:
    image: evecus/easypanel:latest
    container_name: easypanel
    restart: unless-stopped
    # 注意：使用 network_mode: host 时，-p 端口映射会被忽略
    # 但为了清晰起见，可以在这里标注该应用监听 3088 端口
    network_mode: host
    pid: host
    environment:
      - HOST_PROC=/host/proc
      - HOST_SYS=/host/sys
    volumes:
      # 持久化数据与配置
      - /root/data:/app/data
      - /root/config:/app/config
      # 系统信息采集
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      # Docker 守护进程管理
      - /var/run/docker.sock:/var/run/docker.sock
      # 温度传感器
      - /sys/class/thermal:/sys/class/thermal:ro
```

```bash
docker compose up -d
```

---

### 方式三：从源码构建

**前置要求：** Go 1.21+、Node.js 18+

```bash
# 克隆仓库
git clone https://github.com/evecus/EasyPanel.git
cd EasyPanel

# 一键构建
chmod +x build.sh
./build.sh

# 运行
./easypanel
```

或手动构建：

```bash
cd frontend
npm install
npm run build
cd ..
go build -o easypanel .
./easypanel
```

---

## ⚙️ 配置

首次运行后会自动生成 `config/easypanel.yaml`：

```yaml
port: 3088          # 监听端口
jwt_secret: ...     # 自动生成，勿手动修改
public_mode: false  # 是否开启公开访问模式
users:
  - username: admin
    password: ...   # bcrypt 加密
    nickname: Admin
    is_admin: true
```

修改端口后重启生效，无需重新编译。

---

## 🔑 默认账号

```
账号: admin
密码: admin
```

> ⚠️ 首次登录后请立即前往「系统设置 → 我的信息」修改密码

---

## 🛠️ 技术栈

- **前端**：Vue 3 (Composition API) + Vite
- **后端**：Go + Gin，通过 `embed` 将前端打包进单一二进制文件

---

## 📦 项目结构

```
EasyPanel/
├── frontend/               # Vue 3 前端
│   ├── src/
│   │   ├── App.vue
│   │   ├── components/
│   │   └── composables/
│   └── index.html
├── internal/
│   ├── config/             # 配置读写
│   └── handler/            # API 处理
├── web/dist/               # 前端构建产物（自动生成）
├── main.go
├── Dockerfile
└── build.sh
```

---

## 📄 License

MIT
