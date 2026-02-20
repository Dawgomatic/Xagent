import json
import os
import time
import glob

# Configuration
BASE_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "eval_data")
TARGET_FILES = {
    "tips": "tips_companies.jsonl",
    "gov": "gov_programs_supplement.jsonl",
    "criteria": "criteria_supplement.jsonl",
    "success": "success_cases_web.jsonl",
    "vc": "vc_investment.jsonl"
}

# Keywords
KEYWORDS = {
    "tips": [
        "TIPS 합격 후기 2025", "TIPS 합격 사업계획서", "TIPS 선정기업 2025", 
        "TIPS 심사기준 변경 2026", "TIPS 운영사 추천", "TIPS R&D 정산", 
        "팁스 합격 비결", "팁스 면접 후기", "TIPS 선정 후 진행", "팁스 기술사업화"
    ],
    "gov": [
        "창업도약패키지 2026 공고", "예비창업패키지 2026", "정부 R&D 과제 신청 방법", 
        "SBIR 한국", "중기부 지원사업 2026 일정", "정부과제 사업계획서 작성법", 
        "창업성장기술개발 2026", "글로벌 창업사관학교", "소부장 스타트업 100", 
        "민관공동투자"
    ],
    "criteria": [
        "정부과제 심사위원 평가기준", "사업계획서 심사 배점", "기술성 시장성 사업성 평가", 
        "창업지원 면접 질문", "정부과제 탈락 사유", "심사위원 인터뷰"
    ],
    "vc": [
        "스타트업 투자 트렌드 2025 2026", "시드 투자 유치 방법", "Pre-A 라운드 한국", 
        "벤처투자 현황 2025"
    ]
}

# Load existing URLs to avoid duplicates
existing_urls = set()
for filename in glob.glob(os.path.join(BASE_DIR, "*.jsonl")):
    with open(filename, 'r', encoding='utf-8') as f:
        for line in f:
            try:
                data = json.loads(line)
                if 'source' in data:
                    existing_urls.add(data['source'])
            except:
                pass

print(f"Loaded {len(existing_urls)} existing URLs.")

# Function to simulate the search and fetch process (since I can't call tools directly from python script in this env, 
# I will output the plan and let the agent execute it, or I can try to use the python script to generate the list of tasks).
# Actually, I will use this script to manage the 'state' and logic, but I need to execute tools.
# Since I am the agent, I will just loop through keywords manually in blocks to manage context and tool calls.
