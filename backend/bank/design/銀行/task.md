# 実装タスク一覧（銀行・普通口座システム）

## 1. 【要件】DB基盤の構築
**目的**: 負債と履歴を管理できる頑強な土台を作る。
- [x] **DBマイグレーション**: `Accounts`, `Transactions`, `MarketBatches` のスキーマ定義。
    - `balance` はマイナス値を扱えるように符号付き整数（BigInt）で定義。
    

## 2. 【要件】口座の新規開設（0円スタート）
**目的**: ユーザー登録に連動して、空の口座を用意する。
 - [ ] **内部API実装**: `POST /internal/bank-accounts/create`
- [ ] **ロジック**: 残高0、ローン0、スコア3で`Accounts`レコードを作成。履歴は作成しない。
- [ ] **受け入れ基準**: 指定のUUIDで、残高0の口座がDBに存在すること。

## 3. 【要件】初期ローンの実行（100万円の着金演出）
**目的**: 0円の口座に原資を振り込み、通帳の1行目を作る。
 - [ ] **内部API実装**: `POST /internal/bank-accounts/initialize`
- [ ] **ロジック実装**:
        - 同一トランザクション内で `balance += 1,000,000`, `loan_principal += 1,000,000` を実行。
        - `apply_transaction` を通じ、`type: LOAN`, `is_printed: false` で履歴保存。
- [ ] **受け入れ基準**: 実行後、`status`APIで残高100万、`history`APIで未印字の履歴が1件返ること。
 

## 4. 【要件】資産ステータスの可視化と演出同期
**目的**: 「手元にお金はあるが純資産は0」という現実を表示し、印字演出を制御する。
 - [ ] **ステータスAPI**: `GET /api/bank/account/:id/status`
     - [x] **ステータスAPI**: `GET /api/bank/account/:id/status`
          - `net_asset` (-115万等の負値) と `is_debt` を計算して返却。
 - [ ] **履歴・演出API**:
     - `GET /api/v1/bank/account/history`（全履歴取得）
     - `PATCH /api/v1/bank/account/history/print`（演出完了フラグ更新）
 

## 5. 【要件】最終精算と信用スコアの確定
**目的**: 全投資終了後、純資産の成否によってスコアを変動させ、1なら凍結する。
 - [ ] **精算API実装**: `POST /internal/bank-accounts/settle`
- [ ] **ロジック**: 
    - `net_asset` が正ならスコア加算、負ならスコア減算（-2）。
    - スコア1で `is_frozen = true` へ更新。
- [ ] **受け入れ基準**: 債務超過時にスコアが下がり、口座が操作不能（凍結）になること。

---

## 9. 実装フェーズへの移行用チェックリスト（AIプロンプト用）
- [ ] 初期状態（口座作成直後）の `balance` は 0 になっているか？
- [ ] `initialize` (100万着金) 後に、最初の履歴が `is_printed = false` で作成されているか？
- [x] `net_asset` の計算において、`loan_principal`（借りた額）を正しく差し引いているか？
- [ ] すべての資金移動は、DBトランザクションで保護され、ロールバック可能か？