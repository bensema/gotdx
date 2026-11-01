// Package types 提供 `gotdx` 客户端与协议共用的共享类型、枚举常量与代码工具。
//
// 包内集中定义跨协议复用的取值集合，包括市场（`Market`）、板块类型（`BlockType`）、
// 扩展市场品类（`ExCategory`）、排序与过滤（`Sort`*、`Filter`*）、K 线周期（`KLINE_TYPE`_*）、
// 复权方式（`Adjust`*）等枚举，以及股票代码规范化与市场识别工具
// （`StockPrefix`、`DetectMarket`、`DecodeStockCode` 等）。
//
// 这些常量直接对应协议报文中的字段取值，上层方法以本包枚举作为参数即可获得
// 可读的语义，无需记忆原始数值。
package types
