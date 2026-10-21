# 暂停 AI session

回复目标 AI session 中的消息发送 `/suspend [时长|off]`，仅配置中的 `god` 用户可用。
不回复消息时只提示用法，不隐式选择当前 session。时长接受正整数及 `s`、`m`、`h`、`d`， 例如
`/suspend 10s`、`/suspend 1m`、`/suspend 2h`、`/suspend 1d`；无参数永久暂停， `/suspend off`
恢复。重复设置覆盖此前期限，期限从命令执行时开始计算。

群命令菜单将此命令标为 ephemeral，确认、错误及权限提示也使用 ephemeral 回复。 Bot
无法在群内私密回复时不执行、不回退为公开回复；过期的 ephemeral 输入不能修改状态。 私聊沿用 Telegram
普通私聊回复。临时消息 ID 不进入 AI 历史或传统功能上下文。

暂停以 session ID 为单位影响该会话所有参与者，跨话题的同聊天引用也受限；跨聊天引用仍被禁止。
回复、`@会话ID`、`@last` 和已生成但未交付的请求均检查暂停状态。 `@new` 创建独立 session，不受另一个
session 的暂停影响。暂停不删除历史、不改变模型、 不重置 provider state；到期或 off 后可以沿原
session 继续。暂停期间不收集被忽略的 AI 请求。 传统处理器仍先执行，保留传统功能原有行为。

暂停时，已排队请求不再执行，正在生成的请求及其排队请求即使立即 off 也不发送回复。
已经调用的模型可能仍完成计费；成功响应的原始 payload 和 usage 继续保存为 generated，
不交付、不提前提升 provider state。off 不自动重发；恢复后显式重投相同请求可沿用已保存结果。 已提交给
Telegram 的发送无法撤回，本命令不会删除已发出的消息。

新增 TS migration 004，`ai_session_suspensions` 保存 session ID 与 Unix 秒到期时间， NULL
为永久，删除记录表示 off；到期按读取时判断，无后台模型调用或定时删除。 SQL 规范源码及生成流程见
[database.md](database.md)。仅离线临时库验证，不升级线上数据库或重启服务。

本次实现前已刷新远端：TS `origin/master` 与 `origin/codex/retain-features` 为
`ac67c5f7651d3e2a4a2225ae4bd3f3a9f455e979`；Go 最新保留主线 `origin/archive/go/20261013/master` 为
`8f5de9b9161d64d813e85c68a95258f879aa0dd8`， 检查其 `handlers/genbot/session.go` 的
session/历史结构，并以当前 TS session 路由、事务和 ephemeral 实现为基础添加新能力。
