package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// User 用户数据结构
type User struct {
	Phone string `json:"phone"`
	Mail  string `json:"mail"`
}

var users []User

// loadData 从文件加载数据
func loadData(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("无法打开文件: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("无法读取文件: %w", err)
	}

	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("无法解析 JSON: %w", err)
	}

	return nil
}

// usersHandler 处理用户列表请求
func usersHandler(w http.ResponseWriter, r *http.Request) {
	// 检查认证头
	authHeader := r.Header.Get("Authorization")
	if authHeader != "Bearer mock-token" && authHeader != "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "未授权的访问",
		})
		return
	}

	// 重新加载数据（支持热更新）
	if err := loadData("./data.json"); err != nil {
		log.Printf("加载数据失败: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(w).Encode(users)
}

// healthHandler 健康检查端点
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"service":   "mock-api",
		"timestamp": time.Now().Unix(),
	})
}

func main() {
	// 加载初始数据
	dataFile := "./data.json"
	if len(os.Args) > 1 {
		dataFile = os.Args[1]
	}

	if err := loadData(dataFile); err != nil {
		log.Fatalf("加载数据失败: %v", err)
	}

	log.Printf("已加载 %d 个用户", len(users))

	// 注册路由
	http.HandleFunc("/api/users", usersHandler)
	http.HandleFunc("/health", healthHandler)

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Mock API 服务启动在端口 %s", port)
	log.Printf("用户列表端点: http://localhost:%s/api/users", port)
	log.Printf("健康检查端点: http://localhost:%s/health", port)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
