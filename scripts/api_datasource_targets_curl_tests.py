#!/usr/bin/env python3
import os
import sys
import json
import time
import uuid
import subprocess

try:
    import jwt
except ImportError:
    print("Error: PyJWT is required. Please run: pip install PyJWT")
    sys.exit(1)

# ==========================================
# CONFIGURATION
# ==========================================
BASE_URL = os.environ.get("BASE_URL", "http://127.0.0.1:8080/api/v1")
SECRET_KEY = os.environ.get("JWT_SECRET_KEY", "smap-secret-key-at-least-32-chars-long")
ARTIFACTS_DIR = "./artifacts"

# ==========================================
# AUTH SETUP (JWT BYPASS)
# ==========================================
def generate_bypass_token():
    """Generates a valid JWT token bypassing middleware requirements."""
    payload = {
        "email": "qa@test.com",
        "role": "admin",
        "groups": [],
        "iss": "smap",
        "sub": str(uuid.uuid4()),  # User ID
        "exp": int(time.time()) + 3600,
        "iat": int(time.time()),
        "jti": str(uuid.uuid4())
    }
    # Gin middleware reads token from Cookie 'smap_auth_token' or 'Authorization: Bearer <token>'
    return jwt.encode(payload, SECRET_KEY, algorithm="HS256")

TOKEN = generate_bypass_token()

# ==========================================
# TEST RUNNER ENGINE
# ==========================================
class TestCase:
    def __init__(self, case_id, name, method, endpoint, payload=None, expected_status=200, is_infra_500=False, token=None):
        self.case_id = case_id
        self.name = name
        self.method = method
        self.endpoint = endpoint
        self.payload = payload
        self.expected_status = expected_status
        self.is_infra_500 = is_infra_500
        self.token = token if token is not None else TOKEN

def run_curl(test_case: TestCase):
    """Executes a curl command using subprocess, returns status code and response json."""
    url = f"{BASE_URL}{test_case.endpoint}"
    
    headers_file = f"{ARTIFACTS_DIR}/{test_case.case_id}_headers.txt"
    body_file = f"{ARTIFACTS_DIR}/{test_case.case_id}_body.json"

    cmd = ["curl", "-sS", "-D", headers_file, "-o", body_file, "-X", test_case.method, url]
    
    if test_case.token:
        cmd.extend(["-H", f"Authorization: Bearer {test_case.token}"])
        
    if test_case.payload is not None:
        cmd.extend(["-H", "Content-Type: application/json"])
        cmd.extend(["-d", json.dumps(test_case.payload)])
        
    try:
        subprocess.run(cmd, check=True, capture_output=True)
    except subprocess.CalledProcessError as e:
        print(f"[{test_case.case_id}] curl execution failed: {e.stderr.decode()}")
        return 0, None

    # Parse HTTP Status from headers file
    status_code = 0
    if os.path.exists(headers_file):
        with open(headers_file, "r") as f:
            lines = f.readlines()
            # Find the last HTTP/x.x line (to handle redirects/100 Continue)
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
created_ds_id = None
created_target_id = None
project_id = str(uuid.uuid4())

def add_test(*args, **kwargs):
    tests.append(TestCase(*args, **kwargs))

# ==========================================
# 1. AUTHENTICATION & AUTHORIZATION
# ==========================================
add_test("AUTH-01", "Missing Token", "GET", "/datasources", expected_status=401, token="")
add_test("AUTH-02", "Invalid Signature", "GET", "/datasources", expected_status=401, token="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.INVALID")
add_test("AUTH-03", "Expired Token", "GET", "/datasources", expected_status=401, token=jwt.encode({"sub":"test", "exp": int(time.time()) - 3600}, SECRET_KEY, algorithm="HS256"))

# ==========================================
# 3. DATASOURCE CREATE VALIDATION
# ==========================================
add_test("DS-C-01", "Missing project_id", "POST", "/datasources", payload={"name": "Test DS", "source_type": "TIKTOK"}, expected_status=400)
add_test("DS-C-02", "Missing name", "POST", "/datasources", payload={"project_id": "00000000-0000-0000-0000-000000000000", "source_type": "FACEBOOK"}, expected_status=400)
add_test("DS-C-03", "Missing source_type", "POST", "/datasources", payload={"project_id": project_id, "name": "Test"}, expected_status=400)
add_test("DS-C-04", "Invalid source_type", "POST", "/datasources", payload={"project_id": project_id, "name": "Test", "source_type": "INVALID"}, expected_status=400)
add_test("DS-C-05", "CRAWL DS without crawl mode", "POST", "/datasources", payload={"project_id": project_id, "name": "Test", "source_type": "TIKTOK"}, expected_status=400)

# ==========================================
# SETUP VALID DATASOURCE
# ==========================================
def setup_datasource():
    global created_ds_id
    print(">> Setting up Valid Datastore for target testing...")
    tc = TestCase("SETUP-DS", "Create Valid Datasource", "POST", "/datasources", 
                  payload={"project_id": project_id, "name": "QA Valid DS", "source_type": "TIKTOK", "source_category": "CRAWL", "crawl_mode": "NORMAL", "crawl_interval_minutes": 11})
    status, body = run_curl(tc)
    if status == 200 and body and "data" in body and "data_source" in body["data"]:
        created_ds_id = body["data"]["data_source"]["id"]
        print(f"   Created DS ID: {created_ds_id}")
    else:
        print(f"Failed to create DS: Status {status}, Body: {body}")
        sys.exit(1)

# ==========================================
# 3. TARGET CREATE VALIDATION
# ==========================================
def dynamic_target_tests():
    global created_target_id
    add_test("TG-C-01", "Missing values", "POST", f"/datasources/{created_ds_id}/targets/keywords", payload={"is_active": True}, expected_status=400)
    add_test("TG-C-02", "Empty values array", "POST", f"/datasources/{created_ds_id}/targets/keywords", payload={"values": []}, expected_status=400)
    
    # Valid Target Create
    add_test("TG-C-04", "Valid Target Create", "POST", f"/datasources/{created_ds_id}/targets/keywords", payload={"values": ["QA Test Keyword"], "is_active": True, "crawl_interval_minutes": 10}, expected_status=200)

# ==========================================
# RUN EXECUTOR
# ==========================================
os.makedirs(ARTIFACTS_DIR, exist_ok=True)
print("==========================================")
print(" QA AUTOMATION - STARTING TEST SUITE")
print("==========================================\n")

# Run Auth & Initial Validation
for tc in tests:
    status, body = run_curl(tc)
    passed = status == tc.expected_status
    if tc.expected_status < 500 and status >= 500 and not tc.is_infra_500:
        passed = False # ANY unexplained 5xx is a FAIL
    
    results.append((tc, passed, status))

# Setup
setup_datasource()
tests.clear() # Clear initial tests
dynamic_target_tests()

for tc in tests:
    status, body = run_curl(tc)
    passed = status == tc.expected_status
    if tc.expected_status < 500 and status >= 500 and not tc.is_infra_500:
        passed = False
    
    if tc.case_id == "TG-C-04" and status == 200 and body and "data" in body and "target" in body["data"]:
        created_target_id = body["data"]["target"]["id"]
        print(f"   Created Target ID: {created_target_id}")

    results.append((tc, passed, status))

# List Tests
tests.clear()
add_test("TG-L-01", "List Targets Default", "GET", f"/datasources/{created_ds_id}/targets", expected_status=200)
add_test("TG-L-02", "List Targets Filter KEYWORD", "GET", f"/datasources/{created_ds_id}/targets?target_type=KEYWORD", expected_status=200)
add_test("TG-L-03", "List Targets Filter INVALID", "GET", f"/datasources/{created_ds_id}/targets?target_type=INVALID", expected_status=400)

if created_target_id:
    # Update Tests
    add_test("TG-U-01", "Update valid Target", "PUT", f"/datasources/{created_ds_id}/targets/{created_target_id}", payload={"values": ["Updated QA Keyword"]}, expected_status=200)
    add_test("TG-U-02", "Update Target Negative Interval", "PUT", f"/datasources/{created_ds_id}/targets/{created_target_id}", payload={"crawl_interval_minutes": -5}, expected_status=400)
    # Delete Tests
    add_test("TG-D-01", "Delete Target", "DELETE", f"/datasources/{created_ds_id}/targets/{created_target_id}", expected_status=200)
    add_test("TG-D-02", "Delete Same Target Again (Idempotent 404/400)", "DELETE", f"/datasources/{created_ds_id}/targets/{created_target_id}", expected_status=400)

for tc in tests:
    status, body = run_curl(tc)
    passed = status == tc.expected_status
    if tc.expected_status < 500 and status >= 500 and not tc.is_infra_500:
        passed = False
    results.append((tc, passed, status))

print("\n==========================================")
print(" TEST SUMMARY")
print("==========================================")
total = len(results)
passed = sum(1 for r in results if r[1])
failed = total - passed

print(f"Total: {total} | Passed: {passed} | Failed: {failed}\n")

for tc, is_pass, status in results:
    mark = "[PASS]" if is_pass else "[FAIL]"
    if not is_pass:
        print(f"{mark} | [{tc.case_id}] {tc.name} | Exp: {tc.expected_status}, Got: {status}")
        print(f"       -> Details: ./artifacts/{tc.case_id}_body.json")
    else:
        print(f"{mark} | [{tc.case_id}] {tc.name}")
