# AdjustBalanceTx 処理概要

## 概要
`AdjustBalanceTx` は口座残高を原子的に更新し、更新後の残高（balance_after）を返す低レイヤーのDB操作です。
この処理は必ずトランザクション内で実行し、同一トランザクション内で `CreateTransaction`（履歴挿入）を行うことで「残高更新＋履歴挿入」を原子性を保って実現します。

## 前提
- 対象口座（`accounts` の行）が存在すること。
- `userID` は UUID で一意に特定できること。
- `delta` は加算する金額（正の値で着金、負の値で出金）で `int64` を想定。
- この関数はトランザクション（`tx *gorm.DB`）を受け取る設計を推奨。呼び出し側がトランザクションを開始して渡すことで、`CreateTransaction` と組み合わせた原子処理が可能になる。

## 推奨シグネチャ（Go / GORM）
```
// tx は必須（nil を許さない）
func (r *gormAccountRepo) AdjustBalanceTx(tx *gorm.DB, userID uuid.UUID, delta int64) (balanceAfter int64, err error)
```

## 処理手順（詳細）
1. 引数検証: `userID` が有効か、`delta` がオーバーフローしないかの簡易チェック。
2. トランザクション確認: `tx` が nil ならエラー（呼び出し側でトランザクションを作ることを明確にするため）。
3. 行ロック取得: 対象アカウント行を SELECT ... FOR UPDATE で取得。
この処理は必ずトランザクション内で実行し、同一トランザクション内で `CreateTransaction`（履歴挿入）を行うことで「残高更新＋履歴挿入」を原子性を保って実現します。
4. 処理可否チェック:
   - `is_frozen == true` の場合はエラーを返す（凍結口座へは更新不可）。
8. 履歴挿入はこの関数内で直接行わず、呼び出し元（サービス層）で `CreateTransaction(tx, ...)` を同一 `tx` 上で実行することを想定する。

呼び出し側がトランザクションを開始して渡すことで、`CreateTransaction` と組み合わせた原子処理が可能になる。
- `SELECT FOR UPDATE` により同一行への並列更新はブロッキングされるが、デッドロックやシリアライズ失敗が発生する可能性あり。
- 推奨: 呼び出し側（サービス層）で短いリトライ（指数バックオフ、最大3回程度）を実装する。

8. 履歴挿入はこの関数内で直接行わず、呼び出し元（サービス層）で `CreateTransaction(tx, ...)` を同一 `tx` 上で実行することを想定する。
- 入力不正、行未存在、凍結、UPDATE失敗等はエラーを返す。
- トランザクションのコミット/ロールバックは呼び出し側が管理する（この関数は tx を受け取り、エラーを返すのみ）。

3. `repo.CreateTransaction(tx, tr)` を呼んで履歴を挿入（同一 tx 上）
- 正常系: delta=+1000 で balance が期待通り増える。
- 負値許容: delta=-500 で負残高に遷移可能であること（仕様が許す場合）。
if err := repo.CreateTransaction(tx, &tr); err != nil {
- 並列更新: 複数ゴルーチンで同時に `AdjustBalanceTx` を呼び、整合性が保たれること（統合テストで検証）。
- リトライ: デッドロック発生時にリトライで回復できること（模擬的にシリアライズ失敗を発生させるテスト）。

## 実装上の注意点
- `AdjustBalanceTx` は「残高の一貫性」を守る低レイヤー関数であり、ビジネスルール（スコア更新、精算判定、演出フラグ等）はサービス層で担当する。
- 履歴（`transactions`）の挿入は必ず同一トランザクション内で行うこと。そうしないと残高と履歴が不整合になる可能性がある。
- 監査や障害対応のため、更新前後の値をログに残す方がデバッグしやすい。

## トランザクション管理の責任（誰がやるか）
- トランザクションの開始、コミット、ロールバック、およびリトライのポリシーはサービス層（呼び出し元）が責任を持ちます。
- `AdjustBalanceTx` は既に開始された `tx *gorm.DB` を受け取り、その `tx` 上でのみ動作します。内部で `tx.Commit()` や `tx.Rollback()` を呼ぶべきではありません。
- サービス層の推奨フロー:
  1. `tx := db.Begin()`（または `db.Transaction(func(tx *gorm.DB) error { ... })` のようなパターン）でトランザクションを開始
  2. `repo.AdjustBalanceTx(tx, userID, delta)` を呼んで残高更新（行ロックを取得）
3. `repo.CreateTransaction(tx, tr)` を呼んで履歴を挿入（同一 tx 上）
  4. エラーがなければ `tx.Commit()`、エラー時は `tx.Rollback()` を行う
- リトライはサービス層で実装します（例: トランザクション失敗時に最大 N 回の指数バックオフで再試行）。
- 参考となるサービス側の擬似コード:

```go
// 例: サービス層でのトランザクション管理
tx := db.Begin()
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
        panic(r)
    }
}()

balanceAfter, err := repo.AdjustBalanceTx(tx, userID, delta)
if err != nil {
    tx.Rollback()
    return err
}

tr := entity.Transaction{UserID: userID, Amount: delta, BalanceAfter: balanceAfter, /* ... */}
if err := repo.CreateTransaction(tx, &tr); err != nil {
    tx.Rollback()
    return err
}

# AdjustBalanceTx 処理概要

## 概要
`AdjustBalanceTx` は口座残高を原子的に更新し、更新後の残高（balance_after）を返す低レイヤーのDB操作です。
この処理は必ずトランザクション内で実行し、同一トランザクション内で `CreateTransaction`（履歴挿入）を行うことで「残高更新＋履歴挿入」を原子性を保って実現します。

## 前提
- 対象口座（`accounts` の行）が存在すること。
- `userID` は UUID で一意に特定できること。
- `delta` は加算する金額（正の値で着金、負の値で出金）で `int64` を想定。
- この関数はトランザクション（`tx *gorm.DB`）を受け取る設計を推奨。呼び出し側がトランザクションを開始して渡すことで、`CreateTransaction` と組み合わせた原子処理が可能になる。

## 推奨シグネチャ（Go / GORM）
```
// tx は必須（nil を許さない）
func (r *gormAccountRepo) AdjustBalanceTx(tx *gorm.DB, userID uuid.UUID, delta int64) (balanceAfter int64, err error)
```

## 処理手順（詳細）
1. 引数検証: `userID` が有効か、`delta` がオーバーフローしないかの簡易チェック。
2. トランザクション確認: `tx` が nil ならエラー（呼び出し側でトランザクションを作ることを明確にするため）。
3. 行ロック取得: 対象アカウント行を SELECT ... FOR UPDATE で取得。
   - SQL 例: `SELECT balance, loan_principal, is_frozen FROM accounts WHERE user_id = $1 FOR UPDATE`
4. 処理可否チェック:
   - `is_frozen == true` の場合はエラーを返す（凍結口座へは更新不可）。
   - 必要に応じて利用者の制約（上限/下限）を確認。
5. 残高計算: `newBalance := currentBalance + delta`。
6. 更新: `UPDATE accounts SET balance = $1, updated_at = now() WHERE user_id = $2` を実行。
7. 戻り値: 更新後の `newBalance` を返す。
8. 履歴挿入はこの関数内で直接行わず、呼び出し元（サービス層）で `CreateTransaction(tx, ...)` を同一 `tx` 上で実行することを想定する。

## 競合・リトライ戦略
- `SELECT FOR UPDATE` により同一行への並列更新はブロッキングされるが、デッドロックやシリアライズ失敗が発生する可能性あり。
- 推奨: 呼び出し側（サービス層）で短いリトライ（指数バックオフ、最大3回程度）を実装する。

## エラー処理
- 入力不正、行未存在、凍結、UPDATE失敗等はエラーを返す。
- トランザクションのコミット/ロールバックは呼び出し側が管理する（この関数は tx を受け取り、エラーを返すのみ）。

## テストケース（推奨）
- 正常系: delta=+1000 で balance が期待通り増える。
- 負値許容: delta=-500 で負残高に遷移可能であること（仕様が許す場合）。
- 凍結口座: `is_frozen=true` の場合はエラー。
- 並列更新: 複数ゴルーチンで同時に `AdjustBalanceTx` を呼び、整合性が保たれること（統合テストで検証）。
- リトライ: デッドロック発生時にリトライで回復できること（模擬的にシリアライズ失敗を発生させるテスト）。

## 実装上の注意点
- `AdjustBalanceTx` は「残高の一貫性」を守る低レイヤー関数であり、ビジネスルール（スコア更新、精算判定、演出フラグ等）はサービス層で担当する。
- 履歴（`transactions`）の挿入は必ず同一トランザクション内で行うこと。そうしないと残高と履歴が不整合になる可能性がある。
- 監査や障害対応のため、更新前後の値をログに残す方がデバッグしやすい。

## トランザクション管理の責任（誰がやるか）
- トランザクションの開始、コミット、ロールバック、およびリトライのポリシーはサービス層（呼び出し元）が責任を持ちます。
- `AdjustBalanceTx` は既に開始された `tx *gorm.DB` を受け取り、その `tx` 上でのみ動作します。内部で `tx.Commit()` や `tx.Rollback()` を呼ぶべきではありません。
- サービス層の推奨フロー:
  1. `tx := db.Begin()`（または `db.Transaction(func(tx *gorm.DB) error { ... })` のようなパターン）でトランザクションを開始
  2. `repo.AdjustBalanceTx(tx, userID, delta)` を呼んで残高更新（行ロックを取得）
  3. `repo.CreateTransaction(tx, tr)` を呼んで履歴を挿入（同一 tx 上）
  4. エラーがなければ `tx.Commit()`、エラー時は `tx.Rollback()` を行う
- リトライはサービス層で実装します（例: トランザクション失敗時に最大 N 回の指数バックオフで再試行）。
- 参考となるサービス側の擬似コード:

```
// 例: サービス層でのトランザクション管理
tx := db.Begin()
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
        panic(r)
    }
}()

balanceAfter, err := repo.AdjustBalanceTx(tx, userID, delta)
if err != nil {
    tx.Rollback()
    return err
}

tr := entity.Transaction{UserID: userID, Amount: delta, BalanceAfter: balanceAfter, /* ... */}
if err := repo.CreateTransaction(tx, &tr); err != nil {
    tx.Rollback()
    return err
}

if err := tx.Commit().Error; err != nil {
    return err
}
```

## 簡易コード例（擬似）
```
func (r *gormAccountRepo) AdjustBalanceTx(tx *gorm.DB, userID uuid.UUID, delta int64) (int64, error) {
    var acc entity.Account
    if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&acc).Error; err != nil {
        return 0, err
    }
    if acc.IsFrozen {
        return 0, ErrAccountFrozen
    }
    newBalance := acc.Balance + delta
    if err := tx.Model(&entity.Account{}).Where("user_id = ?", userID).Updates(map[string]interface{}{"balance": newBalance}).Error; err != nil {
        return 0, err
    }
    return newBalance, nil
}
```

---

このドキュメントは実装時のガイドラインです。実装の際はリトライ方針やエラーマッピング（アプリ内エラー型）をプロジェクト標準に合わせてください。
