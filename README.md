# HideVideo - Personal Lightweight Video Management

HideVideo is a personal lightweight video management system. Built with Go + Gin + GORM + SQLite + Vue3.

## Tech Stack

- **Backend**: Go, Gin, GORM, SQLite
- **Frontend**: Vue3, Pinia, Vue Router, Axios
- **Video Processing**: FFmpeg

## Features

1. **User Authentication** - Login/logout with admin and member roles
2. **Video Library Management** - Add/remove local video libraries, scan videos, generate covers
3. **Folder Mode** - Browse video hierarchy like a local file manager
4. **Tag Management** - Multi-tag filtering, tag sorting
5. **Actor Management** - Actor library management, video association, view actor works
6. **Sorting** - Sort by creation time, play count, rating, random
7. **Search** - Search by tag/video name/video ID
8. **Video Display** - Pagination, grid configuration (3x4, 4x3, 5x3, 6x3)
9. **Video Playback** - Popup player, fast forward, playback speed, fullscreen, click to play/pause
10. **Rating & Comments** - 10-point rating system, comment functionality
11. **Icon Generation** - Auto-generate video icons
12. **User Management** (Admin only) - Add/remove users, view plaintext passwords
13. **Settings Page** - Basic settings, security settings (change account password)
14. **Preference Persistence** - Home sorting, items per page, grid columns, tag columns auto-save
15. **Video Streaming** - Public access, no login required
16. **Clean Invalid Indexes** - Clean database indexes for deleted videos

## Project Structure

```
HideVideo/
├── backend/           # Go Backend
│   ├── main.go       # Entry point
│   ├── config/       # Configuration
│   ├── models/       # Data models
│   ├── handlers/     # API handlers
│   ├── database/     # Database operations
│   └── utils/        # Utilities (FFmpeg)
├── frontend/         # Vue3 Frontend
│   ├── src/
│   │   ├── views/   # Pages (Home, Login, Libraries, FileManager, Settings, ActorVideos)
│   │   ├── components/ # Components (TopNav, VideoPlayer, AppFooter)
│   │   ├── stores/  # State management (auth, video)
│   │   ├── api/     # API calls
│   │   └── router/  # Route configuration
│   └── package.json
└── data/             # Data directory (SQLite database, covers, icons)
```

## Quick Start

### Prerequisites

1. **Go** 1.18+
2. **Node.js** 18+ (for frontend)
3. **FFmpeg** (for video processing and cover generation)
4. **FFprobe** (for video information parsing)

### Installation

#### 1. Clone the project

```bash
git clone <repository-url>
cd HideVideo
```

#### 2. Start the backend

```bash
cd backend
go mod tidy
go run main.go
```

The backend will start at http://localhost:49377

#### 3. Start the frontend

```bash
cd frontend
npm install
npm run dev
```

The frontend dev server will start at http://localhost:49378

#### 4. Access the system

Open browser to http://localhost:49378

Default admin account:
- Username: admin
- Password: admin123

## Configuration

### Backend Configuration

Modify `backend/config/config.go`:

- `ServerConfig.Port`: Server port (default 49377)
- `DatabaseConfig.Path`: Database file path (default ./data/hidevideo.db)
- `ServerConfig.StaticPath`: Cover storage path (default ./data/covers)
- `ServerConfig.UploadPath`: Video file path (default ./data)

### Frontend Configuration

Modify `frontend/vite.config.js`:

- `server.port`: Frontend port (default 49378)
- `server.allowedHosts`: Allowed hosts (for reverse proxy)

### FFmpeg Path

Ensure FFmpeg is installed and added to system PATH. If using a non-standard path, modify the command paths in `backend/utils/utils.go`.

## User Roles

- **Admin**: Can manage users, view all users' plaintext passwords
- **Member**: Can only modify their own account and password

## API Endpoints

### Authentication
- `POST /api/auth/login` - Login
- `POST /api/auth/logout` - Logout
- `GET /api/auth/check` - Check login status

### Video Library
- `GET /api/libraries` - Get library list
- `POST /api/libraries` - Add library
- `DELETE /api/libraries/:id` - Delete library
- `POST /api/libraries/:id/scan` - Scan library
- `POST /api/libraries/:id/cover` - Generate covers
- `POST /api/libraries/:id/icon` - Generate icons
- `POST /api/libraries/clean-invalid` - Clean invalid indexes
- `GET /api/libraries/:id/files` - Get library file list
- `GET /api/libraries/:id/path` - Get library path

### Videos
- `GET /api/videos` - Get video list
- `GET /api/videos/folders` - Get folder tree
- `GET /api/videos/by-path` - Get videos by path
- `GET /api/videos/:id` - Get video details
- `GET /videos/:id/stream` - Video streaming
- `PUT /api/videos/:id/rating` - Update rating
- `PUT /api/videos/:id/filename` - Rename video
- `POST /api/videos/:id/play` - Increment play count
- `DELETE /api/videos/:id` - Delete video

### Video Tags
- `GET /api/videos/:id/tags` - Get video tags
- `POST /api/videos/:id/tags` - Add video tag
- `DELETE /api/videos/:id/tags/:tagId` - Remove video tag

### Video Actors
- `GET /api/videos/:id/actors` - Get video actors
- `POST /api/videos/:id/actors` - Add video actor
- `DELETE /api/videos/:id/actors/:actorId` - Remove video actor

### Comments
- `GET /api/videos/:id/comments` - Get comments
- `POST /api/videos/:id/comments` - Add comment
- `DELETE /api/comments/:id` - Delete comment

### Tags
- `GET /api/tags` - Get tag list
- `POST /api/tags` - Add tag
- `PUT /api/tags/reorder` - Reorder tags
- `PUT /api/tags/:id` - Update tag
- `DELETE /api/tags/:id` - Delete tag

### Actors
- `GET /api/actors` - Get actor list
- `POST /api/actors` - Add actor
- `PUT /api/actors/reorder` - Reorder actors
- `PUT /api/actors/:id` - Update actor
- `DELETE /api/actors/:id` - Delete actor
- `GET /api/actors/:id/videos` - Get actor's videos

### User Management (Admin only)
- `GET /api/users` - Get user list (with plaintext passwords)
- `POST /api/users` - Add user
- `DELETE /api/users/:id` - Delete user
- `PUT /api/users/:id/password` - Admin change user password
- `PUT /api/users/:id/info` - Admin change user info
- `PUT /api/users/password` - User change own password
- `PUT /api/users/info` - User change own info
- `GET /api/users/me` - Get current user info

## License

MIT License

## GitHub

https://github.com/pureages/HideVideo
