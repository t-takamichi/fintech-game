# 銀行・普通口座システム 実装詳細仕様書（ライフサイクル重視版）

## 1. サービス設計の前提
- **責務の分離**: ユーザー基本情報は「Userサービス」が管理し、本サービスは「口座と金銭の流れ」のみを管理する。
- **ライフサイクル**: 
    1. **誕生**: 口座のみ作成（残高0, ローン0）。通帳は白紙。
    2. **着金**: 初期ローン実行。100万円が振り込まれ、通帳に1行目の履歴が刻まれる。

## 2. データベース構造（Database Schema）
整合性を保つため、更新は必ずトランザクション内で実行する。

### 2.1 Accounts（マスタ／残高の分離）
- 概要: マスタ情報と可変な残高情報を分離します。残高更新はロック粒度を下げるため `accounts_balance` に限定して行い、マスタ情報は `accounts_master` に保持します。

- `accounts_master`（口座の固定情報）
    - `user_id`: UUID (PK) - Userサービスと共通
    - `created_at`: Timestamp
    - `credit_score`: Integer (1-10。初期値 3)
    - `is_frozen`: Boolean (デフォルトFalse)
    - `current_turn`: Integer (0-2。現在の経済サイクル)

- `accounts_balance`（可変の残高情報）
    - `user_id`: UUID (PK, FK -> accounts_master.user_id)
    - `balance`: BigInt (現在の現金残高。初期値0。マイナス値を許容)
    - `loan_principal`: BigInt (初期借入元本。初期値0)
    - `updated_at`: Timestamp

注: 既存 `accounts` テーブルからの段階的な移行を想定しています（バックフィル→切替→旧テーブル削除）。

### 2.2 Transactionsテーブル（不変履歴・通帳データ）
**Update/Delete禁止（Insert Only）**。
- `id`: BigInt (PK)
- `user_id`: UUID (FK)
- `type`: Enum (LOAN, BUY, SELL, INTEREST, SETTLE)
- `amount`: BigInt (正：入金 / 負：出金)
- `balance_after`: BigInt (処理後の残高。通帳の「差し引き残高」欄用)
- `description`: String(15) (AI生成の摘要メッセージ)
- `is_printed`: Boolean (デフォルトFalse。フロント演出完了後にTrueへ)
- `created_at`: Timestamp

### 2.3 MarketBatchesテーブル（経済イベント管理）
- `batch_id`: Integer (PK)
- `news_summary`: Text (AI生成ニュース)
- `interest_rate`: Float (適用金利)

---

## 3. エンドポイント定義（API Endpoints）

### 3.1 プレイヤー用 API（フロントエンド向け）
| メソッド | エンドポイント | 説明 |
| :--- | :--- | :--- |
| **GET** | `/api/bank/account/:id/status` | 口座状況（純資産・借金フラグ等）を取得 |
| **GET** | `/api/bank/account/history` | 通帳の履歴リストを取得 |
| **PATCH** | `/api/bank/account/history/print` | 記帳演出（カチャカチャ音）完了の通知 |

### 3.2 システム内部用 API（他サービス・AIエンジン向け）
| メソッド | エンドポイント | 説明 | 発生タイミング |
| :--- | :--- | :--- | :--- |
| **POST** | `/internal/bank-accounts/create` | **口座のみ作成（残高0）** | ユーザー登録完了直後 |
| **POST** | `/internal/bank-accounts/initialize` | **初期ローン100万実行** | チュートリアル開始時 |
| **POST** | `/internal/bank-accounts/transaction` | 資金移動（売買・利息等） | 株式売買・バッチ処理時 |
| **POST** | `/internal/bank-accounts/settle` | 最終精算とスコア判定 | 6日目終了時 |

---

## 4. ロジック・算出ルール
- **純資産（net_asset）**: `balance - loan_principal`
- **借入フラグ（is_debt）**: `net_asset < 0`
- **資金反映（apply_transaction）**: `balance`を更新し、その結果（`balance_after`）を`Transactions`に記録。必ず同一トランザクションで処理する。