import os
import sys
import json
import time
import uuid
import subprocess

try:
    import jwt
except ImportError:
    print("Error: PyJWT is required.")
    sys.exit(1)

BASE_URL = os.environ.get("BASE_URL", "http://127.0.0.1:8080/api/v1")
SECRET_KEY = os.environ.get("JWT_SECRET_KEY", "smap-secret-key-at-least-32-chars-long")
ARTIFACTS_DIR = "./artifacts"

def generate_bypass_token():
    payload = {
        "email": "qa@test.com",
        "role": "admin",
        "groups": [],
        "iss": "smap",
        "sub": str(uuid.uuid4()),
        "exp": int(time.time()) + 3600,
        "iat": int(time.time()),
        "jti": str(uuid.uuid4())
    }
    return jwt.encode(payload, SECRET_KEY, algorithm="HS256")

TOKEN = generate_bypass_token()

class TestCase:
    def __init__(self, case_id, name, method, endpoint, payload=None, expected_status=200, is_infra_500=False, token=None, is_internal=False):
        self.case_id = case_id
        self.name = name
        self.method = method
        self.endpoint = endpoint
        self.payload = payload
        self.expected_status = expected_status
        self.is_infra_500 = is_infra_500
        self.is_internal = is_internal
        self.token = token if token is not None else TOKEN

def run_curl(test_case: TestCase):
    url = f"{BASE_URL}{test_case.endpoint}"
    
    headers_file = f"{ARTIFACTS_DIR}/{test_case.case_id}_headers.txt"
    body_file = f"{ARTIFACTS_DIR}/{test_case.case_id}_body.json"

    cmd = ["curl", "-sS", "-D", headers_file, "-o", body_file, "-X", test_case.method, url]
    if test_case.is_internal:
        cmd.extend(["-H", "X-Internal-Key: ingest-service-key"])
    elif test_case.token:
        cmd.extend(["-H", f"Authorization: Bearer {test_case.token}"])
    if test_case.payload is not None:
        cmd.extend(["-H", "Content-Type: application/json"])
        cmd.extend(["-d", json.dumps(test_case.payload)])
        
    try:
        subprocess.run(cmd, check=True, capture_output=True)
    except subprocess.CalledProcessError as e:
        print(f"[{test_case.case_id}] curl execution failed: {e.stderr.decode()}")
        return 0, None

    status_code = 0
    if os.path.exists(headers_file):
        with open(headers_file, "r") as f:
            lines = f.readlines()
            for line in reversed(lines):
                if line.startswith("HTTP/"):
                    parts = line.split()
                    if len(parts) >= 2:
                        status_code = int(parts[1])
                    break
    
    body = None
    if os.path.exists(body_file):
        with open(body_file, "r") as f:
            try:
                body = json.load(f)
            except json.JSONDecodeError:
                f.seek(0)
                body = f.read()

    return status_code, body

tests = []
results = []
project_id = str(uuid.uuid4())
created_ds_id = None
created_target_id = None

def add_test(*args, **kwargs):
    tests.append(TestCase(*args, **kwargs))

os.makedirs(ARTIFACTS_DIR, exist_ok=True)

print(">> Setting up Data Source and Target...")
tc_ds = TestCase("SETUP-DS", "Setup DS", "POST", "/datasources", payload={"project_id": project_id, "name": "Phase 2 Setup", "source_type": "TIKTOK", "source_category": "CRAWL", "crawl_mode": "NORMAL", "crawl_interval_minutes": 10})
status_ds, body_ds = run_curl(tc_ds)
if status_ds == 200 and isinstance(body_ds, dict):
    if "data" in body_ds and "data_source" in body_ds["data"]:
        created_ds_id = body_ds["data"]["data_source"]["id"]
    elif "data_source" in body_ds:
        created_ds_id = body_ds["data_source"]["id"]

if not created_ds_id:
    print(f"Setup Failed. Status: {status_ds}, Body: {body_ds}")
    sys.exit(1)

tc_tg = TestCase("SETUP-TG", "Setup Target", "POST", f"/datasources/{created_ds_id}/targets/keywords", payload={"values": ["test_phase2"], "is_active": True, "crawl_interval_minutes": 10})
status_tg, body_tg = run_curl(tc_tg)
if status_tg == 200 and isinstance(body_tg, dict):
    if "data" in body_tg and "target" in body_tg["data"]:
        created_target_id = body_tg["data"]["target"]["id"]
    elif "target" in body_tg:
        created_target_id = body_tg["target"]["id"]

if not created_target_id:
    print(f"Setup Target Failed. Status: {status_tg}, Body: {body_tg}")
    sys.exit(1)

# phase 2 endpoints tests
add_test("CM-01", "Update CrawlMode (Fails due to PENDING status)", "PUT", f"/ingest/datasources/{created_ds_id}/crawl-mode", payload={"crawl_mode": "CRISIS", "trigger_type": "MANUAL", "reason": "Test"}, expected_status=400, is_internal=True)
add_test("CM-02", "Update CrawlMode Invalid Mode", "PUT", f"/ingest/datasources/{created_ds_id}/crawl-mode", payload={"crawl_mode": "INVALID", "trigger_type": "MANUAL"}, expected_status=400, is_internal=True)
add_test("DR-01", "Trigger Dryrun without Target", "POST", f"/datasources/{created_ds_id}/dryrun", payload={"force": False}, expected_status=400)
add_test("DR-02", "Trigger Dryrun with Target", "POST", f"/datasources/{created_ds_id}/dryrun", payload={"target_id": created_target_id})
add_test("DR-03", "Get Latest Dryrun", "GET", f"/datasources/{created_ds_id}/dryrun/latest")
add_test("DR-04", "List Dryrun History", "GET", f"/datasources/{created_ds_id}/dryrun/history")

print("==========================================")
print(" QA AUTOMATION - PHASE 2 (CrawlMode, Dryrun)")
print("==========================================\n")

for tc in tests:
    status, body = run_curl(tc)
    passed = status == tc.expected_status
    results.append((tc, passed, status))

print("\n==========================================")
print(" TEST SUMMARY")
print("==========================================")
total = len(results)
passed_count = sum(1 for r in results if r[1])
failed_count = total - passed_count

print(f"Total: {total} | Passed: {passed_count} | Failed: {failed_count}\n")

for tc, is_pass, status in results:
    mark = "[PASS]" if is_pass else "[FAIL]"
    if not is_pass:
        print(f"{mark} | [{tc.case_id}] {tc.name} | Exp: {tc.expected_status}, Got: {status}")
        print(f"       -> Details: {ARTIFACTS_DIR}/{tc.case_id}_body.json")
    else:
        print(f"{mark} | [{tc.case_id}] {tc.name}")
