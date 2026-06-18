#!/bin/bash

# エラー時にスクリプトを終了
set -e

BASE_URL="http://localhost:8080"
# ランダムなユーザーIDを生成（現在のエポック秒を使用）
USER_ID="user-$(date +%s)"

echo "=== 1. 口座を新規作成します (ID: $USER_ID) ==="
CREATE_RES=$(curl -s -X POST "$BASE_URL/internal/bank-accounts/create" \
  -H "Content-Type: application/json" \
  -d "{\"subject_id\": \"$USER_ID\", \"initial_score\": 3}")
echo "Response: $CREATE_RES"
echo ""

echo "=== 2. 初期ローンを実行します ==="
INIT_RES=$(curl -s -X POST "$BASE_URL/internal/bank-accounts/initialize" \
  -H "Content-Type: application/json" \
  -d "{\"subject_id\": \"$USER_ID\"}")
echo "Response: $INIT_RES"
echo ""

echo "=== 3. 最初の取引履歴を取得します ==="
HISTORY_RES1=$(curl -s "$BASE_URL/api/bank/account/$USER_ID/history")
echo "Response: $HISTORY_RES1"
echo ""

# jq が入っていればそれを使用し、無ければ grep/sed で id を抽出
if command -v jq >/dev/null 2>&1; then
  LOAN_TX_ID=$(echo "$HISTORY_RES1" | jq '.[0].id')
else
  LOAN_TX_ID=$(echo "$HISTORY_RES1" | grep -o '"id":[0-9]*' | head -n 1 | cut -d':' -f2)
fi
echo "LOAN Transaction ID: $LOAN_TX_ID"
echo ""

echo "=== 4. 株式を購入します (手元現金-20万) ==="
BUY_RES=$(curl -s -X POST "$BASE_URL/internal/bank-accounts/transaction/execute" \
  -H "Content-Type: application/json" \
  -d "{\"subject_id\": \"$USER_ID\", \"amount\": -200000, \"type\": \"BUY\", \"description\": \"ロケット株購入\"}")
echo "Response: $BUY_RES"
echo ""

echo "=== 5. べき等性 (Idempotency) のテストを実行します ==="
# 新規UUIDを生成
if command -v uuidgen >/dev/null 2>&1; then
  IDEMPOTENCY_KEY=$(uuidgen | tr '[:upper:]' '[:lower:]')
else
  IDEMPOTENCY_KEY="d58434a9-45e0-49bf-811c-$(date +%s%N | cut -c 1-12)"
fi
echo "Generated Idempotency Key: $IDEMPOTENCY_KEY"

# 1回目の取引実行 (手元現金-10万)
RES_IDEMP1=$(curl -s -X POST "$BASE_URL/internal/bank-accounts/transaction/execute" \
  -H "Content-Type: application/json" \
  -d "{\"subject_id\": \"$USER_ID\", \"amount\": -100000, \"type\": \"BUY\", \"description\": \"べき等性テスト株\", \"idempotency_key\": \"$IDEMPOTENCY_KEY\"}")
echo "1st execution response: $RES_IDEMP1"

if command -v jq >/dev/null 2>&1; then
  BALANCE_BEFORE=$(echo "$RES_IDEMP1" | jq '.Balance')
else
  BALANCE_BEFORE=$(echo "$RES_IDEMP1" | grep -o '"Balance":[0-9]*' | head -n 1 | cut -d':' -f2)
fi

# 2回目の同一取引実行 (全く同じ idempotency_key)
RES_IDEMP2=$(curl -s -X POST "$BASE_URL/internal/bank-accounts/transaction/execute" \
  -H "Content-Type: application/json" \
  -d "{\"subject_id\": \"$USER_ID\", \"amount\": -100000, \"type\": \"BUY\", \"description\": \"べき等性テスト株\", \"idempotency_key\": \"$IDEMPOTENCY_KEY\"}")
echo "2nd execution response: $RES_IDEMP2"

if command -v jq >/dev/null 2>&1; then
  BALANCE_AFTER=$(echo "$RES_IDEMP2" | jq '.Balance')
else
  BALANCE_AFTER=$(echo "$RES_IDEMP2" | grep -o '"Balance":[0-9]*' | head -n 1 | cut -d':' -f2)
fi

# 二重引き落としされていないことの検証
if [ "$BALANCE_BEFORE" = "$BALANCE_AFTER" ]; then
  echo "=> SUCCESS: Idempotency is working perfectly! No duplicate balance reduction detected."
else
  echo "=> FAILURE: Idempotency failed! Balance was reduced twice (before: $BALANCE_BEFORE, after: $BALANCE_AFTER)."
  exit 1
fi
echo ""

echo "=== 6. 再度、取引履歴を取得します ==="
HISTORY_RES2=$(curl -s "$BASE_URL/api/bank/account/$USER_ID/history")
echo "Response: $HISTORY_RES2"
echo ""

if command -v jq >/dev/null 2>&1; then
  BUY_TX_ID=$(echo "$HISTORY_RES2" | jq '.[1].id')
  IDEMP_TX_ID=$(echo "$HISTORY_RES2" | jq '.[2].id')
else
  BUY_TX_ID=$(echo "$HISTORY_RES2" | grep -o '"id":[0-9]*' | head -n 2 | tail -n 1 | cut -d':' -f2)
  IDEMP_TX_ID=$(echo "$HISTORY_RES2" | grep -o '"id":[0-9]*' | head -n 3 | tail -n 1 | cut -d':' -f2)
fi
echo "BUY Transaction ID: $BUY_TX_ID"
echo "IDEMP Transaction ID: $IDEMP_TX_ID"
echo ""

echo "=== 7. 記帳演出を完了させます (ID: $LOAN_TX_ID, $BUY_TX_ID, $IDEMP_TX_ID) ==="
PRINT_RES=$(curl -s -w "%{http_code}" -o /dev/null -X PATCH "$BASE_URL/api/bank/account/$USER_ID/history/print" \
  -H "Content-Type: application/json" \
  -d "{\"ids\": [$LOAN_TX_ID, $BUY_TX_ID, $IDEMP_TX_ID]}")
echo "HTTP Status Code (Expect 204): $PRINT_RES"
echo ""

echo "=== 8. 利息加算バッチを実行します ==="
INTEREST_RES=$(curl -s -X POST "$BASE_URL/internal/bank-accounts/batch/interest")
echo "Response: $INTEREST_RES"
echo ""

echo "=== 9. 利息加算後の口座ステータスを確認します ==="
STATUS_RES1=$(curl -s "$BASE_URL/api/bank/account/$USER_ID/status")
echo "Response: $STATUS_RES1"
echo ""

echo "=== 10. 最終精算を実行します (残高不足で凍結されることを期待) ==="
SETTLE_RES=$(curl -s -X POST "$BASE_URL/internal/bank-accounts/settle" \
  -H "Content-Type: application/json" \
  -d "{\"subject_id\": \"$USER_ID\"}")
echo "Response: $SETTLE_RES"
echo ""

echo "=== 11. 精算後の口座ステータスを確認します ==="
STATUS_RES2=$(curl -s "$BASE_URL/api/bank/account/$USER_ID/status")
echo "Response: $STATUS_RES2"
echo ""

echo "=== 12. 整合性検算バッチを実行します ==="
RECONCILE_RES=$(curl -s -X POST "$BASE_URL/internal/bank-accounts/batch/reconcile")
echo "Response: $RECONCILE_RES"
echo ""

echo "=== すべてのテストフローが終了しました ==="
