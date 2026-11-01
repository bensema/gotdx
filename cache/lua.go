// 文件 lua.go 集中定义缓存机制使用的全部 `Lua` 脚本，借助 `Redis`
// 服务端原子执行保证并发场景不丢更新、不重复流转、不错删锁。
//
// 脚本以 `redis.NewScript` 封装：首次调用时服务端 `SCRIPT LOAD` 缓存，
// 之后走 `EVALSHA` 只传摘要，节省高频路径（tick 更新）的带宽。

package cache

import "github.com/redis/go-redis/v9"

// liveScriptSrc 完成未闭合根的单次原子更新，把「同根覆盖 / 前一根闭合
// 流转 / 续写 TTL」合并为一步：
//
//   - 读取 `String` 实时槽旧值并解析首段分数；
//   - 旧值分数不小于新分数时，判定新 `tick` 过期或乱序，直接忽略；
//   - 旧值分数小于新分数时，判定旧值（前一根）已闭合，把其 base64
//     数据原样 ZADD 进 `ZSet`，随后用新值覆盖实时槽并续写 `TTL`。
//
// 参数：KEYS[1]=实时槽键，KEYS[2]=`ZSet` 键；ARGV[1]=新值（分数|更新
// 时刻|base64 数据），ARGV[2]=新分数文本，ARGV[3]=`TTL` 秒数。返回 1
// 表示更新生效，0 表示被忽略。
const liveScriptSrc = `
local old = redis.call('GET', KEYS[1])
if old then
  local sep1 = string.find(old, '|')
  if sep1 then
    local oldscore = tonumber(string.sub(old, 1, sep1 - 1))
    if oldscore >= tonumber(ARGV[2]) then
      return 0
    end
    local sep2 = string.find(old, '|', sep1 + 1)
    if sep2 then
      redis.call('ZADD', KEYS[2], oldscore, string.sub(old, sep2 + 1))
    end
  end
end
redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[3])
return 1`

// rewriteScriptSrc 完成历史修正写入：先按分数删除同分数的旧成员再 ZADD，
// 保证修正覆盖而不是同分数堆积重复。
//
// 参数：KEYS[1]=`ZSet` 键；ARGV[1]=分数文本，ARGV[2]=新成员（base64
// 数据）。
const rewriteScriptSrc = `
redis.call('ZREMRANGEBYSCORE', KEYS[1], ARGV[1], ARGV[1])
redis.call('ZADD', KEYS[1], ARGV[1], ARGV[2])
return 1`

// unlockScriptSrc 释放分布式锁：仅当锁值仍为本进程持有的 token 时删除，
// 防止误删其他进程新获取的锁。
//
// 参数：KEYS[1]=锁键；ARGV[1]=token。
const unlockScriptSrc = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0`

// liveScript 为实时槽原子更新的封装脚本。
var liveScript = redis.NewScript(liveScriptSrc)

// rewriteScript 为历史修正写入的封装脚本。
var rewriteScript = redis.NewScript(rewriteScriptSrc)

// unlockScript 为分布式锁安全释放的封装脚本。
var unlockScript = redis.NewScript(unlockScriptSrc)
