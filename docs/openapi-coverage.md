# OpenAPI 覆盖与责任清单

本清单记录 **CLI 已实现的范围与已知缺口**，不是 AtomGit 全部端点目录，也不是服务端兼容性认证。基线为 2026-09-20 的 `main`（`c514720`）；后续修改必须随代码更新对应行。命令参数见[生成的命令参考](command-reference.md)，使用示例见[使用指南](usage.md)，测试机制见[API 契约测试](api-contracts.md)。三者不能互相替代。

## 状态、责任与证据

- `implemented`：本行列出的范围已在代码中注册/实现，不表示整个 OpenAPI 模块完整覆盖。
- `partial`：有可用实现，但仍有本页列明的缺口。
- `missing`：已知需求尚无合入实现；开放 PR 不算完成。
- `deferred`：有意暂不实现，必须说明原因及重新评估条件。
- `unverified`：接口契约/授权范围未核实，不据此猜测端点或承诺命令。
- `project`：本项目维护；`partner`：AtomGit 合作方负责的模块，未经明确移交不自动加入本项目待办。责任归属与已有代码维护分开：合作方模块中的既有 CLI 能力仍由本项目维护。

验证标记不是逐级自动升级的认证：

- `source`：核对命令注册、实现和仓库文档；不等同于核对每个官方接口。
- `mock`：有离线模拟 HTTP/注入依赖测试证据；不代表每个参数和权限组合均已覆盖。
- `local`：本地行为有离线测试，不涉及 AtomGit API。
- `docs`：仅核对官方文档，尚无实现或运行证据。
- `live`：仅在附有日期、提交、端点/只读场景、脱敏结果及可访问记录时使用。一次成功不能推广到整个模块。

本次清单不执行真实 API 验收，**下面各行的在线验证均为未记录**，不把历史 PR 的测试声明升级为当前在线结果。`license` 只有命令结构测试，特别不标为 HTTP mock 验证。维护者若补充在线证据，不得写入凭据、真实响应或私有资源名。

官方 [OpenAPI 目录](https://docs.atomgit.com/docs/apis/) 用于模块分类（2026-09-17 核对）：Repositories、Branch、Issues、Search、Pull Requests、Commit、Tag、Labels、Milestone、Users、Organizations、Webhooks、Member、Release、Enterprise、Dashboard、Actions、Discussion、OAuth、AI Hub。下面路径为从代码核对的**端点家族示意**，省略 API 基址和部分参数，不应直接当作请求模板。未知或未覆盖模块不计入覆盖率分母；本项目不发布误导性的“百分之百覆盖”数字。

## 已注册命令族

本表是人工维护的唯一清单源。每个根命令恰好一行，包括隐藏兼容命令；隐式帮助/自动补全和别名不作为独立 API 能力。八列顺序固定，单元格内不用竖线。离线测试会检查字段、重复/遗漏根命令及本页仓库相对链接，但不会自动证明端点语义正确。

### API v5

<!-- coverage:start -->
| 命令族 | 协议 | API 家族与已实现范围 | 状态 | 责任 | 验证 | 代码/测试证据 | 关联工作与边界 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| doctor | local,v5,v8 | 配置、凭据和可选只读连通性检查 | implemented | project | source,mock | [命令与测试](../pkg/cmd/doctor/) | [#131](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/131)；不含逐请求 debug 跟踪、自动修复或写权限验证 |
| schema | local | 基于 Cobra 命令树的版本化 JSON 命令说明 | implemented | project | source,mock | [命令与测试](../pkg/cmd/schema/)、[元数据](../internal/commandschema/) | [#151](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/151)；不执行目标命令；未描述的业务规则、响应结构和权限明确标记 |
| api | v5 | 通用 v5 请求、JSON、分页、无网络和凭据访问的脱敏 dry-run 预览；没有 v8 版本选项 | implemented | project | source,mock | [实现与测试](../pkg/cmd/api/) | [#28](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/28)、[#150](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/150)；预览不验证远端权限或业务规则，不能用它代替 v8 专用客户端 |
| repo | v5 | Repositories：元数据、clone/fork/sync、collaborators、hooks、contents、push_rule、remote_mirrors、transfer、六项 insights、policy 的 permission/code-review/pull-request 查看与编辑 | implemented | project | source,mock | [命令与测试](../pkg/cmd/repo/)、[policy API](../internal/api/repository_policy.go) | [#89](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/89)、[#90](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/90)、[#117](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/117)、[#118](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/118)、[#119](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/119)；仅列明范围，不代表全部仓库 API |
| branch | v5 | Branch：/repos/{owner}/{repo}/branches 及保护规则；列表、详情、创建、删除、保护设置 | implemented | project | source,mock | [命令与测试](../pkg/cmd/branch/) | [#109](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/109)、[#111](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/111)；仅列明能力 |
| commit | v5 | Commit：提交列表/详情、compare、diff、patch，以及评论列表/详情/创建/编辑/删除 | implemented | project | source,mock | [命令与测试](../pkg/cmd/commit/)、[评论命令与测试](../pkg/cmd/commit/comment/)、[API](../internal/api/commits.go) | [#84](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/84)、[#91](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/91)、[#113](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/113) |
| issue | v5 | Issues：CRUD、标签、评论、关联 PR、related_branches、activity/history/reactions；包含 /repos/{owner}/issues/{number} 路径 | implemented | project | source,mock | [命令与测试](../pkg/cmd/issue/)、[API](../internal/api/issue_collaboration.go) | [#86](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/86)；活动和历史限制为本地截断，表态按页获取；仅列明范围 |
| tag | v5 | Tag：仓库 tags 创建/列举/删除与保护规则 | implemented | project | source,mock | [命令与测试](../pkg/cmd/tag/) | [#76](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/76)、[#103](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/103) |
| label | v5 | Labels：仓库标签列表、创建、编辑、删除 | implemented | project | source,mock | [命令与测试](../pkg/cmd/label/) | [#26](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/26) |
| milestone | v5 | Milestone：列表、详情、创建、编辑、关闭、重开、删除 | implemented | project | source,mock | [命令与测试](../pkg/cmd/milestone/) | [#67](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/67) |
| notification | v5 | /repos/{owner}/{repo}/notifications：仓库通知列表、标记已读 | implemented | project | source,mock | [命令与测试](../pkg/cmd/notification/)、[API](../internal/api/notifications.go) | [#92](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/92)；不是全站消息中心 |
| release | v5 | Release：列举/详情/创建/编辑，附件上传、下载；含独立附件传输 | implemented | project | source,mock | [命令与测试](../pkg/cmd/release/)、[附件 API](../internal/api/release_assets.go) | [#32](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/32)；不代表签名/SBOM 已发布 |
| org | v5,v8 | Organizations/Member v5：/users/orgs、/orgs/{org}、members、repos；Actions v8：组织 runner-group 的列表/详情及 runners、runner-sets、shared-namespaces，只读发现 | partial | project | source,mock | [命令与测试](../pkg/cmd/org/)、[v8 API](../internal/api/actions/client.go) | [#75](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/75)、[#116](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/116)；不含 runner-group 写操作 |
| user | v5 | Users：/user、/users/{name}、/emails、namespaces、starred/subscriptions、events；显式字段更新个人资料 | implemented | project | source,mock | [命令与测试](../pkg/cmd/user/) | [#80](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/80)、[PR #275](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/275) 已合入；不是账号管理全覆盖 |
| search | v5 | Search：/search/repositories、users、issues | implemented | project | source,mock | [命令与测试](../pkg/cmd/search/) | [#31](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/31) |
| discussion | v5 | Discussion：/repos/{owner}/{repo}/discuss，列表/详情、评论与回复读取 | partial | project | source,mock | [命令与测试](../pkg/cmd/discussion/)、[API](../internal/api/discussion.go) | [#71](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/71)、[#72](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/72)；没有专用写命令 |
| ssh-key | v5 | Users SSH keys：认证用户 SSH 公钥添加、列举与删除 | implemented | project | source,mock | [命令与测试](../pkg/cmd/ssh-key/) | [#27](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/27)；不代表所有 SSH key 端点已覆盖 |
| kanban | v5 | Dashboard：/org/{org}/kanban/list、detail、item_list；仅组织看板只读发现 | partial | partner | source,mock | [命令与测试](../pkg/cmd/kanban/)、[API](../internal/api/kanban.go) | [#87](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/87) 已实现部分由本项目维护；完整模块/写操作由合作方负责 |

### API v8 与跨版本命令

| 命令族 | 协议 | API 家族与已实现范围 | 状态 | 责任 | 验证 | 代码/测试证据 | 关联工作与边界 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| pr | v5,v8 | Pull Requests v5：CRUD（编辑正文支持文件/stdin）、merge/review、diff/files/commits、评论/回复、协作元数据、关联 Issue、activity/history/reactions；checks 经 Actions v8，checkout 调用本地 Git | partial | project | source,mock | [命令与测试](../pkg/cmd/pr/)、[v5 API](../internal/api/pr_details.go)、[v8 API](../internal/api/actions/client.go) | [#93](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/93)、[#154](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/154)；activity、reactions 分页，history 本地截断；正式 review 仅批准 |
| run | v8 | Actions：runs、jobs、logs/download_log、artifacts 元数据/下载/删除 | implemented | project | source,mock | [命令与测试](../pkg/cmd/run/)、[API 与测试](../internal/api/actions/) | [#110](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/110)、[#112](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/112)、[#114](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/114) |
| workflow | v8 | Actions：workflows 列举、dispatches、validate（调用远端校验，不是纯本地 YAML 检查） | implemented | project | source,mock | [命令与测试](../pkg/cmd/workflow/)、[API](../internal/api/actions/client.go) | [#68](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/68)、[#88](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/88) |
| runner | v8 | Actions：仓库 actions/runners 与 shared-runners，只读列举 | partial | project | source,mock | [命令与测试](../pkg/cmd/runner/)、[API](../internal/api/actions/client.go) | [#115](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/115)、[PR #222](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/222)；不是组织 runner-group 管理 |

### OAuth、外部服务与本地行为

| 命令族 | 协议 | API 家族与已实现范围 | 状态 | 责任 | 验证 | 代码/测试证据 | 关联工作与边界 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| auth | oauth,v5,local | OAuth 登录/刷新、token 用户验证；账号/凭据存储、切换、status（离线/JSON/只读在线身份验证）/token、setup-git 与隐藏 git-credential | implemented | project | source,mock | [命令与测试](../pkg/cmd/auth/)、[OAuth](../internal/oauth/)、[配置](../internal/config/) | [#59](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/59)、[#97](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/97)、[PR #260](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/260) |
| browse | v5,local | 构造浏览器 URL；按需通过 v5 解析仓库默认分支或 Issue/PR 编号 | implemented | project | source,mock | [命令与测试](../pkg/cmd/browse/) | [#29](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/29)、[#108](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/108)；不能因打开 Actions 页面而计为 v8 调用 |
| license | external | openEuler compliance /check；不属于 AtomGit OpenAPI | implemented | project | source | [实现](../pkg/cmd/license/check.go)、[结构测试](../pkg/cmd/license/license_test.go) | 外部服务响应尚无本清单可引用的 HTTP mock/在线验证 |
| update | v5,external,local | AtomGit Release 版本检查；按安装来源处理 npm/Homebrew 更新与本地制品验证 | implemented | project | source,mock | [实现与注入测试](../pkg/cmd/update/) | [#53](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/53)、[PR #230](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/230)；不是通用包管理 API |
| check-update | v5,external,local | 已弃用的兼容入口：只检查更新，复用 update 的版本发现逻辑 | implemented | project | source,mock | [实现与测试](../pkg/cmd/update/) | [#70](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/70)；兼容命令不重复计算 API 覆盖 |
| alias | local | 本地 ag-cli JSON 配置中的内置 ag-cli 命令别名；不支持 shell 别名，不使用 Git config | implemented | project | source,local | [命令与测试](../pkg/cmd/alias/)、[存储](../internal/config/aliases.go)、[存储测试](../internal/config/aliases_test.go) | [PR #130](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/130)；无 AtomGit API |
| version | local | 显示 Version/Commit/BuildDate 与 Go 构建信息回退 | implemented | project | source,local | [命令与测试](../pkg/cmd/version/)、[元数据](../internal/version/) | [#61](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/61)；无 AtomGit API |
<!-- coverage:end -->

## 已知缺口、合作方模块与不支持项

此表是范围索引，不另造与现有 Issue 重复的任务。`docs` 只表示存在文档入口/需求依据；具体字段、分页、权限仍须实现前确认。没有命令的条目不纳入上面的注册命令表。

| API/模块 | 状态 | 责任 | 未交付范围/重新评估条件 | 验证与跟踪 |
| --- | --- | --- | --- | --- |
| v5 Commit comments | implemented | project | 专用 list/view/create/edit/delete 已合入；支持文件/stdin 正文和 JSON 输出 | source,mock；[命令与测试](../pkg/cmd/commit/comment/)、[#113](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/113)、[PR #276](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/276) |
| v5 Issue activity/history/reactions | implemented | project | branches 及三项只读查询已实现；activity/history 无文档化分页参数，reactions 支持分页；未做在线验收 | source,mock；[#86](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/86) |
| v5 PR activity/history/reactions | implemented | project | 三项只读查询已实现：activity、reactions 支持分页，history 无文档化分页参数按本地截断；reactions 字段按真实响应核对（字符串 id、emoji/emoji_name）；未做在线验收 | source,mock；[#93](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/93) |
| Enterprise | deferred | partner | 企业模块不作为本地 backlog；合作方与项目明确移交后再拆任务，具体版本/端点未逐一核实 | docs（仅官方目录）；[#124](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/124)，无在线证据 |
| Dashboard/Kanban 完整模块 | partial | partner | 本地已有 v5 只读 list/view/items；更广覆盖、写操作待合作方推进或明确移交 | source,mock（仅已有只读部分）；[#87](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/87)、[#124](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/124) |
| AI Hub | deferred | partner | 不把目录中的模块名称当作已支持端点；明确移交并验证契约后再立项 | docs（仅官方目录）；[#124](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/124)，具体契约未验证 |
| PR request-changes / 正式拒绝审查 | unverified | project | 当前正式 review 只支持 approve；未确认公开 API 契约前不模拟服务端审查状态 | source；[review 实现](../pkg/cmd/pr/review.go)，不猜测未文档化端点 |
| 通用 ag-cli api 的 v8 访问 | deferred | project | 当前固定 v5；如新增版本选择，须单独明确认证、路径和状态码策略 | source,mock；[API 命令](../pkg/cmd/api/api.go)，v8 现经专用 Actions 命令访问 |
| Discussion 专用写命令 | deferred | project | 当前范围仅 list/view；待单独确定写入流程和安全交互，不因通用 v5 请求可用而标为专用命令已实现 | source；[#72](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/72) |

### Issues #113–#122 对账

状态描述的是基线代码，不依赖 Issue 是否自动关闭；下列链接指向唯一任务，避免重复规划。

| Issue | 基线交付状态 | 代码/PR 证据及剩余边界 |
| --- | --- | --- |
| [#113](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/113) | implemented | [Commit 评论命令与测试](../pkg/cmd/commit/comment/)、[PR #276](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/276)；已实现 list/view/create/edit/delete |
| [#114](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/114) | implemented | [artifact 命令](../pkg/cmd/run/artifact.go)，[PR #219](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/219)；v8 删除/确认 |
| [#115](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/115) | implemented | [runner 测试](../pkg/cmd/runner/runner_test.go)，[PR #222](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/222)；只读仓库/共享 runner |
| [#116](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/116) | implemented | [组织 runner-group 命令与测试](../pkg/cmd/org/runner_group.go)、[v8 客户端](../internal/api/actions/client.go)；只读 list/detail/runners/runner-sets/shared-namespaces，不被 #115 覆盖 |
| [#117](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/117) | implemented | [fork API 与分页](../internal/api/forks.go)，[PR #221](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/221) |
| [#118](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/118) | implemented | [mirror API](../internal/api/remote_mirror.go)，[PR #231](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/231)；只读，不含写管理 |
| [#119](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/119) | implemented | [transfer API](../internal/api/repository_transfer.go)，[PR #229](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/229) |
| [#120](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/120) | implemented | [共享错误处理](../internal/api/error_response.go)，[PR #232](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/232)；v5/v8 已共用脱敏，不标作待实现 |
| [#121](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/121) | implemented | [Factory context](../pkg/cmdutil/factory.go)，[PR #233](https://atomgit.com/hust-open-atom-club/atomgit-cli/merge_requests/233)；取消与元数据超时 |
| [#122](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/122) | partial | 已有部分 domain helpers 与 [Factory](../pkg/cmdutil/factory.go)，统一认证/路径边界、参考命令迁移和规范仍待完成 |

## 如何更新与验证

1. 新增/删除/重命名根命令或新增子命令、API 家族、字段行为时，同一 PR 更新本清单的范围、状态、证据与任务链接。根命令必须一行且唯一；子命令变更须人工检查对应家族范围，并同步[命令参考](command-reference.md)。
2. OpenAPI 文档变化或契约测试失败时，先确定受影响家族，再修正状态与证据；仅修改 fixture 不能证明接口已兼容。契约样例和在线 smoke 的维护遵循[契约文档](api-contracts.md)。
3. 责任转移必须链接明确的 Issue/Discussion 决定，写清模块/只读或写入边界与接收方；不得仅因已有少数命令就将合作方整个模块改为 project。
4. 合入关联 PR 后更新 missing/partial；关闭或放弃 PR 不等于已实现。保留历史任务链接，不复制新 Issue。
5. 新增 `live` 证据需注明验证日期、commit、端点/操作范围及脱敏记录链接；其他场景仍未验证。没有证据的能力使用 unverified/deferred，不据接口名称猜测。
6. 运行 `go test ./pkg/cmd/root -run TestOpenAPICoverage -count=1` 检查清单，再运行 `go test ./...` 和 `make docs-reference-check`。该检查自动进入已有 Go 测试/CI，不新建 workflow。外部网页及邮件服务可达性不由离线测试保证。

仓库文档、代码、测试之间一律使用相对链接，保证 AtomGit 与 GitHub mirror 的文件导航一致。GitHub mirror 不承载本项目的 Issue/PR 协作；活跃的 Issue/PR 记录仅在 AtomGit 上维护，因此相关链接使用完整 AtomGit URL。此文件是人工清单，不运行生成器覆盖它。
