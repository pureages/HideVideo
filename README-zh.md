[中文](README-zh.md) | [English](README.md) 

# HideVideo 个人轻量视频管理

个人轻量视频管理系统，与其他视频刮削的库不同，HideVideo更注重个人一些杂七杂八的视频的收藏。

<table style="width: 100%;">
  <tr>
    <td><img src="screenshots/1.png" alt="img1" style="width: 100%;"></td>
    <td><img src="screenshots/2.png" alt="img2" style="width: 100%;"></td>
  </tr>
  <tr>
    <td><img src="screenshots/3.png" alt="img3" style="width: 100%;"></td>
    <td><img src="screenshots/4.png" alt="img4" style="width: 100%;"></td>
  </tr>
</table>

## 技术栈

- **后端**: Go, Gin, GORM, SQLite3
- **视频处理**: FFmpeg

## 快速开始

### 安装步骤

#### 一、Docker（推荐）
##### 1.首先运行
```
docker run -d \
  -v $(pwd)/data:/app/data \
  -p 49377:49377 \
  --name hidevideo \
  --restart unless-stopped \
  pureages/hidevideo:latest
```
##### 2.然后安装ffmpeg
```
docker exec hidevideo apt-get update
docker exec hidevideo apt-get install -y ffmpeg
docker exec hidevideo rm -rf /var/lib/apt/lists/*
```
注意：添加视频库的目录为 ```/app/data/…… ```，你的视频文件夹应该放在 ```$(pwd)/data``` 里面！```$(pwd)```默认是你用户名，也可以自定义路径！


#### 二、本地部署

##### 1.前置要求

1. **Go** 1.18+
2. **FFmpeg** (用于视频处理和封面生成)
3. **FFprobe** (用于视频信息解析)

##### 2. 克隆项目

```bash
git clone https://github.com/pureages/HideVideo.git
cd HideVideo
```

##### 3. 启动服务器

```bash
cd backend
go mod tidy
go run main.go
```

后端服务将在 http://localhost:49377 启动

### 访问系统

打开浏览器访问 http://localhost:49377 （或者：<你的服务器IP>:49377）

默认管理员账号：
```
- 用户名: admin
- 密码: admin123
```

## 配置说明

## 用户角色

- **管理员 (admin)**: 可以管理用户、添加删除用户、添加标签
- **普通成员 (member)**: 只能修改自己的账号和密码

## 功能特性

1. **用户认证** - 登录/登出功能，支持管理员和普通成员角色
2. **视频库管理** - 添加/删除本地视频库，扫描视频，获取封面
3. **文件夹模式** - 像本地文件管理器一样浏览视频层级目录
4. **标签管理** -（仅管理员）多标签筛选，标签排序
5. **演员管理** -（仅管理员）演员库管理，关联视频，查看演员作品
6. **排序功能** - 按创建时间、播放次数、评分、随机排序
7. **搜索功能** - 按标签/视频名/视频ID搜索
8. **视频展示** - 分页、网格配置（3x4、4x3、5x3、6x3）
9. **视频播放** - 弹窗播放、快进、倍速、全屏，支持点击播放/暂停
10. **评分评论** - 10分制评分、评论功能
11. **图标生成** - 自动生成视频图标
12. **用户管理**（仅管理员）- 添加/删除用户，查看明文密码
13. **设置页面** - 基本设置、安全设置（修改账号密码）
14. **偏好设置持久化** - 首页排序、每页数量、网格列数、标签列数自动保存
15. **视频流式播放** - 公开访问，无需登录
16. **清理无效索引** - 清理已删除视频的数据库索引，错误封面等

## 项目结构

```
HideVideo/
├── backend/           # Go 后端
│   ├── main.go       # 入口文件
│   ├── config/       # 配置
│   ├── models/       # 数据模型
│   ├── handlers/     # API 处理器
│   ├── database/     # 数据库操作
│   └── utils/        # 工具函数 (FFmpeg调用)
├── frontend/         # Vue3 前端
│   ├── src/
│   │   ├── views/   # 页面 (Home, Login, Libraries, FileManager, Settings, ActorVideos)
│   │   ├── components/ # 组件 (TopNav, VideoPlayer, AppFooter)
│   │   ├── stores/  # 状态管理 (auth, video)
│   │   ├── api/     # API 调用
│   │   └── router/  # 路由配置
│   └── package.json
└── data/             # 数据目录（SQLite数据库、封面、图标）
```

## API 接口

### 认证
- `POST /api/auth/login` - 登录
- `POST /api/auth/logout` - 登出
- `GET /api/auth/check` - 检查登录状态

### 视频库
- `GET /api/libraries` - 获取视频库列表
- `POST /api/libraries` - 添加视频库
- `DELETE /api/libraries/:id` - 删除视频库
- `POST /api/libraries/:id/scan` - 扫描视频库
- `POST /api/libraries/:id/cover` - 生成封面
- `POST /api/libraries/:id/icon` - 生成图标
- `POST /api/libraries/clean-invalid` - 清理无效索引
- `GET /api/libraries/:id/files` - 获取库文件列表
- `GET /api/libraries/:id/path` - 获取库路径

### 视频
- `GET /api/videos` - 获取视频列表
- `GET /api/videos/folders` - 获取文件夹树
- `GET /api/videos/by-path` - 按路径获取视频
- `GET /api/videos/:id` - 获取视频详情
- `GET /videos/:id/stream` - 视频流式播放
- `PUT /api/videos/:id/rating` - 更新评分
- `PUT /api/videos/:id/filename` - 重命名视频
- `POST /api/videos/:id/play` - 增加播放次数
- `DELETE /api/videos/:id` - 删除视频

### 视频标签
- `GET /api/videos/:id/tags` - 获取视频标签
- `POST /api/videos/:id/tags` - 添加视频标签
- `DELETE /api/videos/:id/tags/:tagId` - 移除视频标签

### 视频演员
- `GET /api/videos/:id/actors` - 获取视频演员
- `POST /api/videos/:id/actors` - 添加视频演员
- `DELETE /api/videos/:id/actors/:actorId` - 移除视频演员

### 评论
- `GET /api/videos/:id/comments` - 获取评论
- `POST /api/videos/:id/comments` - 添加评论
- `DELETE /api/comments/:id` - 删除评论

### 标签（仅管理员）
- `GET /api/tags` - 获取标签列表
- `POST /api/tags` - 添加标签
- `PUT /api/tags/reorder` - 排序标签
- `PUT /api/tags/:id` - 更新标签
- `DELETE /api/tags/:id` - 删除标签

### 演员（仅管理员）
- `GET /api/actors` - 获取演员列表
- `POST /api/actors` - 添加演员
- `PUT /api/actors/reorder` - 排序演员
- `PUT /api/actors/:id` - 更新演员
- `DELETE /api/actors/:id` - 删除演员
- `GET /api/actors/:id/videos` - 获取演员参演的视频

### 用户管理（仅管理员）
- `GET /api/users` - 获取用户列表（含明文密码）
- `POST /api/users` - 添加用户
- `DELETE /api/users/:id` - 删除用户
- `PUT /api/users/:id/password` - 管理员修改用户密码
- `PUT /api/users/:id/info` - 管理员修改用户信息
- `PUT /api/users/password` - 用户修改自己密码
- `PUT /api/users/info` - 用户修改自己信息
- `GET /api/users/me` - 获取当前用户信息

## 许可证

MIT License

## GitHub

https://github.com/pureages/HideVideo
