#include "textflag.h"

// func DotProductASM(a []float64, b []float64) float64
TEXT ·DotProductASM(SB), NOSPLIT, $0
	// 加载切片信息
	MOVD a_base+0(FP), R0    // a的数据指针
	MOVD a_len+8(FP), R2     // a的长度
	MOVD b_base+24(FP), R1   // b的数据指针

	// 使用4个寄存器作为累加器，最终合并
	FMOVD $0.0, F0           // 累加器1
	FMOVD $0.0, F1           // 累加器2
	FMOVD $0.0, F2           // 累加器3
	FMOVD $0.0, F3           // 累加器4

	// 检查长度是否足够进行向量化处理（至少4个元素）
	CMP $4, R2
	BLT small_array          // 如果少于4个元素，跳到特殊处理

	// 计算主循环迭代次数（每次处理4个元素）
	MOVD R2, R4              // 保存原始长度
	LSR $2, R2               // R2 = R2 / 4（向量化循环次数）

vector_loop:
	// 批量处理4个元素
	FMOVD (R0), F4           // 加载a[0]
	ADD $8, R0
	FMOVD (R0), F5           // 加载a[1]
	ADD $8, R0
	FMOVD (R0), F6           // 加载a[2]
	ADD $8, R0
	FMOVD (R0), F7           // 加载a[3]
	ADD $8, R0

	FMOVD (R1), F8           // 加载b[0]
	ADD $8, R1
	FMOVD (R1), F9           // 加载b[1]
	ADD $8, R1
	FMOVD (R1), F10          // 加载b[2]
	ADD $8, R1
	FMOVD (R1), F11          // 加载b[3]
	ADD $8, R1

	// 乘法+累加
	FMULD F4, F8, F12
	FMULD F5, F9, F13
	FMULD F6, F10, F14
	FMULD F7, F11, F15

	FADDD F12, F0, F0
	FADDD F13, F1, F1
	FADDD F14, F2, F2
	FADDD F15, F3, F3

	// 循环控制
	SUBS $1, R2
	BNE vector_loop

	// 合并四个累加器结果
	FADDD F1, F0, F0
	FADDD F2, F0, F0
	FADDD F3, F0, F0

	// 处理剩余元素（0-3个）
	AND $3, R4, R2           // R2 = R4 % 4（剩余元素）
	CBZ R2, done             // 如果没有剩余元素，结束
	B remainder

small_array:
	// 小数组的特殊处理（少于4个元素）
	CBZ R2, done             // 如果长度为0，直接返回

remainder:
	// 处理剩余元素（1个1个处理）
	FMOVD (R0), F4           // 加载a[i]
	FMOVD (R1), F5           // 加载b[i]
	FMULD F4, F5, F6         // F6 = a[i] * b[i]
	FADDD F6, F0, F0         // 累加到结果中

	// 更新指针
	ADD $8, R0
	ADD $8, R1

	// 更新计数并继续处理剩余元素
	SUBS $1, R2
	BNE remainder

done:
	// 返回最终结果
	FMOVD F0, ret+48(FP)
	RET

