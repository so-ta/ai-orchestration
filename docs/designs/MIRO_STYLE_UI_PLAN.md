# Miroスタイル ワークフローエディタ UI設計

## コンセプト

**「全てフローティング」** - キャンバスが100%、UIは全てキャンバス上にフローティング

---

## Miro UIの特徴（実際の確認結果）

1. **フローティングヘッダー** - 上部に半透明で浮いている
2. **縦型フローティングツールバー** - 左端にアイコンのみ
3. **フローティングズームコントロール** - 右下
4. **キャンバス100%** - 背景全体がキャンバス
5. **選択時のコンテキストメニュー** - オブジェクト近くにポップアップ

---

## 新しいレイアウト設計

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ [⚡] Project ▼ [v2]              [💾▼] [▶実行] [履歴][変数] [⚙️][👤] │    │ ← フローティングヘッダー
│  └─────────────────────────────────────────────────────────────────────┘    │   (背景: rgba白90%)
│                                                                              │
│  ┌────┐                                                                      │
│  │ 🔍 │                                                                      │
│  ├────┤                                                                      │
│  │ 🤖 │                                                                      │
│  │ ⚡ │                         ┌───────────────────┐                        │
│  │ 📱 │                         │ ノード            │                        │
│  │ 🔧 │                         └───────────────────┘                        │
│  ├────┤                              ↓                                       │
│  │ ⫲ │                         ┌───────────────────┐     ┌─────────────┐    │
│  │ ∀ │      キャンバス          │ ノード            │────→│ ノード      │    │
│  ├────┤      (100%)            └───────────────────┘     └─────────────┘    │
│  │ + │                                                                      │
│  └────┘                                                                      │
│    ↑                                                                         │
│  フローティング                                                               │
│  ツールバー                                                                   │
│  (56px)                                                   ┌────────────────┐ │
│                                                           │ MiniMap        │ │
│                                                           └────────────────┘ │
│                                                           [100%] [🔍][+][-]  │ ← フローティングズーム
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
                              ↑
                     キャンバス背景（グリッドパターン）
```

### 選択時のコンテキストパネル

```
                         ┌───────────────────┐
                         │ ● LLM Process     │
                         └───────────────────┘
                              ↓ 選択
        ┌───────────────────────────────────────┐
        │ ● LLM Process                      ✕ │  ← フローティングパネル
        ├───────────────────────────────────────┤     (ノードの右に表示)
        │ モデル:     [gpt-4           ▼]      │
        │ プロバイダー: [OpenAI          ▼]      │
        │                                       │
        │ プロンプト:                            │
        │ ┌────────────────────────────────┐   │
        │ │ Summarize the following...     │   │
        │ └────────────────────────────────┘   │
        │                                       │
        │ ▶ 出力スキーマ                        │
        │ ▶ 詳細設定                            │
        ├───────────────────────────────────────┤
        │ [🗑️ 削除]               [💾 保存]    │
        └───────────────────────────────────────┘
```

---

## コンポーネント設計

### 1. FloatingHeader.vue（新規）

```vue
<template>
  <header class="floating-header">
    <div class="header-left">
      <div class="logo">⚡</div>
      <EditorProjectSelector :project="project" />
      <StatusBadge :status="project?.status" :version="project?.version" />
    </div>

    <div class="header-right">
      <SaveDropdown @save="$emit('save')" @saveDraft="$emit('saveDraft')" />
      <button class="btn-run" @click="$emit('run')">
        <PlayIcon /> 実行
      </button>
      <button class="btn-icon" @click="$emit('openRuns')" title="実行履歴">
        <HistoryIcon />
      </button>
      <button class="btn-icon" @click="$emit('openVariables')" title="変数">
        <VariableIcon />
      </button>
      <button class="btn-icon" @click="$emit('openSettings')" title="設定">
        <SettingsIcon />
      </button>
      <UserMenu />
    </div>
  </header>
</template>

<style scoped>
.floating-header {
  position: fixed;
  top: 12px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 100;

  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;

  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);

  min-width: 600px;
  max-width: 900px;
}

.btn-run {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #10b981;
  color: white;
  border: none;
  border-radius: 8px;
  font-weight: 500;
  cursor: pointer;
}

.btn-run:hover {
  background: #059669;
}

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  color: #6b7280;
}

.btn-icon:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #111827;
}
</style>
```

### 2. FloatingToolbar.vue（新規）

```vue
<template>
  <div class="floating-toolbar">
    <!-- 検索 -->
    <button class="toolbar-btn" @click="$emit('openSearch')" title="検索 (⌘K)">
      <SearchIcon />
    </button>

    <div class="toolbar-divider" />

    <!-- ブロックカテゴリ -->
    <div
      v-for="category in categories"
      :key="category.id"
      class="toolbar-btn-wrapper"
      @mouseenter="hoveredCategory = category.id"
      @mouseleave="hoveredCategory = null"
    >
      <button class="toolbar-btn" :title="category.name">
        <component :is="category.icon" />
      </button>

      <!-- ホバー時のサブメニュー -->
      <Transition name="slide">
        <div v-if="hoveredCategory === category.id" class="toolbar-submenu">
          <div
            v-for="block in category.blocks"
            :key="block.slug"
            class="submenu-item"
            draggable="true"
            @dragstart="handleDragStart($event, block)"
          >
            <span class="block-indicator" :style="{ background: block.color }" />
            <span class="block-name">{{ block.name }}</span>
          </div>
        </div>
      </Transition>
    </div>

    <div class="toolbar-divider" />

    <!-- コントロールフローグループ -->
    <button class="toolbar-btn" title="Parallel" draggable="true" @dragstart="handleGroupDrag($event, 'parallel')">
      ⫲
    </button>
    <button class="toolbar-btn" title="ForEach" draggable="true" @dragstart="handleGroupDrag($event, 'foreach')">
      ∀
    </button>

    <div class="toolbar-divider" />

    <!-- ブロック追加 -->
    <button class="toolbar-btn" @click="$emit('openBlockPicker')" title="ブロックを追加">
      <PlusIcon />
    </button>
  </div>
</template>

<style scoped>
.floating-toolbar {
  position: fixed;
  top: 80px;
  left: 16px;
  z-index: 100;

  display: flex;
  flex-direction: column;
  gap: 4px;

  padding: 8px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.toolbar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  color: #6b7280;
  font-size: 18px;
}

.toolbar-btn:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #111827;
}

.toolbar-divider {
  height: 1px;
  margin: 4px 0;
  background: rgba(0, 0, 0, 0.08);
}

.toolbar-btn-wrapper {
  position: relative;
}

.toolbar-submenu {
  position: absolute;
  left: 100%;
  top: 0;
  margin-left: 8px;

  min-width: 200px;
  padding: 8px;
  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}

.submenu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: grab;
}

.submenu-item:hover {
  background: rgba(0, 0, 0, 0.04);
}

.block-indicator {
  width: 4px;
  height: 20px;
  border-radius: 2px;
}

.block-name {
  font-size: 13px;
  color: #374151;
}

.slide-enter-active,
.slide-leave-active {
  transition: all 0.15s ease;
}

.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  transform: translateX(-8px);
}
</style>
```

### 3. FloatingZoomControl.vue（新規）

```vue
<template>
  <div class="floating-zoom">
    <!-- ミニマップ（折りたたみ可能） -->
    <div v-if="showMinimap" class="minimap-container">
      <MiniMap />
    </div>

    <div class="zoom-controls">
      <button class="zoom-btn" @click="$emit('toggleMinimap')" :class="{ active: showMinimap }">
        <MapIcon />
      </button>
      <div class="zoom-divider" />
      <button class="zoom-btn" @click="$emit('zoomOut')">
        <MinusIcon />
      </button>
      <span class="zoom-value">{{ Math.round(zoom * 100) }}%</span>
      <button class="zoom-btn" @click="$emit('zoomIn')">
        <PlusIcon />
      </button>
      <button class="zoom-btn" @click="$emit('fitView')" title="全体表示">
        <FitIcon />
      </button>
    </div>
  </div>
</template>

<style scoped>
.floating-zoom {
  position: fixed;
  bottom: 16px;
  right: 16px;
  z-index: 100;

  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
}

.minimap-container {
  width: 180px;
  height: 120px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
  overflow: hidden;
}

.zoom-controls {
  display: flex;
  align-items: center;
  gap: 4px;

  padding: 6px 8px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.zoom-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  color: #6b7280;
}

.zoom-btn:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #111827;
}

.zoom-btn.active {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.zoom-value {
  min-width: 48px;
  text-align: center;
  font-size: 12px;
  font-weight: 500;
  color: #374151;
}

.zoom-divider {
  width: 1px;
  height: 20px;
  background: rgba(0, 0, 0, 0.08);
}
</style>
```

### 4. ContextPropertyPanel.vue（新規）

```vue
<template>
  <Teleport to="body">
    <Transition name="panel">
      <div
        v-if="step"
        class="context-panel"
        :style="panelStyle"
        ref="panelRef"
      >
        <!-- ヘッダー -->
        <div class="panel-header">
          <div class="step-indicator" :style="{ background: stepColor }" />
          <input
            v-model="formName"
            class="step-name-input"
            @blur="handleNameChange"
          />
          <button class="close-btn" @click="$emit('close')">
            <XIcon />
          </button>
        </div>

        <!-- コンテンツ -->
        <div class="panel-content">
          <DynamicConfigForm
            v-if="configSchema"
            v-model="formConfig"
            :schema="configSchema"
            :ui-config="uiConfig"
          />

          <!-- 詳細設定（折りたたみ） -->
          <details class="advanced-section">
            <summary>詳細設定</summary>
            <div class="advanced-content">
              <!-- フロー設定、エラーハンドリングなど -->
              <FlowTab :step="step" @update="handleFlowUpdate" />
            </div>
          </details>
        </div>

        <!-- フッター -->
        <div class="panel-footer">
          <button class="btn-delete" @click="$emit('delete')">
            <TrashIcon /> 削除
          </button>
          <button class="btn-save" @click="handleSave" :disabled="saving">
            <SaveIcon /> 保存
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.context-panel {
  position: fixed;
  z-index: 150;

  width: 340px;
  max-height: 70vh;

  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);

  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.step-indicator {
  width: 4px;
  height: 24px;
  border-radius: 2px;
  flex-shrink: 0;
}

.step-name-input {
  flex: 1;
  border: none;
  background: transparent;
  font-size: 15px;
  font-weight: 600;
  color: #111827;
  outline: none;
}

.step-name-input:focus {
  background: rgba(0, 0, 0, 0.03);
  border-radius: 4px;
  padding: 4px 8px;
  margin: -4px -8px;
}

.close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  color: #9ca3af;
}

.close-btn:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #6b7280;
}

.panel-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.advanced-section {
  margin-top: 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  padding-top: 12px;
}

.advanced-section summary {
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  color: #6b7280;
  user-select: none;
}

.advanced-section summary:hover {
  color: #374151;
}

.advanced-content {
  margin-top: 12px;
}

.panel-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
}

.btn-delete {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: transparent;
  border: 1px solid #fecaca;
  border-radius: 8px;
  color: #dc2626;
  cursor: pointer;
  font-size: 13px;
}

.btn-delete:hover {
  background: #fef2f2;
}

.btn-save {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #3b82f6;
  border: none;
  border-radius: 8px;
  color: white;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
}

.btn-save:hover {
  background: #2563eb;
}

.btn-save:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Transitions */
.panel-enter-active,
.panel-leave-active {
  transition: all 0.2s ease;
}

.panel-enter-from,
.panel-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-10px);
}
</style>
```

### 5. QuickSearchModal.vue（新規）

```vue
<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="quick-search-overlay" @click.self="$emit('close')">
        <div class="quick-search-modal">
          <!-- 検索入力 -->
          <div class="search-header">
            <SearchIcon class="search-icon" />
            <input
              ref="inputRef"
              v-model="query"
              type="text"
              class="search-input"
              placeholder="ブロックを検索..."
              @keydown="handleKeydown"
            />
            <kbd class="search-shortcut">ESC</kbd>
          </div>

          <!-- 結果 -->
          <div class="search-results">
            <!-- 最近使用 -->
            <div v-if="!query && recentBlocks.length" class="result-section">
              <div class="section-title">最近使用</div>
              <div
                v-for="(block, index) in recentBlocks"
                :key="block.slug"
                :class="['result-item', { selected: selectedIndex === index }]"
                @click="selectBlock(block)"
                @mouseenter="selectedIndex = index"
              >
                <span class="block-indicator" :style="{ background: block.color }" />
                <span class="block-name">{{ block.name }}</span>
                <span class="block-category">{{ block.category }}</span>
              </div>
            </div>

            <!-- 検索結果 -->
            <div v-if="filteredBlocks.length" class="result-section">
              <div v-if="query" class="section-title">検索結果</div>
              <div
                v-for="(block, index) in filteredBlocks"
                :key="block.slug"
                :class="['result-item', { selected: selectedIndex === index + recentBlocks.length }]"
                @click="selectBlock(block)"
                @mouseenter="selectedIndex = index + recentBlocks.length"
              >
                <span class="block-indicator" :style="{ background: block.color }" />
                <span class="block-name">{{ block.name }}</span>
                <span class="block-category">{{ block.category }}</span>
              </div>
            </div>

            <!-- 空状態 -->
            <div v-if="query && !filteredBlocks.length" class="empty-state">
              「{{ query }}」に一致するブロックがありません
            </div>
          </div>

          <!-- フッター -->
          <div class="search-footer">
            <span><kbd>↑↓</kbd> 選択</span>
            <span><kbd>Enter</kbd> 追加</span>
            <span><kbd>ESC</kbd> 閉じる</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.quick-search-overlay {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 15vh;
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(2px);
}

.quick-search-modal {
  width: 100%;
  max-width: 520px;
  background: white;
  border-radius: 16px;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.2);
  overflow: hidden;
}

.search-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid #e5e7eb;
}

.search-icon {
  width: 20px;
  height: 20px;
  color: #9ca3af;
}

.search-input {
  flex: 1;
  border: none;
  font-size: 16px;
  color: #111827;
  outline: none;
}

.search-input::placeholder {
  color: #9ca3af;
}

.search-shortcut {
  padding: 4px 8px;
  background: #f3f4f6;
  border-radius: 4px;
  font-size: 11px;
  color: #6b7280;
}

.search-results {
  max-height: 400px;
  overflow-y: auto;
}

.result-section {
  padding: 8px 0;
}

.section-title {
  padding: 8px 20px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #9ca3af;
}

.result-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 20px;
  cursor: pointer;
}

.result-item:hover,
.result-item.selected {
  background: #f9fafb;
}

.block-indicator {
  width: 4px;
  height: 24px;
  border-radius: 2px;
}

.block-name {
  flex: 1;
  font-size: 14px;
  color: #111827;
}

.block-category {
  font-size: 12px;
  color: #9ca3af;
}

.empty-state {
  padding: 32px 20px;
  text-align: center;
  color: #9ca3af;
  font-size: 14px;
}

.search-footer {
  display: flex;
  gap: 16px;
  padding: 12px 20px;
  background: #f9fafb;
  border-top: 1px solid #e5e7eb;
  font-size: 12px;
  color: #6b7280;
}

.search-footer kbd {
  padding: 2px 6px;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  font-size: 11px;
}

/* Transition */
.modal-enter-active,
.modal-leave-active {
  transition: all 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .quick-search-modal,
.modal-leave-to .quick-search-modal {
  transform: scale(0.95) translateY(-20px);
}
</style>
```

---

## ページ構成（pages/index.vue）

```vue
<template>
  <div class="miro-editor">
    <!-- キャンバス（全画面・背景） -->
    <div class="canvas-wrapper">
      <DagEditor
        ref="dagEditor"
        v-model:steps="steps"
        v-model:edges="edges"
        :selected-step-id="selectedStepId"
        @select-step="handleSelectStep"
        @viewport-change="handleViewportChange"
      />
    </div>

    <!-- フローティングヘッダー -->
    <FloatingHeader
      :project="project"
      :saving="saving"
      @save="handleSave"
      @save-draft="handleSaveDraft"
      @run="handleRun"
      @open-runs="showRuns = true"
      @open-variables="showVariables = true"
      @open-settings="showSettings = true"
    />

    <!-- フローティングツールバー -->
    <FloatingToolbar
      :blocks="blocks"
      @open-search="showQuickSearch = true"
      @drag-start="handleDragStart"
      @drag-end="handleDragEnd"
    />

    <!-- コンテキストプロパティパネル -->
    <ContextPropertyPanel
      v-if="selectedStep"
      :step="selectedStep"
      :position="contextPanelPosition"
      :block-definition="selectedBlockDef"
      @save="handleSaveStep"
      @delete="handleDeleteStep"
      @close="selectedStepId = null"
    />

    <!-- フローティングズームコントロール -->
    <FloatingZoomControl
      :zoom="zoom"
      :show-minimap="showMinimap"
      @zoom-in="handleZoomIn"
      @zoom-out="handleZoomOut"
      @fit-view="handleFitView"
      @toggle-minimap="showMinimap = !showMinimap"
    />

    <!-- クイック検索（Cmd+K） -->
    <QuickSearchModal
      v-model:open="showQuickSearch"
      :blocks="blocks"
      :recent-blocks="recentBlocks"
      @select="handleQuickAddBlock"
    />

    <!-- スライドアウトパネル -->
    <SlideOutPanel v-model="showRuns" title="実行履歴">
      <RunHistoryPanel :project-id="project?.id" />
    </SlideOutPanel>
    <SlideOutPanel v-model="showVariables" title="変数">
      <VariablesPanel :project-id="project?.id" />
    </SlideOutPanel>
    <SlideOutPanel v-model="showSettings" title="設定">
      <SettingsPanel :project="project" />
    </SlideOutPanel>
  </div>
</template>

<style scoped>
.miro-editor {
  position: relative;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: #f5f5f5;
}

.canvas-wrapper {
  position: absolute;
  inset: 0;
}

/* VueFlowのグリッド背景 */
:deep(.vue-flow) {
  background-color: #fafafa;
  background-image:
    radial-gradient(circle, #ddd 1px, transparent 1px);
  background-size: 20px 20px;
}
</style>
```

---

## 実装ステップ

### Phase 1: フローティングUI基盤
1. `FloatingHeader.vue` 作成
2. `FloatingToolbar.vue` 作成
3. `FloatingZoomControl.vue` 作成
4. `pages/index.vue` 新レイアウト

### Phase 2: コンテキストパネル
5. `ContextPropertyPanel.vue` 作成
6. 位置計算ロジック（ノードの右に配置）
7. PropertiesPanel / StepPalette 削除

### Phase 3: クイック検索
8. `QuickSearchModal.vue` 作成
9. キーボードショートカット（⌘K）

### Phase 4: 仕上げ
10. アニメーション調整
11. レスポンシブ対応
12. i18n対応

---

## 比較表

| 要素 | Before | After (Miro Style) |
|------|--------|-------------------|
| ヘッダー | 固定（画面上端） | **フローティング（中央上部）** |
| 左パネル | 固定（280px） | **フローティングツールバー（56px）** |
| 右パネル | 固定（360px） | **コンテキストパネル（選択時のみ）** |
| ズーム | ミニマップ内 | **フローティング（右下）** |
| キャンバス | 約60% | **100%** |
| 背景 | 白 | **グリッドパターン** |
