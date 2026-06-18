import requests
import uuid

subject_id = str(uuid.uuid4())
print("subject_id:", subject_id)

url = "http://localhost:8080/internal/bank-accounts/Create"
payload = {
    "subject_id": subject_id,
    "initial_score": 3
}
resp = requests.post(url, json=payload, timeout=10)
print(resp.status_code)
print(resp.headers)
print(resp.text)