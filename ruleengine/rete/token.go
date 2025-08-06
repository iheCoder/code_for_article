package rete

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"

	"github.com/ihewe/code_for_article/ruleengine/model"
)

// Token 是 β 网络中向下传播的“事实组合链”。
// 通过 Facts 切片串联起 join 过程中累计的事实，用于后续节点继续匹配。
// Hash 字段用于在 BetaMemory 中做去重与检索。

type Token struct {
	Facts []model.Fact
	hash  string
}

// NewToken 创建一个新的 Token，并生成唯一哈希。
func NewToken(facts []model.Fact) Token {
	t := Token{Facts: facts}
	t.hash = t.computeHash()
	return t
}

// Hash 返回 token 的唯一标识。
func (t Token) Hash() string { return t.hash }

func (t Token) computeHash() string {
	var keys []string
	for _, f := range t.Facts {
		keys = append(keys, f.Key())
	}
	joined := strings.Join(keys, "|")
	sum := sha1.Sum([]byte(joined))
	return hex.EncodeToString(sum[:])
}
