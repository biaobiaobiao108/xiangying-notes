# DESIGN.md — 象映笔记 (Xiangying Notes)

> 本文档为象映笔记桌面客户端 UI/UX 规范与视觉基准。
> 所有界面实现严格遵循本文档中定义的色彩、字阶、留白和组件规则，坚决杜绝 AI 生成式模板风格（AI slop）。

更新日期: 2026-11-09

## 1. 视觉方向与艺术指导 (Direction: 砚宣沉香)

- **概念定位 (Concept):** 砚宣沉香 (The Ink & Xuan Paper Craft)。温润、考究、沉静的纯原生桌面 Markdown 笔记，像一件经年相伴的书写文具。
- **三大目标特质 (Target Qualities):**
  - **文具质感 (Crafted Stationery):** 质朴温润，界面如展开的熟宣与沉香砚台，色彩沉着内敛。
  - **视平线统一 (Unified Horizon):** 顶栏全局贯通 44px 高度与 1px 微痕底线，左中右三栏浑然一体。
  - **自然文本流 (Un-carded Content Flow):** 严格依靠严谨的字阶、灰阶和间距呼吸组织层级，拒绝无意义卡片堆叠与视觉噪音。
- **三大坚决拒绝 (Anti-Qualities):**
  - **拒绝卡片套叠 (No Cardocalypse):** 列表中不使用悬浮白色卡片叠加外边框与弥散阴影；弹窗中严禁卡片套卡片。
  - **拒绝浮夸动效 (No Frivolous Motion):** 卡片 hover 严禁 3px 上浮放大；状态切换采用自然的物理微反馈。
  - **拒绝视线遮挡 (No Floating Text Obscurity):** 正文编辑区严禁悬浮遮挡文本的悬浮胶囊；辅助信息平铺于底部专用状态栏。

## 2. 标准 7 级字阶体系 (Strict 7-level Typography)

| 角色 (Role) | 字号 (px) | 行高 (Line Height) | 字重 (Weight) | 适用场景 |
| :--- | :--- | :--- | :--- | :--- |
| **Display** | 20 | 1.4 | 700 Bold | 空状态醒目引导、卡片视图大标题 |
| **Title** | 16 | 1.4 | 700 Bold | 模态弹窗主标题、卡片笔记标题 |
| **Subtitle** | 14 | 1.4 | 600 SemiBold | 笔记列表栏顶栏标题（“全部笔记”）、大纲面板标题 |
| **Body-Primary** | 13 | 1.5 | 500 Medium / 600 | 笔记列表主标题、侧栏主导航项、弹窗操作按钮、表单输入 |
| **Body-Secondary**| 12 | 1.5 | 400 Regular | 侧栏二级条目、笔记摘要正文、引用上下文、导出项说明 |
| **Meta** | 11 | 1.3 | 400 Regular | 标签文本、弹窗次级提示、搜索输入辅助 |
| **Caption** | 10 | 1.2 | 400 Regular | 列表时间戳、快捷键徽标 (Ctrl N)、字数与统计数值 |

### 字体栈规范

- **UI 界面字体:** `Microsoft YaHei UI, Segoe UI`
- **标题与正文字体:** `Noto Serif SC, LXGW WenKai, Microsoft YaHei UI`
- **代码与快捷键:** `Maple Mono NF CN, Cascadia Code, Consolas`

## 3. 颜色语义规范 (Color System)

### 浅色模式 (Light - 宣纸墨韵)

| 角色 | 颜色值 | 说明 |
| :--- | :--- | :--- |
| **Canvas** | `#F3EFE6` | 全局背景衬底、表单输入框底色 |
| **Sidebar** | `#F7F4EC` | 侧边栏底色、沉香笔搁收纳 |
| **List** | `#FAF8F3` | 笔记列表栏底色、目录书目 |
| **Editor** | `#FDFCF9` | 熟宣书写区表面底色 |
| **Surface** | `#FDFCF9` | 弹窗表面、笔记卡片表面 |
| **SurfaceHover**| `#EFECE2` | 条目悬浮态微光反馈 |
| **SurfaceSelect**| `#E6E1D4` | 沉静温润的平整选中底色 |
| **Ink** | `#1C221E` | 松烟浓墨：主标题、核心文字 |
| **InkSoft** | `#2F3832` | 笔墨正文：次级标题、正文文本 |
| **QuietInk** | `#535F56` | 淡墨摘要：笔记预览摘录、次要正文 |
| **Muted** | `#7A877D` | 余墨说明：标签、图标、辅助说明 |
| **SoftMuted** | `#A2ADA4` | 轻痕文字：时间戳、快捷键、未激活图标 |
| **Border** | `rgba(30, 38, 32, 0.10)` | 弹窗轮廓线、输入框外框 |
| **BorderSoft** | `rgba(30, 38, 32, 0.05)` | 顶栏贯穿分割线、行列表微痕线 |
| **Accent** | `#2B5340` | 冷松墨绿：主行动按钮、激活图标、选中指示 |
| **AccentDark** | `#1F4030` | 墨绿深色：主按钮悬浮态 |
| **AccentSoft** | `#E7F0E9` | 柔墨绿浅底：激活胶囊、高亮标记背景 |
| **Star** | `#B8842E` | 泥金：收藏星标 |
| **Danger** | `#B53D2A` | 朱砂：删除、清空警示操作 |

### 深色模式 (Dark - 墨砚沉静)

| 角色 | 颜色值 | 说明 |
| :--- | :--- | :--- |
| **Canvas** | `#121316` | 深色总衬底 |
| **Sidebar** | `#16171B` | 深色侧栏 |
| **List** | `#191A1F` | 深色列表栏 |
| **Editor** | `#1D1E24` | 深色编辑器书写表面 |
| **Surface** | `#22242D` | 弹窗与卡片表面 |
| **SurfaceHover**| `#2A2C37` | 悬浮微光 |
| **SurfaceSelect**| `#313441` | 选中底色 |
| **Ink** | `#E8E9E4` | 象牙白墨：主标题 |
| **InkSoft** | `#D4D7D0` | 次级浅白：正文 |
| **QuietInk** | `#B6BBB3` | 浅灰文本：摘要 |
| **Muted** | `#9AA097` | 次级浅灰：标签说明 |
| **SoftMuted** | `#6C726B` | 极浅微灰：时间戳、元数据 |
| **Border** | `rgba(255, 255, 255, 0.08)` | 容器边框 |
| **BorderSoft** | `rgba(255, 255, 255, 0.04)` | 分割线 |
| **Accent** | `#6CAE8E` | 墨玉青绿：深色高亮与主操作 |
| **AccentDark** | `#8AC4A8` | 悬浮浅绿 |
| **AccentSoft** | `rgba(108, 174, 142, 0.14)` | 柔绿半透明衬底 |
| **Star** | `#DAB062` | 暗金收藏星标 |
| **Danger** | `#DE705C` | 柔红警示色 |

## 4. 几何布局与空间规范 (Geometry & Layout)

- **顶栏视平线 (Top Bar Horizon):** 全局高度固定为 **44px**，三栏（侧栏、列表栏、主工作区）底部贯穿一条统一的 1px `BorderSoft` 分割线。
- **列表项 (Note List Item):**
  - 不使用外边框与卡片投影；
  - 选中态采用纯平纸韵底色 (`SurfaceSelect`)，内嵌左侧 2px 墨绿柔和指示条 (`Accent`)；
  - 内边距规范为垂向 8px、水平 10px，圆角固定为 6px。
- **卡片瀑布流 (Card Grid):**
  - 三列响应式 Masonry 布局；
  - 卡片内部移除生硬贯穿横线，依靠间距与柔和字阶区分标题、摘要与底部元数据；
  - 悬浮态使用平滑微色差过渡 (`SurfaceHover` + 柔和墨绿边框微显)，严禁大幅上浮或尺寸跳动。
- **底部状态栏 (Bottom Status Bar):**
  - 编辑器底部独立 26px 高度状态条，左侧/右侧平铺展示字数统计与大纲切换入口；
  - 彻底杜绝悬浮胶囊覆盖正文的重叠排版缺陷。

## 5. 交互与动效规范 (Motion & Touch)

- **弹窗动效:** `180ms EaseOut` 缩放淡入 (`scale: 0.975 -> 1.0`, `opacity: 0 -> 1`)。
- **状态颜色过渡:** `150ms EaseOut` 属性过渡。
- **按压与回弹:** 点击按压微缩 `scale: 0.985`，释放阻尼回弹 (`ui.Snappy`)。
- **平滑滚动:** 280ms 临界阻尼动画 (`listWheelAnimation`)。
