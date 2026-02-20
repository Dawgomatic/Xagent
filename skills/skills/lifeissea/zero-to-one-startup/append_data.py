import json
import os

# Define the data to be added
data_map = {
    "eval_data/gov_programs_supplement.jsonl": [
        {
            "type": "TIPS",
            "source": "https://www.unicornfactory.co.kr/article/2025111014022099783",
            "title": "2026년 TIPS 개편안: 일반 트랙 단일화 및 지원금 상향",
            "content": "2026년부터 TIPS 프로그램이 '일반 트랙' 하나로 통합됨. 기존 딥테크/글로벌 트랙 폐지. 일반 트랙 지원금은 최대 5억원에서 8억원으로 상향. 운영사 선투자 요건은 수도권 2억원, 비수도권 1억원으로 변경. 졸업 기업은 '포스트 팁스(사업화 7억)' 또는 '딥테크 팁스(R&D 추가지원)' 중 선택 가능. 글로벌 진출 기업을 위한 '글로벌 팁스(해외 VC 100만불 투자 유치 기업 대상, 50억 지원)' 신설.",
            "year": "2026",
            "tags": ["TIPS", "개편", "지원금상향", "일반트랙", "글로벌팁스"]
        },
        {
            "type": "TIPS",
            "source": "https://llumos.co/collection/tips-2026-reform-5-changes",
            "title": "2026년 TIPS 대개편 5가지 핵심 변화",
            "content": "1. 트랙 통합: 일반/딥테크/글로벌 -> 일반트랙 단일화. 2. 지원금 증액: 최대 8억원(정부) + 2억원(민간) = 10억원 확보 가능. 3. 포스트TIPS 진화: 포스트TIPS(사업화) vs 딥테크TIPS(R&D) 선택 가능. 4. 글로벌TIPS 강화: 해외 VC 선투자 100만 달러 이상 기업 대상 최대 50억원 지원. 5. 지원 기업 수 확대: 520개사 -> 620개사.",
            "year": "2026",
            "tags": ["TIPS", "개편", "지원규모", "포스트TIPS", "딥테크TIPS"]
        },
        {
            "type": "창업도약패키지",
            "source": "https://www.nextunicorn.kr/insight/84f0a77ac6b126a3",
            "title": "2026년 창업도약패키지 모집 공고 분석",
            "content": "2026년 창업도약패키지는 일반형(300개사, 최대 2억), 딥테크 특화형(75개사, 최대 3억), 투자연계형(50개사, 최대 2억)으로 구분. 일반형 접수 마감 2026.02.13. 지방 우대 정책 강화(특별지원지역 자부담 10%). 3년 초과 7년 이내 기업 대상. 패스트트랙(초기창업패키지 우수 기업 등) 제도 운영.",
            "year": "2026",
            "tags": ["창업도약패키지", "도약기", "지원금", "자부담", "딥테크"]
        },
        {
            "type": "창업도약패키지",
            "source": "https://gongysd.com/startup-wiki/?idx=169695325&bmode=view",
            "title": "2026 창업도약패키지 가이드: 신청 및 평가 절차",
            "content": "서류평가(2배수 내외, 2026년 3월) -> 발표평가(2026년 3~4월) -> 최종선정(4월). 협약기간 10개월. 자기부담금은 현금 10% 필수. 주관기관 선택 중요(권역 내 선택 필수). BM 혁신 및 스케일업 전략 중요 평가.",
            "year": "2026",
            "tags": ["창업도약패키지", "평가절차", "일정", "주관기관"]
        },
        {
             "type": "TIPS",
             "source": "https://storiesheaven.tistory.com/entry/TIPS-%ED%94%84%EB%A1%9C%EA%B7%B8%EB%9E%A8-%EC%97%B0%EA%B3%84-%EC%A1%B0%EA%B1%B4%EA%B3%BC-%EC%A0%88%EC%B0%A8",
             "title": "TIPS 프로그램 연계 조건과 절차 (2025)",
             "content": "선 민간투자(운영사 1~2억) -> 후 정부지원 구조. 창업 7년 이내 법인. 운영사 추천 필수. 정부 R&D(5억) + 창업사업화(1억) + 해외마케팅(1억). 성공 판정 시 R&D 자금의 20% 기술료 납부(우수/보통). 실패 시 성실수행 평가 중요.",
             "year": "2025",
             "tags": ["TIPS", "절차", "기술료", "성실수행"]
        }
    ],
    "eval_data/tips_companies.jsonl": [
        {
            "type": "TIPS_Operator",
            "source": "https://besuccess.com/?p=177917",
            "title": "퓨처플레이, 2025 TIPS 우수운영사 선정",
            "content": "퓨처플레이가 2025년 TIPS 우수 운영사로 선정. 11년간 딥테크, 스케일업 팁스 등 운영. 글로벌 진출 지원 강화(글로벌 특화형 운영사). 2026년 제도 개편 대비 전 트랙 추천 역량 확보.",
            "year": "2025",
            "tags": ["TIPS운영사", "퓨처플레이", "우수운영사", "글로벌"]
        },
        {
            "type": "TIPS_Company",
            "source": "https://thevc.kr/smart-collections/67047ffccd63fa69c53f9de0",
            "title": "2025 딥테크 팁스 선정 기업 (일부)",
            "content": "페블스퀘어, 소서릭스코리아, 임팩티브에이아이 등 딥테크 팁스 선정. (출처: TheVC 리스트)",
            "year": "2025",
            "tags": ["딥테크팁스", "선정기업", "페블스퀘어", "소서릭스코리아", "임팩티브에이아이"]
        },
        {
             "type": "TIPS_Company",
             "source": "https://thevc.kr/smart-collections/67039de691caeea8d0e5c299",
             "title": "2025 일반형 팁스 선정 기업 (일부)",
             "content": "노벨티노빌리티, 클리카, 제클린 등 일반형 팁스 선정. (출처: TheVC 리스트)",
             "year": "2025",
             "tags": ["팁스", "선정기업", "노벨티노빌리티", "클리카", "제클린"]
        }
    ],
    "eval_data/vc_investment.jsonl": [
         {
            "type": "Investment_Trend",
            "source": "https://www.venturesquare.net/1029921/",
            "title": "2025년 벤처투자 결산 및 2026년 전망",
            "content": "2025년 비수도권 13개 스타트업 100억원 이상 투자 유치(에이바이오머티리얼즈 등). '승자독식' 구조 심화. 2026년은 글로벌 진출이 필수 생존 요건. BDC(기업성장집합투자기구) 도입 논의 활성화. M&A형 기업승계 특별법 논의.",
            "year": "2026",
            "tags": ["벤처투자", "트렌드", "비수도권", "BDC", "M&A"]
         }
    ]
}

# Write to files
base_dir = os.path.dirname(os.path.abspath(__file__))
for file_rel_path, entries in data_map.items():
    file_path = os.path.join(base_dir, file_rel_path)
    # Ensure directory exists
    os.makedirs(os.path.dirname(file_path), exist_ok=True)
    
    with open(file_path, 'a', encoding='utf-8') as f:
        for entry in entries:
            f.write(json.dumps(entry, ensure_ascii=False) + '\n')

print("Data append completed.")
