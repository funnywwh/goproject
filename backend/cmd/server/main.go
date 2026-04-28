package main

import (
	"compress/gzip"
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"prjflow/internal/api"
	"prjflow/internal/config"
	"prjflow/internal/middleware"
	"prjflow/internal/utils"
	"prjflow/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

//go:embed frontend-dist
var frontendFS embed.FS

// 版本信息（可以通过构建时注入）
var (
	Version   = "v0.5.8"  // 版本号
	BuildTime = "unknown" // 构建时间
	GitCommit = "unknown" // Git提交哈希
)

// getContentType 根据文件扩展名返回正确的 Content-Type
func getContentType(filePath string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	// 根据扩展名设置 MIME 类型
	switch ext {
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".woff", ".woff2":
		return "font/woff"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	case ".otf":
		return "font/otf"
	case ".wasm":
		return "application/wasm"
	default:
		// 尝试使用 Go 的 mime 包检测
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			return contentType
		}
		// 最后使用 http.DetectContentType 作为回退
		return http.DetectContentType(data)
	}
}

// setupExternalFrontend 设置外部前端文件系统（作为 embed 失败时的回退方案）
func setupExternalFrontend(r *gin.Engine) {
	// 获取前端 dist 目录路径（支持从项目根目录或 backend 目录运行）
	var frontendDistPath string
	// 尝试从项目根目录查找
	if _, err := os.Stat("./frontend/dist"); err == nil {
		frontendDistPath = "./frontend/dist"
	} else if _, err := os.Stat("../frontend/dist"); err == nil {
		// 从 backend 目录运行时
		frontendDistPath = "../frontend/dist"
	} else {
		// 尝试获取可执行文件所在目录
		exePath, err := os.Executable()
		if err == nil {
			exeDir := filepath.Dir(exePath)
			// 尝试从可执行文件目录向上查找
			for _, path := range []string{
				filepath.Join(exeDir, "frontend", "dist"),
				filepath.Join(exeDir, "..", "frontend", "dist"),
				filepath.Join(exeDir, "..", "..", "frontend", "dist"),
			} {
				if _, err := os.Stat(path); err == nil {
					frontendDistPath = path
					break
				}
			}
		}
	}

	// 如果找到了前端 dist 目录，配置静态文件服务
	if frontendDistPath != "" {
		if utils.Logger != nil {
			utils.Logger.Infof("使用外部前端静态文件目录: %s", frontendDistPath)
		} else {
			log.Printf("使用外部前端静态文件目录: %s", frontendDistPath)
		}
		// 提供静态资源文件（如 JS、CSS、图片等）
		r.Static("/assets", filepath.Join(frontendDistPath, "assets"))
		r.StaticFile("/vite.svg", filepath.Join(frontendDistPath, "vite.svg"))

		// SPA 路由处理：所有非 API 路由都返回 index.html
		// 这样前端路由（如 /dashboard, /login 等）可以正常工作
		r.NoRoute(func(c *gin.Context) {
			// 如果请求的是 API 路径，返回 404
			path := c.Request.URL.Path
			if len(path) >= 4 && path[:4] == "/api" {
				c.JSON(http.StatusNotFound, gin.H{
					"code":    404,
					"message": "API endpoint not found",
				})
				return
			}
			// 否则返回前端 index.html（用于 SPA 路由）
			c.File(filepath.Join(frontendDistPath, "index.html"))
		})
	} else {
		if utils.Logger != nil {
			utils.Logger.Warn("警告: 未找到前端 dist 目录，静态文件服务未启用")
		} else {
			log.Println("警告: 未找到前端 dist 目录，静态文件服务未启用")
		}
		// 即使没有前端文件，也要处理 API 404
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if len(path) >= 4 && path[:4] == "/api" {
				c.JSON(http.StatusNotFound, gin.H{
					"code":    404,
					"message": "API endpoint not found",
				})
			} else {
				c.JSON(http.StatusNotFound, gin.H{
					"code":    404,
					"message": "Not found",
				})
			}
		})
	}
}

// backupDatabase 备份数据库
func backupDatabase() error {
	// 加载配置
	if err := config.LoadConfig(""); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 检查数据库类型，只支持 SQLite 备份
	if config.AppConfig.Database.Type != "sqlite" {
		return fmt.Errorf("backup only supports SQLite database, current type: %s", config.AppConfig.Database.Type)
	}

	// 获取数据库文件路径
	dbPath := config.AppConfig.Database.DSN
	if !filepath.IsAbs(dbPath) {
		// 相对路径，从当前工作目录或可执行文件目录查找
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			// 尝试从可执行文件目录查找
			exePath, err := os.Executable()
			if err == nil {
				exeDir := filepath.Dir(exePath)
				absPath := filepath.Join(exeDir, dbPath)
				if _, err := os.Stat(absPath); err == nil {
					dbPath = absPath
				}
			}
		}
	}

	// 检查数据库文件是否存在
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("database file not found: %s", dbPath)
	}

	// 创建备份目录（在数据库文件同目录下）
	dbDir := filepath.Dir(dbPath)
	backupDir := filepath.Join(dbDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 生成备份文件名（带时间戳）
	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("data_%s.db", timestamp)
	backupPath := filepath.Join(backupDir, backupFileName)

	// 复制数据库文件
	log.Printf("Backing up database: %s -> %s", dbPath, backupPath)
	srcFile, err := os.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dstFile.Close()

	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		dstFile.Close()
		os.Remove(backupPath)
		return fmt.Errorf("failed to copy database file: %w", err)
	}

	// 获取文件大小
	fileSize := float64(written)
	unit := "B"
	if fileSize >= 1024*1024 {
		fileSize /= 1024 * 1024
		unit = "MB"
	} else if fileSize >= 1024 {
		fileSize /= 1024
		unit = "KB"
	}

	log.Printf("✓ Backup created: %s (%.2f %s)", backupPath, fileSize, unit)

	// 尝试压缩备份文件
	compressedPath := backupPath + ".gz"
	if err := compressFile(backupPath, compressedPath); err == nil {
		// 压缩成功，删除原文件
		os.Remove(backupPath)
		log.Printf("✓ Backup compressed: %s", compressedPath)
	} else {
		log.Printf("Warning: Failed to compress backup: %v", err)
	}

	// 清理旧备份（保留最近7天）
	if err := cleanupOldBackups(backupDir, 7); err != nil {
		log.Printf("Warning: Failed to cleanup old backups: %v", err)
	}

	// 显示备份统计
	backupCount, totalSize, err := getBackupStats(backupDir)
	if err == nil {
		log.Printf("Backup statistics: %d backups, total size: %s", backupCount, formatSize(totalSize))
	}

	return nil
}

// compressFile 压缩文件
func compressFile(srcPath, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	_, err = io.Copy(gzWriter, srcFile)
	return err
}

// cleanupOldBackups 清理旧备份
func cleanupOldBackups(backupDir string, keepDays int) error {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return err
	}

	cutoffTime := time.Now().AddDate(0, 0, -keepDays)
	removedCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只处理备份文件
		if !strings.HasPrefix(entry.Name(), "data_") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 如果文件修改时间早于保留期限，删除
		if info.ModTime().Before(cutoffTime) {
			filePath := filepath.Join(backupDir, entry.Name())
			if err := os.Remove(filePath); err == nil {
				removedCount++
			}
		}
	}

	if removedCount > 0 {
		log.Printf("Cleaned up %d old backup(s) (older than %d days)", removedCount, keepDays)
	}

	return nil
}

// getBackupStats 获取备份统计信息
func getBackupStats(backupDir string) (int, int64, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return 0, 0, err
	}

	count := 0
	totalSize := int64(0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasPrefix(entry.Name(), "data_") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		count++
		totalSize += info.Size()
	}

	return count, totalSize, nil
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
	unit := "B"
	fsize := float64(size)
	if fsize >= 1024*1024*1024 {
		fsize /= 1024 * 1024 * 1024
		unit = "GB"
	} else if fsize >= 1024*1024 {
		fsize /= 1024 * 1024
		unit = "MB"
	} else if fsize >= 1024 {
		fsize /= 1024
		unit = "KB"
	}
	return fmt.Sprintf("%.2f %s", fsize, unit)
}

// findProcessByPort 通过端口查找进程ID
func findProcessByPort(port int) (int, error) {
	// 尝试使用 lsof 命令
	cmd := exec.Command("lsof", "-t", fmt.Sprintf("-i:%d", port))
	output, err := cmd.Output()
	if err == nil {
		pidStr := strings.TrimSpace(string(output))
		if pidStr != "" {
			pid, err := strconv.Atoi(pidStr)
			if err == nil {
				return pid, nil
			}
		}
	}

	// 尝试使用 ss 命令
	cmd = exec.Command("ss", "-tlnp")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, fmt.Sprintf(":%d", port)) {
				// 尝试从输出中提取 PID
				// ss 输出格式: LISTEN 0 128 *:8080 *:* users:(("server",pid=12345,fd=3))
				parts := strings.Split(line, "pid=")
				if len(parts) > 1 {
					pidPart := strings.Split(parts[1], ",")[0]
					pid, err := strconv.Atoi(pidPart)
					if err == nil {
						return pid, nil
					}
				}
			}
		}
	}

	// 尝试使用 netstat 命令（如果可用）
	cmd = exec.Command("netstat", "-tlnp")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, fmt.Sprintf(":%d", port)) {
				// netstat 输出格式可能不同，尝试提取 PID
				fields := strings.Fields(line)
				for _, field := range fields {
					if strings.Contains(field, "/") {
						parts := strings.Split(field, "/")
						if len(parts) > 0 {
							pid, err := strconv.Atoi(parts[0])
							if err == nil {
								return pid, nil
							}
						}
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("process not found on port %d", port)
}

// stopServer 停止正在运行的服务器
func stopServer(port int) error {
	pid, err := findProcessByPort(port)
	if err != nil {
		return fmt.Errorf("server not running on port %d: %w", port, err)
	}

	log.Printf("Found server process: PID %d", pid)

	// 检查进程是否存在
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}

	// 发送 SIGTERM 信号（优雅关闭）
	log.Printf("Sending SIGTERM to process %d...", pid)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send signal to process %d: %w", pid, err)
	}

	// 等待进程退出（最多等待10秒）
	log.Printf("Waiting for process %d to exit...", pid)
	for i := 0; i < 20; i++ {
		// 检查进程是否还在运行
		if err := process.Signal(syscall.Signal(0)); err != nil {
			// 进程已退出
			log.Printf("Process %d has exited", pid)
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 如果进程还在运行，发送 SIGKILL
	log.Printf("Process %d did not exit, sending SIGKILL...", pid)
	if err := process.Signal(syscall.SIGKILL); err != nil {
		return fmt.Errorf("failed to kill process %d: %w", pid, err)
	}

	// 再等待一下
	time.Sleep(1 * time.Second)
	if err := process.Signal(syscall.Signal(0)); err != nil {
		log.Printf("Process %d has been killed", pid)
		return nil
	}

	return fmt.Errorf("failed to stop process %d", pid)
}

// showVersion 显示版本信息
func showVersion() {
	fmt.Printf("Project Management System\n")
	fmt.Printf("Version: %s\n", Version)
	if BuildTime != "unknown" {
		fmt.Printf("Build Time: %s\n", BuildTime)
	}
	if GitCommit != "unknown" {
		fmt.Printf("Git Commit: %s\n", GitCommit)
	}
	fmt.Printf("Go Version: %s\n", getGoVersion())
}

// getGoVersion 获取Go版本信息
func getGoVersion() string {
	return runtime.Version()
}

// stopServerCommand 停止服务器（命令行参数）
func stopServerCommand() error {
	// 加载配置以获取端口
	if err := config.LoadConfig(""); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	port := config.AppConfig.Server.Port
	log.Printf("Stopping server on port %d...", port)

	// 停止服务器
	if err := stopServer(port); err != nil {
		return err
	}

	log.Printf("Server stopped successfully")
	return nil
}

// restartServer 重启服务器
func restartServer() error {
	// 加载配置以获取端口
	if err := config.LoadConfig(""); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	port := config.AppConfig.Server.Port
	log.Printf("Restarting server on port %d...", port)

	// 停止现有服务器
	if err := stopServer(port); err != nil {
		// 如果服务器没有运行，这不是错误，继续启动
		log.Printf("Warning: %v", err)
	}

	// 等待一下确保端口释放
	time.Sleep(1 * time.Second)

	// 获取可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// 获取可执行文件所在目录
	exeDir := filepath.Dir(exePath)

	// 启动新服务器（在后台）
	log.Printf("Starting new server: %s", exePath)
	cmd := exec.Command(exePath)
	cmd.Dir = exeDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 在后台启动
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	log.Printf("Server restarted successfully (PID: %d)", cmd.Process.Pid)
	return nil
}

func main() {
	// 定义命令行参数
	var (
		backup    = flag.Bool("backup", false, "备份数据库")
		stop      = flag.Bool("stop", false, "停止服务器")
		restart   = flag.Bool("restart", false, "重启服务器")
		version   = flag.Bool("version", false, "显示版本信息")
		versionV  = flag.Bool("v", false, "显示版本信息（简写）")
		versionV2 = flag.Bool("V", false, "显示版本信息（简写）")
	)

	// 自定义 Usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "项目管理系统服务器\n\n")
		fmt.Fprintf(os.Stderr, "使用方法:\n")
		fmt.Fprintf(os.Stderr, "  %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s --backup      # 备份数据库\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --stop        # 停止服务器\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --restart     # 重启服务器\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --version     # 显示版本信息\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s               # 启动服务器\n", os.Args[0])
	}

	// 解析命令行参数
	flag.Parse()

	// 处理命令行参数
	if *backup {
		// 加载配置
		if err := config.LoadConfig(""); err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		// 初始化数据库
		db, err := utils.InitDB()
		if err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
		// 执行备份
		if err := utils.BackupDatabase(db); err != nil {
			log.Fatalf("Backup failed: %v", err)
		}
		log.Println("Backup completed successfully")
		os.Exit(0)
	}

	if *stop {
		if err := stopServerCommand(); err != nil {
			log.Fatalf("Stop failed: %v", err)
		}
		os.Exit(0)
	}

	if *restart {
		if err := restartServer(); err != nil {
			log.Fatalf("Restart failed: %v", err)
		}
		os.Exit(0)
	}

	if *version || *versionV || *versionV2 {
		showVersion()
		os.Exit(0)
	}

	// 如果还有其他未解析的参数，显示帮助信息
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "错误: 未知参数: %v\n\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	// 加载配置
	if err := config.LoadConfig(""); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 设置Gin模式
	gin.SetMode(config.AppConfig.Server.Mode)

	// 初始化数据库（先初始化数据库，因为日志级别配置存储在数据库中）
	db, err := utils.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 初始化日志系统（在数据库初始化后，这样可以从数据库读取日志级别配置）
	if err := utils.InitLogger(db); err != nil {
		// 日志初始化失败，回退到标准库log，但不中断启动
		log.Printf("Warning: Failed to initialize logger: %v, using standard log", err)
	} else {
		// 配置Gin框架的日志输出
		gin.DefaultWriter = utils.Logger.Writer()
		gin.DefaultErrorWriter = utils.Logger.WriterLevel(logrus.ErrorLevel)
	}

	// 自动迁移数据库
	if err := utils.AutoMigrate(db); err != nil {
		if utils.Logger != nil {
			utils.Logger.Fatalf("Failed to migrate database: %v", err)
		} else {
			log.Fatalf("Failed to migrate database: %v", err)
		}
	}
	if utils.Logger != nil {
		utils.Logger.Info("Database migrated successfully")
	} else {
		log.Println("Database migrated successfully")
	}

	// 初始化审计日志数据库（默认使用独立的审计数据库）
	auditDB, err := utils.InitAuditDB()
	if err != nil {
		if utils.Logger != nil {
			utils.Logger.Fatalf("Failed to initialize audit database: %v", err)
		} else {
			log.Fatalf("Failed to initialize audit database: %v", err)
		}
	}
	// 设置全局审计日志数据库连接
	utils.AuditDB = auditDB
	// 迁移审计日志数据库
	if err := utils.MigrateAuditDB(db, auditDB); err != nil {
		if utils.Logger != nil {
			utils.Logger.Fatalf("Failed to migrate audit database: %v", err)
		} else {
			log.Fatalf("Failed to migrate audit database: %v", err)
		}
	}
	if utils.Logger != nil {
		utils.Logger.Info("Audit database initialized and migrated successfully")
	} else {
		log.Println("Audit database initialized and migrated successfully")
	}

	// 创建Gin引擎
	r := gin.New()

	// 注册中间件
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	// 将数据库连接存储到上下文（供中间件使用）
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// 健康检查接口
	r.GET("/health", func(c *gin.Context) {
		utils.Success(c, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 版本信息接口（不需要认证）
	r.GET("/api/version", func(c *gin.Context) {
		utils.Success(c, gin.H{
			"version":    Version,
			"build_time": BuildTime,
			"git_commit": GitCommit,
			"go_version": runtime.Version(),
		})
	})

	// WebSocket路由
	r.GET("/ws", websocket.HandleWebSocket)

	// 微信验证文件路由（不需要认证，必须放在根路径）
	// 支持格式：/MP_verify_xxxxx.txt
	// 注意：这个路由必须放在其他路由之前，但只匹配 MP_verify_ 开头的文件
	wechatVerifyHandler := api.NewWeChatVerifyHandler(db)
	// 使用参数路由，匹配 /MP_verify_:code.txt 格式
	// 注意：Gin 不支持在参数名中包含点，所以我们需要使用 :code 参数
	r.GET("/MP_verify_:code", wechatVerifyHandler.HandleVerifyFile)       // 匹配 /MP_verify_xxxxx.txt
	r.POST("/api/wechat/verify-file", wechatVerifyHandler.SaveVerifyFile) // 保存验证文件内容

	// 系统初始化路由（不需要认证）
	initHandler := api.NewInitHandler(db)
	initCallbackHandler := api.NewInitCallbackHandler(db)
	initGroup := r.Group("/api/init")
	{
		initGroup.GET("/status", initHandler.CheckInitStatus)
		initGroup.POST("/wechat-config", initHandler.SaveWeChatConfig)  // 第一步：保存微信配置
		initGroup.GET("/qrcode", initHandler.GetInitQRCode)             // 获取初始化二维码
		initGroup.GET("/callback", initCallbackHandler.HandleCallback)  // 微信回调处理
		initGroup.POST("", initHandler.InitSystem)                      // 第二步：通过微信登录完成初始化（保留兼容）
		initGroup.POST("/password", initHandler.InitSystemWithPassword) // 第二步：通过密码登录完成初始化
	}

	// 认证相关路由
	authHandler := api.NewAuthHandler(db)
	userHandler := api.NewUserHandler(db)
	authGroup := r.Group("/api/auth")
	{
	authGroup.POST("/login", authHandler.Login)                    // 用户名密码登录
	authGroup.POST("/refresh", authHandler.RefreshToken)          // 刷新Token（不需要认证）
	// GetQRCode 不需要中间件认证：登录场景允许未登录访问，添加用户场景在函数内部检查权限
	authGroup.GET("/wechat/qrcode", authHandler.GetQRCode)
		authGroup.GET("/wechat/callback", authHandler.WeChatCallback)                   // 微信登录回调接口（GET请求，微信直接重定向到这里）
		authGroup.GET("/wechat/add-user/callback", userHandler.AddUserByWeChatCallback) // 微信添加用户回调接口（GET请求，微信直接重定向到这里）
		authGroup.POST("/wechat/login", authHandler.WeChatLogin)                        // 微信登录（POST请求，保留用于其他场景）
		authGroup.GET("/user/info", middleware.Auth(), authHandler.GetUserInfo)
		authGroup.POST("/logout", middleware.Auth(), authHandler.Logout)
		authGroup.POST("/change-password", middleware.Auth(), authHandler.ChangePassword) // 修改密码
		// 微信绑定相关路由
		authGroup.GET("/wechat/bind/qrcode", middleware.Auth(), authHandler.GetWeChatBindQRCode) // 获取微信绑定二维码
		authGroup.GET("/wechat/bind/callback", authHandler.WeChatBindCallback)                   // 微信绑定回调接口（GET请求，微信直接重定向到这里）
		authGroup.POST("/wechat/unbind", middleware.Auth(), authHandler.UnbindWeChat)            // 解绑微信
	}

	// 权限管理路由
	permHandler := api.NewPermissionHandler(db)
	permGroup := r.Group("/api/permissions", middleware.Auth())
	{
		// 角色管理
		permGroup.GET("/roles", permHandler.GetRoles)
		permGroup.GET("/roles/:id", permHandler.GetRole)
		permGroup.POST("/roles", middleware.RequirePermission(db, "permission:manage"), permHandler.CreateRole)
		permGroup.PUT("/roles/:id", middleware.RequirePermission(db, "permission:manage"), permHandler.UpdateRole)
		permGroup.DELETE("/roles/:id", middleware.RequirePermission(db, "permission:manage"), permHandler.DeleteRole)
		permGroup.GET("/roles/:id/permissions", permHandler.GetRolePermissions)
		permGroup.POST("/roles/:id/permissions", middleware.RequirePermission(db, "permission:manage"), permHandler.AssignRolePermissions)

		// 权限管理
		permGroup.GET("/permissions", permHandler.GetPermissions)
		permGroup.GET("/permissions/:id", permHandler.GetPermission)
		permGroup.POST("/permissions", middleware.RequirePermission(db, "permission:manage"), permHandler.CreatePermission)
		permGroup.PUT("/permissions/:id", middleware.RequirePermission(db, "permission:manage"), permHandler.UpdatePermission)
		permGroup.DELETE("/permissions/:id", middleware.RequirePermission(db, "permission:manage"), permHandler.DeletePermission)

		// 用户角色管理
		permGroup.GET("/users/:id/roles", permHandler.GetUserRoles)
		permGroup.POST("/users/:id/roles", middleware.RequirePermission(db, "permission:manage"), permHandler.AssignUserRoles)

		// 当前用户权限
		permGroup.GET("/me", permHandler.GetUserPermissions)

		// 获取菜单树
		permGroup.GET("/menus", permHandler.GetMenus)
		permGroup.GET("/menus/all", middleware.RequirePermission(db, "permission:manage"), permHandler.GetAllMenus)
	}

	// 用户管理路由
	userGroup := r.Group("/api/users", middleware.Auth())
	{
		userGroup.GET("", middleware.RequirePermission(db, "user:read"), userHandler.GetUsers)                      // 查看用户列表
		userGroup.POST("", middleware.RequirePermission(db, "user:create"), userHandler.CreateUser)                 // 创建用户需要权限
		userGroup.POST("/wechat/add", middleware.RequirePermission(db, "user:create"), userHandler.AddUserByWeChat) // 扫码添加用户需要权限
		// 注意：绑定微信接口需要在 /:id 之前，避免路由冲突
		userGroup.GET("/:id/wechat/bind/qrcode", middleware.RequirePermission(db, "user:update"), userHandler.GetUserWeChatBindQRCode) // 获取用户绑定微信二维码（管理员操作）
		userGroup.GET("/:id", middleware.RequirePermission(db, "user:read"), userHandler.GetUser)                                      // 查看用户详情
		userGroup.PUT("/:id", middleware.RequirePermission(db, "user:update"), userHandler.UpdateUser)                                 // 更新用户需要权限
		userGroup.DELETE("/:id", middleware.RequirePermission(db, "user:delete"), userHandler.DeleteUser)                              // 删除用户需要权限
	}

	// 部门管理路由
	deptHandler := api.NewDepartmentHandler(db)
	deptGroup := r.Group("/api/departments", middleware.Auth())
	{
		deptGroup.GET("", middleware.RequirePermission(db, "department:read"), deptHandler.GetDepartments)
		deptGroup.POST("", middleware.RequirePermission(db, "department:create"), deptHandler.CreateDepartment)
		// 部门成员管理（必须在 /:id 之前，因为路由按顺序匹配）
		deptGroup.GET("/:id/members", middleware.RequirePermission(db, "department:read"), deptHandler.GetDepartmentMembers)
		deptGroup.POST("/:id/members", middleware.RequirePermission(db, "department:update"), deptHandler.AddDepartmentMembers)
		deptGroup.DELETE("/:id/members/:user_id", middleware.RequirePermission(db, "department:update"), deptHandler.RemoveDepartmentMember)
		// 部门CRUD（放在成员管理之后）
		deptGroup.GET("/:id", middleware.RequirePermission(db, "department:read"), deptHandler.GetDepartment)
		deptGroup.PUT("/:id", middleware.RequirePermission(db, "department:update"), deptHandler.UpdateDepartment)
		deptGroup.DELETE("/:id", middleware.RequirePermission(db, "department:delete"), deptHandler.DeleteDepartment)
	}

	// 个人工作台路由
	dashboardHandler := api.NewDashboardHandler(db)
	dashboardGroup := r.Group("/api/dashboard", middleware.Auth())
	{
		dashboardGroup.GET("", dashboardHandler.GetDashboard)
		dashboardGroup.GET("/config", dashboardHandler.GetDashboardConfig)
		dashboardGroup.POST("/config", dashboardHandler.SaveDashboardConfig)
	}

	// 标签管理路由（标签是系统资源，使用项目权限）
	tagHandler := api.NewTagHandler(db)
	tagGroup := r.Group("/api/tags", middleware.Auth())
	{
		tagGroup.GET("", middleware.RequirePermission(db, "project:read"), tagHandler.GetTags)
		tagGroup.GET("/:id", middleware.RequirePermission(db, "project:read"), tagHandler.GetTag)
		tagGroup.POST("", middleware.RequirePermission(db, "project:update"), tagHandler.CreateTag)
		tagGroup.PUT("/:id", middleware.RequirePermission(db, "project:update"), tagHandler.UpdateTag)
		tagGroup.DELETE("/:id", middleware.RequirePermission(db, "project:delete"), tagHandler.DeleteTag)
	}

	// 项目管理路由
	projectHandler := api.NewProjectHandler(db)

	// 看板管理路由（需要在项目路由之前定义，因为项目路由中会用到）
	boardHandler := api.NewBoardHandler(db)

	projectGroup := r.Group("/api/projects", middleware.Auth())
	{
		projectGroup.GET("", middleware.RequirePermission(db, "project:read"), projectHandler.GetProjects)
		// 注意：统计接口、看板接口和甘特图接口需要在详情接口之前，避免路由冲突
		projectGroup.GET("/:id/statistics", middleware.RequirePermission(db, "project:read"), projectHandler.GetProjectStatistics)
		projectGroup.GET("/:id/progress", middleware.RequirePermission(db, "project:read"), projectHandler.GetProjectProgress)
		projectGroup.GET("/:id/gantt", middleware.RequirePermission(db, "project:read"), projectHandler.GetProjectGantt)
		// 项目看板路由（需要在详情路由之前）
		projectGroup.GET("/:id/boards", middleware.RequirePermission(db, "project:read"), boardHandler.GetProjectBoards)
		projectGroup.POST("/:id/boards", middleware.RequirePermission(db, "project:manage"), boardHandler.CreateBoard)
		projectGroup.GET("/:id", middleware.RequirePermission(db, "project:read"), projectHandler.GetProject)
		projectGroup.POST("", middleware.RequirePermission(db, "project:create"), projectHandler.CreateProject)
		projectGroup.PUT("/:id", middleware.RequirePermission(db, "project:update"), projectHandler.UpdateProject)
		projectGroup.DELETE("/:id", middleware.RequirePermission(db, "project:delete"), projectHandler.DeleteProject)
		// 项目历史记录
		projectGroup.GET("/:id/history", middleware.RequirePermission(db, "project:read"), projectHandler.GetProjectHistory)
		projectGroup.POST("/:id/history/note", middleware.RequirePermission(db, "project:update"), projectHandler.AddProjectHistoryNote)
		// 项目成员管理
		projectGroup.GET("/:id/members", middleware.RequirePermission(db, "project:read"), projectHandler.GetProjectMembers)
		projectGroup.POST("/:id/members", middleware.RequirePermission(db, "project:manage"), projectHandler.AddProjectMembers)
		projectGroup.PUT("/:id/members/:member_id", middleware.RequirePermission(db, "project:manage"), projectHandler.UpdateProjectMember)
		projectGroup.DELETE("/:id/members/:member_id", middleware.RequirePermission(db, "project:manage"), projectHandler.RemoveProjectMember)
	}

	// 需求管理路由
	requirementHandler := api.NewRequirementHandler(db)
	requirementGroup := r.Group("/api/requirements", middleware.Auth())
	{
		requirementGroup.GET("/statistics", middleware.RequirePermission(db, "requirement:read"), requirementHandler.GetRequirementStatistics)
		requirementGroup.GET("", middleware.RequirePermission(db, "requirement:read"), requirementHandler.GetRequirements)
		requirementGroup.GET("/:id", middleware.RequirePermission(db, "requirement:read"), requirementHandler.GetRequirement)
		requirementGroup.POST("", middleware.RequirePermission(db, "requirement:create"), requirementHandler.CreateRequirement)
		requirementGroup.PUT("/:id", middleware.RequirePermission(db, "requirement:update"), requirementHandler.UpdateRequirement)
		requirementGroup.DELETE("/:id", middleware.RequirePermission(db, "requirement:delete"), requirementHandler.DeleteRequirement)
		requirementGroup.PATCH("/:id/status", middleware.RequirePermission(db, "requirement:update"), requirementHandler.UpdateRequirementStatus)
		requirementGroup.POST("/:id/assign", middleware.RequirePermission(db, "requirement:update"), requirementHandler.AssignRequirement)
		// 需求历史记录
		requirementGroup.GET("/:id/history", middleware.RequirePermission(db, "requirement:read"), requirementHandler.GetRequirementHistory)
		requirementGroup.POST("/:id/history/note", middleware.RequirePermission(db, "requirement:update"), requirementHandler.AddRequirementHistoryNote)
	}

	// 功能模块管理路由（模块是系统资源，使用项目权限）
	moduleHandler := api.NewModuleHandler(db)
	moduleGroup := r.Group("/api/modules", middleware.Auth())
	{
		moduleGroup.GET("", middleware.RequirePermission(db, "project:read"), moduleHandler.GetModules)
		moduleGroup.GET("/:id", middleware.RequirePermission(db, "project:read"), moduleHandler.GetModule)
		moduleGroup.POST("", middleware.RequirePermission(db, "project:update"), moduleHandler.CreateModule)
		moduleGroup.PUT("/:id", middleware.RequirePermission(db, "project:update"), moduleHandler.UpdateModule)
		moduleGroup.DELETE("/:id", middleware.RequirePermission(db, "project:delete"), moduleHandler.DeleteModule)
	}

	// Bug管理路由
	bugHandler := api.NewBugHandler(db)
	bugGroup := r.Group("/api/bugs", middleware.Auth())
	{
		bugGroup.GET("/statistics", middleware.RequirePermission(db, "bug:read"), bugHandler.GetBugStatistics)
		// 列设置路由（必须在 /:id 之前）
		bugGroup.GET("/column-settings", middleware.RequirePermission(db, "bug:read"), bugHandler.GetBugColumnSettings)
		bugGroup.POST("/column-settings", middleware.RequirePermission(db, "bug:read"), bugHandler.SaveBugColumnSettings)
		bugGroup.GET("", middleware.RequirePermission(db, "bug:read"), bugHandler.GetBugs)
		// 历史记录路由（必须在 /:id 之前）
		bugGroup.GET("/:id/history", middleware.RequirePermission(db, "bug:read"), bugHandler.GetBugHistory)
		bugGroup.POST("/:id/history/note", middleware.RequirePermission(db, "bug:update"), bugHandler.AddBugHistoryNote)
		bugGroup.GET("/:id", middleware.RequirePermission(db, "bug:read"), bugHandler.GetBug)
		bugGroup.POST("", middleware.RequirePermission(db, "bug:create"), bugHandler.CreateBug)
		bugGroup.PUT("/:id", middleware.RequirePermission(db, "bug:update"), bugHandler.UpdateBug)
		bugGroup.DELETE("/:id", middleware.RequirePermission(db, "bug:delete"), bugHandler.DeleteBug)
		bugGroup.PATCH("/:id/status", middleware.RequirePermission(db, "bug:update"), bugHandler.UpdateBugStatus)
		bugGroup.POST("/:id/assign", middleware.RequirePermission(db, "bug:assign"), bugHandler.AssignBug)
		bugGroup.POST("/:id/confirm", middleware.RequirePermission(db, "bug:update"), bugHandler.ConfirmBug)
	}

	// 任务管理路由
	taskHandler := api.NewTaskHandler(db)
	taskGroup := r.Group("/api/tasks", middleware.Auth())
	{
		taskGroup.GET("", middleware.RequirePermission(db, "task:read"), taskHandler.GetTasks)
		taskGroup.GET("/:id", middleware.RequirePermission(db, "task:read"), taskHandler.GetTask)
		taskGroup.POST("", middleware.RequirePermission(db, "task:create"), taskHandler.CreateTask)
		taskGroup.PUT("/:id", middleware.RequirePermission(db, "task:update"), taskHandler.UpdateTask)
		taskGroup.DELETE("/:id", middleware.RequirePermission(db, "task:delete"), taskHandler.DeleteTask)
		taskGroup.PATCH("/:id/status", middleware.RequirePermission(db, "task:update"), taskHandler.UpdateTaskStatus)
		taskGroup.POST("/:id/assign", middleware.RequirePermission(db, "task:update"), taskHandler.AssignTask)
		// 任务历史记录
		taskGroup.GET("/:id/history", middleware.RequirePermission(db, "task:read"), taskHandler.GetTaskHistory)
		taskGroup.POST("/:id/history/note", middleware.RequirePermission(db, "task:update"), taskHandler.AddTaskHistoryNote)
		taskGroup.PATCH("/:id/progress", middleware.RequirePermission(db, "task:update"), taskHandler.UpdateTaskProgress)
	}

	// 看板管理路由（看板属于项目的一部分）
	boardGroup := r.Group("/api/boards", middleware.Auth())
	{
		boardGroup.GET("/:id", middleware.RequirePermission(db, "project:read"), boardHandler.GetBoard)
		boardGroup.PUT("/:id", middleware.RequirePermission(db, "project:manage"), boardHandler.UpdateBoard)
		boardGroup.DELETE("/:id", middleware.RequirePermission(db, "project:manage"), boardHandler.DeleteBoard)
		boardGroup.GET("/:id/tasks", middleware.RequirePermission(db, "project:read"), boardHandler.GetBoardTasks)
		boardGroup.PATCH("/:id/tasks/:task_id/move", middleware.RequirePermission(db, "project:manage"), boardHandler.MoveTask)
		boardGroup.POST("/:id/columns", middleware.RequirePermission(db, "project:manage"), boardHandler.CreateBoardColumn)
		boardGroup.PUT("/:id/columns/:column_id", middleware.RequirePermission(db, "project:manage"), boardHandler.UpdateBoardColumn)
		boardGroup.DELETE("/:id/columns/:column_id", middleware.RequirePermission(db, "project:manage"), boardHandler.DeleteBoardColumn)
	}

	// 版本管理路由（版本属于项目的一部分）
	versionHandler := api.NewVersionHandler(db)
	versionGroup := r.Group("/api/versions", middleware.Auth())
	{
		versionGroup.GET("", middleware.RequirePermission(db, "project:read"), versionHandler.GetVersions)
		versionGroup.GET("/:id", middleware.RequirePermission(db, "project:read"), versionHandler.GetVersion)
		versionGroup.POST("", middleware.RequirePermission(db, "project:update"), versionHandler.CreateVersion)
		versionGroup.PUT("/:id", middleware.RequirePermission(db, "project:update"), versionHandler.UpdateVersion)
		versionGroup.DELETE("/:id", middleware.RequirePermission(db, "project:delete"), versionHandler.DeleteVersion)
		versionGroup.PATCH("/:id/status", middleware.RequirePermission(db, "project:update"), versionHandler.UpdateVersionStatus)
		versionGroup.POST("/:id/release", middleware.RequirePermission(db, "project:update"), versionHandler.ReleaseVersion)
	}

	// 测试单管理路由（测试用例属于项目的一部分）
	testCaseHandler := api.NewTestCaseHandler(db)
	testCaseGroup := r.Group("/api/test-cases", middleware.Auth())
	{
		testCaseGroup.GET("/statistics", middleware.RequirePermission(db, "project:read"), testCaseHandler.GetTestCaseStatistics)
		testCaseGroup.GET("", middleware.RequirePermission(db, "project:read"), testCaseHandler.GetTestCases)
		testCaseGroup.GET("/:id", middleware.RequirePermission(db, "project:read"), testCaseHandler.GetTestCase)
		testCaseGroup.POST("", middleware.RequirePermission(db, "test-case:create"), testCaseHandler.CreateTestCase)
		testCaseGroup.PUT("/:id", middleware.RequirePermission(db, "test-case:update"), testCaseHandler.UpdateTestCase)
		testCaseGroup.DELETE("/:id", middleware.RequirePermission(db, "test-case:delete"), testCaseHandler.DeleteTestCase)
		testCaseGroup.PATCH("/:id/status", middleware.RequirePermission(db, "test-case:update"), testCaseHandler.UpdateTestCaseStatus)
	}

	// 资源管理路由 (统计、冲突检测、利用率分析)
	resourceHandler := api.NewResourceHandler(db)
	resourceGroup := r.Group("/api/resources", middleware.Auth())
	{
		resourceGroup.GET("/statistics", middleware.RequirePermission(db, "resource:read"), resourceHandler.GetResourceStatistics)
		resourceGroup.GET("/utilization", middleware.RequirePermission(db, "resource:read"), resourceHandler.GetResourceUtilization)
		resourceGroup.GET("/conflict", middleware.RequirePermission(db, "resource:read"), resourceHandler.CheckResourceConflict)
		// resourceGroup.GET("", resourceHandler.GetResources) // Removed
		// resourceGroup.GET("/:id", resourceHandler.GetResource) // Removed
		// resourceGroup.POST("", resourceHandler.CreateResource) // Removed
		// resourceGroup.PUT("/:id", resourceHandler.UpdateResource) // Removed
		// resourceGroup.DELETE("/:id", resourceHandler.DeleteResource) // Removed
	}

	// 资源分配管理路由
	resourceAllocationHandler := api.NewResourceAllocationHandler(db)
	resourceAllocationGroup := r.Group("/api/resource-allocations", middleware.Auth())
	{
		resourceAllocationGroup.GET("/calendar", middleware.RequirePermission(db, "resource:read"), resourceAllocationHandler.GetResourceCalendar)
		resourceAllocationGroup.GET("/conflict", middleware.RequirePermission(db, "resource:read"), resourceAllocationHandler.CheckResourceConflict)
		resourceAllocationGroup.GET("", middleware.RequirePermission(db, "resource:read"), resourceAllocationHandler.GetResourceAllocations)
		resourceAllocationGroup.GET("/:id", middleware.RequirePermission(db, "resource:read"), resourceAllocationHandler.GetResourceAllocation)
		resourceAllocationGroup.POST("", middleware.RequirePermission(db, "resource:manage"), resourceAllocationHandler.CreateResourceAllocation)
		resourceAllocationGroup.PUT("/:id", middleware.RequirePermission(db, "resource:manage"), resourceAllocationHandler.UpdateResourceAllocation)
		resourceAllocationGroup.DELETE("/:id", middleware.RequirePermission(db, "resource:manage"), resourceAllocationHandler.DeleteResourceAllocation)
	}

	// 工作报告路由（日报和周报）
	reportHandler := api.NewReportHandler(db)
	reportGroup := r.Group("/api/reports", middleware.Auth())
	{
		reportGroup.GET("/work-summary", reportHandler.GetWorkSummary) // 获取工作内容汇总
	}
	dailyReportGroup := r.Group("/api/daily-reports", middleware.Auth())
	{
		dailyReportGroup.GET("", reportHandler.GetDailyReports)
		dailyReportGroup.GET("/:id", reportHandler.GetDailyReport)
		dailyReportGroup.POST("", reportHandler.CreateDailyReport)
		dailyReportGroup.PUT("/:id", reportHandler.UpdateDailyReport)
		dailyReportGroup.DELETE("/:id", reportHandler.DeleteDailyReport)
		dailyReportGroup.PATCH("/:id/status", reportHandler.UpdateDailyReportStatus)
		dailyReportGroup.POST("/:id/approve", reportHandler.ApproveDailyReport)
	}
	weeklyReportGroup := r.Group("/api/weekly-reports", middleware.Auth())
	{
		weeklyReportGroup.GET("", reportHandler.GetWeeklyReports)
		weeklyReportGroup.GET("/:id", reportHandler.GetWeeklyReport)
		weeklyReportGroup.POST("", reportHandler.CreateWeeklyReport)
		weeklyReportGroup.PUT("/:id", reportHandler.UpdateWeeklyReport)
		weeklyReportGroup.DELETE("/:id", reportHandler.DeleteWeeklyReport)
		weeklyReportGroup.PATCH("/:id/status", reportHandler.UpdateWeeklyReportStatus)
		weeklyReportGroup.POST("/:id/approve", reportHandler.ApproveWeeklyReport)
	}

	// 附件管理路由
	attachmentHandler := api.NewAttachmentHandler(db)
	attachmentGroup := r.Group("/api/attachments", middleware.Auth())
	{
		attachmentGroup.POST("/upload", middleware.RequirePermission(db, "attachment:upload"), attachmentHandler.UploadFile)
		attachmentGroup.GET("/:id", attachmentHandler.GetAttachment)
		attachmentGroup.GET("/:id/download", attachmentHandler.DownloadFile)
		attachmentGroup.GET("/:id/preview", attachmentHandler.PreviewFile)
		attachmentGroup.DELETE("/:id", middleware.RequirePermission(db, "attachment:delete"), attachmentHandler.DeleteAttachment)
		attachmentGroup.GET("", attachmentHandler.GetAttachments)
		attachmentGroup.POST("/:id/attach", attachmentHandler.AttachToEntity)
	}

	// 静态文件服务（上传的文件）
	// 配置上传文件的静态服务，路径为 /uploads/*
	storagePath := config.AppConfig.Upload.StoragePath
	if !filepath.IsAbs(storagePath) {
		storagePath = filepath.Join(".", storagePath)
	}
	// 确保目录存在
	if err := os.MkdirAll(storagePath, 0755); err == nil {
		r.Static("/uploads", storagePath)
		log.Printf("上传文件静态服务目录: %s", storagePath)
	} else {
		log.Printf("警告: 无法创建上传文件目录 %s: %v", storagePath, err)
	}

	// 系统设置路由（微信配置等）
	wechatHandler := api.NewWeChatHandler(db)
	systemHandler := api.NewSystemHandler(db)
	// 使用可选认证中间件：系统未初始化时允许访问，已初始化时需要认证
	systemGroup := r.Group("/api/system", middleware.AuthOptional(db))
	{
		systemGroup.GET("/wechat-config", wechatHandler.GetWeChatConfig)
		systemGroup.POST("/wechat-config", middleware.RequirePermissionOptional(db, "wechat:settings"), wechatHandler.SaveWeChatConfig)
		// 备份配置路由
		systemGroup.GET("/backup-config", middleware.RequirePermissionOptional(db, "system:settings"), systemHandler.GetBackupConfig)
		systemGroup.POST("/backup-config", middleware.RequirePermissionOptional(db, "system:settings"), systemHandler.SaveBackupConfig)
		systemGroup.POST("/backup/trigger", middleware.RequirePermissionOptional(db, "system:settings"), systemHandler.TriggerBackup)
		// 日志管理路由
		systemGroup.GET("/log-level", systemHandler.GetLogLevel)
		systemGroup.POST("/log-level", middleware.RequirePermissionOptional(db, "log:settings"), systemHandler.SetLogLevel)
		systemGroup.GET("/log-files", middleware.RequirePermissionOptional(db, "log:settings"), systemHandler.GetLogFiles)
		systemGroup.GET("/log-files/:filename", middleware.RequirePermissionOptional(db, "log:settings"), systemHandler.DownloadLogFile)
	}

	// 审计日志路由
	auditLogHandler := api.NewAuditLogHandler(db)
	auditLogGroup := r.Group("/api/audit-logs", middleware.Auth())
	{
		auditLogGroup.GET("", middleware.RequirePermission(db, "audit:read"), auditLogHandler.GetAuditLogs)
		auditLogGroup.GET("/:id", middleware.RequirePermission(db, "audit:read"), auditLogHandler.GetAuditLog)
	}

	// 静态文件服务（前端构建后的文件，使用 embed 嵌入）
	// 注意：必须在所有 API 路由之后，但在 catch-all 路由之前
	// 从 embed.FS 中获取前端文件系统
	// embed 路径是 frontend-dist，所以文件系统直接包含 dist 目录的内容
	// 使用 Sub 获取 frontend-dist 子目录
	distFS, err := fs.Sub(frontendFS, "frontend-dist")
	if err != nil {
		if utils.Logger != nil {
			utils.Logger.Warnf("警告: 无法获取前端文件系统子目录: %v，尝试使用外部文件", err)
		} else {
			log.Printf("警告: 无法获取前端文件系统子目录: %v，尝试使用外部文件", err)
		}
		setupExternalFrontend(r)
	} else {
		// 检查是否有文件（至少应该有 index.html）
		if _, err := fs.Stat(distFS, "index.html"); err != nil {
			if utils.Logger != nil {
				utils.Logger.Warnf("警告: 嵌入的前端文件系统中未找到 index.html: %v，尝试使用外部文件", err)
			} else {
				log.Printf("警告: 嵌入的前端文件系统中未找到 index.html: %v，尝试使用外部文件", err)
			}
			setupExternalFrontend(r)
		} else {
			if utils.Logger != nil {
				utils.Logger.Info("使用嵌入的前端静态文件（embed）")
			} else {
				log.Println("使用嵌入的前端静态文件（embed）")
			}

			// 提供静态资源文件（如 JS、CSS、图片等）
			// 使用自定义处理器确保正确的 MIME 类型
			r.GET("/assets/*filepath", func(c *gin.Context) {
				// 获取请求的文件路径
				filePath := c.Param("filepath")
				// 移除开头的斜杠（因为 filepath 参数包含 /）
				if len(filePath) > 0 && filePath[0] == '/' {
					filePath = filePath[1:]
				}
				// 构建完整路径（assets/xxx.js）
				fullPath := "assets/" + filePath

				// 读取文件
				data, err := fs.ReadFile(distFS, fullPath)
				if err != nil {
					c.Status(http.StatusNotFound)
					return
				}

				// 根据文件扩展名设置正确的 Content-Type
				contentType := getContentType(fullPath, data)
				c.Data(http.StatusOK, contentType, data)
			})

			// 提供 vite.svg 文件（如果存在）
			r.GET("/vite.svg", func(c *gin.Context) {
				data, err := fs.ReadFile(distFS, "vite.svg")
				if err != nil {
					c.Status(http.StatusNotFound)
					return
				}
				c.Data(http.StatusOK, "image/svg+xml", data)
			})

			// SPA 路由处理：所有非 API 路由都返回 index.html
			// 这样前端路由（如 /dashboard, /login 等）可以正常工作
			r.NoRoute(func(c *gin.Context) {
				// 如果请求的是 API 路径，返回 404
				path := c.Request.URL.Path
				if len(path) >= 4 && path[:4] == "/api" {
					c.JSON(http.StatusNotFound, gin.H{
						"code":    404,
						"message": "API endpoint not found",
					})
					return
				}

				// 尝试直接提供请求的文件（如 /assets/xxx.js）
				if path != "/" && path != "" {
					// 移除开头的斜杠
					filePath := path[1:]
					if data, err := fs.ReadFile(distFS, filePath); err == nil {
						// 根据文件扩展名设置正确的 Content-Type
						contentType := getContentType(filePath, data)
						c.Data(http.StatusOK, contentType, data)
						return
					}
				}

				// 否则返回前端 index.html（用于 SPA 路由）
				indexData, err := fs.ReadFile(distFS, "index.html")
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{
						"code":    404,
						"message": "Frontend not found",
					})
					return
				}
				c.Data(http.StatusOK, "text/html; charset=utf-8", indexData)
			})
		}
	}

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.AppConfig.Server.Port),
		Handler:      r,
		ReadTimeout:  time.Duration(config.AppConfig.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.AppConfig.Server.WriteTimeout) * time.Second,
	}

	// 启动备份定时任务
	scheduler := utils.GetBackupScheduler(db)
	scheduler.Start()
	if utils.Logger != nil {
		utils.Logger.Info("Backup scheduler started")
	} else {
		log.Println("Backup scheduler started")
	}

	// 启动服务器（异步）
	go func() {
		if utils.Logger != nil {
			utils.Logger.Infof("Server starting on port %d", config.AppConfig.Server.Port)
		} else {
			log.Printf("Server starting on port %d", config.AppConfig.Server.Port)
		}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			if utils.Logger != nil {
				utils.Logger.Fatalf("Failed to start server: %v", err)
			} else {
				log.Fatalf("Failed to start server: %v", err)
			}
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if utils.Logger != nil {
		utils.Logger.Info("Shutting down server...")
	} else {
		log.Println("Shutting down server...")
	}

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		if utils.Logger != nil {
			utils.Logger.Fatalf("Server forced to shutdown: %v", err)
		} else {
			log.Fatalf("Server forced to shutdown: %v", err)
		}
	}

	if utils.Logger != nil {
		utils.Logger.Info("Server exited")
	} else {
		log.Println("Server exited")
	}
}
