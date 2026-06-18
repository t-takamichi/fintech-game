## 概要

本 MR は、ゲームの口座管理システムにおける**「二重決済の防止」「同時アクセス時のレースコンディション解消」「バッチ実行時のメモリ負荷軽減」**を目的とした、BaaS（Banking-as-a-Service）基準の堅牢化リファクタリングおよび機能追加です。

これまでの暫定的なモック構造から、並行アクセスやリトライ耐性を備えた本格的な金融決済システム水準へ設計を強化し、さらに不要なDBアクセスや重複コードを排除してコードクオリティを高めました。また、リポジトリ層、サービス層、マッパー層、ハンドラ層の全レイヤーに詳細なユニットテストを追加し、テストカバレッジを大幅に強化して品質を検証済みです。

---

## 解決した課題 (Background & Issues)

1. **二重決済 (Double Spend) のリスク**
   - NW遅延や再送処理により同一取引リクエストが重複した場合、二重に引き落としが走ってしまう課題。
   - 利息計算バッチが多重実行された際に、過剰に利息が加算されてしまう課題。
2. **同時アクセス時のデータ競合 (Race Condition)**
   - 同一口座に対して、ミリ秒単位で取引処理やバッチ処理が重なった際、残高（キャッシュ）の更新値が上書きされ整合性が崩れる課題。
3. **スケーラビリティの限界（OOMの懸念）**
   - 利息加算や検算バッチにおいて、全ユーザーのデータをメモリ上に一括ロードしていたため、ユーザー規模拡大時にサーバーがメモリ不足（Out of Memory）でクラッシュする課題。
4. **アーキテクチャの密結合（ドメイン依存の流出）**
   - GORMのDB定義モデル（`entity.Transaction`）がAPIのレスポンス（大文字JSON）として直接Web層へ流出していた課題。
   - `main.go` にルーティングやDB初期化がベタ書きされ、コードの見通しが悪くなっていた課題。
5. **コードベースの冗長性・非一貫性と不要なルート**
   - トランザクション処理の直前・直後でDBから無駄にレコードを再フェッチしている無駄クエリの存在。
   - べき等キーの文字列パース処理などがハンドラ内に重複して存在し、コードの見通しが悪く、また自明なコメントが多く残されていた課題。
   - 大文字 `/Create` と小文字 `/create` の重複した余計なルーティング定義の存在。
   - リポジトリ層での一時変数への代入など、書き方の不揃いさ。
   - トランザクションヘルパー `withTx` 内での無駄な匿名ラッパー関数の存在。
   - 口座作成 (`CreateAccount`) にて、一時的なDB接続遮断によるエラーを「口座がない」と誤認して作成処理を強行し、整合性崩壊を招くバグ。
6. **返済取引 (`REPAYMENT`) 時の DB Enum 制約違反バグ (新規検出 & 修正)**
   - APIで `"REPAYMENT"` という取引タイプを指定した際、PostgreSQL 側の `transaction_type` Enum 制約（`LOAN`, `BUY`, `SELL`, `INTEREST`, `SETTLE`）に存在しないため、DB保存時に制約エラーとなるバグ。これまでは結合テストで返済が検証されておらず未検出でした。本MRで追加したユニットテストによって検出され、速やかに修正されました。

---

## 主な変更内容 (Key Changes)

### 1. べき等性（Idempotency）保証の実装
- **DBスキーマ更新**: `transactions` テーブルに一意制約付きの `idempotency_key` (UUID) カラムを追加（DBマイグレーション適用済）。
- **取引APIの二重処理防止**: リクエスト時にべき等キーを受け取り、登録済みのキーであれば取引更新を安全にスキップして前回の結果を返すよう改修。
- **決定論的バッチべき等キー**: 利息加算バッチにおいて、「UserID ＋ 現在のターン数」から生成した SHA-1 UUID をべき等キーとして利用し、バッチの多重実行から口座を保護。
- **DBストアド更新**: `fn_apply_transaction` 内でもべき等キーの重複チェックと早期リターンを行うよう更新。

### 2. SELECT FOR UPDATE による排他制御（同時実行制御）
- トランザクション開始直後に `GetMasterForUpdateTx` を用いて口座行の悲観的ロックを取得し、並行するリクエストが直列化されるように実装。

### 3. バッチ処理のスモールバッチ化（スケーラビリティ対策）
- `ApplyInterestBatch` および `ReconcileAccountsBatch` において、`Limit` と `Offset`（100件ずつ）を用いた分割ページネーションループ処理に変更。メモリ消費量を常に一定に維持。

### 4. DTOマッピングによる境界分離
- `domain.Transaction` を新たに定義し、[mapper.go](file:///Users/tanakakoushin/Documents/workspace/projects/2026/projects/fintech-game/backend/bank/internal/service/mapper.go) を通じて DB エンティティからドメインオブジェクトへマッピング。外部 API へのレスポンスを標準的な小文字의 JSON キーに統一。

### 5. 冗長なDBアクセス・コード重複・重複エンドポイントの排除とクリーンアップ（追加）
- **不要なDB再フェッチの排除**: 各サービスメソッドのトランザクション完了直前に実行していた `tx.First` での口座マスタ再ロード処理をすべて廃止。メモリ上でロード＆更新された状態の構造体を直接マッピングして返却するよう最適化し、無駄なSELECTクエリを削減。
- **DB接続エラー処理の厳密化 (バグ修正)**: `CreateAccount` 時、DBから口座マスターが引けなかった場合の接続エラーと `gorm.ErrRecordNotFound`（正常系）を明確に区別し、DBエラー時は直ちにエラーとして早期リターンするよう修正。
- **REPAYMENT 取引タイプのマッピング (バグ修正)**: `"REPAYMENT"` 取引要求時、ビジネスロジックでローン返済計算を行った後、DBの Enum 制約を満たすために内部的に `"LOAN"` タイプへマッピング変換して保存するよう修正。
- **withTx での無駄な匿名関数の排除**: [account_service.go](file:///Users/tanakakoushin/Documents/workspace/projects/2026/projects/fintech-game/backend/bank/internal/service/account_service.go) 内の `withTx` にて、GORMの `Transaction` メソッドにコールバック `fn` を直接渡すようにし、冗長な無駄クロージャアロケーションをカット。
- **べき等パースヘルパーの導入**: `Initialize` および `ExecuteTransaction` ハンドラ内に重複していた `idempotency_key` 文字列のパース・検証ロジックを、ハンドラ共通の `parseIdempotencyKey` ヘルパーメソッドとしてカプセル化。
- **重複ルートとテストURLのクリーンアップ**: 歴史的経緯で重複登録されていた大文字 `/Create` エンドポイントを [router.go](file:///Users/tanakakoushin/Documents/workspace/projects/2026/projects/fintech-game/backend/bank/internal/handler/router.go) から削除し、小文字の `/create` に一本化。これに伴い [test.py](file:///Users/tanakakoushin/Documents/workspace/projects/2026/projects/fintech-game/test/test.py) のテストリクエストURLも小文字へ修正。
- **リポジトリコードの統一とインライン化**: [account_repository.go](file:///Users/tanakakoushin/Documents/workspace/projects/2026/projects/fintech-game/backend/bank/internal/repository/account_repository.go) の `CreateMasterTx` 内の記述スタイルをインラインエラーチェックに統一し、[transaction_repository.go](file:///Users/tanakakoushin/Documents/workspace/projects/2026/projects/fintech-game/backend/bank/internal/repository/transaction_repository.go) の `MarkAsPrintedTx` もエラー発生の明示チェックを省いて直接 `Error` を返す形へ簡略化。
- **バッチページング時のソート順保証**: [account_repository.go](file:///Users/tanakakoushin/Documents/workspace/projects/2026/projects/fintech-game/backend/bank/internal/repository/account_repository.go) の `GetMastersPage` において、`Order("user_id ASC")` を明示的に指定。バッチ内でのデータ更新に伴いPostgreSQL内の物理順序が変わっても、Offsetページングがスキップや重複なく正確に行われるよう保証。
- **未使用定数の削除とマッピング整理**: 使用されていなかった `DebtThreshold` 定数を削除。また `GetAccountStatus` で行っていた struct 詰め替え処理を `mapper.go` へ統一。
- **コメントのクリーンアップ**: 不要なインラインコメントや自明なドキュメントコメントを全面的にクリーンアップし、セルフドキュメンティングなコードへ整理。

---

## 動作検証 (Verification)

1. **各レイヤーのユニットテスト (新規追加・全面網羅)**
   Go 標準の `testing` パッケージを使用し、テスト用DBを自動でクローズ・初期化して稼働する実機統合ユニットテストを追加しました。
   - `account_service_test.go` (サービス層)
     - `TestCreateAccount` (作成成功、重複エラー、バリデーションエラー)
     - `TestInitializeAccount` (初期ローン成功、べき等チェック、重複初期化防止)
     - `TestExecuteTransaction` (一般取引、べき等、ローン返済、過剰返済上限クリッピングの検証)
     - `TestSettleAccount` (精算処理、決定論的べき等スキップの検証)
     - `TestApplyInterestBatch` (利息バッチの多重実行保護と複利計算の検証)
     - `TestReconcileAccountsBatch` (残高不整合の検知と口座凍結の検証)
   - `account_repository_test.go` & `transaction_repository_test.go` & `account_balance_repositry_test.go` (リポジトリ層)
     - `TestAccountRepository` (CRUD、トランザクション内の更新、GetMastersPage のソート順検証)
     - `TestTransactionRepository` (取引履歴の登録、取得、記帳印字、金額総和、べき等キー取得の検証)
     - `TestAccountBalanceRepository` (口座残高の登録・更新処理の検証)
   - `mapper_test.go` (サービス/マッピング層)
     - `TestMapper` (entity から domain struct への各マッピングロジックの検証)
   - `account_handler_test.go` & `internal_bank_account_handler_test.go` & `error_test.go` (ハンドラ/WebAPI層)
     - `TestAccountHandler_GetAccountStatus` (ステータス取得の200/404レスポンス検証)
     - `TestAccountHandler_MarkAsPrinted` (記帳完了の204/404レスポンス検証)
     - `TestAccountHandler_GetAccountHistory` (取引履歴取得APIの200レスポンス検証)
     - `TestCustomHTTPErrorHandler` (一般的なエラー時の500エラーやEcho HTTPエラー時のステータスコードマッピング検証)
     - `TestInternalBankAccountHandler_Create` (内部API用口座作成の201/409レスポンス検証)
     - `TestInternalBankAccountHandler_Initialize` (初期ローンAPIの200レスポンス検証)
     - `TestInternalBankAccountHandler_ExecuteTransaction` (取引実行APIの200レスポンス検証)
     - `TestInternalBankAccountHandler_Batches` (利息加算・検算バッチ起動APIの検証)
     - `TestInternalBankAccountHandler_Settle` (口座最終精算APIの200レスポンスおよび信用スコア増減の検証)
   ```bash
   go test -v ./...
   ```
   **全テストケース PASS 確認済み。**

2. **自動統合検証スクリプトによるテスト**
   `./test/test_api_flow.sh` を実行し、以下のシナリオがすべてパスすることを確認しました。
   - 各種APIフロー（新規開設、初期ローン、株式購入、記帳演出の演出完了/204、最終精算と口座凍結、複利計算バッチ、整合性検算バッチ）の動作。
   - 同一べき等キーを用いた再送リクエストが無視され、二重引き落としされないアサーションチェック。

---

## 関連ドキュメント / タスク
- [task.md](file:///Users/tanakakoushin/Documents/workspace/projects/2026/projects/fintech-game/backend/bank/design/銀行/task.md) （全タスク完了済）
