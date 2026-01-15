
## プロジェクト概要

このリポジトリは「銀行・普通口座」システムのバックエンド実装です。主な目的は以下のとおりです。

- プレイヤーの口座情報管理（`Accounts`）と不変な取引履歴（`Transactions`）の保存。
- バッチ処理やAIエンジンからの内部APIで口座を操作する仕組み（利息、ローン、精算など）。
- 開発者向けにローカルでPostgresを立ち上げてマイグレーションを適用できる環境を提供。

関連ドキュメント: `design/銀行/desing.md` に仕様とAPI設計が記載されています。

## ゲームシステム概要

このプロジェクトで実装するゲームの仕組み（簡潔）:

- プレイヤーは「口座」を持ち、`balance`（現金）と`loan_principal`（借入元本）を管理します。
- 初期化時に所定のローン（例: 1,000,000）が付与され、`Transactions` に LOAN レコードが挿入されてゲームが始まります。
- 市場変動や利息、売買は内部バッチ／AI により `apply_transaction` を通じて口座残高に反映され、履歴は `Transactions` に挿入（Insert-only）。
- 通帳表示では `is_printed` フラグを利用してフロント側で印字アニメーションを制御します。
- 精算フェーズ（2回目の市場変動後）で純資産がマイナスの場合、信用スコアが下がり凍結（ゲームオーバー）に繋がります。
- 目的は資産を増やして借入をカバーすること。負の `balance` は「返済義務」を意味します。

詳しい仕様・API フローは [design/銀行/desing.md](design/銀行/desing.md) を参照してください。


## ローカル開発用 Postgres とマイグレーション

簡単な使い方（Docker と Docker Compose が必要です）:

1) Postgres コンテナを起動します:

```bash
docker compose up -d db
```

2) `migrate` ツールで使う `DATABASE_URL` を環境変数に設定します（例）:

```bash
export DATABASE_URL="postgres://bank_user:bank_pass@localhost:5432/bank_db?sslmode=disable"
```

3) `migrate` コンテナを使ってマイグレーションを適用します（`./migrations` をマウントしています）:

```bash
docker compose run --rm migrate -path=/migrations -database "$DATABASE_URL" up
```

4) 新しい SQL を追加する手順:
- `migrations/` に `0004_description.up.sql` のようなファイルを追加します（`migrate` の命名規則に合わせる）。
- 追加後に上記のマイグレーションコマンドを再実行します。

注意事項:
- `docker-compose.yml` は `./migrations` を `db` と `migrate` の両方にマウントします。
- Amazon Aurora（DSQL / Data API）にデプロイする場合は、業務ロジックを `SECURITY DEFINER` 関数にまとめ、CI/CD からマイグレーションを実行する運用を推奨します。

Makefile の補助タスクと推奨ワークフロー
--------------------------------------

- DB を起動して準備が整うのを確認する:

```bash
docker compose up -d db
docker compose exec db pg_isready -U bank_user -d bank_db
```

- マイグレーションの適用（`Makefile` の `migrate-up` は `pg_isready` の待機ループを実行します）:

```bash
make migrate-up
```

- マイグレーションのバージョン確認:

```bash
make migrate-status
```

- DB コンテナの psql に入る:

```bash
make psql
```

- 新規マイグレーション雛形の作成（編集後に `make migrate-up` を実行）:

```bash
make new-migration NAME=add_some_table
```

接続コンテキストの注意:
- `Makefile` 経由で `migrate` をコンテナ内部から実行する場合、`db` サービス名（ホスト名 `db`）を使って接続します。ホスト側から実行する場合は `localhost` を使用してください。

Aurora にデプロイする場合:
- 業務ロジックを `SECURITY DEFINER` 関数にまとめ、CI/CD パイプラインからマイグレーションを実行する方法が安全で運用しやすいです。
