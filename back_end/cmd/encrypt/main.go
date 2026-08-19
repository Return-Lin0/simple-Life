// 数据库密码加密工具：使用环境变量 VIBE_DB_KEY 加密明文，输出密文。
// 用法：
//   go run ./cmd/encrypt -plaintext '你的数据库密码'
// 将输出写入 config/config.yaml 的 mysql.password_cipher 字段。
package main

import (
	"flag"
	"fmt"
	"os"

	"vibe/internal/auth"
)

func main() {
	plaintext := flag.String("plaintext", "", "待加密的数据库密码明文")
	flag.Parse()
	if *plaintext == "" {
		fmt.Fprintln(os.Stderr, "请通过 -plaintext 传入数据库密码")
		os.Exit(1)
	}
	key := os.Getenv("VIBE_DB_KEY")
	if key == "" {
		fmt.Fprintln(os.Stderr, "缺少环境变量 VIBE_DB_KEY（64 位十六进制）")
		os.Exit(1)
	}
	cipher, err := auth.EncryptAESGCM(key, *plaintext)
	if err != nil {
		fmt.Fprintln(os.Stderr, "加密失败:", err)
		os.Exit(1)
	}
	fmt.Println(cipher)
}
