# 銀行・普通口座システム 実装詳細仕様書（最終確定版）

## 1. サービス設計の前提
- **責務の分離**: ユーザー基本情報は「Userサービス」が管理し、本サービスは「預金・負債の事実（ジャーナル）」と「残高キャッシュ」のみを管理する。
- **不変性（Immutability）**: 履歴は一度書き込んだら更新・削除しない（Insert Only）。
- **高可用性**: 分散環境でのロック競合を避けるため、行更新（Update）を最小化し、冪等性（Idempotency）を保証する。

---

## 2. データベース構造（Database Schema）

### 2.1 Accounts（口座マスタ：キャッシュ・参照用）
参照パフォーマンス向上のためのマテリアライズドビュー。

| カラム名 | 型 | 説明 |
| :--- | :--- | :--- |
| `user_id` | UUID (PK) | ユーザーID |
| `deposit_balance` | BigInt | 現在の預金残高キャッシュ |
| `loan_balance` | BigInt | 現在の負債総額キャッシュ |
| `credit_score` | Integer | 信用スコア(1-10) |
| `is_frozen` | Boolean | 不整合または事故時の凍結フラグ |
| `updated_at` | Timestamp | キャッシュ最終更新日時 |

### 2.2 Transaction_Log（取引管理：分散トランザクション用）
預金とローンの両方が動く取引の整合性を管理する。

| カラム名 | 型 | 説明 |
| :--- | :--- | :--- |
| `tx_id` | UUID (PK) | クライアント生成の一意な取引ID（冪等性キー） |
| `tx_type` | Enum | LOAN_INIT, REPAYMENT, INTEREST, BUY, SELL, SETTLE |
| `status` | Enum | PENDING, SUCCESS, FAILED |
| `created_at` | Timestamp | 取引開始日時 |

### 2.3 Deposit_Journal（預金元帳：不変履歴）
**Update/Delete禁止。**

| カラム名 | 型 | 説明 |
| :--- | :--- | :--- |
| `id` | BigInt (PK) | 物理連番ID |
| `tx_id` | UUID (UQ) | Transaction_Log.tx_id（重複挿入防止） |
| `user_id` | UUID (Index) | ユーザーID |
| `amount` | BigInt | 増減額（正：入金 / 負：出金） |
| `balance_snap` | BigInt | 処理後の預金残高（通帳表示用） |
| `created_at` | Timestamp | 記録日時 |

### 2.4 Loan_Journal（ローン元帳：不変履歴）
**Update/Delete禁止。**

| カラム名 | 型 | 説明 |
| :--- | :--- | :--- |
| `id` | BigInt (PK) | 物理連番ID |
| `tx_id` | UUID (UQ) | Transaction_Log.tx_id |
| `loan_id` | BigInt (Index) | 紐付くローン契約ID |
| `amount` | BigInt | 負債増減（正：利息等による増 / 負：返済による減） |
| `loan_snap` | BigInt | 処理後の負債残高（負債推移用） |
| `created_at` | Timestamp | 記録日時 |

---

## 3. ロジック・整合性維持ルール

### 3.1 冪等性（Idempotency）の強制
- APIリクエスト時に必ず `tx_id` を受け取る。
- 各Journalテーブルの `tx_id` ユニーク制約により、分散環境でのリトライによる二重記帳をDBレベルで防ぐ。

### 3.2 銀行グレードの検算（Reconciliation）
- **検算ロジック**: `Accounts.deposit_balance == SUM(Deposit_Journal.amount)` を定期実行。
- **不整合時**: 自動的に `is_frozen = true` をセットし、管理者にアラート。

### 3.3 資産の定義
- **純資産（Net Asset）**: `deposit_balance - loan_balance`
- **債務超過判定**: `Net Asset < 0`（ゲーム上のリスク状態）

---

## 4. エンドポイント定義（API Endpoints）

### 4.1 プレイヤー用 API（フロント・UI向け）
| メソッド | エンドポイント | 説明 |
| :--- | :--- | :--- |
| **GET** | `/api/bank/account/status` | `deposit_balance` と `loan_balance` を取得 |
| **GET** | `/api/bank/account/history/deposit` | 預金通帳の履歴を取得（`Deposit_Journal`） |
| **GET** | `/api/bank/account/history/loan` | ローンの推移履歴を取得（`Loan_Journal`） |
| **PATCH** | `/api/bank/account/history/print` | 記帳演出完了の通知 |

### 4.2 システム内部用 API（他サービス・バッチ向け）
| メソッド | エンドポイント | 説明 | 発生タイミング |
| :--- | :--- | :--- | :--- |
| **POST** | `/internal/bank/account/create` | 口座開設（初期値0） | ユーザー登録完了直後 |
| **POST** | `/internal/bank/account/initialize` | 初期ローン100万実行 | チュートリアル開始時 |
| **POST** | `/internal/bank/transaction/execute` | 汎用資金移動（売買・返済等） | 株取引・任意返済時 |
| **POST** | `/internal/bank/batch/interest` | 利息計算・反映（負債増） | ターン終了時 |
| **POST** | `/internal/bank/batch/reconcile` | 全口座整合性チェック実行 | 定期メンテナンス時 |